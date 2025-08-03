package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/config"
	"github.com/yurifrl/ynabu/pkg/csv"
	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/parser"
	"github.com/yurifrl/ynabu/pkg/reconcile"
	"github.com/yurifrl/ynabu/pkg/ynab"
)

// Server handles HTTP requests for YNAB file processing
type Server struct {
	config       *config.Config
	logger       *log.Logger
	mux          *http.ServeMux
	template     *template.Template
	parser       *parser.Parser
	transactions sync.Map
}

// New creates a new HTTP server
func New(config *config.Config, logger *log.Logger) *Server {
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	return &Server{
		config:   config,
		logger:   logger,
		mux:      http.NewServeMux(),
		template: tmpl,
		parser:   parser.New(logger),
	}
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	s.setupRoutes()
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) setupRoutes() {
    // New home page for reconciliation
    s.mux.HandleFunc("/", s.handleHome)
    // Existing CSV converter available under /csv
    s.mux.HandleFunc("/csv", s.handleIndex)

    s.mux.HandleFunc("/api/reconcile", s.handleReconcile)
    s.mux.HandleFunc("/api/convert", s.handleConvert)
    s.mux.HandleFunc("/api/files/", s.handleFiles)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
    // Legacy CSV converter page
    s.template.ExecuteTemplate(w, "csv.html", nil)
}

// handleHome serves the reconciliation page located at templates/index.html
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
    s.template.ExecuteTemplate(w, "index.html", nil)
}

// handleReconcile processes an uploaded statement, fetches remote YNAB
// transactions, performs a reconciliation diff and returns a JSON payload
// similar to the CLI plan output.
func (s *Server) handleReconcile(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Extract credentials & identifiers
    token := r.FormValue("token")
    budgetID := r.FormValue("budget_id")
    accountID := r.FormValue("account_id")
    if token == "" || budgetID == "" || accountID == "" {
        http.Error(w, "token, budget_id and account_id are required", http.StatusBadRequest)
        return
    }

    // Read the uploaded file
    file, header, err := r.FormFile("statement")
    if err != nil {
        http.Error(w, "Failed to read file", http.StatusBadRequest)
        return
    }
    defer file.Close()

    data, err := io.ReadAll(file)
    if err != nil {
        http.Error(w, "Failed to read file", http.StatusInternalServerError)
        return
    }

    // Parse local transactions
    localTxs, err := s.parser.ProcessBytes(data, header.Filename)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to process file: %v", err), http.StatusBadRequest)
        return
    }

    // Fetch remote transactions via YNAB API
    ynabClient := ynab.New(token)
    remoteTxs, err := ynabClient.Transaction().GetTransactionsByAccount(budgetID, accountID, nil)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to fetch remote transactions: %v", err), http.StatusInternalServerError)
        return
    }

    // Build reconciliation report
    report := reconcile.Build(localTxs, remoteTxs, s.config.UseCustomID)

    // Prepare diff-like lines
    lines := make([]string, 0, len(report.Items))
    for _, entry := range report.Items {
        prefix := "="
        if entry.Status == reconcile.ToAdd {
            prefix = "+"
        }
        line := fmt.Sprintf("%s %s | %-30s | R$ %.2f", prefix, entry.Local.Date(), entry.Local.Payee(), entry.Local.Amount())
        lines = append(lines, line)
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]interface{}{
        "status":  "success",
        "lines":   lines,
        "to_add":  report.MissingCount(),
        "in_sync": report.InSyncCount(),
    })
}

type Transaction struct {
	Date   string  `json:"date"`
	Payee  string  `json:"payee"`
	Amount float64 `json:"amount"`
	Memo   string  `json:"memo"`
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("statement")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	transactions, err := s.parser.ProcessBytes(data, header.Filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process file: %v", err), http.StatusBadRequest)
		return
	}

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Date() < transactions[j].Date()
	})

	filename := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)) + "-ynabu.csv"
	s.transactions.Store(filename, transactions)

	// Convert to simplified transaction format for JSON
	txs := make([]Transaction, len(transactions))
	for i, t := range transactions {
		txs[i] = Transaction{
			Date:   t.Date(),
			Payee:  t.Payee(),
			Memo:   t.Memo(),
			Amount: t.Amount(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"file":   filename,
		"data":   txs,
	})
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if filename == "" {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	transactions, ok := s.transactions.Load(filename)
	if !ok {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	txs := transactions.([]*models.Transaction)
	csvData := csv.Create(txs, nil)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(csvData)
}

