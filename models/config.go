package models

// Config ...
type Config struct {
	ConnectionString string
}

// Cache
type Cached map[string]struct{}
