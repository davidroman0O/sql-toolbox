package sqltoolbox

import (
	"database/sql"
	"fmt"
	"reflect"

	adaptersqlite3 "github.com/davidroman0O/sql-toolbox/adapters/sqlite3"
	"github.com/davidroman0O/sql-toolbox/data"
)

type Toolbox struct {
	// db     *sql.DB
	*data.MuxDb
	config *initConfig
}

// func (t *Toolbox) Do(cb data.DoFn) error {
// 	return cb(t.db)
// }

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

	for _, middleware := range t.config.middlewareManager.Middlewares {
		if reflect.TypeOf(middleware) == reflect.TypeFor[*T]() {
			var inter interface{} = middleware
			base := inter.(*T)
			return base, nil
		}
	}

	return nil, fmt.Errorf("middleware not found")
}

type initConfig struct {
	connector         DatabaseConnector
	middlewareManager *data.MiddlewareManager
}

type initOpts func(*initConfig) error

func WithSqlite3(opts ...adaptersqlite3.SqliteOption) initOpts {
	return func(config *initConfig) error {
		config.connector = *adaptersqlite3.NewSqlite3Connector(opts...)
		return nil
	}
}

func WithMiddleware(middleware data.Middleware) initOpts {
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
		middlewareManager: &data.MiddlewareManager{},
	}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, err
		}
	}

	toolbox := &Toolbox{
		config: config,
	}

	var err error

	if toolbox.MuxDb, err = config.connector.Open(config.middlewareManager); err != nil {
		return nil, err
	}

	if toolbox.MuxDb == nil {
		return nil, fmt.Errorf("sql connector critically failed")
	}

	if err := toolbox.MuxDb.Do(func(db *sql.DB) error {
		if err := db.Ping(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Initialize middlewares
	if err := config.middlewareManager.RunOnInit(toolbox.MuxDb); err != nil {
		return nil, err
	}

	return toolbox, nil
}

func (t *Toolbox) Close() error {
	if t.config.connector != nil {
		// Close middlewares
		if err := t.config.middlewareManager.RunOnClose(); err != nil {
			return err
		}

		if err := t.config.connector.Close(); err != nil {
			return err
		}

		t.MuxDb.Db.Close()
	}
	return nil
}
