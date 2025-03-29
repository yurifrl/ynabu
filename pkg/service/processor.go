package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/parser"
	"github.com/yurifrl/ynabu/pkg/types"
)

type Processor struct {
	config types.Config
	logger *log.Logger
}

func NewProcessor(config types.Config, logger *log.Logger) *Processor {
	return &Processor{
		config: config,
		logger: logger,
	}
}

func (p *Processor) ProcessDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}

	for _, entry := range entries {
		if err := p.processEntry(dir, entry); err != nil {
			p.logger.Error("failed to process entry", "file", entry.Name(), "error", err)
		}
	}

	return nil
}

func (p *Processor) processEntry(dir string, entry os.DirEntry) error {
	if entry.IsDir() {
		return nil
	}

	fileName := strings.ToLower(entry.Name())
	if !strings.HasSuffix(fileName, ".xls") && !strings.HasSuffix(fileName, ".txt") {
		return nil
	}

	inputPath := filepath.Join(dir, entry.Name())
	outFile := p.determineOutputPath(inputPath, entry.Name())

	fileType, err := parser.DetectFileType(inputPath)
	if err != nil {
		return fmt.Errorf("error detecting file type: %w", err)
	}

	p.logger.Info("processing file", "path", inputPath, "type", fileType)

	if err := p.processFile(fileType, inputPath, outFile); err != nil {
		return err
	}

	p.logger.Info("processed file successfully", "input", inputPath, "output", outFile)
	return nil
}

func (p *Processor) determineOutputPath(inputPath, fileName string) string {
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)
	if p.config.GetOutputPath() != "" {
		return filepath.Join(p.config.GetOutputPath(), baseName+"-ynabu.csv")
	}
	return strings.TrimSuffix(inputPath, ext) + "-ynabu.csv"
}

func (p *Processor) processFile(fileType parser.FileType, inputPath, outputPath string) error {
	var transactions []models.Transaction
	var err error

	switch fileType {
	case parser.ExtratoItauXls:
		transactions, err = parser.ParseExtrato(inputPath)
	case parser.FaturaItauXls, parser.FaturaItauTxt:
		transactions, err = parser.ParseFatura(inputPath)
	default:
		return fmt.Errorf("unknown file type")
	}

	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer output.Close()

	if err := parser.WriteYNABCSV(output, transactions); err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}

	return nil
}
