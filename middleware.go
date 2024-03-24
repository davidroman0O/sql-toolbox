package sqlitetoolbox

import (
	"database/sql"

	"github.com/mattn/go-sqlite3"
)

type Middleware interface {
	OnInit(db *sql.DB) error
	OnClose(db *sql.DB) error
	OnInsert(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error
	OnUpdate(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error
	OnDelete(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error
}

type MiddlewareManager struct {
	middlewares []Middleware
}

func (mm *MiddlewareManager) Register(middleware Middleware) {
	mm.middlewares = append(mm.middlewares, middleware)
}

func (mm *MiddlewareManager) RunOnInit(db *sql.DB) error {
	for _, middleware := range mm.middlewares {
		if err := middleware.OnInit(db); err != nil {
			return err
		}
	}
	return nil
}

func (mm *MiddlewareManager) RunOnClose(db *sql.DB) error {
	for _, middleware := range mm.middlewares {
		if err := middleware.OnClose(db); err != nil {
			return err
		}
	}
	return nil
}

func (mm *MiddlewareManager) RunOnInsert(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	for _, middleware := range mm.middlewares {
		if err := middleware.OnInsert(conn, db, table, rowid); err != nil {
			return err
		}
	}
	return nil
}

func (mm *MiddlewareManager) RunOnUpdate(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	for _, middleware := range mm.middlewares {
		if err := middleware.OnUpdate(conn, db, table, rowid); err != nil {
			return err
		}
	}
	return nil
}

func (mm *MiddlewareManager) RunOnDelete(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	for _, middleware := range mm.middlewares {
		if err := middleware.OnDelete(conn, db, table, rowid); err != nil {
			return err
		}
	}
	return nil
}
