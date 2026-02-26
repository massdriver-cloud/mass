package config

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile represents a single profile configuration
type Profile struct {
	OrganizationID string `yaml:"organization_id"`
	APIKey         string `yaml:"api_key"`
	URL            string `yaml:"url"`
	TemplatesPath  string `yaml:"templates_path"`
}

// Config represents the CLI configuration file
type Config struct {
	Version  int                `yaml:"version"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// ErrTemplatesPathNotConfigured is returned when templates path is not set via env var or config file
var ErrTemplatesPathNotConfigured = errors.New("templates path not configured: set MD_TEMPLATES_PATH environment variable or templates_path in profile in ~/.config/massdriver/config.yaml. See https://docs.massdriver.cloud/guides/bundle-templates for more info")

// GetActiveProfileName returns the active profile name from MASSDRIVER_PROFILE env var or "default"
func GetActiveProfileName() string {
	if profile := os.Getenv("MASSDRIVER_PROFILE"); profile != "" {
		return profile
	}
	return "default"
}

// Load loads the configuration from ~/.config/massdriver/config.yaml
func Load() (*Config, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Profiles: make(map[string]Profile)}, nil
		}
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}

	return cfg, nil
}

// GetActiveProfile returns the active profile based on MASSDRIVER_PROFILE env var or "default"
func GetActiveProfile() (*Profile, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	profileName := GetActiveProfileName()
	profile, exists := cfg.Profiles[profileName]
	if !exists {
		return nil, errors.New("profile not found: " + profileName)
	}

	return &profile, nil
}

// GetTemplatesPath returns the configured templates path with the following precedence:
// 1. MD_TEMPLATES_PATH environment variable
// 2. templates_path from active profile in ~/.config/massdriver/config.yaml
func GetTemplatesPath() (string, error) {
	// Check environment variable first (highest precedence)
	if envPath := os.Getenv("MD_TEMPLATES_PATH"); envPath != "" {
		return envPath, nil
	}

	// Try to load from active profile
	profile, err := GetActiveProfile()
	if err != nil {
		return "", ErrTemplatesPathNotConfigured
	}

	if profile.TemplatesPath == "" {
		return "", ErrTemplatesPathNotConfigured
	}

	return profile.TemplatesPath, nil
}

func getConfigFilePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}

// GetConfigDir returns the path to the massdriver config directory (~/.config/massdriver),
// creating it if it doesn't exist
func GetConfigDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(usr.HomeDir, ".config", "massdriver")

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if mkdirErr := os.MkdirAll(configDir, 0755); mkdirErr != nil {
			return "", mkdirErr
		}
	}

	return configDir, nil
}
