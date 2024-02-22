package config

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type Config struct {
	AzureClientID           string `envconfig:"AZURE_APP_CLIENT_ID" required:"true"`
	DatabaseURL             string `envconfig:"DATABASE_URL" required:"true" default:"postgres://gemini:gemini@localhost:5432/gemini"`
	BindAddress             string `envconfig:"BIND_ADDRESS" default:"127.0.0.1:3000"`
	MetricsBindAddress      string `envconfig:"METRICS_BIND_ADDRESS" default:"127.0.0.1:3001"`
	DatabaseConnectionLimit int    `envconfig:"DATABASE_CONNECTION_LIMIT" default:"1"`
	AzureClientSecret       string `envconfig:"AZURE_APP_CLIENT_SECRET" required:"true"`
	AzureEndpoint           string `envconfig:"AZURE_APP_TENANT_ID" required:"true"`
	AzureRedirectURL        string `envconfig:"REDIRECT_URL" default:"http://localhost:3000/oauth/callback"`
}

func FromEnvironment() (*Config, error) {

	// load config from environment
	cfg := &Config{}
	err := envconfig.Process("", cfg)
	if err != nil {
		return nil, err
	}

	// inject database connection limit into db url
	if len(cfg.DatabaseURL) > 0 && cfg.DatabaseConnectionLimit > 0 {
		cfg.DatabaseURL, err = dbURLWithConnectionLimit(cfg.DatabaseURL, cfg.DatabaseConnectionLimit)
		if err != nil {
			return nil, fmt.Errorf("add connection limit to database url: %w", err)
		}
	}

	return cfg, nil
}

func (cfg *Config) OAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.AzureClientID,
		ClientSecret: cfg.AzureClientSecret,
		Endpoint:     microsoft.AzureADEndpoint(cfg.AzureEndpoint),
		RedirectURL:  cfg.AzureRedirectURL,
		Scopes: []string{
			"Calendars.Read",
			"offline_access",
		},
	}
}

func dbURLWithConnectionLimit(dsn string, limit int) (string, error) {
	// hack to limit connections to database
	dburl, err := url.Parse(dsn)
	if err != nil {
		return dsn, err
	}
	q := dburl.Query()
	q.Add("pool_max_conns", strconv.Itoa(limit))
	dburl.RawQuery = q.Encode()
	return dburl.String(), nil
}
