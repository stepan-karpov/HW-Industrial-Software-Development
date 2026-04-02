package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configPath = "/app/config/config.json"
	logPath    = "/app/logs/app.log"
)

type Config struct {
	Port     int    `json:"port"`
	Greeting string `json:"greeting"`
	LogLevel string `json:"log_level"`
}

type Paths struct {
	ConfigPath string
	LogPath    string
}

func ResolvePaths() Paths {
	return Paths{ConfigPath: configPath, LogPath: logPath}
}

func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	if cfg.Port <= 0 || cfg.Port >= 65536 {
		return Config{}, fmt.Errorf("invalid config: port must be 1..65535")
	}
	if strings.TrimSpace(cfg.Greeting) == "" {
		return Config{}, fmt.Errorf("invalid config: greeting must be non-empty")
	}
	if strings.TrimSpace(cfg.LogLevel) == "" {
		return Config{}, fmt.Errorf("invalid config: log_level must be non-empty")
	}

	return cfg, nil
}

func EnsureLogFileExists(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
