package executors

import (
	"github.com/k0kubun/pp/v3"
	"github.com/yurifrl/ynabu/pkg/models"
)

func (e *Executor) Plan(manifest *models.Manifest) error {
	e.logger.Debug("planning manifest")
	pp.Println(e.config)
	pp.Println(manifest)
	return nil
} 