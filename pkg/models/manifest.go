package models

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest represents the structure of the YAML manifest file.
type Manifest struct {
	YNAB       YNABConfig  `yaml:"ynab"`
	Statements []Statement `yaml:"statements"`
}

// YNABConfig holds the YNAB specific configurations.
type YNABConfig struct {
	BudgetID string            `yaml:"budget_id"`
	TokenEnv string            `yaml:"token_env"`
	Accounts map[string]string `yaml:"accounts"`
}

// Statement represents a single statement to be processed.
type Statement struct {
	Type    string `yaml:"type"`
	File    string `yaml:"file"`
	Account string `yaml:"account"`
}

// FromFile reads a manifest from a YAML file.
func FromFile(filePath string) (*Manifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	expandedData := os.ExpandEnv(string(data))

	var manifest Manifest
	err = yaml.Unmarshal([]byte(expandedData), &manifest)
	if err != nil {
		return nil, err
	}

	for i := range manifest.Statements {
		if strings.HasPrefix(manifest.Statements[i].File, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			manifest.Statements[i].File = filepath.Join(home, manifest.Statements[i].File[2:])
		}
	}

	return &manifest, nil
} 