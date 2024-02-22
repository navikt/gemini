package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AzureClientID           string `envconfig:"AZURE_APP_CLIENT_ID" required:"true"`
	DatabaseURL             string `envconfig:"DATABASE_URL" default:"postgres://gemini:gemini@localhost:5432/gemini"`
	BindAddress             string `envconfig:"BIND_ADDRESS" default:"127.0.0.1:3000"`
	MetricsBindAddress      string `envconfig:"METRICS_BIND_ADDRESS" default:"127.0.0.1:3001"`
	DatabaseConnectionLimit int    `envconfig:"DATABASE_CONNECTION_LIMIT" default:"1"`
	AzureClientSecret       string `envconfig:"AZURE_APP_CLIENT_SECRET" required:"true"`
	AzureEndpoint           string `envconfig:"AZURE_APP_TENANT_ID" required:"true"`
	AzureRedirectURL        string `envconfig:"REDIRECT_URL" default:"http://localhost:3000/oauth/callback"`
}

func FromEnvironment() (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process("", cfg)
	return cfg, err
}
