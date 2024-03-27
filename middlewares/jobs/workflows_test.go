package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/k0kubun/pp/v3"
)

type TestWorkflowInput struct{}

// that's exactly how i like it
func TestWorkflowInstance(t *testing.T) {
	workflow, err := WorkflowParam(
		func(ctx context.Context, input TestWorkflowInput) error {
			return nil
		},
		WorkflowName("test"),
		WorkflowWithRetries(3),
		WorkflowWithTimeout(time.Now().Add(time.Hour)),
		WorkflowWithCron("* * * * *"),
		// WorkflowWithInput[TestWorkflowInput](),
	)
	if err != nil {
		t.Error(err)
	}
	pp.Println(workflow)
}
