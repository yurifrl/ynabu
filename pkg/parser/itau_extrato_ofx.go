package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yurifrl/ynabu/pkg/models"
)

func (p *Parser) ParseItauExtratoOFX(data []byte) ([]*models.Transaction, error) {
	// Skip header until empty line
	reader := bufio.NewReader(bytes.NewReader(data))
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read header: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	// Read remaining content
	content, err := reader.ReadBytes(0)
	if err != nil && err.Error() != "EOF" {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Extract transactions using regex
	var transactions []*models.Transaction
	trnRegex := regexp.MustCompile(`<STMTTRN>(?s)(.*?)</STMTTRN>`)
	matches := trnRegex.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		block := match[1]
		
		// Extract fields
		getField := func(tag string) string {
			r := regexp.MustCompile(fmt.Sprintf("<%s>([^<\n]*)", tag))
			if m := r.FindStringSubmatch(block); len(m) > 1 {
				return strings.TrimSpace(m[1])
			}
			return ""
		}

		// Get transaction data
		date := getField("DTPOSTED")
		amount := getField("TRNAMT")
		memo := getField("MEMO")

		// Clean up date: convert from YYYYMMDDHHMMSS[-TZ:TZ] to DD/MM/YYYY
		if len(date) >= 8 {
			year := date[0:4]
			month := date[4:6]
			day := date[6:8]
			date = fmt.Sprintf("%s/%s/%s", day, month, year)
		}

		// Build transaction
		tx, err := models.NewTransaction(memo).
			AsExtrato().
			SetValueFromExtrato(amount).
			SetDate(date).
			Build()
		
		if err != nil {
			p.logger.Debug("error building transaction", "error", err)
			continue
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
} 