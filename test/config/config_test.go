package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hluaguo/commity/internal/config"
)

func TestDefault(t *testing.T) {
	cfg := config.Default()

	// Test general defaults
	if cfg.General.Mode != "auto" {
		t.Errorf("expected default mode 'auto', got %q", cfg.General.Mode)
	}
	if cfg.General.SplitThreshold != 5 {
		t.Errorf("expected default split threshold 5, got %d", cfg.General.SplitThreshold)
	}

	// Test AI defaults (empty)
	if cfg.AI.Model != "" {
		t.Errorf("expected empty default model, got %q", cfg.AI.Model)
	}
	if cfg.AI.BaseURL != "" {
		t.Errorf("expected empty default base URL, got %q", cfg.AI.BaseURL)
	}
	if cfg.AI.APIKey != "" {
		t.Errorf("expected empty default API key, got %q", cfg.AI.APIKey)
	}

	// Test commit defaults
	if !cfg.Commit.Conventional {
		t.Error("expected conventional commits to be enabled by default")
	}
	expectedTypes := []string{"feat", "fix", "docs", "style", "refactor", "test", "chore"}
	if len(cfg.Commit.Types) != len(expectedTypes) {
		t.Errorf("expected %d commit types, got %d", len(expectedTypes), len(cfg.Commit.Types))
	}
	for i, typ := range expectedTypes {
		if cfg.Commit.Types[i] != typ {
			t.Errorf("expected type %q at index %d, got %q", typ, i, cfg.Commit.Types[i])
		}
	}

	// Test UI defaults
	if cfg.UI.Theme != "tokyonight" {
		t.Errorf("expected default theme 'tokyonight', got %q", cfg.UI.Theme)
	}
}

func TestLoadNonExistent(t *testing.T) {
	// Loading a non-existent config should return defaults
	cfg, err := config.Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("Load should not error for non-existent file: %v", err)
	}

	// Should have default values
	if cfg.General.Mode != "auto" {
		t.Errorf("expected default mode 'auto', got %q", cfg.General.Mode)
	}
	if cfg.UI.Theme != "tokyonight" {
		t.Errorf("expected default theme 'tokyonight', got %q", cfg.UI.Theme)
	}
}

func TestLoadValidConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[general]
mode = "manual"
split_threshold = 10

[ai]
model = "gpt-4"
base_url = "https://api.example.com"
api_key = "test-key"
custom_instructions = "Be concise"

[commit]
conventional = false
types = ["feat", "fix"]

[ui]
theme = "dracula"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded values
	if cfg.General.Mode != "manual" {
		t.Errorf("expected mode 'manual', got %q", cfg.General.Mode)
	}
	if cfg.General.SplitThreshold != 10 {
		t.Errorf("expected split threshold 10, got %d", cfg.General.SplitThreshold)
	}
	if cfg.AI.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %q", cfg.AI.Model)
	}
	if cfg.AI.BaseURL != "https://api.example.com" {
		t.Errorf("expected base URL 'https://api.example.com', got %q", cfg.AI.BaseURL)
	}
	if cfg.AI.APIKey != "test-key" {
		t.Errorf("expected API key 'test-key', got %q", cfg.AI.APIKey)
	}
	if cfg.AI.CustomInstructions != "Be concise" {
		t.Errorf("expected custom instructions 'Be concise', got %q", cfg.AI.CustomInstructions)
	}
	if cfg.Commit.Conventional {
		t.Error("expected conventional to be false")
	}
	if len(cfg.Commit.Types) != 2 {
		t.Errorf("expected 2 commit types, got %d", len(cfg.Commit.Types))
	}
	if cfg.UI.Theme != "dracula" {
		t.Errorf("expected theme 'dracula', got %q", cfg.UI.Theme)
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Write invalid TOML
	if err := os.WriteFile(configPath, []byte("invalid toml [[["), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := config.Load(configPath)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestLoadPartialConfig(t *testing.T) {
	// Partial config should merge with defaults
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[ai]
model = "gpt-3.5-turbo"
api_key = "my-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Custom values should be loaded
	if cfg.AI.Model != "gpt-3.5-turbo" {
		t.Errorf("expected model 'gpt-3.5-turbo', got %q", cfg.AI.Model)
	}
	if cfg.AI.APIKey != "my-key" {
		t.Errorf("expected API key 'my-key', got %q", cfg.AI.APIKey)
	}

	// Defaults should still apply for unset values
	if cfg.General.Mode != "auto" {
		t.Errorf("expected default mode 'auto', got %q", cfg.General.Mode)
	}
	if cfg.UI.Theme != "tokyonight" {
		t.Errorf("expected default theme 'tokyonight', got %q", cfg.UI.Theme)
	}
}

func TestLoadEmptyPath(t *testing.T) {
	// Empty path should use default XDG path (may or may not exist)
	// This test just verifies it doesn't panic
	_, err := config.Load("")
	// We don't check error here as it depends on whether the user has a config
	_ = err
}

func TestConfigPath(t *testing.T) {
	path := config.ConfigPath()
	if path == "" {
		t.Error("ConfigPath should not return empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("ConfigPath should return absolute path, got %q", path)
	}
	if filepath.Base(path) != "config.toml" {
		t.Errorf("ConfigPath should end with config.toml, got %q", filepath.Base(path))
	}
}

func TestLoadEnvVars(t *testing.T) {
	// Set environment variables
	t.Setenv("OPENAI_API_KEY", "env-api-key")
	t.Setenv("OPENAI_BASE_URL", "https://env.example.com")
	t.Setenv("OPENAI_MODEL", "env-model")

	cfg, err := config.Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Env vars should be loaded
	if cfg.AI.APIKey != "env-api-key" {
		t.Errorf("expected API key 'env-api-key', got %q", cfg.AI.APIKey)
	}
	if cfg.AI.BaseURL != "https://env.example.com" {
		t.Errorf("expected base URL 'https://env.example.com', got %q", cfg.AI.BaseURL)
	}
	if cfg.AI.Model != "env-model" {
		t.Errorf("expected model 'env-model', got %q", cfg.AI.Model)
	}
}

func TestLoadEnvVarsOverrideConfig(t *testing.T) {
	// Create a config file with values
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[ai]
model = "config-model"
base_url = "https://config.example.com"
api_key = "config-api-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Set environment variables - these should take priority
	t.Setenv("OPENAI_API_KEY", "env-api-key")
	t.Setenv("OPENAI_BASE_URL", "https://env.example.com")
	t.Setenv("OPENAI_MODEL", "env-model")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Env vars should override config file values
	if cfg.AI.APIKey != "env-api-key" {
		t.Errorf("expected API key 'env-api-key' (from env), got %q", cfg.AI.APIKey)
	}
	if cfg.AI.BaseURL != "https://env.example.com" {
		t.Errorf("expected base URL 'https://env.example.com' (from env), got %q", cfg.AI.BaseURL)
	}
	if cfg.AI.Model != "env-model" {
		t.Errorf("expected model 'env-model' (from env), got %q", cfg.AI.Model)
	}
}

func TestLoadPartialEnvVars(t *testing.T) {
	// Create a config file with some values
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[ai]
model = "config-model"
api_key = "config-api-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Only set one env var
	t.Setenv("OPENAI_API_KEY", "env-api-key")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// API key should come from env
	if cfg.AI.APIKey != "env-api-key" {
		t.Errorf("expected API key 'env-api-key' (from env), got %q", cfg.AI.APIKey)
	}
	// Model should come from config file (no env override)
	if cfg.AI.Model != "config-model" {
		t.Errorf("expected model 'config-model' (from config), got %q", cfg.AI.Model)
	}
}
