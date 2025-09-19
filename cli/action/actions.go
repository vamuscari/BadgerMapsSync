package action

import (
	"badgermaps/api"
	"badgermaps/database"
	"fmt"
	"os"
	"os/exec"

	"gopkg.in/yaml.v3"
)

// Executor is responsible for executing actions.
type Executor struct {
	DB  database.DB
	API *api.APIClient
}

// NewExecutor creates a new Executor.
func NewExecutor(db database.DB, api *api.APIClient) *Executor {
	return &Executor{
		DB:  db,
		API: api,
	}
}

// Action is the interface for all event actions.
type Action interface {
	Execute(executor *Executor) error
	Validate() error
}

// ActionConfig is a generic struct for unmarshalling actions from YAML.
type ActionConfig struct {
	Type string                 `yaml:"type"`
	Args map[string]interface{} `yaml:"args"`
}

// EventAction is the top-level struct for an event-triggered action.
type EventAction struct {
	Name   string         `yaml:"name"`
	Event  string         `yaml:"event"`
	Source string         `yaml:"source,omitempty"`
	Run    []ActionConfig `yaml:"run"`
}

// NewActionFromConfig creates a specific action implementation from a generic ActionConfig.
func NewActionFromConfig(config ActionConfig) (Action, error) {
	bytes, err := yaml.Marshal(config.Args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal action args: %w", err)
	}

	var action Action
	switch config.Type {
	case "exec":
		action = &ExecAction{}
	case "db":
		action = &DbAction{}
	case "api":
		action = &ApiAction{}
	default:
		return nil, fmt.Errorf("unknown action type: %s", config.Type)
	}

	if err := yaml.Unmarshal(bytes, action); err != nil {
		return nil, fmt.Errorf("failed to unmarshal args for action type '%s': %w", config.Type, err)
	}

	return action, nil
}

// ExecAction executes a shell command.
type ExecAction struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

// Execute runs the command.
func (a *ExecAction) Execute(executor *Executor) error {
	cmd := exec.Command("sh", "-c", a.Command)
	cmd.Dir, _ = os.Getwd()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// Validate checks if the action is configured correctly.
func (a *ExecAction) Validate() error {
	if a.Command == "" {
		return fmt.Errorf("exec action requires a 'command'")
	}
	return nil
}

// DbAction executes a database function, procedure, or raw query.
type DbAction struct {
	config ActionConfig
}

// Execute runs the database action.
func (a *DbAction) Execute(executor *Executor) error {
	dbActionConfig := database.ActionConfig{
		Type: a.config.Type,
		Args: a.config.Args,
	}
	return executor.DB.RunAction(dbActionConfig)
}

// Validate checks if the action is configured correctly.
func (a *DbAction) Validate() error {
	args := a.config.Args
	_, hasCommand := args["command"]
	_, hasFunction := args["function"]
	_, hasProcedure := args["procedure"]
	_, hasQuery := args["query"]

	if !hasCommand && !hasFunction && !hasProcedure && !hasQuery {
		return fmt.Errorf("db action requires one of 'command', 'function', 'procedure', or 'query'")
	}
	return nil
}

// UnmarshalYAML is a custom unmarshaler to capture the raw config.
func (a *DbAction) UnmarshalYAML(value *yaml.Node) error {
	a.config.Type = "db"
	return value.Decode(&a.config.Args)
}

// ApiAction makes an API call.
type ApiAction struct {
	Endpoint string            `yaml:"endpoint"`
	Method   string            `yaml:"method"`
	Data     map[string]string `yaml:"data,omitempty"`
}

// Execute makes the API call.
func (a *ApiAction) Execute(executor *Executor) error {
	_, err := executor.API.RawRequest(a.Method, a.Endpoint, a.Data)
	return err
}

// Validate checks if the action is configured correctly.
func (a *ApiAction) Validate() error {
	if a.Endpoint == "" {
		return fmt.Errorf("api action requires an 'endpoint'")
	}
	if a.Method == "" {
		return fmt.Errorf("api action requires a 'method'")
	}
	return nil
}
