package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/yurifrl/ynabu/pkg/config"
	"github.com/yurifrl/ynabu/pkg/server"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "ynabu-server",
	Short: "Run ynabu HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.NewWithOptions(os.Stderr, log.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			Prefix:          "ynabu",
		})

		cfg, err := config.Build(cfgFile, cmd.Flags())
		if err != nil {
			return err
		}

		srv := server.New(cfg, logger)
		addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
		logger.Info("starting server", "addr", addr)
		return srv.Start(addr)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file (default is config.yaml)")
	rootCmd.Flags().String("port", "", "Server port (overrides config)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
