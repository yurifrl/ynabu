package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/brunomvsouza/ynab.go/api"
)

type Transaction struct {
	date            string
	payee           string
	memo            string
	amount          float64
	docType         string
	cardType        string
	cardNumber      string
	err             error
}

func NewTransaction() *Transaction {
	return &Transaction{}
}

func (t *Transaction) SetPayee(payee string) *Transaction {
	t.payee = payee
	return t
}

func (t *Transaction) SetFatura(cardType, cardNumber string) *Transaction {
	t.docType = "fatura"
	t.cardType = strings.TrimSpace(cardType)
	t.cardNumber = strings.TrimSpace(cardNumber)
	return t
}

func (t *Transaction) SetExtrato() *Transaction {
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
		t.memo = fmt.Sprintf("\"%s,%s,%s\"", t.ID(), t.cardType, t.cardNumber)
	} else {
		t.memo = fmt.Sprintf("\"%s,extrato\"", t.ID())
	}
	return t, nil
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

func (t *Transaction) ID() string {
	data := fmt.Sprintf("%s-%s-%.2f", t.date, t.payee, t.amount)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

func (t *Transaction) Date() string {
	return t.date
}

func (t *Transaction) Payee() string {
	transformed := strings.TrimSpace(t.payee)
	if len(transformed) > 5 {
		if match := strings.LastIndex(transformed, "/"); match > 0 && match == len(transformed)-3 {
			if _, err := strconv.Atoi(transformed[match-2:match] + transformed[match+1:]); err == nil {
				transformed = strings.TrimSpace(transformed[:match-2])
			}
		}
	}
	return strings.ToUpper(transformed)
}

func (t *Transaction) Memo() string {
	return t.memo
}

func (t *Transaction) Amount() float64 {
    return t.amount
}

// PayeePointer returns a pointer to the formatted payee or nil when empty.
func (t *Transaction) PayeePointer() *string {
    p := t.Payee()
    if p == "" {
        return nil
    }
    return &p
}

// MemoPointer returns a pointer to the memo or nil when empty.
func (t *Transaction) MemoPointer() *string {
    if t.memo == "" {
        return nil
    }
    m := t.memo
        return &m
}

// APIDate converts the internal date string (yyyy/mm/dd) into an api.Date
// understood by the YNAB SDK.
func (t *Transaction) APIDate() (api.Date, error) {
    return api.DateFromString(strings.ReplaceAll(t.date, "/", "-"))
}

// AmountMilliunits converts the float amount into the integer milliunits used
// by the YNAB API (1000 milliunits == 1 currency unit).
func (t *Transaction) AmountMilliunits() int64 {
    return int64(t.amount * 1000)
}
