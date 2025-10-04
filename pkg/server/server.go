package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/yurifrl/ynabu/pkg/csv"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/config"
	"github.com/yurifrl/ynabu/pkg/executors"
	"github.com/yurifrl/ynabu/pkg/models"
	"github.com/yurifrl/ynabu/pkg/parser"
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
	// homepage
	s.mux.HandleFunc("/", s.withLogging(s.handleHome))

	// consolidated endpoint
	s.mux.HandleFunc("/api/process", s.withLogging(s.handleProcess))
	s.mux.HandleFunc("/api/apply", s.withLogging(s.handleApply))
	s.mux.HandleFunc("/api/files/", s.withLogging(s.handleFiles))
	s.mux.HandleFunc("/api/budgets", s.withLogging(s.handleBudgets))
	s.mux.HandleFunc("/api/budgets/", s.withLogging(s.handleBudgetAccounts))

}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if err := s.template.ExecuteTemplate(w, "index.html", nil); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "failed to render page", err)
		return
	}
}

func (s *Server) handleBudgets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		s.respondError(w, r, http.StatusBadRequest, "token required", nil)
		return
	}

	ynabClient := ynab.New(token)
	budgetsResponse, err := ynabClient.Budget().GetBudgets()
	if err != nil {
		s.respondError(w, r, http.StatusBadGateway, "failed to fetch budgets", err)
		return
	}

	// Convert budgets slice to interface{} slice for JSON serialization
	var budgets []interface{}
	if budgetsResponse != nil {
		budgets = make([]interface{}, len(budgetsResponse))
		for i, budget := range budgetsResponse {
			budgets[i] = budget
		}
	}

	// Debug logging
	s.logger.Info("budgets response", "budgets_count", len(budgets))

	if err := s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"budgets": budgets,
	}); err != nil {
		s.logger.Warn("failed to write json response", "err", err)
	}
}

func (s *Server) handleBudgetAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}

	budgetID := strings.TrimPrefix(r.URL.Path, "/api/budgets/")
	if budgetID == "" {
		s.respondError(w, r, http.StatusBadRequest, "budget_id required", nil)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		s.respondError(w, r, http.StatusBadRequest, "token required", nil)
		return
	}

	ynabClient := ynab.New(token)
	accountsResponse, err := ynabClient.Account().GetAccounts(budgetID, nil)
	if err != nil {
		s.respondError(w, r, http.StatusBadGateway, "failed to fetch accounts", err)
		return
	}

	// Extract accounts array from SearchResultSnapshot
	var accounts []interface{}
	if accountsResponse != nil && accountsResponse.Accounts != nil {
		// Convert the accounts slice to interface{} slice for JSON serialization
		accounts = make([]interface{}, len(accountsResponse.Accounts))
		for i, account := range accountsResponse.Accounts {
			accounts[i] = account
		}
	}

	// Debug logging
	s.logger.Info("accounts response", "budget_id", budgetID, "accounts_count", len(accounts))

	if err := s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"accounts": accounts,
	}); err != nil {
		s.logger.Warn("failed to write json response", "err", err)
	}
}

// ---------------- consolidated handler ----------------
func (s *Server) handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}

	// read file
	file, header, err := r.FormFile("statement")
	if err != nil {
		s.respondError(w, r, http.StatusBadRequest, "failed to read file", err)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "failed to read file", err)
		return
	}

	// parse local transactions once
	localTxs, err := s.parser.ProcessBytes(data, header.Filename)
	if err != nil {
		s.respondError(w, r, http.StatusBadRequest, "failed to process file", err)
		return
	}

	// build csv list for html table
	sort.Slice(localTxs, func(i, j int) bool { return localTxs[i].Date() < localTxs[j].Date() })
	filename := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)) + "-ynabu.csv"
	s.transactions.Store(filename, localTxs)

	txs := make([]Transaction, len(localTxs))
	for i, t := range localTxs {
		txs[i] = Transaction{Date: t.Date(), Payee: t.Payee(), Memo: t.Memo(), Amount: t.Amount()}
	}

	// check if reconciliation requested (token + budget + account provided)
	token := r.FormValue("token")
	budgetID := r.FormValue("budget_id")
	accountID := r.FormValue("account_id")

	var lines []string
	var toAdd, inSync int

	if token != "" && budgetID != "" && accountID != "" {
		ynabClient := ynab.New(token)
		remoteTxs, err := ynabClient.Transaction().GetTransactionsByAccount(budgetID, accountID, nil)
		if err != nil {
			s.respondError(w, r, http.StatusBadGateway, "failed to fetch remote transactions", err)
			return
		}
		report := executors.BuildReport(localTxs, remoteTxs, s.config.UseCustomID)
		lines = make([]string, 0, len(report.Items))
		for _, entry := range report.Items {
			prefix := "="
			if entry.Status == executors.ToAdd {
				prefix = "+"
			}
			line := fmt.Sprintf("%s %s | %-30s | R$ %.2f | %s", prefix, entry.Local.Date(), entry.Local.Payee(), entry.Local.Amount(), entry.Local.ID())
			lines = append(lines, line)
		}
		toAdd = report.MissingCount()
		inSync = report.InSyncCount()
		s.logger.Info("reconciliation complete", "file", header.Filename, "to_add", toAdd, "in_sync", inSync)
	}

	if err := s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"file":    filename,
		"data":    txs,
		"lines":   lines,
		"to_add":  toAdd,
		"in_sync": inSync,
	}); err != nil {
		s.logger.Warn("failed to write json response", "err", err)
	}
}

