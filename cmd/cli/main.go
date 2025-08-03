package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/k0kubun/pp/v3"
	"github.com/spf13/cobra"
	"github.com/subosito/gotenv"

	"github.com/yurifrl/ynabu/pkg/config"
	"github.com/yurifrl/ynabu/pkg/csv"
	"github.com/yurifrl/ynabu/pkg/executors"
	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/parser"
	"github.com/yurifrl/ynabu/pkg/ynab"
)

var (
	cliFilters filters
	cfgFile    string
	file       string
)

type contextKey string

const (
	loggerKey contextKey = "logger"
	configKey contextKey = "config"
)

var _ = pp.Println

var rootCmd = &cobra.Command{
	Use:   "ynabu-cli",
	Short: "YNABu command-line interface",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Build(cfgFile, cmd.Flags())
		if err != nil {
			return err
		}

		// Determine log level, defaulting to info
		lvl := log.InfoLevel
		switch strings.ToLower(cfg.LogLevel) {
		case "debug":
			lvl = log.DebugLevel
		case "info":
			lvl = log.InfoLevel
		case "warn", "warning":
			lvl = log.WarnLevel
		case "error":
			lvl = log.ErrorLevel
		}

		logger := log.NewWithOptions(os.Stderr, log.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			Prefix:          "ynabu-cli",
			Level:           lvl,
		})

		ctx := context.WithValue(cmd.Context(), loggerKey, logger)
		ctx = context.WithValue(ctx, configKey, cfg)
		cmd.SetContext(ctx)

		return nil
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

var convertCmd = &cobra.Command{
	Use:   "convert [flags]",
	Short: "Convert bank statements to YNAB CSV format",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		logger := cmd.Context().Value(loggerKey).(*log.Logger)
		file := cmd.Flag("file").Value.String()

		fileBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		parser := parser.New(logger)
		transactions, err := parser.ProcessBytes(fileBytes, filepath.Base(file))
		if err != nil {
			return fmt.Errorf("failed to process file: %w", err)
		}

		sort.Slice(transactions, func(i, j int) bool {
			return transactions[i].Date() < transactions[j].Date()
		})

		outputBytes := csv.Create(transactions, cliFilters.toFilterFunc())

		fmt.Println(string(outputBytes))
		return nil
	},
}

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Preview a YAML plan of statements (dry-run)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := cmd.Context().Value(loggerKey).(*log.Logger)
		cfg := cmd.Context().Value(configKey).(*config.Config)
		file := cmd.Flag("file").Value.String()

		manifest, err := models.FromFile(file)
		if err != nil {
			return fmt.Errorf("failed to read manifest: %w", err)
		}

		logger.Debug("plan", "planPath", file)

		ynabClient := ynab.New(cfg.YNAB.Token)

		exec := executors.New(logger, cfg, ynabClient)
		err = exec.Plan(manifest)
		if err != nil {
			return fmt.Errorf("failed to plan: %w", err)
		}

		return nil
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file (default is config.yaml)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "Log level (debug, info, warn, error)")

	// Filter flags (global)
	rootCmd.PersistentFlags().StringVar(&cliFilters.startDate, "start", "", "Start date (YYYY/MM/DD)")
	rootCmd.PersistentFlags().StringVar(&cliFilters.endDate, "end", "", "End date (YYYY/MM/DD)")
	rootCmd.PersistentFlags().Float64Var(&cliFilters.minAmount, "min", 0, "Minimum amount")
	rootCmd.PersistentFlags().Float64Var(&cliFilters.maxAmount, "max", 0, "Maximum amount")
	rootCmd.PersistentFlags().StringVar(&cliFilters.payee, "payee", "", "Filter by payee (case insensitive)")
	rootCmd.PersistentFlags().StringVarP(&file, "file", "f", "", "Input path (supports glob patterns)")

	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(planCmd)

	convertCmd.MarkFlagRequired("file")
	planCmd.MarkFlagRequired("file")
}

func main() {
	gotenv.Load()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
