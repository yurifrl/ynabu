package ynab

import (
	"strings"

	"github.com/brunomvsouza/ynab.go"
	"github.com/brunomvsouza/ynab.go/api/account"
	"github.com/brunomvsouza/ynab.go/api/budget"
	"github.com/brunomvsouza/ynab.go/api/transaction"
)

// YNABClient wraps the original YNAB client and adds custom functionality
type YNABClient struct {
	client ynab.ClientServicer
}

// TransactionService wraps the original transaction service
type TransactionService struct {
	client   *YNABClient
	original *transaction.Service
}

// Transaction wraps the core YNAB transaction adding CustomID extracted from
// the memo first CSV field.
type Transaction struct {
	*transaction.Transaction
	customID string
}

// TODO: centralise custom ID generation/usage in one place.
func extractCustomID(tx *transaction.Transaction) string {
	if tx == nil || tx.Memo == nil {
		return ""
	}
	memo := strings.Trim(*tx.Memo, "\"")
	if idx := strings.Index(memo, ","); idx > 0 {
		return memo[:idx]
	}
	return ""
}

func New(token string) *YNABClient {
	return &YNABClient{
		client: ynab.NewClient(token),
	}
}

func (c *YNABClient) Transaction() *TransactionService {
	return &TransactionService{
		client:   c,
		original: c.client.Transaction(),
	}
}

func (c *YNABClient) Budget() *budget.Service {
	return c.client.Budget()
}

func (c *YNABClient) Account() *account.Service {
	return c.client.Account()
}

func (ts *TransactionService) GetTransactionsByAccount(budgetID, accountID string, filter interface{}) ([]*Transaction, error) {
	// Call the original client
	var filterPtr *transaction.Filter
	if filter != nil {
		if f, ok := filter.(*transaction.Filter); ok {
			filterPtr = f
		}
	}
	originalTransactions, err := ts.original.GetTransactionsByAccount(budgetID, accountID, filterPtr)
	if err != nil {
		return nil, err
	}

	// Convert to our extended Transaction type with TransactionID
	transactions := make([]*Transaction, 0, len(originalTransactions))
	for _, tx := range originalTransactions {
		customID := extractCustomID(tx)
		transactions = append(transactions, &Transaction{Transaction: tx, customID: customID})
	}

	return transactions, nil
}

// CreateTransactions creates multiple transactions in one API call
func (ts *TransactionService) CreateTransactions(budgetID string, payloads []transaction.PayloadTransaction) error {
	if len(payloads) == 0 {
		return nil
	}
	_, err := ts.original.CreateTransactions(budgetID, payloads)
	return err
}

func (t *Transaction) CustomID() string {
	return t.customID
}
