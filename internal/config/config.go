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

func Get() *Config {
	ctx := context.Background()
	err := envconfig.Process(ctx, &c)
	if err != nil {
		msg := fmt.Sprintf("Required environment variable not set: %s", err)
		panic(msg)
	}

	setDefaults(&c)

	return &c
}

func setDefaults(conf *Config) {
	if conf.URL == "" {
		conf.URL = api.Endpoint
	}
}
