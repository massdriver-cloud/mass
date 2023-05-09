package config

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	OrgID  string `env:"MASSDRIVER_ORG_ID,required"`
	APIKey string `env:"MASSDRIVER_API_KEY,required"`
	URL    string `env:"MASSDRIVER_API_URL"`
}

var c Config

func Get() (*Config, error) {
	ctx := context.Background()
	err := envconfig.Process(ctx, &c)
	if err != nil {
		return nil, fmt.Errorf("required environment variable not set: %s", err)
	}

	setDefaults(&c)

	return &c, nil
}

func setDefaults(conf *Config) {
	if conf.URL == "" {
		conf.URL = api.Endpoint
	}
}
