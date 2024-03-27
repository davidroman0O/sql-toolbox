package data

import (
	"github.com/mattn/go-sqlite3"
)

type MiddlewareManager struct {
	Middlewares []Middleware
}

// TODO make a getter for all

func (mm *MiddlewareManager) Register(middleware Middleware) {
	mm.Middlewares = append(mm.Middlewares, middleware)
}

func (mm *MiddlewareManager) RunOnInit(muxdb *MuxDb) error {
	for _, middleware := range mm.Middlewares {
		if err := middleware.OnInit(muxdb); err != nil {
			return err
		}
	}
	return nil
}

func (mm *MiddlewareManager) RunOnClose() error {
	for _, middleware := range mm.Middlewares {
		if err := middleware.OnClose(); err != nil {
			return err
		}
	}
	return nil
}

/// TODO: for now, i will just use `*sqlite3.SQLiteConn` because that's what i'm using...
/// I'm lacking some inspiration on how i can make this more generic, i will come back to it

func (mm *MiddlewareManager) RunOnInsert(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	for _, middleware := range mm.Middlewares {
		if err := middleware.OnInsert(conn, db, table, rowid); err != nil {
			return err
		}
	}
	return nil
}

func (mm *MiddlewareManager) RunOnUpdate(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	for _, middleware := range mm.Middlewares {
		if err := middleware.OnUpdate(conn, db, table, rowid); err != nil {
			return err
		}
	}
	return nil
}

func (mm *MiddlewareManager) RunOnDelete(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	for _, middleware := range mm.Middlewares {
		if err := middleware.OnDelete(conn, db, table, rowid); err != nil {
			return err
		}
	}
	return nil
}
