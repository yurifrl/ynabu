package executors

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/yurifrl/ynabu/pkg/models"
)

// Plan generates a reconciliation report for a single statement and prints a
// human-readable preview. It is a thin, side-effecting wrapper around the pure
// BuildReport function – all heavy lifting is delegated there.  The caller is
// responsible for looping over multiple statements when needed.
func (e *Executor) Plan(statement *models.Statement) error {
    e.logger.Debug("planning statement", "file", statement.FilePath)

    // Parse local transactions
    localTxs, err := statement.Transactions(e.parser)
    if err != nil {
        return err
    }

    if statement.AccountID == "" {
        return fmt.Errorf("statement %s missing account_id", statement.FilePath)
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
            continue // nothing to add
        }

        line := fmt.Sprintf("%s | %-30s | %s | %s | R$ %.2f", m.Local.Date(), m.Local.Payee(), m.Local.ID(), "xxxxxxxxxxxxxxxx", m.Local.Amount())
        fmt.Println(addedStyle.Render("+ " + line))
    }

    if report.MissingCount() == 0 {
        fmt.Printf("\nPlan: All %d transaction(s) are in sync\n", report.InSyncCount())
    } else {
        fmt.Printf("\nPlan: %d transaction(s) will be added, %d already in sync\n", report.MissingCount(), report.InSyncCount())
    }

    return nil
}
