package executors

import (
	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/config"
)

type Executor struct {
	logger *log.Logger
	config *config.Config
}

func New(logger *log.Logger, config *config.Config) *Executor {
	return &Executor{
		logger: logger,
		config: config,
	}
} 