package executors

import (
	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/config"
	"github.com/yurifrl/ynabu/pkg/ynab"
)

type Executor struct {
	logger *log.Logger
	config *config.Config
	ynab   *ynab.YNABClient
}

func New(logger *log.Logger, config *config.Config, ynab *ynab.YNABClient) *Executor {
	return &Executor{
		logger: logger,
		config: config,
		ynab:   ynab,
	}
} 