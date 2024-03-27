package sqltoolbox

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	adaptersqlite3 "github.com/davidroman0O/sql-toolbox/adapters/sqlite3"
	"github.com/davidroman0O/sql-toolbox/middlewares/logger"
	"github.com/davidroman0O/sql-toolbox/middlewares/tasks"
	"github.com/k0kubun/pp/v3"
)

func TestOpenCloseMemory(t *testing.T) {
	var toolbox *Toolbox
	var err error

	if toolbox, err = New(
		WithSqlite3(
			adaptersqlite3.WithMemory()..., // gives a default
		), // should instantiate a default in memory sqlite3 database
	); err != nil {
		t.Error(err)
	}

	defer func() {
		if err := toolbox.Close(); err != nil {
			t.Error(err)
		}
	}()

	didPing := false

	if err := toolbox.Do(func(db *sql.DB) error {
		if err := db.Ping(); err != nil {
			return err
		}
		slog.Info("pinged")
		didPing = true
		return nil
	}); err != nil {
		t.Error(err)
	}

	if !didPing {
		t.Error("did not ping")
	}

}

type MyData struct {
	ID        int64
	Natural   string
	CreatedAt time.Time
	UpdatedAt sql.NullTime
	DeletedAt sql.NullTime
	Data      map[string]interface{}
}

func TestCreateTableMemory(t *testing.T) {

	var toolbox *Toolbox
	var err error
	if toolbox, err = New(
		WithMiddleware(logger.New()),
		WithSqlite3(),
	); err != nil {
		t.Error(err)
	}

	defer func() {
		if err := toolbox.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Let's test a basic table with a few columns
	if err := toolbox.Do(func(db *sql.DB) error {
		_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS mytable (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	natural TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NULL,
	deleted_at DATETIME NULL,
	data JSON
);
		`)
		return err
	}); err != nil {
		t.Error(err)
	}

	if err := toolbox.Do(func(db *sql.DB) error {
		_, err := db.Exec("INSERT INTO mytable (natural, data) VALUES (?, ?)", "Example 1", `{"key": "value", "numbers": [1, 2, 3]}`)
		return err
	}); err != nil {
		t.Error(err)
	}

	myData := MyData{
		Natural: "Example 2",
		Data: map[string]interface{}{
			"key":     "value",
			"numbers": []int{1, 2, 3},
		},
		CreatedAt: time.Now(),
	}

	dataJSON, err := json.Marshal(myData.Data)
	if err != nil {
		t.Error(err)
	}

	if err := toolbox.Do(func(db *sql.DB) error {
		_, err := db.Exec(`
        INSERT INTO mytable (natural, data, created_at)
        VALUES (?, ?, ?)
    `, myData.Natural, dataJSON, myData.CreatedAt)
		return err
	}); err != nil {
		t.Error(err)
	}

	var results []MyData
	if err := toolbox.Do(func(db *sql.DB) error {
		rows, err := db.Query("SELECT id, natural, created_at, updated_at, deleted_at, data FROM mytable")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var row MyData
			var dataJSON []byte

			err = rows.Scan(&row.ID, &row.Natural, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &dataJSON)
			if err != nil {
				return err
			}

			err = json.Unmarshal(dataJSON, &row.Data)
			if err != nil {
				return err
			}

			results = append(results, row)
		}

		if err = rows.Err(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Error(err)
	}

	// pp.Println(results)

	if len(results) == 0 {
		t.Error("no results")
	}

	slog.Info("closing")

}

type MyData2 struct {
	Msg string
}

func TestJobsMiddleware(t *testing.T) {
	var toolbox *Toolbox
	var err error

	// prepare the whole toolbox
	if toolbox, err = New(
		WithSqlite3(
			adaptersqlite3.WithMemory()..., // gives a default
		),
		WithMiddleware(
			tasks.New(),
		),
	); err != nil {
		t.Error(err)
	}

	defer func() {
		if err := toolbox.Close(); err != nil {
			t.Error(err)
		}
	}()

	// that's how you should find it back
	jobBox, err := FindMiddleware[tasks.TasksMiddleware](toolbox)
	if err != nil {
		t.Error(err)
	}

	if err := jobBox.Register(tasks.NewHandler[MyData2](
		func() func(data MyData2) error {
			return func(data MyData2) error {
				pp.Println("ping", data)
				return nil
			}
		},
	)); err != nil {
		t.Error(err)
	}

	if err := jobBox.Send(MyData2{Msg: "hello"}); err != nil {
		t.Error(err)
	}

	jobs, err := jobBox.GetTasksByState(tasks.Completed)
	if err != nil {
		t.Error(err)
	}

	for _, v := range jobs {
		fmt.Println(v)
	}

}
