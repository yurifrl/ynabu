package models

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
)

type Transaction struct {
	date       string
	payee      string
	memo       string
	amount     float64
	docType    string
	cardType   string
	cardNumber string
	logger     *log.Logger
}

func NewTransaction(date string, payee string, value float64) *Transaction {
	return &Transaction{
		date:   strings.TrimSpace(date),
		payee:  strings.TrimSpace(payee),
		amount: value,
	}
}

func (t *Transaction) AsFatura(cardType, cardNumber string) *Transaction {
	t.docType = "fatura"
	t.cardType = strings.TrimSpace(cardType)
	t.cardNumber = strings.TrimSpace(cardNumber)
	return t
}

func (t *Transaction) AsExtrato() *Transaction {
	t.docType = "extrato"
	return t
}

func (t *Transaction) Build() (*Transaction, error) {
	// Validation
	if t.docType == "" {
		return nil, fmt.Errorf("docType is required")
	}
	if t.payee == "" {
		return nil, fmt.Errorf("payee is required")
	}
	if len(t.date) != 10 {
		return nil, fmt.Errorf("date is required")
	}

	// Generate memo
	if t.docType == "fatura" {
		t.memo = fmt.Sprintf("\"%s,%s,%s\"", t.genID(), t.cardType, t.cardNumber)
	} else {
		t.memo = fmt.Sprintf("\"%s,extrato,\"", t.genID())
	}
	return t, nil
}

func (t *Transaction) genID() string {
	data := fmt.Sprintf("%s-%s", t.date, t.payee)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

func (t *Transaction) ToCSV() []byte {
	return []byte(fmt.Sprintf("%s,%s,%s,%.2f\n",
		t.date,
		t.payee,
		t.memo,
		t.amount))
}

func ToCSV(transactions []Transaction) []byte {
	var buf bytes.Buffer
	buf.WriteString("Date,Payee,Memo,Amount\n")
	for _, t := range transactions {
		buf.Write(t.ToCSV())
	}
	return buf.Bytes()
}
