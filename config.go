package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Repo             string        `yaml:"repo"`
	Branch           string        `yaml:"branch"`
	PollInterval     time.Duration `yaml:"poll_interval"`
	DeployDir        string        `yaml:"deploy_dir"`
	SrcDir           string        `yaml:"src_dir"`
	BuildTimeout     time.Duration `yaml:"build_timeout"`
	RollbackVersions int           `yaml:"rollback_versions"`
	BinaryName       string        `yaml:"binary_name"`
	ServiceName      string        `yaml:"service_name"`
	ProtectFiles     []string      `yaml:"protect_files"`
}

func DefaultConfig() *Config {
	return &Config{
		Repo:             "https://github.com/tillknuesting/phubot.git",
		Branch:           "main",
		PollInterval:     60 * time.Second,
		DeployDir:        "/opt/phubot",
		SrcDir:           "/opt/phubot/src",
		BuildTimeout:     120 * time.Second,
		RollbackVersions: 3,
		BinaryName:       "phubot",
		ServiceName:      "phubot",
		ProtectFiles:     []string{"config.json", ".phubot"},
	}
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found: %s", path)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
