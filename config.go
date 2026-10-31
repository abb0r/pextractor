package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type connectionConfig struct {
	Server   string `json:"server"`
	Token    string `json:"token"`
	Insecure bool   `json:"insecure"`
}

func (u *ui) loadConfig() {
	cfg, err := readConfig(configPath())
	if err != nil {
		return
	}
	u.server.SetText(cfg.Server)
	u.token.SetText(cfg.Token)
	u.insecure.Value = cfg.Insecure
	u.saved = cfg
}

func (u *ui) persistConfig() {
	cfg := connectionConfig{
		Server:   u.server.Text(),
		Token:    u.token.Text(),
		Insecure: u.insecure.Value,
	}
	if cfg == u.saved {
		return
	}
	if err := writeConfig(configPath(), cfg); err != nil {
		return
	}
	u.saved = cfg
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "pExtractor", "config.json")
}

func readConfig(path string) (connectionConfig, error) {
	var cfg connectionConfig
	if path == "" {
		return cfg, errors.New("no config path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return connectionConfig{}, err
	}
	return cfg, nil
}

func writeConfig(path string, cfg connectionConfig) error {
	if path == "" {
		return errors.New("no config path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
