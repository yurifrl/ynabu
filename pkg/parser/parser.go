package parser

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/models"
)

type FileType string

const (
	ItauExtratoXLS FileType = "itau_extrato_xls"
	ItauFaturaXLS  FileType = "itau_fatura_xls"
	ItauExtratoTXT FileType = "itau_extrato_txt"
	ItauExtratoOFX FileType = "itau_extrato_ofx"
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
	fileType := detectType(filename)
	p.logger.Debug("detected file type", "type", fileType, "filename", filename)

	switch fileType {
	case ItauExtratoXLS:
		return p.ParseItauExtratoXLS(data)
	case ItauFaturaXLS:
		return p.ParseItauFaturaXLS(data)
	case ItauExtratoTXT:
		return p.ParseItauExtratoTXT(data)
	case ItauExtratoOFX:
		return p.ParseItauExtratoOFX(data)
	default:
		p.logger.Debug("unknown file type", "filename", filename)
		return nil, fmt.Errorf("unknown file type")
	}
}

func detectType(filename string) FileType {
	lowerFilename := strings.ToLower(filename)

	// Prioritize explicit keywords when available
	if strings.Contains(lowerFilename, "fatura") && strings.HasSuffix(lowerFilename, ".xls") {
		return ItauFaturaXLS
	}

	if strings.HasSuffix(lowerFilename, ".txt") {
		return ItauExtratoTXT
	}
	if strings.HasSuffix(lowerFilename, ".ofx") {
		return ItauExtratoOFX
	}
	if strings.HasSuffix(lowerFilename, ".xls") {
		return ItauExtratoXLS
	}

	return ""
}
