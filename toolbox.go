package sqlitetoolbox

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"reflect"

	"github.com/mattn/go-sqlite3"
)

type Toolbox struct {
	db     *sql.DB
	config *initConfig
}

func (t *Toolbox) Do(cb DoFn) error {
	return cb(t.db)
}

type findConfig struct {
}

type findOptions func(*findConfig) error

func FindMiddleware[T any](t *Toolbox, opts ...findOptions) (*T, error) {
	config := &findConfig{}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, err
		}
	}

	for _, middleware := range t.config.middlewareManager.middlewares {
		if reflect.TypeOf(middleware) == reflect.TypeFor[*T]() {
			var inter interface{} = middleware
			base := inter.(*T)
			return base, nil
		}
	}

	return nil, fmt.Errorf("middleware not found")
}

// func (t *Toolbox) FindMiddleware(middlewareType reflect.Type) (Middleware, error) {

// 	for _, middleware := range t.config.middlewareManager.middlewares {
// 		if reflect.TypeOf(middleware) == middlewareType {
// 			return middleware, nil
// 		}
// 	}

// 	return nil, fmt.Errorf("middleware not found")
// }

type initConfig struct {
	db                *dbConfig
	middlewareManager *MiddlewareManager
}

type initOpts func(*initConfig) error

func WithDBConfig(opts ...dbOption) initOpts {
	return func(config *initConfig) error {
		config.db = NewSettingConfig(opts...)
		return nil
	}
}

func WithDBMemory() initOpts {
	return func(config *initConfig) error {
		config.db = NewSettingConfig(
			DBWithMode(Memory),
		)
		return nil
	}
}

func WithMiddleware(middleware Middleware) initOpts {
	return func(ic *initConfig) error {
		if reflect.TypeOf(middleware).Kind() != reflect.Ptr {
			return fmt.Errorf("middleware must be a pointer to a struct")
		}
		ic.middlewareManager.Register(middleware)
		return nil
	}
}

func New(opts ...initOpts) (*Toolbox, error) {

	config := &initConfig{
		middlewareManager: &MiddlewareManager{},
	}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, err
		}
	}

	var err error

	var connectionString string
	if connectionString, err = ConnectionString(config.db); err != nil {
		return nil, err
	}

	slog.Info("connection string ", slog.String("value", connectionString))

	sql.Register(
		config.db.name,
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				// register callback
				conn.RegisterUpdateHook(
					func(op int, db string, table string, rowid int64) {

						// TODO @droman: i need a tool to unmarshall the rows from `conn` to have match the sql.DB api

						switch op {

						case sqlite3.SQLITE_INSERT:
							if err := config.middlewareManager.RunOnInsert(conn, db, table, rowid); err != nil {
								// TODO: Handle error - i have no idea how
								slog.Error("SQLITE_INSERT error: %v", err)
							}

						case sqlite3.SQLITE_UPDATE:
							if err := config.middlewareManager.RunOnUpdate(conn, db, table, rowid); err != nil {
								// TODO: Handle error - i have no idea how
								slog.Error("SQLITE_UPDATE error: %v", err)
							}

						case sqlite3.SQLITE_DELETE:
							if err := config.middlewareManager.RunOnDelete(conn, db, table, rowid); err != nil {
								// TODO: Handle error - i have no idea how
								slog.Error("SQLITE_DELETE error: %v", err)
							}

						}
					},
				)

				return nil
			}})

	var db *sql.DB
	if db, err = sql.Open(config.db.name, connectionString); err != nil {
		return nil, err
	}

	// TODO @droman: might only enable that for `memory` mode?
	db.SetMaxOpenConns(1)

	if err = db.Ping(); err != nil {
		return nil, err
	}

	// Initialize middlewares
	if err := config.middlewareManager.RunOnInit(db); err != nil {
		return nil, err
	}

	toolbox := &Toolbox{
		db:     db,
		config: config,
	}

	return toolbox, nil
}

func (t *Toolbox) Close() error {

	if t.db != nil {

		// Close middlewares
		if err := t.config.middlewareManager.RunOnClose(t.db); err != nil {
			return err
		}

		// Check if the file exists
		if _, err := os.Stat(t.config.db.filePath); err == nil {
			// Remove the file
			if err := os.Remove(t.config.db.filePath); err != nil {
				return err
			}
		}

		if err := t.db.Close(); err != nil {
			return err
		}

		t.db = nil
	}

	return nil
}
