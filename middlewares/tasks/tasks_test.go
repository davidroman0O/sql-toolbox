package tasks

import (
	"database/sql"
	"testing"
	"time"

	"github.com/davidroman0O/sqlite-toolbox/data"
	"github.com/k0kubun/pp/v3"
	"github.com/mattn/go-sqlite3"
)

func TestApiSimple(t *testing.T) {

	middleware := New()

	middleware.OnInit(nil)

	defer func() {
		middleware.OnClose()
		time.Sleep(time.Second * 2)
	}()

	middleware.Register(
		NewHandler(
			func() func(data interface{}) error {
				counter := 0
				return func(data interface{}) error {
					counter++
					return nil
				}
			},
		))

	time.Sleep(time.Second * 2)
}

type PingMsg struct {
	Msg string
}

func TestSimpleTask(t *testing.T) {
	var db *sql.DB
	var err error
	middleware := New(
		WithTicker(time.Second*1),
		WithLimit(2),
	)

	sql.Register(
		"sqlite3-extended",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				// make so something with `conn`?
				conn.RegisterUpdateHook(
					func(op int, db string, table string, rowid int64) {
						switch op {

						case sqlite3.SQLITE_INSERT:
							middleware.OnInsert(conn, db, table, rowid)

						case sqlite3.SQLITE_UPDATE:
							middleware.OnUpdate(conn, db, table, rowid)

						case sqlite3.SQLITE_DELETE:
							middleware.OnDelete(conn, db, table, rowid)

						}
					},
				)
				return nil
			}})

	// var connectionString = "file:./db.db?cache=shared"
	var connectionString = "file::memory:?cache=shared"
	if db, err = sql.Open("sqlite3-extended", connectionString); err != nil {
		t.Error(err)
	}

	if err := middleware.OnInit(data.NewMuxDb(db)); err != nil {
		t.Error(err)
	}

	defer func() {
		middleware.OnClose()
		time.Sleep(time.Second * 2)
	}()

	counter := 0

	if err := middleware.Register(NewHandler[PingMsg](
		func() func(data PingMsg) error {
			return func(data PingMsg) error {
				counter++
				pp.Println("ping", data)
				// return fmt.Errorf("failed")
				return nil
			}
		},
	)); err != nil {
		t.Error(err)
	}

	// for range 10 {
	if err := middleware.Send(PingMsg{Msg: "hello"}); err != nil {
		t.Error(err)
	}
	// }

	time.Sleep(time.Second * 2)

	data, err := middleware.GetTasksByState(Enqueued)
	if err != nil {
		t.Error(err)
	}
	pp.Println(data)
	data, err = middleware.GetTasksByState(Archived)
	if err != nil {
		t.Error(err)
	}
	pp.Println(data)
	data, err = middleware.GetTasksByState(Completed)
	if err != nil {
		t.Error(err)
	}
	pp.Println(data)
}
