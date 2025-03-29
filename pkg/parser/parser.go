package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/extrame/xls"
	"github.com/yurifrl/ynabu/pkg/models"
)

type FileType string

const (
	ItauExtratoXLS FileType = "itau_extrato_xls"
	ItauFaturaXLS  FileType = "itau_fatura_xls"
	ItauFaturaTXT  FileType = "itau_fatura_txt"
	ItauExtratoTXT FileType = "itau_extrato_txt"
)

type Parser struct {
	logger *log.Logger
}

func New(logger *log.Logger) *Parser {
	return &Parser{
		logger: logger,
	}
}

func (p *Parser) ProcessBytes(data []byte, filename string) ([]byte, error) {
	fileType := detectType(filename)
	p.logger.Debug("detected file type", "type", fileType, "filename", filename)
	var transactions []models.Transaction
	var err error

	switch fileType {
	case ItauExtratoXLS:
		transactions, err = p.ParseItauExtratoXLS(data)
	case ItauFaturaXLS:
		transactions, err = p.ParseItauFaturaXLS(data)
	case ItauExtratoTXT:
		transactions, err = p.ParseItauExtratoTXT(data)
	default:
		return nil, fmt.Errorf("unknown file type")
	}

	if err != nil {
		return nil, err
	}

	return models.ToCSV(transactions), nil
}

func detectType(filename string) FileType {
	lowerFilename := strings.ToLower(filename)
	if strings.Contains(lowerFilename, "extrato conta corrente") {
		if strings.HasSuffix(lowerFilename, ".xls") {
			return ItauExtratoXLS
		}
		if strings.HasSuffix(lowerFilename, ".txt") {
			return ItauExtratoTXT
		}
	}
	if strings.Contains(lowerFilename, "fatura") && strings.HasSuffix(lowerFilename, ".xls") {
		return ItauFaturaXLS
	}
	return ""
}

// TODO REMOVE
func openXLSFile(filePath string) (*xls.WorkBook, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	workbook, err := xls.OpenReader(file, "cp1252")
	if err != nil {
		return nil, fmt.Errorf("error creating workbook: %w", err)
	}

	return workbook, nil
}
