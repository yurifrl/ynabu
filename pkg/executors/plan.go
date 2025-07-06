package executors

import (
	"github.com/k0kubun/pp/v3"
	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/parser"
)

func (e *Executor) Plan(manifest *models.Manifest) error {
	e.logger.Debug("planning manifest")
	p := parser.New(e.logger)
	pp.Println(manifest)
	pp.Println(e.config)
	for _, statement := range manifest.Statements {
		transactions, err := statement.Transactions(p)
		if err != nil {
			return err
		}

		pp.Println(transactions[0])
		filePath, err := statement.File()
		if err != nil {
			return err
		}
		pp.Println(filePath)
	}

	return nil
} 