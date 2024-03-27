ABANDONNED UNTIL FUTHER NOTICE



- User define queues
- Task state:
    - scheduled: created and waitinf for being taken by a worker
    - pending: picked up by worker
    - active: being processed
    - retry: failed, waiting for retry
    - archived: failed max retry, stored for inspection
    - completed: sucessfully processed


TODO: rename that package `jobs` to `workflows` or `flows` or whatever

```go

// New "interface" activity
activity := jobs.Activity[T](
    func(ctx jobs.Context, params T) error {
        return nil
    }
    jobs.ActivityWithName(""),
    jobs.ActivityWithInput[T](), // if input, required to add second param
    jobs.ActivityWithRetries(5), 
)

middlewareJob.Register(activity)

// New "interface" workflow
workflow := jobs.Workflow(
    // Can also be: func(ctx workflow.Context) error 
    func(ctx jobs.Context, param T) error {

        activity := ctx.GetOneActivity[Y](Y{ something: params.thing })

        //  wait for the activity to be finished
        result, err := jobs.Execute(ctx, actvity) // Execute[Y](ctx, fnActivity) (Result[Y], error)

        switch result.State {
            case jobs.Success:
                fmt.Println(result.data) // type Y
            case jobs.Failed:
        }

        return nil
    },
    jobs.WorkflowWithName(""),
    jobs.WorkflowWithInput[T](), // if input, required to add second param
    jobs.WorkflowWithRetries(5), 
    jobs.WorkflowWithTimeout(time.Second * 5),
    // jobs.WorkflowWithCron(time.Second * 5),
)

middlewareJob.Register(workflow)

```

