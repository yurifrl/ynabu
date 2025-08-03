package executors

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/k0kubun/pp/v3"
	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/parser"
	"github.com/yurifrl/ynabu/pkg/ynab"
)

var _ = pp.Println

func (e *Executor) Plan(manifest *models.Manifest) error {
    e.logger.Debug("planning manifest")
    p := parser.New(e.logger)

    for _, statement := range manifest.Statements {
        transactions, err := statement.Transactions(p)
        if err != nil {
            return err
        }

        e.logger.Debug("fetching remote transactions", "budget_id", e.config.YNAB.BudgetID, "account_id", statement.AccountID)
        remoteTransactions, err := e.ynab.Transaction().GetTransactionsByAccount(e.config.YNAB.BudgetID, statement.AccountID, nil)
        if err != nil {
            e.logger.Error("failed to fetch remote transactions", "error", err)
            return err
        }

        e.logger.Debug("fetched remote transactions", "count", len(remoteTransactions))
		e.showPlan(transactions, remoteTransactions)
    }

    return nil
}

func (e *Executor) showPlan(localTxs []*models.Transaction, remoteTxs []*ynab.Transaction) {
    e.logger.Debug("processing transactions", "local_count", len(localTxs), "remote_count", len(remoteTxs))

    for _, remoteTx := range remoteTxs {
        payee := ""
        if remoteTx.PayeeName != nil {
            payee = *remoteTx.PayeeName
        }
        e.logger.Debug("remote transaction", "payee", payee, "amount", remoteTx.Amount, "date", remoteTx.Date)
    }

    missingCount := 0
    syncCount := 0

    syncedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray
    addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // green

    for _, localTx := range localTxs {
        matched := false
        remoteTx := &ynab.Transaction{}
        for _, remoteTx = range remoteTxs {
            if remoteTx.CustomID == localTx.ID() {
                matched = true
                break
            }
        }

        payee := localTx.Payee()
        if len(payee) > 30 {
            payee = payee[:27] + "..."
        }
        if matched {
            line := fmt.Sprintf("%s | %-30s | %s | %s | R$ %.2f", localTx.Date(), payee, localTx.ID(), remoteTx.CustomID, localTx.Amount())
            fmt.Println(syncedStyle.Render("= " + line))
            syncCount++
        } else {
            line := fmt.Sprintf("%s | %-30s | %s | %s | R$ %.2f", localTx.Date(), payee, localTx.ID(), "xxxxxxxxxxxxxxxx", localTx.Amount())
            fmt.Println(addedStyle.Render("+ " + line))
            missingCount++
        }
    }

    if missingCount == 0 {
        fmt.Printf("\nPlan: All %d transaction(s) are in sync\n", syncCount)
    } else {
        fmt.Printf("\nPlan: %d transaction(s) will be added, %d already in sync\n", missingCount, syncCount)
    }
}
