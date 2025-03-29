package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/parser"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Prefix:          "ynabu-cli",
		Level:           log.DebugLevel,
	})

	var outputDir string
	flag.StringVar(&outputDir, "o", "", "Output directory (default: same as input directory)")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		logger.Fatal("Usage: cli [-o output_dir] <input_dir>")
	}

	inputDir := args[0]

	// If no output dir specified, use input dir
	if outputDir == "" {
		outputDir = inputDir
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Fatal("failed to create output directory", "error", err)
	}

	processor := NewFileProcessor(logger)

	// Process directory
	if err := processor.ProcessDirectory(inputDir, outputDir); err != nil {
		logger.Fatal("failed to process directory", "error", err)
	}
}

type FileProcessor struct {
	logger *log.Logger
	parser *parser.Parser
}

func NewFileProcessor(logger *log.Logger) *FileProcessor {
	return &FileProcessor{
		logger: logger,
		parser: parser.New(logger),
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
	// Read file into bytes
	fileBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Process bytes using parser
	outputBytes, err := p.parser.ProcessBytes(fileBytes, filepath.Base(inputPath))
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
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
