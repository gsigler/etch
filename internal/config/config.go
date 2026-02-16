package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	etcherr "github.com/gsigler/etch/internal/errors"
)

const (
	DefaultModel           = "claude-sonnet-4-20250514"
	DefaultComplexityGuide = "small = single focused session, medium = may need iteration, large = multiple sessions likely"

	configPath = ".etch/config.toml"
	envKeyName = "ANTHROPIC_API_KEY"
)

// Config holds all etch configuration.
type Config struct {
	API      APIConfig      `toml:"api"`
	Defaults DefaultsConfig `toml:"defaults"`
}

// APIConfig holds AI provider settings.
type APIConfig struct {
	Model  string `toml:"model"`
	APIKey string `toml:"api_key"`
}

// DefaultsConfig holds default values for plan generation.
type DefaultsConfig struct {
	ComplexityGuide string `toml:"complexity_guide"`
}

// Load reads config from .etch/config.toml relative to the given project root,
// applies defaults, and resolves the API key from the environment if not set
// in the config file.
func Load(projectRoot string) (Config, error) {
	cfg := Config{
		API: APIConfig{
			Model: DefaultModel,
		},
		Defaults: DefaultsConfig{
			ComplexityGuide: DefaultComplexityGuide,
		},
	}

	path := filepath.Join(projectRoot, configPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No config file — use defaults with env var for API key.
		cfg.API.APIKey = os.Getenv(envKeyName)
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, etcherr.WrapConfig("reading config file", err).
			WithHint("check the syntax of " + path)
	}

	// Apply defaults for fields not set in the file.
	if cfg.API.Model == "" {
		cfg.API.Model = DefaultModel
	}
	if cfg.Defaults.ComplexityGuide == "" {
		cfg.Defaults.ComplexityGuide = DefaultComplexityGuide
	}

	// Env var overrides config file API key.
	if envKey := os.Getenv(envKeyName); envKey != "" {
		cfg.API.APIKey = envKey
	}

	return cfg, nil
}

// ResolveAPIKey returns the API key from the config, or an error with a
// helpful message if no key is available.
func (c Config) ResolveAPIKey() (string, error) {
	if c.API.APIKey != "" {
		return c.API.APIKey, nil
	}
	return "", etcherr.Config("no API key found").
		WithHint("set the " + envKeyName + " environment variable or add api_key under [api] in " + configPath)
}
