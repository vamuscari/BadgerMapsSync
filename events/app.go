package events

import (
	"badgermaps/api"
	"badgermaps/app/state"
	"badgermaps/database"
)

// AppConfig defines the configuration for the event dispatcher.
type AppConfig struct {
	Events map[string][]ActionConfig `yaml:"actions"`
}

// AppInterface defines the methods that the event system needs to access from the main app.
// This decouples the events package from the main app package.
type AppInterface interface {
	GetState() *state.State
	GetConfig() *AppConfig
	GetDB() database.DB
	GetAPI() *api.APIClient
}
