package parser

import (
	"strconv"
	"strings"

	"github.com/yurifrl/ynabu/pkg/models"
)

func (p *Parser) ParseItauExtratoTXT(data []byte) ([]models.Transaction, error) {
	var transactions []models.Transaction
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Split(line, ";")
		if len(fields) < 3 {
			continue
		}

		valueStr := strings.TrimSpace(strings.ReplaceAll(fields[2], ",", "."))
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			p.logger.Debug("error parsing value", "row", line, "error", err)
			continue
		}

		date := fields[0]
		payee := fields[1]
		transaction, err := models.NewTransaction(date, payee, value).
			AsExtrato().
			Build()
		if err != nil {
			p.logger.Debug("error building transaction", "row", line, "error", err)
			continue
		}

		transactions = append(transactions, *transaction)
	}

	return transactions, nil
}
