package sqlite

import (
	"context"
	"os"
	"sync"

	"github.com/w6xian/sidecar/internal/config"
	"github.com/w6xian/sidecar/internal/server/router"
	dbStore "github.com/w6xian/sidecar/internal/store"
	"github.com/w6xian/sidecar/internal/utils"

	"github.com/pkg/errors"
	"github.com/w6xian/sqlm"
	"github.com/w6xian/sqlm/store"
)

const ACTION_KEY = "action"

func DBId(ctx context.Context) int64 {
	v := ctx.Value(router.GetRequestIdKey())
	id, ok := v.(int64)
	if !ok {
		return 0
	}
	return id
}

type DB struct {
	db     *sqlm.Db
	option *config.Profile
	mu     sync.Mutex
	mapId  map[int64]*sqlm.Db
	pool   sync.Pool
}

func (s *DB) Lock() bool {
	return s.mu.TryLock()
}
func (s *DB) Unlock() {
	s.mu.Unlock()
}

// NewDB opens a database specified by its database driver name and a
// driver-specific data source name, usually consisting of at least a
// database name and connection information.
func NewDB(option *config.Profile) (dbStore.Driver, error) {
	// Ensure a DSN is set before attempting to open the database.
	if option.Store.DSN == "" {
		return nil, errors.New("dsn required")
	}
	if ok := regSqlm(option.Store, sqlm.DEFAULT_KEY); !ok {
		return nil, errors.New("register sqlm failed")
	}
	if ok := regSqlm(option.Store, ACTION_KEY); !ok {
		return nil, errors.New("register sqlm failed")
	}
	return &DB{
		option: option,
		mapId:  map[int64]*sqlm.Db{},
		pool: sync.Pool{
			New: func() any {
				return sqlm.Major(context.Background())
			},
		},
	}, nil
}

func regSqlm(opt *sqlm.Server, key string) bool {
	// 数据库
	storeOpt := sqlm.Server{}
	utils.Copy(&storeOpt, opt)
	storeDir, err := sqlm.NewOptionsWithServer(storeOpt, key)
	if err != nil {
		os.Exit(0)
	}
	// 使用store
	store, err := store.NewDriver(storeDir)
	if err != nil {
		os.Exit(0)
	}

	return sqlm.Use(store)
}

func checkConnection(d *sqlm.Db) error {
	if db, err := d.Conn(); err == nil {
		if err = db.Ping(); err == nil {
			return nil
		}
	}
	return errors.Errorf("check connection failed")
}
func (d *DB) getInstance(id int64) *sqlm.Db {
	db := d.pool.Get().(*sqlm.Db)
	if err := checkConnection(db); err == nil {
		d.storeId(id, db)
		return db
	}
	// Close the bad connection
	if conn, err := db.Conn(); err == nil {
		conn.Close()
	}
	db = sqlm.Major(context.Background())
	d.storeId(id, db)
	return db
}
func (d *DB) GetConnect(ctx context.Context) context.Context {
	id := DBId(ctx)
	d.getInstance(id)
	return ctx
}

func (d *DB) CloseConnect(ctx context.Context) error {
	id := DBId(ctx)
	d.mu.Lock()
	db, ok := d.mapId[id]
	delete(d.mapId, id)
	d.mu.Unlock()

	if ok && db != nil {
		if err := checkConnection(db); err == nil {
			d.pool.Put(db)
		} else {
			if conn, err := db.Conn(); err == nil {
				conn.Close()
			}
		}
	}
	ctx.Done()
	return nil
}

func (d *DB) GetDb(id int64) *sqlm.Db {
	d.mu.Lock()
	db, ok := d.mapId[id]
	d.mu.Unlock()
	if ok {
		return db
	}
	db = d.getInstance(id)
	return db
}

func (d *DB) storeId(id int64, db *sqlm.Db) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, v := range d.mapId {
		if v == db {
			return
		}
	}
	d.mapId[id] = db
}

func (d *DB) GetAction(ctx context.Context) *sqlm.Db {
	db := d.pool.Get().(*sqlm.Db)
	if err := checkConnection(db); err == nil {
		return db
	}
	if conn, err := db.Conn(); err == nil {
		conn.Close()
	}
	return sqlm.Major(context.Background())
}

func (d *DB) Close() error {
	if d.db != nil {
		d.db.Close()
	}
	d.db = nil
	return nil
}

func (d *DB) IsInitialized(ctx context.Context) (bool, error) {
	// Check if the database is initialized by checking if the memo table exists.
	return true, nil
}

func (d *DB) GetLink(ctx context.Context) sqlm.ITable {
	id := DBId(ctx)
	link := d.GetDb(id)
	return link
}
