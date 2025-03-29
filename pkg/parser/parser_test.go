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
	output, err := parser.ProcessBytes(content, "extrato conta corrente.txt")
	if err != nil {
		t.Fatalf("ProcessBytes failed: %v", err)
	}

	t1, _ := models.NewTransaction("17/03/2025", "PIX TRANSF ID_A15/03", -2327.00).AsExtrato().Build()
	t2, _ := models.NewTransaction("17/03/2025", "MOBILE PAG TIT 426XXXXXX", -287.00).AsExtrato().Build()
	t3, _ := models.NewTransaction("19/03/2025", "PIX TRANSF ID_C19/03", -1900.00).AsExtrato().Build()
	expected := []models.Transaction{*t1, *t2, *t3}

	expectedCSV := models.ToCSV(expected)
	if string(output) != string(expectedCSV) {
		t.Errorf("Expected %s, got %s", string(expectedCSV), string(output))
	}
}

func TestParseSampleFiles(t *testing.T) {
	files := []string{
		filepath.Join(".", "data", "sample", "Extrato Conta Corrente-290320251101.txt"),
		filepath.Join(".", "data", "sample", "sample-Extrato Conta Corrente-290320250850.xls"),
		filepath.Join(".", "data", "sample", "sample-Fatura-Excel.xls"),
	}

	parser := New(log.Default())

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", file, err)
			}

			output, err := parser.ProcessBytes(data, file)
			if err != nil {
				t.Fatalf("Failed to parse file %s: %v", file, err)
			}

			t.Logf("Parsed contents of %s:\n%s", file, string(output))
		})
	}
}
