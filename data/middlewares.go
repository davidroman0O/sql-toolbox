package data

import (
	"github.com/mattn/go-sqlite3"
)

type Middleware interface {
	OnInit(muxdb *MuxDb) error
	OnClose() error
	OnInsert(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error
	OnUpdate(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error
	OnDelete(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error
}