// ---------------- file download handler ----------------

// Transaction represents a simplified transaction for JSON responses.
type Transaction struct {
	Date   string  `json:"date"`
	Payee  string  `json:"payee"`
	Memo   string  `json:"memo"`
	Amount float64 `json:"amount"`
}

// handleFiles serves the generated CSV for a previously processed statement.
// ---------------- apply (plan + create) handler ----------------
func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	// TODO: Work in menory instead of temp file
	file, header, err := r.FormFile("statement")
	if err != nil {
		s.respondError(w, r, http.StatusBadRequest, "statement file required", err)
		return
	}
	defer file.Close()
	data, _ := io.ReadAll(file)

	budgetID := r.FormValue("budget_id")
	accountID := r.FormValue("account_id")
	if budgetID == "" {
		s.respondError(w, r, http.StatusBadRequest, "budget_id required", nil)
		return
	}
	if accountID == "" {
		s.respondError(w, r, http.StatusBadRequest, "account_id required", nil)
		return
	}

	// save temp file for executor to read
	tmp := filepath.Join(os.TempDir(), header.Filename)
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "failed to write temp file", err)
		return
	}

	stmt := &models.Statement{FilePath: tmp, BudgetID: budgetID, AccountID: accountID}

	token := r.FormValue("token")
	if token == "" {
		s.respondError(w, r, http.StatusBadRequest, "token required", nil)
		return
	}

	ynabCli := ynab.New(token)
	exec := executors.New(s.logger, s.config, ynabCli)

	if err := exec.Apply(stmt); err != nil {
		s.respondError(w, r, http.StatusBadGateway, "apply failed", err)
		return
	}

	if err := s.writeJSON(w, http.StatusOK, map[string]any{"status": "applied"}); err != nil {
		s.logger.Warn("failed to write json response", "err", err)
	}
}

// ---------------- file download handler ----------------

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if filename == "" {
		s.respondError(w, r, http.StatusBadRequest, "filename required", nil)
		return
	}

	// Retrieve cached transactions
	value, ok := s.transactions.Load(filename)
	if !ok {
		s.respondError(w, r, http.StatusNotFound, "file not found", nil)
		return
	}
	txs, ok := value.([]*models.Transaction)
	if !ok {
		s.respondError(w, r, http.StatusInternalServerError, "internal type assertion error", nil)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if _, err := w.Write(csv.Create(txs, nil)); err != nil {
		s.logger.Warn("failed to write csv response", "err", err)
	}
}

// --- helpers ---

// writeJSON encodes v as JSON with the given status and writes headers.
func (s *Server) writeJSON(w http.ResponseWriter, status int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// respondError logs the error and returns a minimal JSON error body.
func (s *Server) respondError(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	if err != nil {
		s.logger.Warn("request error", "status", status, "msg", message, "err", err, "method", r.Method, "path", r.URL.Path)
	} else {
		s.logger.Warn("request error", "status", status, "msg", message, "method", r.Method, "path", r.URL.Path)
	}
	_ = s.writeJSON(w, status, map[string]string{
		"status": "error",
		"error":  message,
	})
}

// withLogging wraps a handler to log request start/end and recover panics.
func (s *Server) withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("http request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic recovered", "panic", rec, "method", r.Method, "path", r.URL.Path)
				s.respondError(w, r, http.StatusInternalServerError, "internal server error", fmt.Errorf("panic: %v", rec))
			}
		}()
		next(w, r)
	}
}
