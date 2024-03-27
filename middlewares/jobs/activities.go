package jobs

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

type activityType any

type activityFn struct {
	jobFn
}

type activityFnOption func(*activityFn)

func ActivityName(name string) activityFnOption {
	return func(w *activityFn) {
		w.jobName = jobName(name)
	}
}

func ActivityWithRetries(retries int) activityFnOption {
	return func(w *activityFn) {
		w.jobRetries = jobRetries(retries)
	}
}

func ActivityWithTimeout(timeout time.Time) activityFnOption {
	return func(w *activityFn) {
		w.jobTimeout = jobTimeout(timeout)
	}
}

func ActivityWithInput[T any]() activityFnOption {
	return func(w *activityFn) {
		w.inputFnType = params
		w.jobInputType = reflect.TypeFor[T]()
	}
}

func Activity(fn any, opts ...activityFnOption) (activityType, error) {
	config := activityFn{
		jobFn: jobFn{
			inputFnType: single,
		},
	}

	// Check if fn is a function with one or two parameters
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func || (fnType.NumIn() != 1 && fnType.NumIn() != 2) || fnType.NumOut() != 1 || fnType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		return nil, fmt.Errorf("fn must be a function with one or two parameters and a return type of error %v", "")
	}

	// Check if the first parameter is context.Context
	if fnType.NumIn() > 0 && fnType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return nil, fmt.Errorf("first parameter of fn must be context.Context %v", "")
	}

	// Declare a variable to store the reflect.Type of the second parameter
	var secondParamType reflect.Type

	// Check if the second parameter is the expected type
	if fnType.NumIn() == 2 {
		secondParamType = fnType.In(1)
		config.jobFn.jobInputType = secondParamType
	}

	config.jobFn.jobFnType = jobFnType(fnType)
	config.jobFn.jobFnValue = jobFnValue(reflect.ValueOf(fn))

	for _, opt := range opts {
		opt(&config)
	}

	return activityType(config), nil
}
