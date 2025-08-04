package executors

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/k0kubun/pp/v3"
	"github.com/yurifrl/ynabu/pkg/models"
)

var _ = pp.Println

func (e *Executor) Plan(manifest *models.Manifest) error {
    e.logger.Debug("planning manifest")

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

        e.logger.Debug("processing plan report", "total", len(report.Items), "in_sync", report.InSyncCount(), "to_add", report.MissingCount())

        syncedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray
        addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // green

        for _, m := range report.Items {
            if m.Status == Synced {
                line := fmt.Sprintf("%s | %-30s | %s | %s | R$ %.2f", m.Local.Date(), m.Local.Payee(), m.Local.ID(), m.Remote.CustomID(), m.Local.Amount())
                fmt.Println(syncedStyle.Render("= " + line))
                continue // early continue: nothing to add
            }

            line := fmt.Sprintf("%s | %-30s | %s | %s | R$ %.2f", m.Local.Date(), m.Local.Payee(), m.Local.ID(), "xxxxxxxxxxxxxxxx", m.Local.Amount())
            fmt.Println(addedStyle.Render("+ " + line))
        }

        if report.MissingCount() == 0 {
            fmt.Printf("\nPlan: All %d transaction(s) are in sync\n", report.InSyncCount())
        } else {
            fmt.Printf("\nPlan: %d transaction(s) will be added, %d already in sync\n", report.MissingCount(), report.InSyncCount())
        }
    }

    return nil
}