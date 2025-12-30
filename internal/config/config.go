// internal/config/config.go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub GitHubConfig `yaml:"github"`
	APIs   APIConfig    `yaml:"apis"`
	Fetch  FetchConfig  `yaml:"fetch"`
	Digest DigestConfig `yaml:"digest"`
}

type GitHubConfig struct {
	Token string `yaml:"token"`
}

type APIConfig struct {
	LLMProvider string `yaml:"llm_provider"`
	LLMModel    string `yaml:"llm_model"`
}

type FetchConfig struct {
	Concurrency    int `yaml:"concurrency"`
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

type DigestConfig struct {
	DefaultDays int `yaml:"default_days"`
}

func DefaultConfig() *Config {
	return &Config{
		GitHub: GitHubConfig{
			Token: "",
		},
		APIs: APIConfig{
			LLMProvider: "ollama",
			LLMModel:    "llama3.2",
		},
		Fetch: FetchConfig{
			Concurrency:    5,
			TimeoutSeconds: 30,
		},
		Digest: DigestConfig{
			DefaultDays: 7,
		},
	}
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ghmon")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func DBPath() string {
	return filepath.Join(ConfigDir(), "ghmon.db")
}

func (c *Config) Save() error {
	if err := os.MkdirAll(ConfigDir(), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0600)
}

func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Exists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}
