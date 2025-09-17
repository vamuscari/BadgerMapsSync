package events

import (
	"fmt"
	"os/exec"

	"gopkg.in/yaml.v3"
)

// Action is the interface for all event actions.
type Action interface {
	Execute(app AppInterface) error
	Validate() error
}

// ActionConfig is a generic struct for unmarshalling actions from YAML.
type ActionConfig struct {
	Type string                 `yaml:"type"`
	Args map[string]interface{} `yaml:"args"`
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
func (a *ExecAction) Execute(app AppInterface) error {
	cmd := exec.Command(a.Command, a.Args...)
	return cmd.Run()
}

// Validate checks if the action is configured correctly.
func (a *ExecAction) Validate() error {
	if a.Command == "" {
		return fmt.Errorf("exec action requires a 'command'")
	}
	return nil
}

// DbAction executes a database function.
type DbAction struct {
	Function string `yaml:"function"`
}

// Execute runs the database function.
func (a *DbAction) Execute(app AppInterface) error {
	return app.GetDB().RunFunction(a.Function)
}

// Validate checks if the action is configured correctly.
func (a *DbAction) Validate() error {
	if a.Function == "" {
		return fmt.Errorf("db action requires a 'function'")
	}
	return nil
}

// ApiAction makes an API call.
type ApiAction struct {
	Endpoint string `yaml:"endpoint"`
	Method   string `yaml:"method"`
}

// Execute makes the API call.
func (a *ApiAction) Execute(app AppInterface) error {
	// This can be expanded to support different methods.
	_, err := app.GetAPI().GetRaw(a.Endpoint)
	return err
}

// Validate checks if the action is configured correctly.
func (a *ApiAction) Validate() error {
	if a.Endpoint == "" {
		return fmt.Errorf("api action requires an 'endpoint'")
	}
	return nil
}
