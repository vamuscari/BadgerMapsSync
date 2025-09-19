package events

import (
	"badgermaps/api"
	"badgermaps/app/state"
	"badgermaps/database"
)

// AppInterface defines the methods that the application must implement
// for the event system to interact with it.
type AppInterface interface {
	GetDB() database.DB
	GetAPI() *api.APIClient
	GetConfig() *Config
	GetState() *state.State
	GetEventActions() []EventAction
	RawRequest(method, endpoint string, data map[string]string) ([]byte, error)
}

// Config defines the structure of the application's configuration.
type Config struct {
	Events []EventAction `yaml:"actions"`
}