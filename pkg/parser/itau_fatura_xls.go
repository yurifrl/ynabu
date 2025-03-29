package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/extrame/xls"
	"github.com/yurifrl/ynabu/pkg/models"
)

func (p *Parser) ParseItauFaturaXLS(data []byte) ([]models.Transaction, error) {
	workbook, err := xls.OpenReader(bytes.NewReader(data), "cp1252")
	if err != nil {
		return nil, fmt.Errorf("error creating workbook: %w", err)
	}

	rows := workbook.ReadAllCells(1000)
	if len(rows) == 0 {
		return nil, fmt.Errorf("no data found in sheet")
	}

	var cardNumberRegex = regexp.MustCompile(`final (\d+)`)
	var transactions []models.Transaction
	var cardType string
	var cardNumber string
	var foundTransactions bool

	for _, row := range rows {
		if len(row) < 4 {
			continue
		}

		// Check for card holder section
		text := strings.TrimSpace(row[0])
		if strings.Contains(text, "  final ") {
			if strings.HasSuffix(text, "(titular)") {
				cardType = "titular"
				if matches := cardNumberRegex.FindStringSubmatch(text); len(matches) > 1 {
					cardNumber = matches[1]
				}
				foundTransactions = true
				continue
			}
			if strings.HasSuffix(text, "(adicional)") {
				cardType = "adicional"
				if matches := cardNumberRegex.FindStringSubmatch(text); len(matches) > 1 {
					cardNumber = matches[1]
				}
				foundTransactions = true
				continue
			}
		}

		if !foundTransactions {
			continue
		}

		// Skip header and total rows
		if row[0] == "data" || strings.Contains(strings.ToLower(row[0]), "total") {
			continue
		}

		// Skip empty rows or section headers
		if row[0] == "" || strings.Contains(strings.ToLower(row[0]), "lançamentos") {
			continue
		}

		// ...
		payee := row[1]
		if payee == "" {
			p.logger.Info("payee is empty", "row", row)
			continue
		}

		// ...
		date := row[0]
		if date == "" {
			p.logger.Info("date is empty", "row", row)
			continue
		}

		valueStr := strings.ReplaceAll(strings.ReplaceAll(row[3], "R$ ", ""), ",", ".")
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			p.logger.Info("error parsing value", "row", row, "error", err)
			continue
		}

		// Create transaction
		transaction, err := models.NewTransaction(date, payee, value).
			AsFatura(cardType, cardNumber).
			Build()
		if err != nil {
			p.logger.Debug("error building transaction", "row", row, "error", err)
			continue
		}

		transactions = append(transactions, *transaction)
	}

	return transactions, nil
}
