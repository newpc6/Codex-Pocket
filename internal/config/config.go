package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr      string        `yaml:"listen_addr"`
	CodexPath       string        `yaml:"codex_path"`
	ClaudePath      string        `yaml:"claude_path"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	StateDBPath     string        `yaml:"state_db_path"`
	JWTSecret       string        `yaml:"jwt_secret"`
	Users           []UserConfig  `yaml:"users"`
	WebDistPath     string        `yaml:"web_dist_path"`
}

type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func Load() Config {
	cfg := Config{
		ListenAddr:      "0.0.0.0:7318",
		CodexPath:       "codex",
		ClaudePath:      "claude",
		RefreshInterval: 12 * time.Second,
		StateDBPath:     defaultStateDBPath(),
		JWTSecret:       "codexpocket-default-secret-change-me",
		Users:           []UserConfig{{Username: "admin", Password: "admin123"}},
		WebDistPath:     "",
	}

	// Try to load config file
	configPath := findConfigFile()
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to parse config file %s: %v\n", configPath, err)
			} else {
				fmt.Fprintf(os.Stdout, "loaded config from %s\n", configPath)
			}
		}
	}

	// Environment variables override config file
	if v := os.Getenv("CODEXPOCKET_LISTEN_ADDR"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("CODEXPOCKET_CODEX_PATH"); v != "" {
		cfg.CodexPath = v
	}
	if v := os.Getenv("CODEXPOCKET_CLAUDE_PATH"); v != "" {
		cfg.ClaudePath = v
	}
	if v := os.Getenv("CODEXPOCKET_JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("CODEXPOCKET_WEB_DIST_PATH"); v != "" {
		cfg.WebDistPath = v
	}
	if v := os.Getenv("CODEXPOCKET_STATE_DB_PATH"); v != "" {
		cfg.StateDBPath = v
	}
	if v := os.Getenv("CODEXPOCKET_REFRESH_INTERVAL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil && parsed > 0 {
			cfg.RefreshInterval = parsed
		}
	}

	// Auto-detect web dist path if not set
	if cfg.WebDistPath == "" {
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		candidate := filepath.Join(exeDir, "dist")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			cfg.WebDistPath = candidate
		}
	}

	return cfg
}

func findConfigFile() string {
	candidates := []string{
		"codexpocket.yaml",
		"codexpocket.yml",
		"config.yaml",
		"config.yml",
	}

	// Check executable directory first
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		for _, name := range candidates {
			p := filepath.Join(exeDir, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	// Check current working directory
	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}

	return ""
}

func (c *Config) Authenticate(username, password string) bool {
	for _, u := range c.Users {
		if u.Username == username && u.Password == password {
			return true
		}
	}
	return false
}

func defaultStateDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "./codexpocket-state.db"
	}
	return filepath.Join(home, ".codexpocket", "state.db")
}
