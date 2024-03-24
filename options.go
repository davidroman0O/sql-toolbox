package sqlitetoolbox

import (
	"fmt"
	"os"
)

// Configuration for the database connection
type dbConfig struct {
	name     string
	mode     Option[dbMode]
	file     Option[dbFile]
	mutex    Option[dbMutex]
	cache    Option[dbCache]
	filePath string
}

type dbOption func(*dbConfig)

// Mode of the connection
func DBWithMode(value dbMode) dbOption {
	return func(config *dbConfig) {
		config.mode.Enable(value)
	}
}

func DBWithName(value string) dbOption {
	return func(config *dbConfig) {
		config.name = value
	}
}

// Proactively append `.db` and concatenante `/` between path and name
func DBWithFile(path string, name string) dbOption {
	return func(config *dbConfig) {
		config.file.Enable(dbFile(fmt.Sprintf("%v/%v.db", path, name)))
		config.filePath = fmt.Sprintf("%v/%v.db", path, name)
	}
}

func DBWithNoMutex() dbOption {
	return func(config *dbConfig) {
		config.mutex.Enable(dbMutex(false))
	}
}

func DBWithFullMutex() dbOption {
	return func(config *dbConfig) {
		config.mutex.Enable(dbMutex(true))
	}
}

func DBWithCacheShared() dbOption {
	return func(config *dbConfig) {
		config.cache.Enable(true)
	}
}

func DBWithCachePrivate() dbOption {
	return func(config *dbConfig) {
		config.cache.Enable(false)
	}
}

// Supposed easy configuration through dependency injection
func NewSettingConfig(options ...dbOption) *dbConfig {
	// with defaults
	config := &dbConfig{
		name: "sqlite3-extended",
		mode: Option[dbMode]{
			Env:     "DB_MODE",
			Enabled: true,
			Value:   Memory,
		},
		file: Option[dbFile]{
			Env:     "DB_FILE",
			Enabled: false,
			Value:   dbFile(fmt.Sprintf("%v/%v.db", ".", "gogog")),
		},
		mutex: Option[dbMutex]{
			Env:     "DB_MUTEX",
			Enabled: false,
		},
		cache: Option[dbCache]{
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
		// :memory:?mode=YYY
		return fmt.Sprintf(":memory:?%v", queueStr), nil
	}

	// file:XXX?mode=YYY&{key}={value}&
	return fmt.Sprintf("%v?%v", config.file.String(), queueStr), nil
}

func getEnvDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
