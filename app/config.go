package app

// AdvancedConfig represents advanced application configuration
type AdvancedConfig struct {
	MaxConcurrentRequests int `mapstructure:"MAX_CONCURRENT_REQUESTS"`
}
