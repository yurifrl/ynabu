package executors

import (
	"fmt"

	"github.com/yurifrl/ynabu/pkg/models"
)

func (e *Executor) Apply(manifest *models.Manifest) error {
    e.logger.Debug("applying manifest")

    for _, statement := range manifest.Statements {
		// Parse local transactions
		localTxs, err := statement.Transactions(e.parser)
		if err != nil {
			return err
		}

		if statement.AccountID == "" {
			return fmt.Errorf("manifest error: statement %s missing account_id", statement.FilePath)
		}

		// Fetch remote transactions for the account
		remoteTxs, err := e.ynab.Transaction().GetTransactionsByAccount(e.config.YNAB.BudgetID, statement.AccountID, nil)
		if err != nil {
			return err
		}

		report := BuildReport(localTxs, remoteTxs, e.config.UseCustomID)

        toSync := report.TransactionsToSync()
        e.logger.Info("transactions to create", "count", len(toSync), "account_id", statement.AccountID)

        if len(toSync) == 0 {
            continue
        }

        // Prepare and create each transaction individually
        ts := e.ynab.Transaction()
        batch, err := report.Payloads(statement.AccountID)
        if err != nil {
            return err
        }
        if len(batch) == 0 {
            continue
        }
        if err := ts.CreateTransactions(e.config.YNAB.BudgetID, batch); err != nil {
            return fmt.Errorf("failed to create transactions: %w", err)
        }
        e.logger.Info("created transactions", "count", len(batch), "account_id", statement.AccountID)
    }

    return nil
}
