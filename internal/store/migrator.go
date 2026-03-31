package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/w6xian/sidecar/internal/version"

	"github.com/pkg/errors"
	"github.com/w6xian/sqlm"
)

//go:embed migration
var migrationFS embed.FS

//go:embed seed
var seedFS embed.FS

const (
	// MigrateFileNameSplit is the split character between the patch version and the description in the migration file name.
	// For example, "1__create_table.sql".
	MigrateFileNameSplit = "__"
	LatestSchemaFileName = "LATEST.sql"
)

// Migrate applies the latest schema to the database.
func (s *Store) Migrate(ctx context.Context) error {
	if err := s.preMigrate(ctx); err != nil {
		return errors.Wrap(err, "failed to pre-migrate")
	}
	// 数据比较
	switch s.profile.Mode {
	case "prod":
		currentSchemaVersion, err := s.GetCurrentSchemaVersion()
		if err != nil {
			return errors.Wrap(err, "failed to get current schema version")
		}
		schemaVersion := version.GetCurrentVersion(s.profile.Mode)
		if version.IsVersionGreaterThan(schemaVersion, currentSchemaVersion) {
			slog.Error("cannot downgrade schema version",
				slog.String("databaseVersion", schemaVersion),
				slog.String("currentVersion", currentSchemaVersion),
			)
			return errors.Errorf("cannot downgrade schema version from %s to %s", schemaVersion, currentSchemaVersion)
		}
		if version.IsVersionGreaterThan(currentSchemaVersion, schemaVersion) {
			filePaths, err := fs.Glob(migrationFS, fmt.Sprintf("%s*/*.sql", s.getMigrationBasePath()))
			if err != nil {
				return errors.Wrap(err, "failed to read migration files")
			}
			sort.Strings(filePaths)

			db := sqlm.Major(ctx)
			defer db.Close()
			db.Action(func(tx sqlm.ITable, args ...interface{}) (int64, error) {
				slog.Info("start migration", slog.String("currentSchemaVersion", schemaVersion), slog.String("targetSchemaVersion", currentSchemaVersion))
				for _, filePath := range filePaths {
					fileSchemaVersion, err := s.getSchemaVersionOfMigrateScript(filePath)
					if err != nil {
						return 0, errors.Wrap(err, "failed to get schema version of migrate script")
					}
					if version.IsVersionGreaterThan(fileSchemaVersion, schemaVersion) && version.IsVersionGreaterOrEqualThan(currentSchemaVersion, fileSchemaVersion) {
						bytes, err := migrationFS.ReadFile(filePath)
						if err != nil {
							return 0, errors.Wrapf(err, "failed to read minor version migration file: %s", filePath)
						}
						stmt := string(bytes)
						tx.Exec(stmt)
					}
				}
				return 0, nil
			})
			slog.Info("end migrate")
			if err := s.updateCurrentSchemaVersion(ctx, currentSchemaVersion); err != nil {
				return errors.Wrap(err, "failed to update current schema version")
			}
		}
	case "demo":
		// In demo mode, we should seed the database.
		if err := s.seed(ctx); err != nil {
			return errors.Wrap(err, "failed to seed")
		}
	}
	return nil
}

func (s *Store) preMigrate(ctx context.Context) error {

	filePath := s.getMigrationBasePath() + LatestSchemaFileName
	bytes, err := migrationFS.ReadFile(filePath)
	if err != nil {
		return errors.Errorf("failed to read latest schema file: %s", err)
	}

	db := sqlm.Major(ctx)
	defer db.Close()
	db.Action(func(tx sqlm.ITable, args ...interface{}) (int64, error) {
		if _, err := tx.Exec(string(bytes)); err != nil {
			return 0, errors.Wrapf(err, "failed to execute SQL file %s", filePath)
		}
		return 0, nil
	})

	if s.profile.Mode == "prod" {
		// Upsert current schema version to database.
		schemaVersion, err := s.GetCurrentSchemaVersion()
		if err != nil {
			return errors.Wrap(err, "failed to get current schema version")
		}
		if err := s.updateCurrentSchemaVersion(ctx, schemaVersion); err != nil {
			return errors.Wrap(err, "failed to update current schema version")
		}
	}
	return nil
}

func (s *Store) updateCurrentSchemaVersion(ctx context.Context, schemaVersion string) error {

	return nil
}

func (s *Store) getMigrationBasePath() string {
	return fmt.Sprintf("migration/%s/", s.profile.Store.Protocol)
}

func (s *Store) getSeedBasePath() string {
	return fmt.Sprintf("seed/%s/", s.profile.Store.Protocol)
}

func (s *Store) seed(ctx context.Context) error {
	// Only seed for SQLite.
	if s.profile.Store.Protocol != "sqlite" {
		slog.Warn("seed is only supported for SQLite")
		return nil
	}

	filenames, err := fs.Glob(seedFS, fmt.Sprintf("%s*.sql", s.getSeedBasePath()))
	if err != nil {
		return errors.Wrap(err, "failed to read seed files")
	}

	sort.Strings(filenames)
	// 获取新的数据操作链接
	db := sqlm.Major(ctx)
	defer db.Close()
	db.Action(func(tx sqlm.ITable, args ...interface{}) (int64, error) {
		for _, filename := range filenames {
			bytes, err := seedFS.ReadFile(filename)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to read seed file, filename=%s", filename)
			}
			if _, err := tx.Exec(string(bytes)); err != nil {
				return 0, errors.Wrapf(err, "seed error: %s", filename)
			}
			return 0, nil
		}
		return 0, nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	return nil
}

func (s *Store) GetCurrentSchemaVersion() (string, error) {
	currentVersion := version.GetCurrentVersion(s.profile.Mode)
	minorVersion := version.GetMinorVersion(currentVersion)
	filePaths, err := fs.Glob(migrationFS, fmt.Sprintf("%s%s/*.sql", s.getMigrationBasePath(), minorVersion))
	if err != nil {
		return "", errors.Wrap(err, "failed to read migration files")
	}

	sort.Strings(filePaths)
	if len(filePaths) == 0 {
		return fmt.Sprintf("%s.0", minorVersion), nil
	}
	return s.getSchemaVersionOfMigrateScript(filePaths[len(filePaths)-1])
}

func (s *Store) getSchemaVersionOfMigrateScript(filePath string) (string, error) {
	// If the file is the latest schema file, return the current schema version.
	if strings.HasSuffix(filePath, LatestSchemaFileName) {
		return s.GetCurrentSchemaVersion()
	}

	normalizedPath := filepath.ToSlash(filePath)
	elements := strings.Split(normalizedPath, "/")
	if len(elements) < 2 {
		return "", errors.Errorf("invalid file path: %s", filePath)
	}
	minorVersion := elements[len(elements)-2]
	rawPatchVersion := strings.Split(elements[len(elements)-1], MigrateFileNameSplit)[0]
	patchVersion, err := strconv.Atoi(rawPatchVersion)
	if err != nil {
		return "", errors.Wrapf(err, "failed to convert patch version to int: %s", rawPatchVersion)
	}
	return fmt.Sprintf("%s.%d", minorVersion, patchVersion+1), nil
}
