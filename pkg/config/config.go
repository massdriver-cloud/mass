package config

import (
	"errors"
	"os"
	"path/filepath"

	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
	"gopkg.in/yaml.v3"
)

// profileWithTemplates extends the SDK profile with templates_path
type profileWithTemplates struct {
	TemplatesPath string `yaml:"templates_path"`
}

type configFile struct {
	Version  int                         `yaml:"version"`
	Profiles map[string]profileWithTemplates `yaml:"profiles"`
}

// ErrTemplatesPathNotConfigured is returned when templates path is not set via env var or config file
var ErrTemplatesPathNotConfigured = errors.New("templates path not configured: set MD_TEMPLATES_PATH environment variable or templates_path in profile in ~/.config/massdriver/config.yaml. See https://docs.massdriver.cloud/guides/bundle-templates for more info")

// GetTemplatesPath returns the configured templates path with the following precedence:
// 1. MD_TEMPLATES_PATH environment variable
// 2. templates_path from active profile in ~/.config/massdriver/config.yaml
func GetTemplatesPath() (string, error) {
	// Check environment variable first (highest precedence)
	if envPath := os.Getenv("MD_TEMPLATES_PATH"); envPath != "" {
		return envPath, nil
	}

	// Get active profile name from SDK config
	sdkCfg, err := sdkconfig.Get()
	if err != nil {
		// If SDK config fails, try to read profile directly
		profileName := os.Getenv("MASSDRIVER_PROFILE")
		if profileName == "" {
			profileName = "default"
		}
		return getTemplatesPathFromProfile(profileName)
	}

	profileName := sdkCfg.Profile
	if profileName == "" {
		profileName = "default"
	}

	return getTemplatesPathFromProfile(profileName)
}

func getTemplatesPathFromProfile(profileName string) (string, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return "", ErrTemplatesPathNotConfigured
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", ErrTemplatesPathNotConfigured
	}

	var cfg configFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", ErrTemplatesPathNotConfigured
	}

	profile, exists := cfg.Profiles[profileName]
	if !exists || profile.TemplatesPath == "" {
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

// GetConfigDir returns the path to the massdriver config directory,
// creating it if it doesn't exist. Uses XDG_CONFIG_HOME if set,
// otherwise defaults to ~/.config/massdriver
func GetConfigDir() (string, error) {
	var configDir string

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome != "" {
		configDir = filepath.Join(xdgConfigHome, "massdriver")
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config", "massdriver")
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if mkdirErr := os.MkdirAll(configDir, 0755); mkdirErr != nil {
			return "", mkdirErr
		}
	}

	return configDir, nil
}
