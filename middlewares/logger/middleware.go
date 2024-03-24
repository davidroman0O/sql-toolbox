package logger

import (
	"database/sql"
	"log"

	"github.com/mattn/go-sqlite3"
)

func New() *LoggerMiddleware {
	return &LoggerMiddleware{}
}

type LoggerMiddleware struct{}

func (l *LoggerMiddleware) OnInit(db *sql.DB) error {
	log.Println("Logger middleware initialized")
	return nil
}

func (l *LoggerMiddleware) OnClose(db *sql.DB) error {
	log.Println("Logger middleware closed")
	return nil
}

func (l *LoggerMiddleware) OnInsert(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	log.Printf("Logger Insert operation on table %s with rowid %d", table, rowid)
	return nil
}

func (l *LoggerMiddleware) OnUpdate(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	log.Printf("Logger Update operation on table %s with rowid %d", table, rowid)
	return nil
}

func (l *LoggerMiddleware) OnDelete(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	log.Printf("Logger Delete operation on table %s with rowid %d", table, rowid)
	return nil
}
