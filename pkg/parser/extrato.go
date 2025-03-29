package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/extrame/xls"
	"github.com/yurifrl/ynabu/pkg/models"
)

// FileType represents the type of bank file being processed
type FileType string

const (
	ExtratoItauXls FileType = "extrato itau xls"
	FaturaItauXls  FileType = "fatura itau xls"
	FaturaItauTxt  FileType = "fatura itau txt"
)

// DetectFileType determines the type of file based on extension and content
func DetectFileType(filePath string) (FileType, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".txt" {
		return FaturaItauTxt, nil
	}

	if ext != ".xls" {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}

	workbook, err := xls.OpenReader(file, "cp1252")
	if err != nil {
		return "", fmt.Errorf("error creating workbook: %w", err)
	}

	rows := workbook.ReadAllCells(10) // Only need first few rows to detect type
	if len(rows) == 0 {
		return "", fmt.Errorf("no data found in sheet")
	}

	// Look for characteristic markers in the first few rows
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}

		// Check for Extrato markers
		if row[0] == "lançamentos" {
			return ExtratoItauXls, nil
		}

		// Check for Fatura markers
		if strings.Contains(strings.ToLower(row[0]), "fatura") {
			return FaturaItauXls, nil
		}
	}

	return "", fmt.Errorf("unknown file type")
}

// ParseExtrato parses a bank statement XLS file and returns a slice of transactions
func ParseExtrato(filePath string) ([]models.Transaction, error) {
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

		// Skip header and empty rows
		if row[0] == "data" || row[1] == "SALDO TOTAL DISPONÃ" || row[1] == "SALDO ANTERIOR" {
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

		// Parse value
		valueStr := strings.TrimSpace(strings.ReplaceAll(row[3], ",", "."))
		if valueStr == "" {
			continue
		}

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		// Create transaction - all values are outflow
		transaction := models.Transaction{
			Date:    date,
			Payee:   strings.TrimSpace(row[1]),
			Memo:    fmt.Sprintf("%s,extrato,-", generateTransactionID(date, strings.TrimSpace(row[1]))),
			Outflow: value,
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// WriteYNABCSV writes transactions in YNAB CSV format
func WriteYNABCSV(writer io.Writer, transactions []models.Transaction) error {
	csvWriter := csv.NewWriter(writer)

	// Write header
	if err := csvWriter.Write([]string{"Date", "Payee", "Memo", "Outflow", "Inflow"}); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Write transactions
	for _, t := range transactions {
		outflow := ""
		if t.Outflow > 0 {
			outflow = fmt.Sprintf("R$ %.2f", t.Outflow)
			outflow = strings.ReplaceAll(outflow, ".", ",")
		}

		inflow := ""
		if t.Inflow > 0 {
			inflow = fmt.Sprintf("R$ %.2f", t.Inflow)
			inflow = strings.ReplaceAll(inflow, ".", ",")
		}

		record := []string{
			t.Date.Format("2006-01-02"),
			t.Payee,
			t.Memo,
			outflow,
			inflow,
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("error writing transaction: %w", err)
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

// ParseCSV parses a CSV file and returns a slice of transactions
func ParseCSV(filePath string) ([]models.Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';' // Try semicolon first

	// Read header
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("empty file")
		}
		// Try with comma separator
		file.Seek(0, 0)
		reader = csv.NewReader(file)
		header, err = reader.Read()
		if err != nil {
			return nil, fmt.Errorf("error reading CSV header: %w", err)
		}
	}

	// Find column indices
	dateIdx := -1
	descIdx := -1
	valueIdx := -1

	for i, h := range header {
		h = strings.ToLower(strings.TrimSpace(h))
		switch {
		case strings.Contains(h, "data"):
			dateIdx = i
		case strings.Contains(h, "descrição") || strings.Contains(h, "descricao") || strings.Contains(h, "histórico") || strings.Contains(h, "historico"):
			descIdx = i
		case strings.Contains(h, "valor") || strings.Contains(h, "amount"):
			valueIdx = i
		}
	}

	if dateIdx == -1 || valueIdx == -1 {
		return nil, fmt.Errorf("required columns not found in CSV")
	}

	var transactions []models.Transaction
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV record: %w", err)
		}

		if len(record) <= valueIdx || len(record) <= dateIdx {
			continue // Skip malformed rows
		}

		// Parse date
		dateStr := strings.TrimSpace(record[dateIdx])
		if dateStr == "" {
			continue
		}

		// Try different date formats
		var date time.Time
		dateFormats := []string{
			"02/01/2006",
			"2006-01-02",
			"02-01-2006",
			"02/01/2006 15:04:05",
			"2006-01-02 15:04:05",
		}

		for _, format := range dateFormats {
			if parsedDate, err := time.Parse(format, dateStr); err == nil {
				date = parsedDate
				break
			}
		}

		if date.IsZero() {
			continue // Skip if we couldn't parse the date
		}

		// Parse description/payee
		payee := ""
		if descIdx != -1 && len(record) > descIdx {
			payee = strings.TrimSpace(record[descIdx])
		}

		// Parse value
		valueStr := strings.TrimSpace(record[valueIdx])
		valueStr = strings.ReplaceAll(valueStr, "R$", "")
		valueStr = strings.ReplaceAll(valueStr, ".", "")  // Remove thousand separators
		valueStr = strings.ReplaceAll(valueStr, ",", ".") // Convert decimal separator
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
			Date:  date,
			Payee: payee,
			Memo:  fmt.Sprintf("%s,csv,-", generateTransactionID(date, payee)),
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
