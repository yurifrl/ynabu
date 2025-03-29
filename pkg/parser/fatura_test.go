package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yurifrl/ynabu/pkg/models"
)

func TestParseFatura(t *testing.T) {
	// Create a temporary test file
	content := `17/03/2025;PIX TRANSF ID_A15/03;-2327,00
17/03/2025;MOBILE PAG TIT 426XXXXXX;-287,00
19/03/2025;PIX TRANSF ID_C19/03;-1900,00`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_fatura.txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test the ParseFatura function
	transactions, err := ParseFatura(tmpFile)
	if err != nil {
		t.Fatalf("ParseFatura failed: %v", err)
	}

	// Expected transactions
	expectedDate, _ := time.Parse("02/01/2006", "17/03/2025")
	expected := []models.Transaction{
		{
			Date:    expectedDate,
			Payee:   "PIX TRANSF ID_A15/03",
			Outflow: 2327.00,
			Memo:    generateTransactionID(expectedDate, "PIX TRANSF ID_A15/03") + ",extrato,-",
		},
		{
			Date:    expectedDate,
			Payee:   "MOBILE PAG TIT 426XXXXXX",
			Outflow: 287.00,
			Memo:    generateTransactionID(expectedDate, "MOBILE PAG TIT 426XXXXXX") + ",extrato,-",
		},
		{
			Date:    time.Date(2025, 3, 19, 0, 0, 0, 0, time.UTC),
			Payee:   "PIX TRANSF ID_C19/03",
			Outflow: 1900.00,
			Memo:    generateTransactionID(time.Date(2025, 3, 19, 0, 0, 0, 0, time.UTC), "PIX TRANSF ID_C19/03") + ",extrato,-",
		},
	}

	// Verify the results
	if len(transactions) != len(expected) {
		t.Errorf("Expected %d transactions, got %d", len(expected), len(transactions))
	}

	for i, exp := range expected {
		if i >= len(transactions) {
			break
		}
		got := transactions[i]
		if !got.Date.Equal(exp.Date) {
			t.Errorf("Transaction %d: expected date %v, got %v", i, exp.Date, got.Date)
		}
		if got.Payee != exp.Payee {
			t.Errorf("Transaction %d: expected payee %q, got %q", i, exp.Payee, got.Payee)
		}
		if got.Outflow != exp.Outflow {
			t.Errorf("Transaction %d: expected outflow %.2f, got %.2f", i, exp.Outflow, got.Outflow)
		}
		if got.Memo != exp.Memo {
			t.Errorf("Transaction %d: expected memo %q, got %q", i, exp.Memo, got.Memo)
		}
	}
}
