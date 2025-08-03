package reconcile

// Package reconcile provides minimal reusable logic to compare local parsed
// transactions with the ones that already exist on YNAB.  It is intentionally
// isolated from any UI/CLI so that both the CLI “plan”, “apply” executors and a
// future frontend can reuse the same data-model.

import (
	"fmt"

	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/ynab"
)

// Status indicates the reconciliation result for a given local transaction.
//
//   - Synced: already present remotely.
//   - ToAdd:  missing, needs to be created.
//
// Additional statuses can be added in the future (e.g. ToUpdate, Deleted).
type Status int

const (
    Synced Status = iota
    ToAdd
)

// Match links a local transaction with the corresponding remote (if any) and
// records the reconciliation status.
type Entry struct {
    Local  *models.Transaction
    Remote *ynab.Transaction // nil when status == ToAdd
    Status Status
}

// RemoteCustomID returns the CustomID of the remote txn when present.
func (e Entry) RemoteCustomID() string {
    if e.Remote == nil {
        return ""
    }
    return e.Remote.CustomID()
}

// Report is the high-level structure produced by the reconciliation process.
// It contains every local transaction plus metadata so that callers can decide
// what to display or sync without re-implementing the comparison logic.
type Report struct {
    Items  []Entry
    toSync []*models.Transaction
}

// Build walks through local transactions and tries to find a corresponding
// remote transaction.  Matching can be done by CustomID (memo-based) or by
// value/date/payee matching.
func Build(local []*models.Transaction, remote []*ynab.Transaction, useCustomID bool) *Report {
    items := make([]Entry, 0, len(local))
    toSync := make([]*models.Transaction, 0)

    if useCustomID {
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

// equal compares a local transaction parsed from a statement with a remote
// YNAB transaction using amount, payee, and date.
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

// InSyncCount returns how many local transactions are already present remotely.
func (r *Report) InSyncCount() int {
    return len(r.Items) - len(r.toSync)
}

// MissingCount returns how many local transactions are missing remotely.
func (r *Report) MissingCount() int {
    return len(r.toSync)
}

// TransactionsToSync returns the subset of local transactions that still need
// to be created on YNAB.
func (r *Report) TransactionsToSync() []*models.Transaction {
    return r.toSync
}
