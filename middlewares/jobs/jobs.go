package jobs

import (
	"reflect"
	"time"
)

type jobFnType reflect.Type
type jobFnValue reflect.Value
type jobName string
type jobRetries int
type jobTimeout time.Time
type jobInputType reflect.Type

type jobFn struct {
	jobName
	jobRetries
	jobTimeout
	jobInputType
	jobFnType
	inputFnType
	jobFnValue
}

type State string

var (
	// Job was just created and stored
	Enqueued State = "enqueued"
	// Job tagged as scanned by scheduler
	Scheduled State = "scheduled"
	// Job in the list of a worker
	Pending State = "pending"
	// Job is being processed
	Active State = "active"
	// Job failed and will be retried (scheduled again)
	Retry State = "retry"
	// Job failed and stored for inspection
	Archived State = "archived"
	// Job was successfully processed
	Completed State = "completed"
	// Job is waiting for an external signal
	WaitSignal State = "wait_signal"
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
