package parser

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/models"
)

type Parser struct {
	logger *log.Logger
}

func New(logger *log.Logger) *Parser {
	return &Parser{
		logger: logger,
	}
}

func (p *Parser) ProcessBytes(data []byte, filename string) ([]*models.Transaction, error) {
	lowerFilename := strings.ToLower(filename)
	p.logger.Debug("processing file", "filename", filename)

	// Check for specific file patterns first (most specific to least specific)
	switch {
	case strings.Contains(lowerFilename, "fatura") && strings.HasSuffix(lowerFilename, ".xls"):
		return p.ParseItauFaturaXLS(data)
	case strings.Contains(lowerFilename, "fatura") && strings.HasSuffix(lowerFilename, ".csv"):
		return p.ParseItauFaturaCSV(data)
	case strings.HasSuffix(lowerFilename, ".txt"):
		return p.ParseItauExtratoTXT(data)
	case strings.HasSuffix(lowerFilename, ".ofx"):
		return p.ParseItauExtratoOFX(data)
	case strings.HasSuffix(lowerFilename, ".xls"):
		return p.ParseItauExtratoXLS(data)
	case strings.HasSuffix(lowerFilename, ".csv"):
		return p.ParseYNABCSV(data)
	default:
		p.logger.Debug("unknown file type", "filename", filename)
		return nil, fmt.Errorf("unknown file type")
	}
}
