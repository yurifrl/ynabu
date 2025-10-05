package parser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/yurifrl/ynabu/pkg/models"
)

// ParseItauFaturaCSV parses Itau credit card CSV files with format: data, lançamento, valor
// Expected format: 2025-06-27,IFD*55668457 GABRIEL A,113.98
func (p *Parser) ParseItauFaturaCSV(data []byte) ([]*models.Transaction, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // allow variable columns – we will validate manually

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("csv is empty")
	}

	p.logger.Debug("parsing Itau fatura CSV", "total_records", len(records), "first_record", records[0])

	// Expect header: data, lançamento, valor (3 columns)
	start := 0
	if len(records[0]) >= 3 && (strings.EqualFold(strings.TrimSpace(records[0][0]), "data") ||
		strings.EqualFold(strings.TrimSpace(records[0][0]), "date")) {
		start = 1 // skip header
	}

	txs := make([]*models.Transaction, 0, len(records)-start)
	for i := start; i < len(records); i++ {
		rec := records[i]
		if len(rec) < 3 {
			// skip malformed line but log for debug
			p.logger.Debug("csv line has less than 3 fields, skipping", "line", i)
			continue
		}

		dateCSV := strings.TrimSpace(rec[0])
		payee := strings.TrimSpace(rec[1])
		amountStr := strings.TrimSpace(rec[2])

		// Parse amount as float64 (dot as decimal separator)
		amountStr = strings.ReplaceAll(amountStr, ",", ".")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			p.logger.Debug("invalid amount, skipping", "line", i, "err", err)
			continue
		}

		// Convert date from ISO format (2025-06-27) to dd/mm/yyyy expected by builder
		var dmy string
		if strings.Contains(dateCSV, "-") {
			// ISO format: 2025-06-27 -> 27/06/2025
			dParts := strings.Split(dateCSV, "-")
			if len(dParts) != 3 {
				p.logger.Debug("invalid ISO date format, skipping", "line", i, "date", dateCSV)
				continue
			}
			dmy = fmt.Sprintf("%s/%s/%s", dParts[2], dParts[1], dParts[0])
		} else {
			p.logger.Debug("unsupported date format, skipping", "line", i, "date", dateCSV)
			continue
		}

		// Build transaction as fatura (credit card bill)
		tx, err := models.NewTransaction().
			SetPayee(payee).
			SetFatura("", ""). // CSV format doesn't have card type/number info
			SetValueFromFatura(fmt.Sprintf("%.2f", amount)).
			SetDate(dmy).
			SetLineNumber(i).
			Build()
		if err != nil {
			p.logger.Debug("failed to build transaction from csv line", "line", i, "err", err, "date", dmy, "payee", payee, "amount", amount)
			continue
		}
		p.logger.Debug("created transaction from Itau fatura CSV", "line", i, "date", tx.Date(), "payee", tx.Payee(), "amount", tx.Amount())
		txs = append(txs, tx)
	}

	p.logger.Info("Itau fatura CSV parsing complete", "total_transactions", len(txs), "total_records", len(records))
	return txs, nil
}
