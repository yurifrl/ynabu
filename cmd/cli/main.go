package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/yurifrl/ynabu/pkg/executor"
	"github.com/yurifrl/ynabu/pkg/plan"
	"github.com/yurifrl/ynabu/pkg/statement"

	"github.com/yurifrl/ynabu/pkg/config"
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
	Use:   "convert [flags] <input_path>",
	Short: "Convert bank statements to YNAB CSV format",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.NewWithOptions(os.Stderr, log.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			Prefix:          "ynabu-cli",
			Level:           log.DebugLevel,
		})

		// Load configuration (config file + flag overrides)
		if _, err := config.Build(cfgFile, cmd.Flags()); err != nil {
			return err
		}

		inputPath := args[0]

		importMode, _ := cmd.Flags().GetBool("import")
		if importMode {
			fmt.Println("import flag invoked (not yet implemented)")
			return nil
		}

		// Output directory is no longer used; processing always prints to stdout.
		outputDir := ""

		processor := NewFileProcessor(logger, &cliFilters)

		matches, err := filepath.Glob(inputPath)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return fmt.Errorf("no files found matching pattern %s", inputPath)
		}

		for _, match := range matches {
			fileInfo, err := os.Stat(match)
			if err != nil {
				logger.Warn("failed to stat file", "error", err, "file", match)
				continue
			}

			if fileInfo.IsDir() {
				if err := processor.ProcessDirectory(match, outputDir); err != nil {
					logger.Warn("failed to process directory", "error", err, "dir", match)
				}
			} else {
				if err := processor.ProcessFile(match, outputDir); err != nil {
					logger.Warn("failed to process file", "error", err, "file", match)
				}
			}
		}
		return nil
	},
}

var planCmd = &cobra.Command{
	Use:   "plan <plan_file>",
	Short: "Preview a YAML plan of statements (dry-run)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planPath := args[0]

		p, err := plan.Load(planPath)
		if err != nil {
			return err
		}

		logger := log.NewWithOptions(os.Stderr, log.Options{Prefix: "ynabu-cli"})
		exec := executor.New(logger)

		var stmts []statement.Statement
		for _, spec := range p.Statements {
			st, err := statement.New(spec, p.YNAB, logger)
			if err != nil {
				return err
			}
			stmts = append(stmts, st)
		}

		fmt.Printf("Plan preview for %s\n", planPath)
		p.Print()
		changes, err := exec.Plan(stmts)
		if err != nil {
			return err
		}
		fmt.Println("Summary of changes:")
		for _, c := range changes {
			fmt.Printf("  - statement %s -> account %s : create %d transactions\n", c.StatementID, c.AccountID, c.ToCreate)
		}
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
	convertCmd.Flags().Bool("import", false, "Import mode (placeholder)")

	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(planCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
