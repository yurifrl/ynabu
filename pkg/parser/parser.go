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

// setTransactionPositions assigns a position index to each transaction within its day
func setTransactionPositions(transactions []*models.Transaction) {
	// Group transactions by date
	dateIndex := make(map[string]int)

	for _, tx := range transactions {
		date := tx.Date()
		position := dateIndex[date]
		tx.SetPosition(position)
		dateIndex[date]++
	}
}

func (p *Parser) ProcessBytes(data []byte, filename string) ([]*models.Transaction, error) {
	lowerFilename := strings.ToLower(filename)
	p.logger.Info("processing file", "filename", filename)

	var transactions []*models.Transaction
	var err error

	// Check for specific file patterns first (most specific to least specific)
	switch {
	case strings.Contains(lowerFilename, "fatura") && strings.HasSuffix(lowerFilename, ".xls"):
		p.logger.Info("using parser: ParseItauFaturaXLS")
		transactions, err = p.ParseItauFaturaXLS(data)
	case strings.Contains(lowerFilename, "fatura") && strings.HasSuffix(lowerFilename, ".csv"):
		p.logger.Info("using parser: ParseItauFaturaCSV")
		transactions, err = p.ParseItauFaturaCSV(data)
	case strings.HasSuffix(lowerFilename, ".txt"):
		p.logger.Info("using parser: ParseItauExtratoTXT")
		transactions, err = p.ParseItauExtratoTXT(data)
	case strings.HasSuffix(lowerFilename, ".ofx"):
		p.logger.Info("using parser: ParseItauExtratoOFX")
		transactions, err = p.ParseItauExtratoOFX(data)
	case strings.HasSuffix(lowerFilename, ".xls"):
		p.logger.Info("using parser: ParseItauExtratoXLS")
		transactions, err = p.ParseItauExtratoXLS(data)
	case strings.HasSuffix(lowerFilename, ".csv"):
		p.logger.Info("using parser: ParseYNABCSV")
		transactions, err = p.ParseYNABCSV(data)
	default:
		p.logger.Info("unknown file type", "filename", filename)
		return nil, fmt.Errorf("unknown file type")
	}

	if err != nil {
		return nil, err
	}

	// Set position for each transaction within its day (centralized)
	setTransactionPositions(transactions)

	return transactions, nil
}
