package config

import "github.com/yurifrl/ynabu/pkg/types"

type DefaultConfig struct {
	OutputPath string
}

func (c *DefaultConfig) GetOutputPath() string {
	return c.OutputPath
}

// New creates a new default configuration
func New(outputPath string) types.Config {
	return &DefaultConfig{
		OutputPath: outputPath,
	}
}
