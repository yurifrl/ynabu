package parser

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/extrame/xls"
	"github.com/yurifrl/ynabu/pkg/models"
)

func (p *Parser) ParseItauExtratoXLS(data []byte) ([]models.Transaction, error) {
	workbook, err := xls.OpenReader(bytes.NewReader(data), "cp1252")
	if err != nil {
		return nil, fmt.Errorf("error creating workbook: %w", err)
	}

	rows := workbook.ReadAllCells(1000)
	if len(rows) == 0 {
		return nil, fmt.Errorf("no data found in sheet")
	}

	var transactions []models.Transaction
	var foundTransactions bool

	for _, row := range rows {
		if len(row) < 4 {
			continue
		}

		// Skip until we find the transactions section
		if row[0] == "lançamentos" {
			foundTransactions = true
			continue
		}

		if !foundTransactions {
			continue
		}

		// ...
		date := row[0]
		payee := row[1]
		valueStr := row[3]
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		transaction, err := models.NewTransaction(date, payee, value).
			AsExtrato().
			Build()
		if err != nil {
			p.logger.Debug("error building transaction", "row", row, "error", err)
			continue
		}

		transactions = append(transactions, *transaction)
	}

	return transactions, nil
}
