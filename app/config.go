package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultPort     = 8080
	defaultGreeting = "Welcome to the custom app"
	defaultLogLevel = "info"

	defaultConfigPath = "/app/config/config.json"
	defaultLogPath    = "/app/logs/app.log"
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
	p := Paths{
		ConfigPath: defaultConfigPath,
		LogPath:    defaultLogPath,
	}
	if v := strings.TrimSpace(os.Getenv("APP_CONFIG_PATH")); v != "" {
		p.ConfigPath = v
	}
	if v := strings.TrimSpace(os.Getenv("APP_LOG_PATH")); v != "" {
		p.LogPath = v
	}
	return p
}

func LoadConfig(path string) Config {
	cfg := Config{
		Port:     defaultPort,
		Greeting: defaultGreeting,
		LogLevel: defaultLogLevel,
	}

	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}

	if v := strings.TrimSpace(os.Getenv("APP_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 && p < 65536 {
			cfg.Port = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("APP_GREETING")); v != "" {
		cfg.Greeting = v
	}
	if v := strings.TrimSpace(os.Getenv("APP_LOG_LEVEL")); v != "" {
		cfg.LogLevel = v
	}

	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}
	if strings.TrimSpace(cfg.Greeting) == "" {
		cfg.Greeting = defaultGreeting
	}
	if strings.TrimSpace(cfg.LogLevel) == "" {
		cfg.LogLevel = defaultLogLevel
	}
	return cfg
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
