package adaptersqlite3

import (
	"fmt"

	"github.com/davidroman0O/sql-toolbox/data"
)

// Configuration for the database connection
type dbConfig struct {
	name     string
	mode     data.Option[dbMode]
	file     data.Option[dbFile]
	mutex    data.Option[dbMutex]
	cache    data.Option[dbCache]
	filePath string
}

type SqliteOption func(*dbConfig)

// Mode of the connection
func DBWithMode(value dbMode) SqliteOption {
	return func(config *dbConfig) {
		config.mode.Enable(value)
	}
}

func DBWithName(value string) SqliteOption {
	return func(config *dbConfig) {
		config.name = value
	}
}

// Proactively append `.db` and concatenante `/` between path and name
func DBWithFile(path string, name string) SqliteOption {
	return func(config *dbConfig) {
		config.file.Enable(dbFile(fmt.Sprintf("%v/%v.db", path, name)))
		config.filePath = fmt.Sprintf("%v/%v.db", path, name)
	}
}

func DBWithNoMutex() SqliteOption {
	return func(config *dbConfig) {
		config.mutex.Enable(dbMutex(false))
	}
}

func DBWithFullMutex() SqliteOption {
	return func(config *dbConfig) {
		config.mutex.Enable(dbMutex(true))
	}
}

func DBWithCacheShared() SqliteOption {
	return func(config *dbConfig) {
		config.cache.Enable(true)
	}
}

func DBWithCachePrivate() SqliteOption {
	return func(config *dbConfig) {
		config.cache.Enable(false)
	}
}

// `Memory` profile options
func WithMemory() []SqliteOption {
	opts := []SqliteOption{
		DBWithMode(Memory),
		DBWithCacheShared(),
	}
	return opts
}

func WithFile() []SqliteOption {
	opts := []SqliteOption{
		DBWithMode(OpenCreateReadWrite),
		DBWithName("db"),
		DBWithCacheShared(),
	}
	return opts
}

// Supposed easy configuration through dependency injection
func NewSettingConfig(options ...SqliteOption) *dbConfig {
	// with defaults
	config := &dbConfig{
		name: "sqlite3-extended",
		mode: data.Option[dbMode]{
			Env:     "DB_MODE",
			Enabled: true,
			Value:   Memory,
		},
		file: data.Option[dbFile]{
			Env:     "DB_FILE",
			Enabled: false,
			Value:   dbFile(fmt.Sprintf("%v/%v.db", ".", "gogog")),
		},
		mutex: data.Option[dbMutex]{
			Env:     "DB_MUTEX",
			Enabled: false,
		},
		cache: data.Option[dbCache]{
			Env:     "DB_CACHE",
			Enabled: false,
		},
		filePath: fmt.Sprintf("%v/%v.db", "./", "gogog"),
	}
	for _, option := range options {
		option(config)
	}
	return config
}

func ConnectionString(config *dbConfig) (string, error) {

	options := []string{}

	if config.cache.Enabled {
		fmt.Println("added ", config.cache.String())
		options = append(options, config.cache.String())
	}
	if config.file.Enabled {
		fmt.Println("added ", config.file.String())
		options = append(options, config.file.String())
	}
	if config.mode.Enabled {
		fmt.Println("added ", config.mode.String())
		options = append(options, config.mode.String())
	}

	queueStr := ""
	for _, v := range options {
		if v != "" {
			queueStr += v + "&"
		}
	}

	if config.mode.Value == Memory {
		// if you want to avoid "no such table"
		return fmt.Sprintf("file::memory:?cache=shared&%v", queueStr), nil
		// return fmt.Sprintf(":memory:?%v", queueStr), nil
	}

	// file:XXX?mode=YYY&{key}={value}&
	return fmt.Sprintf("%v?%v", config.file.String(), queueStr), nil
}
