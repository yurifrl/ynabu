package compare

import (
	"fmt"

	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/ynab"
)

// Equal compares a local transaction (parsed from statement) with a remote
// transaction fetched from YNAB using three fields that are stable across
// systems: date (formatted as YYYY/MM/DD), payee (normalised), and the
// monetary amount. Amounts are compared using two-decimal fixed precision so
// that minor floating-point differences do not lead to mismatches.
func Equal(local *models.Transaction, remote *ynab.Transaction) bool {
    if local == nil || remote == nil {
        return false
    }
    // Compare amounts (remote is in milli-units)
    remoteAmount := float64(remote.Amount) / 1000.0
    if fmt.Sprintf("%.2f", local.Amount()) != fmt.Sprintf("%.2f", remoteAmount) {
        return false
    }
    // Compare payees (normalised)
    if local.Payee() != *remote.PayeeName {
        return false
    }
	// Compare dates
    if local.Date() != remote.Date.Format("2006/01/02") {
        return false
    }
    return true
}