package plan

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type YNABConfig struct {
	BudgetID   string            `yaml:"budget_id"`
	TokenEnv   string            `yaml:"token_env"`
	Accounts   map[string]string `yaml:"accounts"`
}

type Plan struct {
	YNAB       YNABConfig  `yaml:"ynab"`
	Statements []Statement `yaml:"statements"`
}

type Statement struct {
	Type    string `yaml:"type"`
	File    string `yaml:"file"`
	Account string `yaml:"account"`
}

func Load(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var p Plan
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	if len(p.Statements) == 0 {
		return nil, fmt.Errorf("plan has no statements")
	}
	return &p, nil
}

func (p *Plan) Print() {
	fmt.Printf("YNAB budget: %s\n", p.YNAB.BudgetID)
	for i, st := range p.Statements {
		fmt.Printf("[%d] type=%s file=%s account=%s\n", i+1, st.Type, st.File, st.Account)
	}
} 