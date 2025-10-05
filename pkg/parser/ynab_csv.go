package parser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/yurifrl/ynabu/pkg/models"
)

// ParseYNABCSV parses a CSV exported/created by this tool (Date,Payee,Memo,Amount)
// and converts each line back into a models.Transaction so that the rest of the
// pipeline (plan, reconcile, etc.) can operate transparently on either the
// original statement formats or a previously generated CSV.
func (p *Parser) ParseYNABCSV(data []byte) ([]*models.Transaction, error) {
    r := csv.NewReader(bytes.NewReader(data))
    r.FieldsPerRecord = -1 // allow variable columns – we will validate manually

    records, err := r.ReadAll()
    if err != nil {
        return nil, fmt.Errorf("failed to read csv: %w", err)
    }
    if len(records) == 0 {
        return nil, fmt.Errorf("csv is empty")
    }

    // Expect header: Date,Payee,Memo,Amount
    start := 0
    if len(records[0]) >= 4 && strings.EqualFold(strings.TrimSpace(records[0][0]), "date") {
        start = 1 // skip header
    }

    txs := make([]*models.Transaction, 0, len(records)-start)
    for i := start; i < len(records); i++ {
        rec := records[i]
        if len(rec) < 4 {
            // skip malformed line but log for debug
            p.logger.Debug("csv line has less than 4 fields, skipping", "line", i)
            continue
        }

        dateCSV := strings.TrimSpace(rec[0])
        payee := strings.TrimSpace(rec[1])
        amountStr := strings.TrimSpace(rec[3])

        // Parse amount as float64 (dot as decimal separator)
        amountStr = strings.ReplaceAll(amountStr, ",", ".")
        amount, err := strconv.ParseFloat(amountStr, 64)
        if err != nil {
            p.logger.Debug("invalid amount, skipping", "line", i, "err", err)
            continue
        }

        // Convert date from yyyy/mm/dd (CSV) to dd/mm/yyyy expected by builder.
        dParts := strings.Split(dateCSV, "/")
        if len(dParts) != 3 {
            p.logger.Debug("invalid date format, skipping", "line", i, "date", dateCSV)
            continue
        }
        dmy := fmt.Sprintf("%s/%s/%s", dParts[2], dParts[1], dParts[0])

        // Build transaction as extrato (simpler & enough for reconciliation)
        tx, err := models.NewTransaction().
            SetPayee(payee).
            SetExtrato().
            SetValueFromExtrato(fmt.Sprintf("%.2f", amount)).
            SetDate(dmy).
            SetLineNumber(i).
            Build()
        if err != nil {
            p.logger.Debug("failed to build transaction from csv line", "line", i, "err", err)
            continue
        }
        txs = append(txs, tx)
    }

    return txs, nil
}
