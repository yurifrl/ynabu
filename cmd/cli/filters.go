package main

import (
	"strings"
	"time"

	"github.com/yurifrl/ynabu/pkg/csv"
	"github.com/yurifrl/ynabu/pkg/models"
)

type filters struct {
	startDate string
	endDate   string
	minAmount float64
	maxAmount float64
	payee     string
}

func (f *filters) toFilterFunc() csv.FilterFunc[*models.Transaction] {
	return func(t *models.Transaction) bool {
		if f.startDate != "" {
			start, _ := time.Parse("2006/01/02", f.startDate)
			date, _ := time.Parse("2006/01/02", t.Date())
			if date.Before(start) {
				return false
			}
		}
		if f.endDate != "" {
			end, _ := time.Parse("2006/01/02", f.endDate)
			date, _ := time.Parse("2006/01/02", t.Date())
			if date.After(end) {
				return false
			}
		}
		if f.minAmount != 0 && t.Amount() < f.minAmount {
			return false
		}
		if f.maxAmount != 0 && t.Amount() > f.maxAmount {
			return false
		}
		if f.payee != "" && !strings.Contains(strings.ToLower(t.Payee()), strings.ToLower(f.payee)) {
			return false
		}
		return true
	}
}