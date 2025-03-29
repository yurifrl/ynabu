# YNABU - YNAB Bank Statement Utility

A command-line tool to convert bank statements from Itaú into YNAB-compatible CSV format.

## Features

- Converts Itaú bank statements (Extrato) to YNAB CSV format
- Converts Itaú credit card statements (Fatura) to YNAB CSV format
- Converts Itaú CSV/TXT exports to YNAB CSV format
- Processes multiple files in a directory
- Preserves transaction details including date, payee, and amount
- Automatically categorizes transactions as inflow or outflow
- Smart detection of CSV formats (supports both comma and semicolon separators)

## Usage

```bash
# Process all XLS/CSV/TXT files in the current directory
ynabu .

# Process all files in a specific directory
ynabu /path/to/directory

# Process files and save output to a specific directory
ynabu -o /path/to/output/dir /path/to/input/dir

# By default, output files are saved in the same directory as the input files
# with "-ynabu.csv" appended to the original filename
```

## Supported File Types

- XLS bank statements (Extrato)
- XLS credit card statements (Fatura)
- CSV/TXT files with transaction data
  - Must contain at least date and value columns
  - Supports various date formats
  - Automatically detects column headers
  - Skips files already in YNAB format

## Output Format

The tool generates CSV files with the following columns:
- Date (YYYY-MM-DD)
- Payee
- Memo (includes import date)
- Outflow
- Inflow

## Building

```bash
go build -o ynabu ./cmd/ynabu
```

## Installation

```bash
go install github.com/yurifreire/ynabu/cmd/ynabu@latest
``` 