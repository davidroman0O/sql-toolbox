package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"

	"github.com/mattn/go-sqlite3"
)

// TODO @droman: add parent ID so we can have a tree of jobs for the saga pattern
var jobsTable = `
CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'enqueued',
    created_at DATETIME,
    updated_at DATETIME NULL,
    payload JSON,
    error TEXT NULL
);
`

func New() *JobsMiddleware {
	return &JobsMiddleware{
		tasks: map[string][]fnTaskCallback{},
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
	tasks map[string][]fnTaskCallback
	db    *sql.DB
}

func (t *JobsMiddleware) Send(data interface{}) error {
	return nil
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

		// if the map is not initialized, we initialize it
		if _, ok := t.tasks[consumer.typeFor]; !ok {
			t.tasks[consumer.typeFor] = []fnTaskCallback{}
		}

		//	we append the consumer to the list of consumers while keeping the type of the data
		t.tasks[consumer.typeFor] = append(t.tasks[consumer.typeFor], consumer)

	default:
		return fmt.Errorf("invalid consumer type %T", consumer)
	}
	return nil
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

func (t *JobsMiddleware) OnInsert(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	log.Printf("Jobs Insert operation on table %s with rowid %d", table, rowid)
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
