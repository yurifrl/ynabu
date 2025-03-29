package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/models"
)

func TestProcessBytes(t *testing.T) {
	content := []byte(`17/03/2025;PIX TRANSF ID_A15/03;-2327,00
17/03/2025;MOBILE PAG TIT 426XXXXXX;-287,00
19/03/2025;PIX TRANSF ID_C19/03;-1900,00`)

	parser := New(log.Default())
	output, err := parser.ProcessBytes(content, "extrato.txt")
	if err != nil {
		t.Fatalf("ProcessBytes failed: %v", err)
	}

	t1, _ := models.NewTransaction("PIX TRANSF ID_A15/03").AsExtrato().SetDate("17/03/2025").SetValueFromExtrato("-2327,00").Build()
	t2, _ := models.NewTransaction("MOBILE PAG TIT 426XXXXXX").AsExtrato().SetDate("17/03/2025").SetValueFromExtrato("-287,00").Build()
	t3, _ := models.NewTransaction("PIX TRANSF ID_C19/03").AsExtrato().SetDate("19/03/2025").SetValueFromExtrato("-1900,00").Build()
	expected := []*models.Transaction{t1, t2, t3}

	if len(output) != len(expected) {
		t.Errorf("Expected %d transactions, got %d", len(expected), len(output))
		return
	}

	for i, exp := range expected {
		if exp.Date() != output[i].Date() ||
			exp.Payee() != output[i].Payee() ||
			exp.Amount() != output[i].Amount() {
			t.Errorf("Transaction %d mismatch:\nExpected: %+v\nGot: %+v", i, exp, output[i])
		}
	}
}

func TestParseSampleFiles(t *testing.T) {
	files := []string{
		"../../data/sample/sample-Extrato Conta Corrente-290320251101.txt",
		"../../data/sample/sample-Extrato Conta Corrente-290320250850.xls",
		"../../data/sample/sample-Fatura-Excel.xls",
	}

	parser := New(log.Default())

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("Failed to read %s: %v", file, err)
			continue
		}

		filename := filepath.Base(file)
		transactions, err := parser.ProcessBytes(content, filename)
		if err != nil {
			t.Errorf("Failed to process %s: %v", filename, err)
			continue
		}

		switch filename {
		case "sample-Extrato Conta Corrente-290320251101.txt":
			assertTransaction(t, transactions[0], "2025/03/17", "PIX TRANSF ID_A15/03", -2327.00)
			assertTransaction(t, transactions[27], "2025/03/31", "PIX TRANSF ID_B29/03", -4000.00)
			assertTransaction(t, transactions[24], "2025/03/28", "PIX TRANSF ID_B28/03", 42000.00)

		case "sample-Extrato Conta Corrente-290320250850.xls":
			assertTransaction(t, transactions[0], "2025/03/17", "CraftCorner Supplies", -2327.00)
			assertTransaction(t, transactions[27], "2025/03/31", "PetPals Emporium", -4000.00)
			assertTransaction(t, transactions[24], "2025/03/28", "StyleHub Apparel", 2000.00)

		case "sample-Fatura-Excel.xls":
			assertTransaction(t, transactions[0], "2025/02/28", "Clix*GadgetGalaxy", -16.00)
			assertTransaction(t, transactions[13], "2025/03/06", "HomeHaven Decor", -289.00)
			assertTransaction(t, transactions[24], "2025/03/22", "Clix*GadgetGalaxy", -194.29)
		}
	}
}

func assertTransaction(t *testing.T, tx *models.Transaction, date, payee string, amount float64) {
	if tx.Date() != date || tx.Payee() != payee || tx.Amount() != amount {
		t.Errorf("Transaction mismatch:\nExpected: date=%s, payee=%s, amount=%.2f\nGot: date=%s, payee=%s, amount=%.2f",
			date, payee, amount,
			tx.Date(), tx.Payee(), tx.Amount())
	}
}
