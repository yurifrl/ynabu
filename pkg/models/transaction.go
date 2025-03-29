package models

import "time"

// Transaction represents a financial transaction in YNAB format
type Transaction struct {
	Date    time.Time
	Payee   string
	Memo    string
	Outflow float64
	Inflow  float64
}

// BankTransaction represents a raw bank transaction from the statement
type BankTransaction struct {
	Date        time.Time
	Description string
	Value       float64
	Balance     float64
}
