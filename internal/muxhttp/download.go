package muxhttp

import (
	"context"
	"os"
	"path/filepath"

	"github.com/w6xian/sidecar/internal/utils/got"
)

func Download(ctx context.Context, uri, storeFile string) error {
	// 文件夹
	folderPath, _ := filepath.Split(storeFile)
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		if mkErr := os.Mkdir(folderPath, os.ModePerm); mkErr != nil {
			return mkErr
		}
		if chErr := os.Chmod(folderPath, os.ModePerm); chErr != nil {
			return chErr
		}
	}

	g := got.NewWithContext(ctx)
	if err := g.Download(uri, storeFile); err != nil {
		return err
	}
	return nil
}
