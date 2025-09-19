package database

// ActionConfig is a generic struct for unmarshalling actions from YAML.
type ActionConfig struct {
	Type string                 `yaml:"type"`
	Args map[string]interface{} `yaml:"args"`
}
