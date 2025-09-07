package executors

import (
	"fmt"

	"github.com/yurifrl/ynabu/pkg/models"
)

// Apply creates the missing transactions for a single statement. The heavy
// reconciliation logic lives in BuildReport; Apply only performs the side
// effects required to bring the account in sync. The caller is responsible for
// looping over multiple statements.
func (e *Executor) Apply(statement *models.Statement) error {
	e.logger.Debug("applying statement", "file", statement.FilePath)

	// Parse local transactions
	localTxs, err := statement.Transactions(e.parser)
	if err != nil {
		return err
	}

	if statement.AccountID == "" {
		return fmt.Errorf("statement %s missing account_id", statement.FilePath)
	}

	// Fetch remote transactions for the account
	remoteTxs, err := e.ynab.Transaction().GetTransactionsByAccount(statement.BudgetID, statement.AccountID, nil)
	if err != nil {
		return err
	}

	report := BuildReport(localTxs, remoteTxs, e.config.UseCustomID)

	toSync := report.TransactionsToSync()
	e.logger.Info("transactions to create", "count", len(toSync), "account_id", statement.AccountID)

	if len(toSync) == 0 {
		return nil // nothing to do
	}

	ts := e.ynab.Transaction()
	batch, err := report.Payloads(statement.AccountID)
	if err != nil {
		return err
	}
	if len(batch) == 0 {
		e.logger.Info("no valid transactions to create after filtering")
		return nil // safety check
	}
	e.logger.Info("sending batch to YNAB API", "count", len(batch), "account_id", statement.AccountID)
	if err := ts.CreateTransactions(statement.BudgetID, batch); err != nil {
		return fmt.Errorf("failed to create transactions: %w", err)
	}
	e.logger.Info("created transactions", "count", len(batch), "account_id", statement.AccountID)

	return nil
}
