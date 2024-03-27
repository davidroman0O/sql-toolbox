package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"time"

	"reflect"

	"github.com/davidroman0O/sqlite-toolbox/data"
	"github.com/mattn/go-sqlite3"
)

/// This middleware can be used to preemptively store tasks in the database
///	- required to give channels to receive tasks, the channels and the tasks are sharing the same Type
/// - when tasks are sent, they are stored in the database, then a scheduler will get those tasks and send them to their channel
/// - you can tweak the scheduler
///
/// This middleware doesn't provide workers, nor scaling solution nor retries, it's just a simple way to store tasks in the database.
/// You will be able to then plugin any kind of other data processing system to handle those tasks.
///
/// I have a personal preference to use [`watermill`](https://github.com/ThreeDotsLabs/watermill/) or [`goakt`](https://github.com/Tochemey/goakt) to process my tasks, but you can use any other system you want.
///

var jobsTable = `
CREATE TABLE IF NOT EXISTS jobs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'enqueued',
	created_at INTEGER,
	updated_at INTEGER NULL,
	payload JSON,
	error TEXT NULL
);
`

func New(opts ...schedulerOptions) *TasksMiddleware {
	task := &TasksMiddleware{
		receivers: map[string]ReceiverHandler{},
		schedulerConfig: schedulerConfig{
			ticker: time.Millisecond * 100,
			limit:  10,
		},
	}

	for _, v := range opts {
		v(&task.schedulerConfig)
	}

	return task
}

type schedulerConfig struct {
	ticker time.Duration
	limit  int
}

func WithTicker(ticker time.Duration) schedulerOptions {
	return func(c *schedulerConfig) {
		c.ticker = ticker
	}
}

func WithLimit(limit int) schedulerOptions {
	return func(c *schedulerConfig) {
		c.limit = limit
	}
}

type schedulerOptions func(*schedulerConfig)

type TasksMiddleware struct {
	schedulerConfig schedulerConfig
	muxdb           *data.MuxDb
	receivers       map[string]ReceiverHandler
	doneScheduler   chan struct{}
}

func (l *TasksMiddleware) OnInit(muxdb *data.MuxDb) error {
	log.Println("Tasks middleware initialized")
	l.muxdb = muxdb
	if err := l.muxdb.Do(func(db *sql.DB) error {
		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			return err
		}

		_, err = tx.Exec(jobsTable)
		if err != nil {
			tx.Rollback()
			return err
		}
		defer slog.Info("Task database initialized")
		return tx.Commit()
	}); err != nil {
		return err
	}

	l.doneScheduler = make(chan struct{})

	go l.scheduler()
	return nil
}

func (l *TasksMiddleware) OnClose() error {
	log.Println("Tasks middleware closed")
	close(l.doneScheduler)
	return nil
}

func (l *TasksMiddleware) OnInsert(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	// slog.Info("inserted", slog.Any("db", db), slog.Any("table", table), slog.Any("rowid", rowid))
	return nil
}

func (l *TasksMiddleware) OnUpdate(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	return nil
}

func (l *TasksMiddleware) OnDelete(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	return nil
}

type ReceiverFn[T any] func() func(data T) error

type ReceiverHandler struct {
	consumer     reflect.Value
	consumerType reflect.Type
	kind         ReceiverKind
}

func NewHandler[T any](fn ReceiverFn[T]) ReceiverHandler {
	initializeState := fn() // allowing the functon to initialize it's own local state
	return ReceiverHandler{
		consumer:     reflect.ValueOf(initializeState),
		consumerType: reflect.TypeFor[T](),
		kind:         Receiver,
	}
}

func (l *TasksMiddleware) Register(receiver ReceiverHandler) error {
	if _, ok := l.receivers[receiver.consumerType.Name()]; ok {
		return fmt.Errorf("that type already exists")
	}
	l.receivers[receiver.consumerType.Name()] = receiver
	return nil
}

// the scheduler will search for tasks to be triggered
func (l *TasksMiddleware) scheduler() {
	ticker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-l.doneScheduler:
			slog.Info("scheduler stopped")
			ticker.Stop()
			return
		case <-ticker.C:
			if err := l.Beat(); err != nil {
				slog.Error("scheduler beat failed %v", err)
			}
		}
	}
}

