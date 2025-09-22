package action

import (
	"badgermaps/api"
	"badgermaps/database"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

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

// ExecAction executes a command using the platform shell by default.
// Set UseShell to false to execute the binary directly with Args.
type ExecAction struct {
	Command  string   `yaml:"command"`
	Args     []string `yaml:"args"`
	UseShell *bool    `yaml:"use_shell"`
}

// Execute runs the command.
func (a *ExecAction) Execute(executor *Executor) error {
	cmd := a.buildCommand()
	if cmd.Dir == "" {
		if cwd, err := os.Getwd(); err == nil {
			cmd.Dir = cwd
		}
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return fmt.Errorf("%w: %s", err, trimmed)
		}
		return err
	}
	return nil
}

// Validate checks if the action is configured correctly.
func (a *ExecAction) Validate() error {
	if strings.TrimSpace(a.Command) == "" {
		return fmt.Errorf("exec action requires a 'command'")
	}
	if a.useShell() && len(a.Args) > 0 {
		return fmt.Errorf("exec action 'args' are only supported when use_shell is set to false")
	}
	return nil
}

// UnmarshalYAML ensures backwards compatibility defaults when decoding from YAML.
func (a *ExecAction) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Command  string   `yaml:"command"`
		Args     []string `yaml:"args"`
		UseShell *bool    `yaml:"use_shell"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	a.Command = raw.Command
	a.Args = raw.Args
	a.UseShell = raw.UseShell
	return nil
}

func (a *ExecAction) useShell() bool {
	return a.UseShell == nil || *a.UseShell
}

func (a *ExecAction) buildCommand() *exec.Cmd {
	if a.useShell() {
		shell, args := defaultShell()
		combined := append(args, a.Command)
		return exec.Command(shell, combined...)
	}
	return exec.Command(a.Command, a.Args...)
}

func defaultShell() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/C"}
	}
	return "sh", []string{"-c"}
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
