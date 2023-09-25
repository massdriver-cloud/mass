package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	OrgID  string `json:"orgID" env:"MASSDRIVER_ORG_ID,required"`
	APIKey string `json:"apiKey" env:"MASSDRIVER_API_KEY,required"`
	URL    string `json:"url" env:"MASSDRIVER_API_URL"`
}

var c Config

func Get() (*Config, error) {
	ctx := context.Background()
	err := envconfig.Process(ctx, &c)
	if err != nil {
		return nil, fmt.Errorf("required environment variable not set: %w", err)
	}

	setDefaults(&c)

	return &c, nil
}

func setDefaults(conf *Config) {
	if conf.URL == "" {
		conf.URL = api.Endpoint
	}
}

func NewHandler() (*Config, error) {
	return Get()
}

func (c *Config) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	out, err := json.Marshal(c)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(out)
	if err != nil {
		slog.Error(err.Error())
	}
}
