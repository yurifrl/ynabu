package parser

import (
	"strings"

	"github.com/yurifrl/ynabu/pkg/models"
)

func (p *Parser) ParseItauExtratoTXT(data []byte) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Split(line, ";")
		if len(fields) < 3 {
			continue
		}

		value := fields[2]
		date := fields[0]
		payee := fields[1]

		transaction, err := models.NewTransaction(payee).
			AsExtrato().
			SetValueFromExtrato(value).
			SetDate(date).
			Build()
		if err != nil {
			p.logger.Debug("error building transaction", "row", line, "error", err)
			continue
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}
