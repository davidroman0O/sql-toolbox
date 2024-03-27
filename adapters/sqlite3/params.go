package adaptersqlite3

import "fmt"

/// TODO @droman: I enjoy that implementation of options for `sqlite3`, might do them all and make a lib out of it. Other devs might save some time.

// type to manage the `mode` key in the connection string
type dbMode string

const (
	ReadWrite           dbMode = "rw"
	ReadOnly            dbMode = "ro"
	OpenCreateReadWrite dbMode = "rwc"
	Memory              dbMode = "memory"
)

func (v dbMode) String(env string) string {
	if env != "" {
		return fmt.Sprintf("mode=%v", env)
	}
	switch v {
	case ReadWrite, ReadOnly, OpenCreateReadWrite, Memory:
		return fmt.Sprintf("mode=%v", string(v))
	default:
		return "mode:unknown"
	}
}

// type to manage the `file` key in the connection string
type dbFile string

func (v dbFile) String(env string) string {
	if env != "" {
		return fmt.Sprintf("file=%v", env)
	}
	return fmt.Sprintf("file:%v", string(v))
}

// type to manage the `_mutex` key in the connection string
type dbMutex bool

func (v dbMutex) String(env string) string {
	if env != "" {
		return fmt.Sprintf("_mutex:%v", env)
	}
	value := "full"
	if !v {
		value = "no"
	}
	return fmt.Sprintf("_mutex:%v", value)
}

// type to manage the `_cache` key in the connection string
type dbCache bool

func (v dbCache) String(env string) string {
	if env != "" {
		return fmt.Sprintf("cache:%v", env)
	}
	value := "shared"
	if !v {
		value = "private"
	}
	return fmt.Sprintf("cache:%v", value)
}
