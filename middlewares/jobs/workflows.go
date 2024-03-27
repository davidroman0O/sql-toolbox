package jobs

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

type workflowCron string
type workflowSimple any
type workflowParam any

///	TODO: how do i want to manage queues? Maybe we could have workers for set of queues?
///	TODO: how do i want to manage retries?

type workflowFn struct {
	ID   int64
	uuid string
	jobFn
	*workflowCron
}

// func (w *workflowFn) Save(sql *sql.DB) error {
// 	var trx *sql.Tx
// 	var err error
// 	if w.ID <= 0 {
// 		if trx, err = sql.Begin(); err != nil {
// 			return err
// 		}
// 		_, err := trx.Exec(`
// 			INSERT INTO jobs (status, type, payload, created_at)
// 			VALUES (?, ?, ?, ?);
// 		`, Enqueued, w.jobName, w.uuid, time.Now().UnixNano())
// 		if err != nil {
// 			return err
// 		}
// 	}
// }

func (w *workflowFn) Call(ctx context.Context) error {

	var result []reflect.Value

	switch w.inputFnType {
	case single:
		result = reflect.Value(w.jobFn.jobFnValue).Call([]reflect.Value{reflect.ValueOf(ctx)})
	default:
		return fmt.Errorf("workflow without params")
	}

	if !result[0].IsNil() {
		if errCast, ok := result[0].Interface().(error); ok {
			return errCast
		} else {
			return fmt.Errorf("consumer function must return an error")
		}
	}

	return nil
}

func (w *workflowFn) CallParam(ctx context.Context, data reflect.Value) error {

	var result []reflect.Value

	switch w.inputFnType {
	case params:
		result = reflect.Value(w.jobFn.jobFnValue).Call([]reflect.Value{reflect.ValueOf(ctx), data})
	default:
		return fmt.Errorf("workflow function must have a single parameter")
	}

	if !result[0].IsNil() {
		if errCast, ok := result[0].Interface().(error); ok {
			return errCast
		} else {
			return fmt.Errorf("consumer function must return an error")
		}
	}

	return nil
}

type workflowFnOption func(*workflowFn) error

func WorkflowName(name string) workflowFnOption {
	return func(w *workflowFn) error {
		w.jobName = jobName(name)
		return nil
	}
}

func WorkflowWithRetries(retries int) workflowFnOption {
	return func(w *workflowFn) error {
		w.jobRetries = jobRetries(retries)
		return nil
	}
}

func WorkflowWithTimeout(timeout time.Time) workflowFnOption {
	return func(w *workflowFn) error {
		w.jobTimeout = jobTimeout(timeout)
		return nil
	}
}

func WorkflowWithCron(cron string) workflowFnOption {
	return func(w *workflowFn) error {
		w.workflowCron = (*workflowCron)(&cron)
		return nil
	}
}

// func WorkflowWithInput[T any]() workflowFnOption {
// 	return func(w *workflowFn) error {
// 		w.inputFnType = params
// 		w.jobInputType = reflect.TypeFor[T]()
// 		return nil
// 	}
// }

func WorkflowWithForParam[T any](fn func(context.Context, T) error) workflowFnOption {
	return func(w *workflowFn) error {

		// Check if fn is a function with one or two parameters
		fnType := reflect.TypeOf(fn)
		if fnType.Kind() != reflect.Func || (fnType.NumIn() != 1 && fnType.NumIn() != 2) || fnType.NumOut() != 1 || fnType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			return fmt.Errorf("fn must be a function with one or two parameters and a return type of error %v", "")
		}

		// Check if the first parameter is context.Context
		if fnType.NumIn() > 0 && fnType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
			return fmt.Errorf("first parameter of fn must be context.Context %v", "")
		}

		// Declare a variable to store the reflect.Type of the second parameter
		var secondParamType reflect.Type

		// Check if the second parameter is the expected type
		if fnType.NumIn() == 2 {
			secondParamType = fnType.In(1)
			w.jobInputType = secondParamType
		}

		w.jobFnType = jobFnType(fnType)
		w.jobFnValue = jobFnValue(reflect.ValueOf(fn))

		return nil
	}
}

func WorkflowWithFor[T any](fn func(context.Context) error) workflowFnOption {
	return func(w *workflowFn) error {

		// Check if fn is a function with one or two parameters
		fnType := reflect.TypeOf(fn)
		if fnType.Kind() != reflect.Func || (fnType.NumIn() != 1 && fnType.NumIn() != 2) || fnType.NumOut() != 1 || fnType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			return fmt.Errorf("fn must be a function with one or two parameters and a return type of error %v", "")
		}

		// Check if the first parameter is context.Context
		if fnType.NumIn() > 0 && fnType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
			return fmt.Errorf("first parameter of fn must be context.Context %v", "")
		}

		// Declare a variable to store the reflect.Type of the second parameter
		var secondParamType reflect.Type

		// Check if the second parameter is the expected type
		if fnType.NumIn() == 2 {
			secondParamType = fnType.In(1)
			w.jobInputType = secondParamType
		}

		w.jobFnType = jobFnType(fnType)
		w.jobFnValue = jobFnValue(reflect.ValueOf(fn))

		return nil
	}
}

func Workflow(fn func(context.Context) error, opts ...workflowFnOption) (workflowSimple, error) {
	config := workflowFn{
		jobFn: jobFn{
			inputFnType: single,
		},
	}
	for _, opt := range opts {
		opt(&config)
	}
	config.jobFnValue = jobFnValue(reflect.ValueOf(fn))
	return workflowSimple(config), nil
}

func WorkflowParam[T any](fn func(context.Context, T) error, opts ...workflowFnOption) (workflowParam, error) {
	config := workflowFn{
		jobFn: jobFn{
			inputFnType: params,
		},
	}
	for _, opt := range opts {
		opt(&config)
	}
	config.jobInputType = reflect.TypeFor[T]()
	config.jobFnValue = jobFnValue(reflect.ValueOf(fn))
	return workflowParam(config), nil
}

type inputFnType string

var (
	single inputFnType = "single"
	params inputFnType = "params"
)
