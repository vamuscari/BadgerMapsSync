package action

import (
	"badgermaps/api"
	"badgermaps/database"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Executor is responsible for executing actions.
type Executor struct {
	DB  database.DB
	API *api.APIClient
	ctx *ExecutionContext
}

// NewExecutor creates a new Executor.
func NewExecutor(db database.DB, api *api.APIClient) *Executor {
	return &Executor{
		DB:  db,
		API: api,
	}
}

// WithContext returns a shallow copy carrying the provided execution context.
func (e *Executor) WithContext(ctx *ExecutionContext) *Executor {
	if e == nil {
		return nil
	}
	clone := *e
	clone.ctx = ctx
	return &clone
}

// Context exposes the execution context for the current action run.
func (e *Executor) Context() *ExecutionContext {
	if e == nil {
		return nil
	}
	return e.ctx
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

// ExecutionContext carries event metadata into action executions.
type ExecutionContext struct {
	EventType string
	Source    string
	Payload   interface{}

	payloadGeneric interface{}
	payloadOnce    sync.Once
	payloadErr     error
}

func (c *ExecutionContext) envelope() map[string]interface{} {
	if c == nil {
		return nil
	}
	return map[string]interface{}{
		"type":    c.EventType,
		"source":  c.Source,
		"payload": c.Payload,
	}
}

// EventJSON returns the JSON encoded event envelope.
func (c *ExecutionContext) EventJSON() (string, error) {
	if c == nil {
		return "", nil
	}
	data, err := json.Marshal(c.envelope())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// PayloadJSON returns the JSON encoded payload component.
func (c *ExecutionContext) PayloadJSON() (string, error) {
	if c == nil {
		return "", nil
	}
	data, err := json.Marshal(c.Payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *ExecutionContext) payloadText() string {
	if c == nil || c.Payload == nil {
		return ""
	}
	return fmt.Sprintf("%v", c.Payload)
}

func (c *ExecutionContext) payloadRoot() interface{} {
	if c == nil || c.Payload == nil {
		return nil
	}
	c.payloadOnce.Do(func() {
		c.payloadGeneric, c.payloadErr = normalisePayloadValue(c.Payload)
	})
	if c.payloadErr != nil {
		return nil
	}
	return c.payloadGeneric
}

func (c *ExecutionContext) payloadFieldValue(path string) (interface{}, bool) {
	root := c.payloadRoot()
	if root == nil || strings.TrimSpace(path) == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var current interface{} = root
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		switch value := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = value[part]
			if !ok {
				return nil, false
			}
		case map[string]string:
			var ok bool
			var str string
			str, ok = value[part]
			if !ok {
				return nil, false
			}
			current = str
		case []interface{}:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(value) {
				return nil, false
			}
			current = value[idx]
		default:
			return nil, false
		}
	}
	return current, true
}

func (c *ExecutionContext) PayloadFieldString(path string) (string, bool) {
	value, ok := c.payloadFieldValue(path)
	if !ok {
		return "", false
	}
	return stringifyValue(value)
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
	ctx := executor.Context()
	command := a.Command
	args := a.Args

	if ctx != nil {
		command = replaceEventTokens(command, ctx)
		if !a.useShell() && len(args) > 0 {
			replaced := make([]string, len(args))
			for i, arg := range args {
				replaced[i] = replaceEventTokens(arg, ctx)
			}
			args = replaced
		}
	}

	cmd := a.buildCommand(command, args)
	if cmd.Dir == "" {
		if cwd, err := os.Getwd(); err == nil {
			cmd.Dir = cwd
		}
	}

	if ctx != nil {
		env := append([]string{}, os.Environ()...)
		env = append(env, formatEventEnv(ctx)...)
		cmd.Env = env
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

func (a *ExecAction) buildCommand(command string, args []string) *exec.Cmd {
	if a.useShell() {
		shell, shellArgs := defaultShell()
		combined := append(shellArgs, command)
		return exec.Command(shell, combined...)
	}
	return exec.Command(command, args...)
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
	ctx := executor.Context()
	args := cloneArgsMap(a.config.Args)

	if ctx != nil {
		applyContextToDBArgs(args, ctx)
	}

	dbActionConfig := database.ActionConfig{
		Type: a.config.Type,
		Args: args,
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
	ctx := executor.Context()
	method := a.Method
	endpoint := a.Endpoint
	data := a.Data

	if ctx != nil {
		method = replaceEventTokens(method, ctx)
		endpoint = replaceEventTokens(endpoint, ctx)
		if len(data) > 0 {
			copied := make(map[string]string, len(data))
			for k, v := range data {
				copied[k] = replaceEventTokens(v, ctx)
			}
			data = copied
		}
	}

	_, err := executor.API.RawRequest(method, endpoint, data)
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

func formatEventEnv(ctx *ExecutionContext) []string {
	if ctx == nil {
		return nil
	}
	env := []string{
		"BADGER_EVENT_TYPE=" + ctx.EventType,
		"BADGER_EVENT_SOURCE=" + ctx.Source,
	}
	if eventJSON, err := ctx.EventJSON(); err == nil {
		env = append(env, "BADGER_EVENT_JSON="+eventJSON)
	}
	if payloadJSON, err := ctx.PayloadJSON(); err == nil {
		env = append(env, "BADGER_EVENT_PAYLOAD_JSON="+payloadJSON)
	}
	if payload := ctx.payloadText(); payload != "" {
		env = append(env, "BADGER_EVENT_PAYLOAD="+payload)
	}
	return env
}

func replaceEventTokens(input string, ctx *ExecutionContext) string {
	if ctx == nil {
		return input
	}
	replacements := map[string]string{
		"$EVENT_TYPE":   ctx.EventType,
		"$EVENT_SOURCE": ctx.Source,
	}
	if eventJSON, err := ctx.EventJSON(); err == nil {
		replacements["$EVENT_JSON"] = eventJSON
	}
	if payloadJSON, err := ctx.PayloadJSON(); err == nil {
		replacements["$EVENT_PAYLOAD_JSON"] = payloadJSON
	}
	if payload := ctx.payloadText(); payload != "" {
		replacements["$EVENT_PAYLOAD"] = payload
	}
	result := replacePayloadFieldTokens(input, ctx)
	for token, value := range replacements {
		if value == "" {
			continue
		}
		result = strings.ReplaceAll(result, token, value)
	}
	return result
}

func cloneArgsMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return map[string]interface{}{}
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = cloneValue(v)
	}
	return dst
}

func cloneValue(v interface{}) interface{} {
	switch typed := v.(type) {
	case []interface{}:
		copySlice := make([]interface{}, len(typed))
		for i, item := range typed {
			copySlice[i] = cloneValue(item)
		}
		return copySlice
	case map[string]interface{}:
		copyMap := make(map[string]interface{}, len(typed))
		for k, item := range typed {
			copyMap[k] = cloneValue(item)
		}
		return copyMap
	default:
		return typed
	}
}

func applyContextToDBArgs(args map[string]interface{}, ctx *ExecutionContext) {
	if args == nil || ctx == nil {
		return
	}
	for _, key := range []string{"command", "function", "procedure", "query"} {
		if value, ok := args[key].(string); ok {
			args[key] = replaceEventTokens(value, ctx)
		}
	}
	if values, ok := args["args"].([]interface{}); ok {
		args["args"] = applyContextToValue(values, ctx)
	}
	if _, ok := args["event_type"]; !ok && ctx.EventType != "" {
		args["event_type"] = ctx.EventType
	}
	if _, ok := args["event_source"]; !ok && ctx.Source != "" {
		args["event_source"] = ctx.Source
	}
	if _, ok := args["event_json"]; !ok {
		if eventJSON, err := ctx.EventJSON(); err == nil {
			args["event_json"] = eventJSON
		}
	}
	if _, ok := args["event_payload_json"]; !ok {
		if payloadJSON, err := ctx.PayloadJSON(); err == nil {
			args["event_payload_json"] = payloadJSON
		}
	}
}

func applyContextToValue(value interface{}, ctx *ExecutionContext) interface{} {
	switch typed := value.(type) {
	case string:
		return replaceEventTokens(typed, ctx)
	case []interface{}:
		copySlice := make([]interface{}, len(typed))
		for i, item := range typed {
			copySlice[i] = applyContextToValue(item, ctx)
		}
		return copySlice
	case map[string]interface{}:
		copyMap := make(map[string]interface{}, len(typed))
		for k, item := range typed {
			copyMap[k] = applyContextToValue(item, ctx)
		}
		return copyMap
	default:
		return typed
	}
}

var payloadFieldTokenPattern = regexp.MustCompile(`\$EVENT_PAYLOAD\[([^\]]+)\]`)

func replacePayloadFieldTokens(input string, ctx *ExecutionContext) string {
	if ctx == nil {
		return payloadFieldTokenPattern.ReplaceAllString(input, "")
	}
	return payloadFieldTokenPattern.ReplaceAllStringFunc(input, func(match string) string {
		submatches := payloadFieldTokenPattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return ""
		}
		path := strings.TrimSpace(submatches[1])
		if path == "" {
			return ""
		}
		if field, ok := ctx.PayloadFieldString(path); ok {
			return field
		}
		return ""
	})
}

func normalisePayloadValue(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case map[string]interface{}:
		return v, nil
	case map[string]string:
		converted := make(map[string]interface{}, len(v))
		for key, val := range v {
			converted[key] = val
		}
		return converted, nil
	case []interface{}:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	case error:
		return v.Error(), nil
	case json.Number:
		return v, nil
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return v, nil
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return v, err
		}
		var decoded interface{}
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			return v, err
		}
		return decoded, nil
	}
}

func stringifyValue(value interface{}) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	case json.Number:
		return v.String(), true
	case bool:
		return strconv.FormatBool(v), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case int:
		return strconv.FormatInt(int64(v), 10), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), true
		}
		return string(bytes), true
	}
}
