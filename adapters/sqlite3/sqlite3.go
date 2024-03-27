package adaptersqlite3

import (
	"database/sql"
	"log/slog"
	"os"

	"github.com/davidroman0O/sql-toolbox/data"
	"github.com/mattn/go-sqlite3"
)

type Sqlite3Connector struct {
	config *dbConfig
}

func NewSqlite3Connector(opts ...SqliteOption) *Sqlite3Connector {
	config := dbConfig{}

	// apply the default
	for _, v := range WithMemory() {
		v(&config)
	}

	for _, opt := range opts {
		opt(&config)
	}

	connector := &Sqlite3Connector{
		config: &config,
	}

	return connector
}

func (c Sqlite3Connector) Open(middlewareManager *data.MiddlewareManager) (*data.MuxDb, error) {

	var db *sql.DB
	var err error

	var connectionString string
	if connectionString, err = ConnectionString(c.config); err != nil {
		return nil, err
	}

	slog.Info(connectionString)

	sql.Register(
		c.config.name,
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				// register callback
				conn.RegisterUpdateHook(
					func(op int, db string, table string, rowid int64) {

						// TODO @droman: i need a tool to unmarshall the rows from `conn` to have match the sql.DB api

						switch op {

						case sqlite3.SQLITE_INSERT:
							if err := middlewareManager.RunOnInsert(conn, db, table, rowid); err != nil {
								// TODO: Handle error - i have no idea how
								slog.Error("SQLITE_INSERT error: %v", err)
							}

						case sqlite3.SQLITE_UPDATE:
							if err := middlewareManager.RunOnUpdate(conn, db, table, rowid); err != nil {
								// TODO: Handle error - i have no idea how
								slog.Error("SQLITE_UPDATE error: %v", err)
							}

						case sqlite3.SQLITE_DELETE:
							if err := middlewareManager.RunOnDelete(conn, db, table, rowid); err != nil {
								// TODO: Handle error - i have no idea how
								slog.Error("SQLITE_DELETE error: %v", err)
							}

						}
					},
				)

				return nil
			}})

	if db, err = sql.Open(c.config.name, connectionString); err != nil {
		return nil, err
	}

	// TODO @droman: might only enable that for `memory` mode?
	db.SetMaxOpenConns(1)
	// db.SetMaxIdleConns(1)

	return data.NewMuxDb(db), nil
}

func (c Sqlite3Connector) Close() error {

	// Check if the file exists
	if _, err := os.Stat(c.config.filePath); err == nil {
		// Remove the file
		if err := os.Remove(c.config.filePath); err != nil {
			return err
		}
	}

	return nil
}
