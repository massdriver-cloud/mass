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
var envvarError = "%s environment variable exists but has no value, please set a value to continue"

func Get() (*Config, error) {
	ctx := context.Background()
	err := envconfig.Process(ctx, &c)
	if err != nil {
		return nil, fmt.Errorf("required environment variable not set: %w", err)
	}

	if c.APIKey == "" {
		return nil, fmt.Errorf(envvarError, "MASSDRIVER_API_KEY")
	}

	if c.OrgID == "" {
		return nil, fmt.Errorf(envvarError, "MASSDRIVER_ORG_ID")
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

// ServeHTTP returns the config
//
//	@Summary		Get the users config
//	@Description	Get the users config
//	@ID				get-config
//	@Produce		json
//	@Success		200	{object}	config.Config
//	@Router			/config [get]
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
