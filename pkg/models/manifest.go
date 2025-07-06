package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser is an interface that defines the contract for parsing statement files.
type Parser interface {
	ProcessBytes(data []byte, filename string) ([]*Transaction, error)
}

// Manifest represents the structure of the YAML manifest file.
type Manifest struct {
	Statements []Statement `yaml:"statements"`
}

// YNABConfig holds the YNAB specific configurations.
type YNABConfig struct {
	AccountID string `yaml:"account_id"`
	TokenEnv  string `yaml:"token_env"`
}

// Statement represents a single statement to be processed.
type Statement struct {
	Type     string `yaml:"type"`
	FilePath string `yaml:"file"`
	BudgetID string `yaml:"budget_id"`
}

// File returns the absolute path to the statement file, expanding ~.
func (s *Statement) File() (string, error) {
	if strings.HasPrefix(s.FilePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, s.FilePath[2:]), nil
	}
	return s.FilePath, nil
}

// Transactions reads the statement file and uses the provided parser to return transactions.
func (s *Statement) Transactions(p Parser) ([]*Transaction, error) {
	filePath, err := s.File()
	if err != nil {
		return nil, err
	}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read statement file %s: %w", filePath, err)
	}

	transactions, err := p.ProcessBytes(fileBytes, filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to process statement file %s: %w", filePath, err)
	}

	return transactions, nil
}

// FromFile reads a manifest from a YAML file.
func FromFile(filePath string) (*Manifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}