package executors

import (
	"github.com/brunomvsouza/ynab.go"
	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/config"
)

type Executor struct {
	logger *log.Logger
	config *config.Config
	ynab   ynab.ClientServicer
}

func New(logger *log.Logger, config *config.Config, ynab ynab.ClientServicer) *Executor {
	return &Executor{
		logger: logger,
		config: config,
		ynab:   ynab,
	}
} 