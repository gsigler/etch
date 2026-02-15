package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	configDir := filepath.Join(dir, ".etch")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		config     string // empty string means no config file
		envKey     string
		wantModel  string
		wantAPIKey string
		wantGuide  string
		wantErr    bool
	}{
		{
			name: "valid full config",
			config: `
[api]
model = "claude-opus-4-20250514"
api_key = "sk-ant-file-key"

[defaults]
complexity_guide = "custom guide"
`,
			wantModel:  "claude-opus-4-20250514",
			wantAPIKey: "sk-ant-file-key",
			wantGuide:  "custom guide",
		},
		{
			name:       "missing config file uses defaults",
			wantModel:  DefaultModel,
			wantAPIKey: "",
			wantGuide:  DefaultComplexityGuide,
		},
		{
			name:       "missing config file with env var",
			envKey:     "sk-ant-env-key",
			wantModel:  DefaultModel,
			wantAPIKey: "sk-ant-env-key",
			wantGuide:  DefaultComplexityGuide,
		},
		{
			name: "env var overrides config file api_key",
			config: `
[api]
api_key = "sk-ant-file-key"
`,
			envKey:     "sk-ant-env-key",
			wantModel:  DefaultModel,
			wantAPIKey: "sk-ant-env-key",
			wantGuide:  DefaultComplexityGuide,
		},
		{
			name:       "empty config file uses defaults",
			config:     "",
			wantModel:  DefaultModel,
			wantAPIKey: "",
			wantGuide:  DefaultComplexityGuide,
		},
		{
			name: "partial config fills defaults",
			config: `
[api]
model = "custom-model"
`,
			wantModel:  "custom-model",
			wantAPIKey: "",
			wantGuide:  DefaultComplexityGuide,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if tt.config != "" || tt.name == "empty config file uses defaults" {
				writeConfig(t, dir, tt.config)
			}

			if tt.envKey != "" {
				t.Setenv(envKeyName, tt.envKey)
			} else {
				t.Setenv(envKeyName, "")
			}

			cfg, err := Load(dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if cfg.API.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", cfg.API.Model, tt.wantModel)
			}
			if cfg.API.APIKey != tt.wantAPIKey {
				t.Errorf("APIKey = %q, want %q", cfg.API.APIKey, tt.wantAPIKey)
			}
			if cfg.Defaults.ComplexityGuide != tt.wantGuide {
				t.Errorf("ComplexityGuide = %q, want %q", cfg.Defaults.ComplexityGuide, tt.wantGuide)
			}
		})
	}
}

func TestResolveAPIKey(t *testing.T) {
	t.Run("returns key when present", func(t *testing.T) {
		cfg := Config{API: APIConfig{APIKey: "sk-ant-test"}}
		key, err := cfg.ResolveAPIKey()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if key != "sk-ant-test" {
			t.Errorf("got %q, want %q", key, "sk-ant-test")
		}
	})

	t.Run("returns helpful error when missing", func(t *testing.T) {
		cfg := Config{}
		_, err := cfg.ResolveAPIKey()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		msg := err.Error()
		if !strings.Contains(msg, envKeyName) {
			t.Errorf("error should mention %s, got: %s", envKeyName, msg)
		}
		if !strings.Contains(msg, configPath) {
			t.Errorf("error should mention %s, got: %s", configPath, msg)
		}
	})
}

func TestLoadInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "this is not valid toml [[[")
	t.Setenv(envKeyName, "")

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid TOML, got nil")
	}
}
