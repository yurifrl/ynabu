package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/yurifrl/ynabu/pkg/csv"
	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/parser"
)

type filters struct {
	startDate string
	endDate   string
	minAmount float64
	maxAmount float64
	payee     string
	print     bool
}

type FileProcessor struct {
	logger  *log.Logger
	parser  *parser.Parser
	filters *filters
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

func NewFileProcessor(logger *log.Logger, filters *filters) *FileProcessor {
	return &FileProcessor{
		logger:  logger,
		parser:  parser.New(logger),
		filters: filters,
	}
}

func (p *FileProcessor) ProcessDirectory(inputDir, outputDir string) error {
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if err := p.ProcessFile(filepath.Join(inputDir, entry.Name()), outputDir); err != nil {
			p.logger.Warn("error processing file", "error", err)
		}
	}

	return nil
}

func (p *FileProcessor) ProcessFile(inputPath, outputDir string) error {
	fileBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	transactions, err := p.parser.ProcessBytes(fileBytes, filepath.Base(inputPath))
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Date() < transactions[j].Date()
	})

	outputBytes := csv.Create(transactions, p.filters.toFilterFunc())

	if p.filters.print {
		fmt.Print(string(outputBytes))
		return nil
	}

	filename := filepath.Base(inputPath)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s-ynabu%s.csv", name, ext))

	if err := writeBytes(outputPath, outputBytes); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func writeBytes(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, bytes.NewReader(data))
	return err
}
