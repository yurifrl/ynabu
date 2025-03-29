package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/config"
	"github.com/yurifrl/ynabu/pkg/service"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Prefix:          "ynabu",
	})

	var outputPath string
	flag.StringVar(&outputPath, "o", "", "Output directory (default: same as input file)")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		logger.Error("invalid usage", "args", args)
		fmt.Fprintf(os.Stderr, "Usage: ynabu [-o output_dir] <directory>\n")
		os.Exit(1)
	}

	config := config.New(outputPath)
	processor := service.NewProcessor(config, logger)

	dir := args[0]
	if err := processor.ProcessDirectory(dir); err != nil {
		logger.Fatal("processing failed", "error", err)
	}
}
