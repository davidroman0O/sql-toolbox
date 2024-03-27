package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/mattn/go-sqlite3"
)

func TestSimpleWorkflowActivity(t *testing.T) {
	var db *sql.DB
	var err error
	middleware := New()

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

	var connectionString = ":memory:?cache=shared"
	if db, err = sql.Open("sqlite3-extended", connectionString); err != nil {
		t.Error(err)
	}

	if err := middleware.OnInit(db); err != nil {
		t.Error(err)
	}

	helloWorkflow, err := Workflow(
		func(ctx context.Context) error {
			fmt.Println("Hello world")
			return nil
		},
		WorkflowName("test"),
		WorkflowWithRetries(3),
		WorkflowWithTimeout(time.Now().Add(time.Hour)),
		WorkflowWithCron("* * * * *"),
	)

	if err != nil {
		t.Error(err)
	}

	if err := middleware.Register(helloWorkflow); err != nil {
		t.Error(err)
	}

	if err := middleware.Execute(helloWorkflow); err != nil {
		t.Error(err)
	}

}
