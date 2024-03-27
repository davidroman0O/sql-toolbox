WIP not for use

Intended to be just a toolbox of redundant SQL things i'm doing for side projects or prototyping

- Middleware supports with hooks
- Multiple adapters (just sqlite3 for now)


Example of one middleware with the toolbox:

```go
var toolbox *Toolbox
var err error

if toolbox, err = New(
    WithSqlite3(
        adaptersqlite3.WithMemory()...,
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

```


# Personal notes

I'm just having fun with my little tooling for now

Note: I changed my mind... it don't want to implement a whole set of libraries but instead i will implement datastorage for common things i need and use, ALSO i will make it adaptable to other providers, so the name of the repo should be `sql-toolbox` 

It should be a simple service that wrap multiple providers i'm using while also providing middlewares of usual type of data i'm storing for pet projects. I just want to be able to iterate quickly on prototypes.

Nothing has to be crazy, just providing simple `go` interfaces to push and pull data (for middlewares, e.g. just want to store and retrieve tasks while having a scheduler to trigger a goroutine OR e.g. storing/retrieving events) while being able to instanciate and use a sql connection.

Questions:
- how the fuck i'm supposed to manage the hooks in a generic way for different middlewares?! I need to try different adapters for other providers

