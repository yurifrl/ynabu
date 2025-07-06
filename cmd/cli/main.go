package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/yurifrl/ynabu/pkg/config"
	"github.com/yurifrl/ynabu/pkg/csv"
	"github.com/yurifrl/ynabu/pkg/parser"
)

var (
	cliFilters filters
	cfgFile    string
)

var rootCmd = &cobra.Command{
	Use:   "ynabu-cli",
	Short: "YNABu command-line interface",
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Show help when no subcommand is provided
		return cmd.Help()
	},
}

var convertCmd = &cobra.Command{
	Use:   "convert [flags]",
	Short: "Convert bank statements to YNAB CSV format",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		logger := log.NewWithOptions(os.Stderr, log.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			Prefix:          "ynabu-cli",
			Level:           log.DebugLevel,
		})

		if _, err := config.Build(cfgFile, cmd.Flags()); err != nil {
			return err
		}

		inputPath, err := cmd.Flags().GetString("file")
		if err != nil {
			return err
		}

		fileBytes, err := os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		parser := parser.New(logger)
		transactions, err := parser.ProcessBytes(fileBytes, filepath.Base(inputPath))
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
	Use:   "plan <plan_file>",
	Short: "Preview a YAML plan of statements (dry-run)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planPath := args[0]

		logger := log.NewWithOptions(os.Stderr, log.Options{Prefix: "ynabu-cli"})

		logger.Debug("plan", "planPath", planPath)

		return nil
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file (default is config.yaml)")

	// Filter flags (global)
	rootCmd.PersistentFlags().StringVar(&cliFilters.startDate, "start", "", "Start date (YYYY/MM/DD)")
	rootCmd.PersistentFlags().StringVar(&cliFilters.endDate, "end", "", "End date (YYYY/MM/DD)")
	rootCmd.PersistentFlags().Float64Var(&cliFilters.minAmount, "min", 0, "Minimum amount")
	rootCmd.PersistentFlags().Float64Var(&cliFilters.maxAmount, "max", 0, "Maximum amount")
	rootCmd.PersistentFlags().StringVar(&cliFilters.payee, "payee", "", "Filter by payee (case insensitive)")

	// Flags specific to the convert subcommand
	convertCmd.Flags().StringP("file", "f", "", "Input path (supports glob patterns)")
	convertCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(planCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
