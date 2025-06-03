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
			switch {
			case strings.Contains(strings.ToLower(text), "(titular)"):
				cardType = "titular"
			case strings.Contains(strings.ToLower(text), "(adicional)"):
				cardType = "adicional"
			default:
				cardType = ""
			}
			continue
		}

		// Skip header and total rows
		if strings.ToLower(row[0]) == "data" || strings.Contains(strings.ToLower(row[0]), "total") {
			continue
		}

		// Skip empty rows, section headers and informational rows
		if row[0] == "" || 
			strings.Contains(strings.ToLower(row[0]), "lançamentos") ||
			strings.Contains(strings.ToLower(row[0]), "encargos") ||
			strings.Contains(strings.ToLower(row[0]), "desta fatura") ||
			strings.Contains(strings.ToLower(row[0]), "juros") ||
			strings.Contains(strings.ToLower(row[0]), "compras parceladas") ||
			strings.Contains(strings.ToLower(row[0]), "taxas") ||
			strings.Contains(strings.ToLower(row[0]), "retirada") ||
			strings.Contains(strings.ToLower(row[0]), "limites") ||
			strings.Contains(strings.ToLower(row[0]), "no país") ||
			strings.Contains(strings.ToLower(row[0]), "no exterior") ||
			strings.Contains(strings.ToLower(row[0]), "pagamentos") {
			continue
		}

		// Skip rows without proper date
		if !regexp.MustCompile(`\d{2}/\d{2}`).MatchString(row[0]) {
			continue
		}

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
			continue
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}
