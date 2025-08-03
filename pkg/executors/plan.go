package executors

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/k0kubun/pp/v3"
	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/parser"
	"github.com/yurifrl/ynabu/pkg/reconcile"
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

        var remoteTransactions []*ynab.Transaction
        if statement.AccountID != "" {
            e.logger.Debug("fetching remote transactions", "budget_id", e.config.YNAB.BudgetID, "account_id", statement.AccountID)
            var err error
            remoteTransactions, err = e.ynab.Transaction().GetTransactionsByAccount(e.config.YNAB.BudgetID, statement.AccountID, nil)
            if err != nil {
                e.logger.Error("failed to fetch remote transactions", "error", err)
                return err
            }
            e.logger.Debug("fetched remote transactions", "count", len(remoteTransactions))
        } else {
            e.logger.Debug("account_id empty, skipping remote fetch")
            remoteTransactions = []*ynab.Transaction{}
        }
		report := reconcile.Build(transactions, remoteTransactions, e.config.UseCustomID)
        e.logger.Debug("processing plan report", "total", len(report.Items), "in_sync", report.InSyncCount(), "to_add", report.MissingCount())

        syncedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray
        addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // green

        for _, m := range report.Items {
            if m.Status == reconcile.Synced {
                line := fmt.Sprintf("%s | %-30s | %s | %s | R$ %.2f", m.Local.Date(), m.Local.Payee(), m.Local.ID(), m.Remote.CustomID(), m.Local.Amount())
                fmt.Println(syncedStyle.Render("= " + line))
            } else {
                line := fmt.Sprintf("%s | %-30s | %s | %s | R$ %.2f", m.Local.Date(), m.Local.Payee(), m.Local.ID(), "xxxxxxxxxxxxxxxx", m.Local.Amount())
                fmt.Println(addedStyle.Render("+ " + line))
            }
        }

        if report.MissingCount() == 0 {
            fmt.Printf("\nPlan: All %d transaction(s) are in sync\n", report.InSyncCount())
        } else {
            fmt.Printf("\nPlan: %d transaction(s) will be added, %d already in sync\n", report.MissingCount(), report.InSyncCount())
        }
    }

    return nil
}