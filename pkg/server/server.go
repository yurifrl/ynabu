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
    s.mux.HandleFunc("/", s.handleHome)

    // consolidated endpoint
    s.mux.HandleFunc("/api/process", s.handleProcess)
    s.mux.HandleFunc("/api/apply", s.handleApply)
    s.mux.HandleFunc("/api/files/", s.handleFiles)


}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
    data := struct{
        Accounts []config.Account
    }{Accounts: s.config.YNAB.Accounts}
    s.template.ExecuteTemplate(w, "index.html", data)
}

// ---------------- consolidated handler ----------------
func (s *Server) handleProcess(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // read file
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

    // parse local transactions once
    localTxs, err := s.parser.ProcessBytes(data, header.Filename)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to process file: %v", err), http.StatusBadRequest)
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
    budgetID := s.config.YNAB.BudgetID
    accountID := r.FormValue("account_id")

    var lines []string
    var toAdd, inSync int

    if token != "" && budgetID != "" && accountID != "" {
        ynabClient := ynab.New(token)
        remoteTxs, err := ynabClient.Transaction().GetTransactionsByAccount(budgetID, accountID, nil)
        if err != nil {
            http.Error(w, fmt.Sprintf("Failed to fetch remote transactions: %v", err), http.StatusInternalServerError)
            return
        }
        report := executors.BuildReport(localTxs, remoteTxs, s.config.UseCustomID)
        lines = make([]string, 0, len(report.Items))
        for _, entry := range report.Items {
            prefix := "="
            if entry.Status == executors.ToAdd {
                prefix = "+"
            }
            line := fmt.Sprintf("%s %s | %-30s | R$ %.2f", prefix, entry.Local.Date(), entry.Local.Payee(), entry.Local.Amount())
            lines = append(lines, line)
        }
        toAdd = report.MissingCount()
        inSync = report.InSyncCount()
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":  "success",
        "file":    filename,
        "data":    txs,
        "lines":   lines,
        "to_add":  toAdd,
        "in_sync": inSync,
    })
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
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // TODO: Work in menory instead of temp file
    file, header, err := r.FormFile("statement")
    if err != nil {
        http.Error(w, "statement file required", http.StatusBadRequest)
        return
    }
    defer file.Close()
    data, _ := io.ReadAll(file)

    accountID := r.FormValue("account_id")
    if accountID == "" {
        http.Error(w, "account_id required", http.StatusBadRequest)
        return
    }

    // save temp file for executor to read
    tmp := filepath.Join(os.TempDir(), header.Filename)
    if err := os.WriteFile(tmp, data, 0600); err != nil {
        http.Error(w, "failed temp write", 500)
        return
    }

    stmt := &models.Statement{FilePath: tmp, AccountID: accountID}

    ynabCli := ynab.New(s.config.YNAB.Token)
    exec := executors.New(s.logger, s.config, ynabCli)

    if err := exec.Apply(stmt); err != nil {
        http.Error(w, err.Error(), 502)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]any{"status": "applied"})
}

// ---------------- file download handler ----------------

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
    filename := strings.TrimPrefix(r.URL.Path, "/api/files/")
    if filename == "" {
        http.Error(w, "filename required", http.StatusBadRequest)
        return
    }

    // Retrieve cached transactions
    value, ok := s.transactions.Load(filename)
    if !ok {
        http.NotFound(w, r)
        return
    }
    txs, ok := value.([]*models.Transaction)
    if !ok {
        http.Error(w, "internal type assertion error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
    w.Write(csv.Create(txs, nil))
}

