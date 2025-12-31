package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
)

type Config struct {
	General GeneralConfig `toml:"general"`
	AI      AIConfig      `toml:"ai"`
	Commit  CommitConfig  `toml:"commit"`
	UI      UIConfig      `toml:"ui"`
}

type UIConfig struct {
	Theme string `toml:"theme"` // tokyonight, dracula, catppuccin, nord
}

type GeneralConfig struct {
	Mode           string `toml:"mode"`            // "auto" or "manual"
	SplitThreshold int    `toml:"split_threshold"` // max files before suggesting split
}

type AIConfig struct {
	Model              string `toml:"model"`
	BaseURL            string `toml:"base_url"`
	APIKey             string `toml:"api_key"`
	CustomInstructions string `toml:"custom_instructions"` // custom prompt additions
}

type CommitConfig struct {
	Conventional bool     `toml:"conventional"`
	Types        []string `toml:"types"`
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	return filepath.Join(xdg.ConfigHome, "commity", "config.toml")
}

// Exists checks if config file exists
func Exists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

func Default() *Config {
	return &Config{
		General: GeneralConfig{
			Mode:           "auto",
			SplitThreshold: 5,
		},
		AI: AIConfig{
			Model:   "",
			BaseURL: "",
			APIKey:  "",
		},
		Commit: CommitConfig{
			Conventional: true,
			Types:        []string{"feat", "fix", "docs", "style", "refactor", "test", "chore"},
		},
		UI: UIConfig{
			Theme: "tokyonight",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	// Determine config path
	if path == "" {
		path = filepath.Join(xdg.ConfigHome, "commity", "config.toml")
	}

	// Try to load config file
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, cfg); err != nil {
			return nil, err
		}
	}

	// Environment variables take priority over config file
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.AI.APIKey = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.AI.BaseURL = v
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		cfg.AI.Model = v
	}

	return cfg, nil
}

// Save writes the config to file
func (c *Config) Save() error {
	path := ConfigPath()

	// Create directory if not exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(c)
}