func (t *TasksMiddleware) propagate(tasks []Task[any]) error {
	for _, task := range tasks {
		result := t.receivers[task.Type].consumer.Call([]reflect.Value{reflect.ValueOf(task.Payload)})
		// var nextState State
		// var err error
		if !result[0].IsNil() {
			if errCast, ok := result[0].Interface().(error); ok {
				t.muxdb.Do(func(db *sql.DB) error {
					if _, err := db.Exec(`UPDATE jobs SET status = ?, updated_at = ?, error = ? WHERE id = ?`, Archived, time.Now().UnixNano(), errCast.Error(), task.ID); err != nil {
						return err
					}
					return nil
				})
				continue
			} else {
				t.muxdb.Do(func(db *sql.DB) error {
					if _, err := db.Exec(`UPDATE jobs SET status = ?, updated_at = ?, error = ? WHERE id = ?`, Archived, time.Now().UnixNano(), "consumer function must return an error", task.ID); err != nil {
						return err
					}
					return nil
				})
				return fmt.Errorf("consumer function must return an error")
			}
		} else {
			t.muxdb.Do(func(db *sql.DB) error {
				if _, err := db.Exec(`UPDATE jobs SET status = ?, updated_at = ? WHERE id = ?`, Completed, time.Now().UnixNano(), task.ID); err != nil {
					return err
				}
				return nil
			})
		}
	}
	return nil
}

// TODO: make dependency injection for the database
func (t *TasksMiddleware) GetTasksByState(state State) ([]Task[any], error) {
	tasks := []Task[any]{}
	err := t.muxdb.Do(func(db *sql.DB) error {

		var rows *sql.Rows
		var err error

		if rows, err = db.Query(`SELECT id, type, status, created_at, updated_at, payload, error FROM jobs WHERE status = ? LIMIT ?`, state, t.schedulerConfig.limit); err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {

			row := Task[any]{}
			var createdAt int64
			var updatedAt sql.Null[int64]
			var payload string
			var errorData sql.Null[string]

			if err := rows.Scan(&row.ID, &row.Type, &row.State, &createdAt, &updatedAt, &payload, &errorData); err != nil {
				return err
			}

			row.CreatedAt = time.Unix(0, createdAt)
			if updatedAt.Valid {
				update := time.Unix(0, updatedAt.V)
				row.UpdatedAt = &update
			}

			if errorData.Valid {
				row.Error = &errorData.V
			}

			// dynamically use the type of the receiver to translate back into the type of the payload
			// so we can keep the generic working
			paramInstancePtr := reflect.New(t.receivers[row.Type].consumerType).Interface()

			//	this will avoid having a `map[string]interface{} cannot be converted to blablablabla` error
			err = json.Unmarshal([]byte(payload), paramInstancePtr)
			if err != nil {
				return err
			}

			row.Payload = reflect.ValueOf(paramInstancePtr).Elem().Interface()

			tasks = append(tasks, row)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return nil
	})

	return tasks, err
}

func (t *TasksMiddleware) Beat() error {
	return t.muxdb.Do(func(db *sql.DB) error {

		tasksEnqueued, err := t.GetTasksByState(Enqueued)
		if err != nil {
			return err
		}

		if len(tasksEnqueued) > 0 {
			if err := t.propagate(tasksEnqueued); err != nil {
				log.Fatal(err)
			}
		}

		return nil
	})
}

func (t *TasksMiddleware) Send(data any) error {

	var err error
	var valueOfWork interface{} = data

	if reflect.TypeOf(data).Kind() == reflect.Ptr {
		valueOfWork = reflect.ValueOf(data).Elem().Interface()
	}

	// extract the name of the type of the data
	nameType := reflect.TypeOf(valueOfWork).Name()

	err = t.muxdb.Do(func(db *sql.DB) error {
		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			tx.Rollback()
			return err
		}
		var dataJson []byte
		if dataJson, err = json.Marshal(valueOfWork); err != nil {
			tx.Rollback()
			return err
		}
		if _, err = tx.Exec(`
			INSERT INTO jobs (status, type, payload, created_at) 
			VALUES (?, ?, ?, ?);
		`, Enqueued, nameType, string(dataJson), time.Now().UnixNano()); err != nil {
			tx.Rollback()
			return err
		}
		// return nil
		return tx.Commit()
	})

	return err
}
