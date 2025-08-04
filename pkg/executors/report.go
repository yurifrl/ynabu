package executors

// This file contains all the reconciliation logic that was previously in the
// separate `pkg/reconcile` package. Keeping it inside the `executors` package
// removes the extra indirection and makes the API surface simpler: callers only
// need the `executors` package when working with statements (plan/apply) or
// when they merely want to compare local and remote transactions.

import (
	"fmt"

	"github.com/brunomvsouza/ynab.go/api/transaction"

	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/ynab"
)

// Status indicates the reconciliation result for a local transaction.
//
//   - Synced: already present remotely.
//   - ToAdd:  missing, needs to be created.
//
// Additional statuses (e.g. ToUpdate, Deleted) can be added later.
//
// NOTE: We preserve the names that were already used across the code-base to
// keep the diff small.
//
// Usage example:
//   report := executors.Build(local, remote, true)
//   for _, entry := range report.Items {
//       if entry.Status == executors.ToAdd {
//           // … create transaction …
//       }
//   }
//
// The entire reconciliation API is now self-contained inside the executors
// package and can be consumed without having to instantiate an Executor – the
// behaviour is pure and stateless.
//
// The stateful parts (HTTP calls to YNAB, manifest parsing, etc.) remain methods
// on *Executor.
//
// This two-layer split keeps the public API minimal while still enabling code
// reuse between the CLI executors and the web server.
// -----------------------------------------------------------------------------

type Status int

const (
    Synced Status = iota
    ToAdd
)

// Entry links a local transaction with its remote counterpart (if any) and
// records the reconciliation status.

type Entry struct {
    Local  *models.Transaction
    Remote *ynab.Transaction // nil when status == ToAdd
    Status Status
}

// RemoteCustomID is a helper that returns the remote CustomID when present.
func (e Entry) RemoteCustomID() string {
    if e.Remote == nil {
        return ""
    }
    return e.Remote.CustomID()
}

// Report is the main reconciled data-structure returned by Build.

type Report struct {
    Items  []Entry
    toSync []*models.Transaction
}

// Build produces a reconciliation report by matching local transactions against
// the remote ones. Matching can be done via CustomID (memo-based) or via
// amount/payee/date heuristics.
func BuildReport(local []*models.Transaction, remote []*ynab.Transaction, useCustomID bool) *Report {
    items := make([]Entry, 0, len(local))
    toSync := make([]*models.Transaction, 0)

    if useCustomID {
        // Fast path: O(n) lookup using the memo-encoded CustomID.
        idx := make(map[string]*ynab.Transaction, len(remote))
        for _, rt := range remote {
            idx[rt.CustomID()] = rt
        }
        for _, lt := range local {
            found := idx[lt.ID()]
            status := ToAdd
            if found != nil {
                status = Synced
            }
            items = append(items, Entry{Local: lt, Remote: found, Status: status})
            if status == ToAdd {
                toSync = append(toSync, lt)
            }
        }
    } else {
        // Fallback matching using amount+payee+date – this is slower but more
        // resilient when the statement does not include the CustomID in the
        // memo line.
        idx := make(map[string]*ynab.Transaction, len(remote))
        for _, rt := range remote {
            payee := ""
            if rt.PayeeName != nil {
                payee = *rt.PayeeName
            }
            key := fmt.Sprintf("%.2f|%s|%s", float64(rt.Amount)/1000.0, payee, rt.Date.Format("2006/01/02"))
            if _, ok := idx[key]; !ok {
                idx[key] = rt
            }
        }
        for _, lt := range local {
            key := fmt.Sprintf("%.2f|%s|%s", lt.Amount(), lt.Payee(), lt.Date())
            found := idx[key]
            if found != nil && !equal(lt, found) {
                // Different transaction despite the same key → treat as missing.
                found = nil
            }
            status := ToAdd
            if found != nil {
                status = Synced
            }
            items = append(items, Entry{Local: lt, Remote: found, Status: status})
            if status == ToAdd {
                toSync = append(toSync, lt)
            }
        }
    }

    return &Report{Items: items, toSync: toSync}
}

// equal checks whether local and remote transactions actually match.
func equal(local *models.Transaction, remote *ynab.Transaction) bool {
    if local == nil || remote == nil {
        return false
    }
    remoteAmount := float64(remote.Amount) / 1000.0
    if fmt.Sprintf("%.2f", local.Amount()) != fmt.Sprintf("%.2f", remoteAmount) {
        return false
    }
    if remote.PayeeName == nil || local.Payee() != *remote.PayeeName {
        return false
    }
    if local.Date() != remote.Date.Format("2006/01/02") {
        return false
    }
    return true
}

// InSyncCount returns how many local transactions already exist remotely.
func (r *Report) InSyncCount() int {
    return len(r.Items) - len(r.toSync)
}

// MissingCount returns how many local transactions still need to be created.
func (r *Report) MissingCount() int {
    return len(r.toSync)
}

// TransactionsToSync returns the subset of local transactions missing remotely.
func (r *Report) TransactionsToSync() []*models.Transaction {
    return r.toSync
}

// Payloads converts the transactions that still need syncing into YNAB API payloads.
func (r *Report) Payloads(accountID string) ([]transaction.PayloadTransaction, error) {
    out := make([]transaction.PayloadTransaction, 0, len(r.toSync))
    for _, lt := range r.toSync {
        dateVal, err := lt.APIDate()
        if err != nil {
            return nil, err
        }
        out = append(out, transaction.PayloadTransaction{
            AccountID: accountID,
            Date:      dateVal,
            Amount:    lt.AmountMilliunits(),
            Cleared:   transaction.ClearingStatusCleared,
            Approved:  true,
            PayeeName: lt.PayeePointer(),
            Memo:      lt.MemoPointer(),
        })
    }
    return out, nil
}

