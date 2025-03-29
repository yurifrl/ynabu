package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/extrame/xls"
	"github.com/yurifrl/ynabu/pkg/models"
)

func (p *Parser) ParseItauFaturaXLS(data []byte) ([]*models.Transaction, error) {
	workbook, err := xls.OpenReader(bytes.NewReader(data), "cp1252")
	if err != nil {
		return nil, fmt.Errorf("error creating workbook: %w", err)
	}

	p.logger.Debug("reading workbook", "sheet_count", workbook.NumSheets())

	rows := workbook.ReadAllCells(1000)
	p.logger.Debug("read rows", "count", len(rows))
	if len(rows) == 0 {
		return nil, fmt.Errorf("no data found in sheet")
	}

	var cardNumberRegex = regexp.MustCompile(`final (\d+)`)
	var transactions []*models.Transaction
	var cardType string
	var cardNumber string
	var foundTransactions bool

	for _, row := range rows {
		if len(row) < 4 {
			continue
		}

		text := strings.TrimSpace(row[0])

		// Check for card holder section
		if strings.Contains(strings.ToLower(text), "total nacional do cartão - final") {
			if matches := cardNumberRegex.FindStringSubmatch(text); len(matches) > 1 {
				cardNumber = matches[1]
			}
			if strings.Contains(strings.ToLower(text), "(titular)") {
				cardType = "titular"
				foundTransactions = true
				continue
			}
			if strings.Contains(strings.ToLower(text), "(adicional)") {
				cardType = "adicional"
				foundTransactions = true
				continue
			}
		}

		if !foundTransactions {
			continue
		}

		// Skip header and total rows
		if strings.ToLower(row[0]) == "data" || strings.Contains(strings.ToLower(row[0]), "total") {
			continue
		}

		// Skip empty rows or section headers
		if row[0] == "" || strings.Contains(strings.ToLower(row[0]), "lançamentos") {
			continue
		}

		// ...
		date := row[0]
		payee := row[1]
		valueStr := row[3]

		// Create transaction
		transaction, err := models.NewTransaction(payee).
			AsFatura(cardType, cardNumber).
			SetValueFromFatura(valueStr).
			SetDate(date).
			Build()
		if err != nil {
			p.logger.Debug("error building transaction", "row", row, "error", err)
			continue
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}
