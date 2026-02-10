package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentx-labs/agentx/internal/branding"
	"github.com/spf13/viper"
)

const (
	fileName = "config"
	fileType = "yaml"
)

// Dir returns the path to the AgentX config directory (~/.agentx/).
func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", branding.HomeDir())
	}
	return filepath.Join(home, branding.HomeDir())
}

// FilePath returns the full path to the config file (~/.agentx/config.yaml).
func FilePath() string {
	return filepath.Join(Dir(), fileName+"."+fileType)
}

// EnsureDir creates the config directory if it does not exist.
func EnsureDir() error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}
	return nil
}

// Load initializes Viper to read from the config file and environment.
func Load() {
	viper.SetConfigFile(FilePath())
	viper.SetConfigType(fileType)
	viper.SetEnvPrefix(branding.EnvPrefix())
	viper.AutomaticEnv()

	// Ignore error if config file doesn't exist yet.
	_ = viper.ReadInConfig()
}

// Get returns a config value by key. Returns empty string if not set.
func Get(key string) string {
	return viper.GetString(key)
}

// Set writes a config key-value pair and saves the config file.
func Set(key, value string) error {
	if err := EnsureDir(); err != nil {
		return err
	}

	viper.Set(key, value)

	configFile := FilePath()

	// Create the file if it doesn't exist.
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		f, err := os.Create(configFile)
		if err != nil {
			return fmt.Errorf("creating config file %s: %w", configFile, err)
		}
		f.Close()
	}

	if err := viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
