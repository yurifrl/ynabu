package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type Transaction struct {
	date            string
	payee           string
	transformedPayee string
	memo            string
	amount          float64
	docType         string
	cardType        string
	cardNumber      string
	err             error
}

func NewTransaction(payee string) *Transaction {
	transformed := payee
	if len(payee) > 5 {
		if match := strings.LastIndex(payee, "/"); match > 0 && match == len(payee)-3 {
			if _, err := strconv.Atoi(payee[match-2:match] + payee[match+1:]); err == nil {
				transformed = strings.TrimSpace(payee[:match-2])
			}
		}
	}
	return &Transaction{
		payee:           payee,
		transformedPayee: strings.ToUpper(transformed),
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
	if t.err != nil {
		return nil, t.err
	}
	if t.docType == "" {
		return nil, fmt.Errorf("docType is required")
	}
	if t.payee == "" {
		return nil, fmt.Errorf("payee is required")
	}

	// Generate memo
	if t.docType == "fatura" {
		t.memo = fmt.Sprintf("\"%s,%s,%s\"", t.genID(), t.cardType, t.cardNumber)
	} else {
		t.memo = fmt.Sprintf("\"%s,extrato\"", t.genID())
	}
	return t, nil
}

func (t *Transaction) genID() string {
	data := fmt.Sprintf("%s-%s-%.2f", t.date, t.payee, t.amount)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

func (t *Transaction) SetValueFromExtrato(valueStr string) *Transaction {
	valueStr = strings.TrimSpace(strings.ReplaceAll(valueStr, ",", "."))
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		t.err = err
		return t
	}

	t.amount = value
	return t
}

func (t *Transaction) SetValueFromFatura(valueStr string) *Transaction {
	valueStr = strings.ReplaceAll(strings.ReplaceAll(valueStr, "R$ ", ""), ",", ".")
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		t.err = err
		return t
	}

	t.amount = -value
	return t
}

func (t *Transaction) SetDate(date string) *Transaction {
	t.date = strings.TrimSpace(date)
	if len(t.date) != 10 {
		t.err = fmt.Errorf("date is required")
		return t
	}

	t.date = fmt.Sprintf("%s/%s/%s", t.date[6:10], t.date[3:5], t.date[0:2])
	return t
}

func (t *Transaction) Date() string {
	return t.date
}

func (t *Transaction) Payee() string {
	return t.transformedPayee
}

func (t *Transaction) Memo() string {
	return t.memo
}

func (t *Transaction) Amount() float64 {
	return t.amount
}
