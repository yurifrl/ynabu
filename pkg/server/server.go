package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/parser"
	"github.com/yurifrl/ynabu/pkg/types"
)

// Server handles HTTP requests for YNAB file processing
type Server struct {
	config types.Config
	logger *log.Logger
	mux    *http.ServeMux
}

// New creates a new HTTP server
func New(config types.Config, logger *log.Logger) *Server {
	s := &Server{
		config: config,
		logger: logger,
		mux:    http.NewServeMux(),
	}

	s.setupRoutes()
	return s
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	s.logger.Info("server starting", "address", addr)
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/process", s.handleProcessFile)
}

func (s *Server) handleProcessFile(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed", fmt.Errorf("expected POST, got %s", r.Method))
		return
	}

	// Parse multipart form with 32MB max memory
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "failed to parse form", err)
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Get the file from the request
	file, header, err := r.FormFile("file")
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "failed to get file", err)
		return
	}
	defer file.Close()

	// Read file bytes
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to read file", err)
		return
	}

	parser := parser.New(s.logger)

	// Process bytes using parser
	outputBytes, err := parser.ProcessBytes(fileBytes, header.Filename)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to process file", err)
		return
	}

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", header.Filename))

	// Write output bytes directly to response
	if _, err := w.Write(outputBytes); err != nil {
		s.logger.Error("failed to write response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string, err error) {
	s.logger.Error(message, "error", err)
	response := map[string]string{
		"error": fmt.Sprintf("%s: %v", message, err),
	}
	s.jsonResponse(w, status, response)
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to encode response", "error", err)
	}
}
