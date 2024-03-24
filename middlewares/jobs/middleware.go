package jobs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"time"

	"github.com/mattn/go-sqlite3"
)

type State string

var (
	Enequeued State = "enqueued"
	Pending   State = "pending"
	Running   State = "running"
	Success   State = "success"
	Failed    State = "failed"
)

// A job can contain any payload and can be hard deleted.
// Only the runtime knows how to handle a job type with a callback.
// A job is a stored item that represent a future Task (runtime)
type job[T any] struct {
	ID        int64      `json:"id"`
	State     State      `json:"status"`
	Type      string     `json:"type"`
	Payload   T          `json:"payload"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	Error     *string    `json:"error"`
}

// TODO @droman: add parent ID so we can have a tree of jobs for the saga pattern
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

func New() *JobsMiddleware {
	return &JobsMiddleware{
		consumers: map[string]fnTaskCallback{},
	}
}

type ConsumerFn interface{}

func Consumer[T any](fn func(ctx context.Context, data T) error) ConsumerFn {
	typeFor := reflect.TypeFor[T]().Name()
	tc := fnTaskCallback{
		fnType: reflect.TypeOf(fn),
		// we have to get extract the function as a reflect.Value to be able to call it
		fn:      reflect.ValueOf(fn),
		typeFor: typeFor,
	}
	return tc
}

type fnTaskCallback struct {
	typeFor string
	fnType  reflect.Type
	fn      reflect.Value
	in      reflect.Type
}

type JobsMiddleware struct {
	consumers map[string]fnTaskCallback
	db        *sql.DB
}

func (t *JobsMiddleware) Push(data interface{}) error {
	var err error
	var valueOfWork interface{} = data

	if reflect.TypeOf(data).Kind() == reflect.Ptr {
		valueOfWork = reflect.ValueOf(data).Elem().Interface()
	}

	// extract the name of the type of the data
	nameType := reflect.TypeOf(valueOfWork).Name()

	tx, err := t.db.BeginTx(context.Background(), nil)
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
	`, Enequeued, nameType, string(dataJson), time.Now().UnixNano()); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (t *JobsMiddleware) On(consumer ConsumerFn) error {
	switch consumer := consumer.(type) {
	case fnTaskCallback:

		// trivial validation: might not be needed since we have a type now for the consumer
		if consumer.fnType.Kind() != reflect.Func {
			return fmt.Errorf("task must be a function")
		}
		if consumer.fnType.NumIn() != 2 || consumer.fnType.NumOut() != 1 || consumer.fnType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			return fmt.Errorf("task function must have two parameters (ctx and data) and return 'error'")
		}

		// attach the second paramter of the function of the consumer
		// we will be able later on to parse it easily without `map[string]interface{}` conflicts
		consumer.in = consumer.fnType.In(1)

		t.consumers[consumer.typeFor] = consumer

	default:
		return fmt.Errorf("invalid consumer type %T", consumer)
	}
	return nil
}

func (t *JobsMiddleware) GetJobs() ([]job[any], error) {

	results, err := t.db.Query("SELECT id, type, payload, status, created_at, updated_at, error FROM jobs")
	if err != nil {
		return nil, err
	}

	defer results.Close()

	jobs := []job[any]{}

	for results.Next() {
		var j job[any]
		var createdAt int64
		var updatedAt int64
		var payload string
		err := results.Scan(&j.ID, &j.Type, &payload, &j.State, &createdAt, &updatedAt, &j.Error)
		if err != nil {
			return nil, err
		}

		j.CreatedAt = time.Unix(0, createdAt)
		if updatedAt != 0 {
			updatedAtTime := time.Unix(0, updatedAt)
			j.UpdatedAt = &updatedAtTime
		}

		oneConsumer := t.consumers[j.Type]

		// Now we're going to parse the payload by leveraging the type we gathered from the function of the consumer
		paramInstancePtr := reflect.New(oneConsumer.in).Interface()
		//	this will avoid having a `map[string]interface{} cannot be converted to blablablabla` error
		err = json.Unmarshal([]byte(payload), paramInstancePtr)
		if err != nil {
			return nil, err
		}

		//	convert it back to element and not pointer
		j.Payload = reflect.ValueOf(paramInstancePtr).Elem().Interface()

		jobs = append(jobs, j)
	}

	if err := results.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (t *JobsMiddleware) OnInit(db *sql.DB) error {
	log.Println("Jobs middleware initialized")
	t.db = db
	return t.initTable()
}

func (t *JobsMiddleware) OnClose(db *sql.DB) error {
	log.Println("Jobs middleware closed")
	return nil
}

// When job was inserted, we will call the consumer while gathering metrics
func (t *JobsMiddleware) OnInsert(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	if table != "jobs" {
		return nil
	}
	log.Printf("Jobs Insert operation on table %s with rowid %d", table, rowid)
	var err error
	// simply get it
	rows, err := conn.Query(`SELECT id, type, payload, status, created_at FROM jobs WHERE id = ?`, []driver.Value{rowid})
	if err != nil {
		return err
	}
	defer rows.Close()

	insertedJobs := []job[any]{} // we don't know the type of the job yet... but we will

	//	we have to use the same order as the query
	// 	`conn` got a different api than sql.DB
	dest := make([]driver.Value, 5)
	for {
		err = rows.Next(dest)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		id, _ := dest[0].(int64)
		typeJob, _ := dest[1].(string)
		payload, _ := dest[2].(string)
		status, _ := dest[3].(string)
		created_at, _ := dest[4].(int64)

		row := job[any]{
			ID:        id,
			State:     State(status),
			Type:      typeJob,
			CreatedAt: time.Unix(0, created_at),
		}

		oneConsumer := t.consumers[typeJob]

		// Now we're going to parse the payload by leveraging the type we gathered from the function of the consumer
		paramInstancePtr := reflect.New(oneConsumer.in).Interface()
		//	this will avoid having a `map[string]interface{} cannot be converted to blablablabla` error
		err = json.Unmarshal([]byte(payload), paramInstancePtr)
		if err != nil {
			return err
		}

		//	convert it back to element and not pointer
		row.Payload = reflect.ValueOf(paramInstancePtr).Elem().Interface()

		insertedJobs = append(insertedJobs, row)
	}

	for _, job := range insertedJobs {
		// call and manage err
		result := t.consumers[job.Type].fn.Call([]reflect.Value{reflect.ValueOf(context.Background()), reflect.ValueOf(job.Payload)})
		var nextState State
		var err error
		if !result[0].IsNil() {
			if errCast, ok := result[0].Interface().(error); ok {
				err = errCast
				nextState = Failed
				if _, err := conn.Exec(`UPDATE jobs SET status = ?, updated_at = ?, error = ? WHERE id = ?`, []driver.Value{string(nextState), time.Now().UnixNano(), err.Error(), job.ID}); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("consumer function must return an error")
			}
		}
		nextState = Success
		if _, err := conn.Exec(`UPDATE jobs SET status = ?, updated_at = ? WHERE id = ?`, []driver.Value{string(nextState), time.Now().UnixNano(), job.ID}); err != nil {
			return err
		}
	}

	return nil
}

func (t *JobsMiddleware) OnUpdate(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	log.Printf("Jobs Update operation on table %s with rowid %d", table, rowid)
	return nil
}

func (t *JobsMiddleware) OnDelete(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	log.Printf("Jobs Delete operation on table %s with rowid %d", table, rowid)
	return nil
}

// initTable creates the jobs table if it does not exist
func (t *JobsMiddleware) initTable() error {

	tx, err := t.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	_, err = tx.Exec(jobsTable)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
