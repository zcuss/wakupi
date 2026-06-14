package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config controls the embedded Wakupi REST/WebSocket API server.
type Config struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`     // listen address, default 127.0.0.1:8787
	Token   string `yaml:"token"`    // bearer token; auto-generated if empty
	TLSCert string `yaml:"tls_cert"` // optional path to TLS cert
	TLSKey  string `yaml:"tls_key"`  // optional path to TLS key
}

// DefaultConfig returns a config with sane defaults (API enabled on loopback).
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Addr:    "127.0.0.1:8787",
	}
}

// LoadConfig reads api.yaml from dataDir. On first run it generates a random
// token, writes the file with 0600 perms, and reports newToken=true so the
// caller can log the token exactly once.
func LoadConfig(dataDir string) (cfg Config, newToken bool, err error) {
	path := filepath.Join(dataDir, "api.yaml")
	cfg = DefaultConfig()

	data, readErr := os.ReadFile(path)
	if readErr == nil {
		if err = yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, false, fmt.Errorf("parse %s: %w", path, err)
		}
		if cfg.Addr == "" {
			cfg.Addr = DefaultConfig().Addr
		}
	} else if !os.IsNotExist(readErr) {
		return cfg, false, fmt.Errorf("read %s: %w", path, readErr)
	}

	if cfg.Token == "" {
		tok, genErr := generateToken()
		if genErr != nil {
			return cfg, false, genErr
		}
		cfg.Token = tok
		newToken = true
		if err = saveConfig(path, cfg); err != nil {
			return cfg, false, err
		}
	}
	return cfg, newToken, nil
}

func saveConfig(path string, cfg Config) error {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
