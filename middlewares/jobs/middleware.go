package jobs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/davidroman0O/sqlite-toolbox/rows"
	"github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
)

func New() *JobsMiddleware {
	return &JobsMiddleware{
		consumers:  map[string]fnTaskCallback{},
		workflows:  map[workflowSimple]workflowFn{},
		activities: map[activityType]activityFn{},
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
	workflows  map[workflowSimple]workflowFn
	activities map[activityType]activityFn

	consumers map[string]fnTaskCallback
	db        *sql.DB

	cron     cron.Cron
	location *time.Location
	// amount of jobs that are supposedly ready to be scheduled
	metricJobsToSchedule atomic.Int32

	metricWorkersActive atomic.Int32

	doneScheduler chan struct{}
}

func (t *JobsMiddleware) ScaleWorker(num int) {
	t.metricWorkersActive.Store(int32(num))
}

func (t *JobsMiddleware) Scheduler() {
	// schedule jobs
	// schedule scale up and down of workers
	ticker := time.NewTicker(time.Millisecond * 500)
	for {
		select {
		case <-t.doneScheduler:
			ticker.Stop()
			return
		case <-ticker.C:
			if err := t.Beat(); err != nil {
				slog.Error("scheduler beat failed %v", err)
			}
			// beat the scheduler
			// check crons
		}
	}
}

func (t *JobsMiddleware) Beat() error {
	return nil
}

func (t *JobsMiddleware) Register(fn any) error {
	switch fn := fn.(type) {
	case workflowFn:
		t.workflows[workflowSimple(fn)] = fn
		return nil
	case activityFn:
		t.activities[activityType(fn)] = fn
		return nil
	}
	return fmt.Errorf("cannot register type %T", fn)
}

type executionConfig struct{}

type executionOption func(*executionConfig)

func (t *JobsMiddleware) Execute(workflow workflowSimple) error {
	def := workflow.(workflowFn)
	// TODO: create a job, related to the workflow, and then let the work call it
	return def.Call(context.Background())
}

func (t *JobsMiddleware) ExecuteParams(workflow workflowSimple, value interface{}) error {
	def := workflow.(workflowFn)
	// TODO: create a job, related to the workflow, and then let the work call it
	// TODO check that value is the right type
	return def.CallParam(context.Background(), reflect.ValueOf(value))
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
	`, Enqueued, nameType, string(dataJson), time.Now().UnixNano()); err != nil {
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
		var errorS *string
		err := results.Scan(&j.ID, &j.Type, &payload, &j.State, &createdAt, &updatedAt, &errorS)
		if err != nil {
			return nil, err
		}

		j.CreatedAt = time.Unix(0, createdAt)
		if updatedAt != 0 {
			updatedAtTime := time.Unix(0, updatedAt)
			j.UpdatedAt = &updatedAtTime
		}
		if errorS != nil {
			j.Error = errorS
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
	// after we check for the table, we start the scheduler
	defer func() {
		go t.Scheduler()
	}()
	if t.location == nil {
		t.location = time.Local
	}
	t.cron = *cron.New(cron.WithLocation(t.location))
	return t.initTable()
}

func (t *JobsMiddleware) OnClose(db *sql.DB) error {
	log.Println("Jobs middleware closed")
	return nil
}

// When job was inserted, we will call the consumer while gathering metrics
// TODO @droman: when we receive a job, we should push it to a queue and have a goroutine that process it so we can have a better performance
func (t *JobsMiddleware) OnInsert(conn *sqlite3.SQLiteConn, db string, table string, rowid int64) error {
	if table != "jobs" {
		return nil
	}

	//	TODO: went hooked, increase the atomic counter of the number of jobs
	t.metricJobsToSchedule.Add(1)

	return nil

	log.Printf("Jobs Insert operation on table %s with rowid %d", table, rowid)
	var err error
	// simply get it
	iterator, err := conn.Query(`SELECT id, type, payload, status, created_at FROM jobs WHERE id = ?`, []driver.Value{rowid})
	if err != nil {
		return err
	}
	defer iterator.Close()

	data, err := rows.GetRowsMap(iterator.Next, rows.WithColumns([]string{"id", "type", "payload", "status", "created_at"}))
	if err != nil {
		return err
	}

	insertedJobs := []job[any]{} // we don't know the type of the job yet... but we will

	for _, value := range data {

		row := job[any]{
			ID:        value["id"].(int64),
			State:     State(value["status"].(string)),
			Type:      value["type"].(string),
			CreatedAt: time.Unix(0, value["created_at"].(int64)),
		}

		if oneConsumer, ok := t.consumers[row.Type]; ok {

			// Now we're going to parse the payload by leveraging the type we gathered from the function of the consumer
			paramInstancePtr := reflect.New(oneConsumer.in).Interface()
			//	this will avoid having a `map[string]interface{} cannot be converted to blablablabla` error
			err = json.Unmarshal([]byte(value["payload"].(string)), paramInstancePtr)
			if err != nil {
				return err
			}

			//	convert it back to element and not pointer
			row.Payload = reflect.ValueOf(paramInstancePtr).Elem().Interface()

			insertedJobs = append(insertedJobs, row)
		} else {
			return fmt.Errorf("consumer not found for job type %s", row.Type)
		}

	}

	for _, job := range insertedJobs {
		if job.ID <= 0 {
			// false positive, failed syntax or else
			slog.Error("job ID is invalid", slog.Any("job", job))
			continue
		}
		// call and manage err
		result := t.consumers[job.Type].fn.Call([]reflect.Value{reflect.ValueOf(context.Background()), reflect.ValueOf(job.Payload)})
		var nextState State
		var err error
		if !result[0].IsNil() {
			if errCast, ok := result[0].Interface().(error); ok {
				nextState = Archived
				if _, err = conn.Exec(`UPDATE jobs SET status = ?, updated_at = ?, error = ? WHERE id = ?`, []driver.Value{string(nextState), time.Now().UnixNano(), errCast.Error(), job.ID}); err != nil {
					return err
				}
				continue
			} else {
				return fmt.Errorf("consumer function must return an error")
			}
		}
		nextState = Completed
		if _, err = conn.Exec(`UPDATE jobs SET status = ?, updated_at = ? WHERE id = ?`, []driver.Value{string(nextState), time.Now().UnixNano(), job.ID}); err != nil {
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
