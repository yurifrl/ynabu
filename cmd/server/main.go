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
		Prefix:          "ynabu",
	})

	var (
		port   = flag.String("port", "3000", "Server port")
		output = flag.String("o", "", "Output directory")
	)
	flag.Parse()

	cfg := config.New(*output)
	srv := server.New(cfg, logger)
	addr := fmt.Sprintf("0.0.0.0:%s", *port)
	logger.Info("starting server", "addr", addr)
	if err := srv.Start(addr); err != nil {
		logger.Fatal("server error", "err", err)
	}
}
