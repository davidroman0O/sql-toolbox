package data

import "os"

type Exposable interface {
	String(env string) string
}

// An option can be enabled or not while having a value if enabled
type Option[T Exposable] struct {
	Env     string
	Enabled bool
	Value   Exposable
}

func (o *Option[T]) Enable(value T) {
	o.Enabled = true
	o.Value = value
}

func (o Option[T]) String() string {
	if value := GetEnvDefault(o.Env, ""); value != "" {
		return o.Value.String(value)
	}
	if o.Enabled {
		return o.Value.String("")
	}
	return ""
}

func GetEnvDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
