package ui

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"hexajobs.dev/hexajobs-cli/internal/client"
)

type Preferences struct {
	Language   string `json:"language"`
	Region     string `json:"region"`
	ScamShield bool   `json:"scamshield"`
}

func DefaultPreferences() Preferences {
	return Preferences{Language: "en", Region: "global", ScamShield: true}
}
func (p Preferences) Validate() error {
	if p.Language != "en" && p.Language != "id" && p.Language != "jp" {
		return errors.New("unsupported UI language")
	}
	switch p.Region {
	case "global", "us", "eu", "asia", "id":
	default:
		return errors.New("unsupported UI region")
	}
	return nil
}

func preferencesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(root) {
		root = filepath.Join(home, ".config")
	}
	return filepath.Join(root, "hexajobs", "ui.json"), nil
}

func LoadPreferences() (Preferences, error) {
	p := DefaultPreferences()
	path, err := preferencesPath()
	if err != nil {
		return p, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return p, err
	}
	if len(data) > 65536 {
		return p, errors.New("UI preferences too large")
	}
	if err = json.Unmarshal(data, &p); err != nil {
		return DefaultPreferences(), errors.New("invalid UI preferences JSON")
	}
	if err = p.Validate(); err != nil {
		return DefaultPreferences(), err
	}
	return p, nil
}

func SavePreferences(p Preferences) error {
	if err := p.Validate(); err != nil {
		return err
	}
	path, err := preferencesPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return client.WritePrivateFile(path, data)
}
