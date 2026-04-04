package server

// ServerConfig represents the configuration for the server
type ServerConfig struct {
	Host             string `mapstructure:"SERVER_HOST"`
	Port             int    `mapstructure:"SERVER_PORT"`
	TLSEnabled       bool   `mapstructure:"SERVER_TLS_ENABLED"`
	TLSCert          string `mapstructure:"SERVER_TLS_CERT"`
	TLSKey           string `mapstructure:"SERVER_TLS_KEY"`
	WebhookSecret    string `mapstructure:"SERVER_WEBHOOK_SECRET"`
	InternalAPIToken string `mapstructure:"SERVER_INTERNAL_API_TOKEN"`
	LogRequests      bool   `mapstructure:"SERVER_LOG_REQUESTS"`
}
