package models

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