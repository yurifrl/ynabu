package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/extrame/xls"
	"github.com/yurifrl/ynabu/pkg/models"
)

// Regular expression to extract card number from the card info
var cardNumberRegex = regexp.MustCompile(`final (\d+)`)

// ParseFatura parses an Itaú credit card statement (XLS or TXT) and returns a slice of transactions
func ParseFatura(filePath string) ([]models.Transaction, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".txt" {
		return parseFaturaTxt(filePath)
	}
	return parseFaturaXls(filePath)
}

// parseFaturaTxt parses a TXT bank statement file
func parseFaturaTxt(filePath string) ([]models.Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	var transactions []models.Transaction

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV record: %w", err)
		}

		if len(record) < 3 {
			continue // Skip malformed rows
		}

		// Parse date
		dateStr := strings.TrimSpace(record[0])
		date, err := time.Parse("02/01/2006", dateStr)
		if err != nil {
			continue // Skip rows with invalid dates
		}

		// Get description
		payee := strings.TrimSpace(record[1])

		// Parse value
		valueStr := strings.TrimSpace(record[2])
		valueStr = strings.ReplaceAll(valueStr, ".", "")  // Remove thousand separators
		valueStr = strings.ReplaceAll(valueStr, ",", ".") // Convert decimal separator

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		// Create transaction
		transaction := models.Transaction{
			Date:  date,
			Payee: payee,
			Memo:  fmt.Sprintf("%s,extrato,-", generateTransactionID(date, payee)),
		}

		// Determine if it's inflow or outflow based on the sign
		if value < 0 {
			transaction.Outflow = -value
		} else {
			transaction.Inflow = value
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// parseFaturaXls parses an XLS credit card statement file
func parseFaturaXls(filePath string) ([]models.Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	workbook, err := xls.OpenReader(file, "cp1252")
	if err != nil {
		return nil, fmt.Errorf("error creating workbook: %w", err)
	}

	rows := workbook.ReadAllCells(1000) // Set max to 1000 rows
	if len(rows) == 0 {
		return nil, fmt.Errorf("no data found in sheet")
	}

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
		if strings.Contains(text, " - final ") {
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

		// Parse date
		dateStr := row[0]
		if dateStr == "" {
			continue
		}

		date, err := time.Parse("02/01/2006", dateStr)
		if err != nil {
			continue
		}

		// Parse payee
		payee := strings.TrimSpace(row[1])
		if payee == "" {
			continue
		}

		// Parse value
		valueStr := strings.TrimSpace(strings.ReplaceAll(row[3], "R$ ", ""))
		valueStr = strings.ReplaceAll(valueStr, ",", ".")
		valueStr = strings.TrimSpace(valueStr)
		if valueStr == "" {
			continue
		}

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		// Create transaction
		transaction := models.Transaction{
			Date:    date,
			Payee:   payee,
			Memo:    fmt.Sprintf("%s,%s,%s", generateTransactionID(date, payee), cardType, cardNumber),
			Outflow: value,
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}
