package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/config"
	"github.com/yurifrl/ynabu/pkg/server"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Prefix:          "ynabu-server",
	})

	var (
		port       int
		outputPath string
	)

	flag.IntVar(&port, "port", 8080, "Port to listen on")
	flag.StringVar(&outputPath, "o", "", "Output directory for processed files")
	flag.Parse()

	config := config.New(outputPath)
	srv := server.New(config, logger)

	addr := fmt.Sprintf(":%d", port)
	logger.Info("starting server", "address", addr)

	if err := srv.Start(addr); err != nil {
		logger.Fatal("server error", "error", err)
	}
}
