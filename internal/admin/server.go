package admin

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"vps-webhook/internal/admin/templates"
	"vps-webhook/internal/db"
)

type Server struct {
	db           *db.DB
	logsDir      string
	webhookToken string
	httpClient   *http.Client
}

func NewServer(database *db.DB, logsDir string, webhookToken string) *Server {
	return &Server{
		db:           database,
		logsDir:      logsDir,
		webhookToken: webhookToken,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("POST /webhooks", s.handleCreateWebhook)
	mux.HandleFunc("GET /webhooks/{id}/edit", s.handleEditWebhookForm)
	mux.HandleFunc("GET /webhooks/{id}", s.handleGetWebhookRow)
	mux.HandleFunc("PUT /webhooks/{id}", s.handleUpdateWebhook)
	mux.HandleFunc("DELETE /webhooks/{id}", s.handleDeleteWebhook)
	mux.HandleFunc("POST /webhooks/test", s.handleTestWebhook)
	mux.HandleFunc("GET /logs", s.handleListLogs)
	mux.HandleFunc("GET /logs/{file}", s.handleViewLog)
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	webhooks, err := s.db.ListWebhooks()
	if err != nil {
		log.Printf("error listing webhooks: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	templates.Index(webhooks).Render(r.Context(), w)
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.FormValue("path"))
	scriptPath := strings.TrimSpace(r.FormValue("script_path"))
	httpMethod := strings.TrimSpace(r.FormValue("http_method"))

	if path == "" || scriptPath == "" {
		http.Error(w, "path and script_path are required", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if httpMethod == "" {
		httpMethod = "POST"
	}

	if _, err := s.db.CreateWebhook(path, scriptPath, httpMethod); err != nil {
		log.Printf("error creating webhook: %v", err)
		http.Error(w, fmt.Sprintf("error: %v", err), http.StatusBadRequest)
		return
	}

	webhooks, err := s.db.ListWebhooks()
	if err != nil {
		log.Printf("error listing webhooks: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	templates.WebhookList(webhooks).Render(r.Context(), w)
}

func (s *Server) handleEditWebhookForm(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	webhook, err := s.db.GetWebhook(id)
	if err != nil {
		log.Printf("error getting webhook: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if webhook == nil {
		http.NotFound(w, r)
		return
	}

	templates.WebhookEditRow(*webhook).Render(r.Context(), w)
}

func (s *Server) handleGetWebhookRow(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	webhook, err := s.db.GetWebhook(id)
	if err != nil {
		log.Printf("error getting webhook: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if webhook == nil {
		http.NotFound(w, r)
		return
	}

	templates.WebhookRow(*webhook).Render(r.Context(), w)
}

func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	path := strings.TrimSpace(r.FormValue("path"))
	scriptPath := strings.TrimSpace(r.FormValue("script_path"))
	activeStr := r.FormValue("active")
	httpMethod := strings.TrimSpace(r.FormValue("http_method"))

	if path == "" || scriptPath == "" {
		http.Error(w, "path and script_path are required", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	active := activeStr == "on" || activeStr == "true" || activeStr == "1"

	if httpMethod == "" {
		httpMethod = "POST"
	}

	if err := s.db.UpdateWebhook(id, path, scriptPath, active, httpMethod); err != nil {
		log.Printf("error updating webhook: %v", err)
		http.Error(w, fmt.Sprintf("error: %v", err), http.StatusBadRequest)
		return
	}

	webhooks, err := s.db.ListWebhooks()
	if err != nil {
		log.Printf("error listing webhooks: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	templates.WebhookList(webhooks).Render(r.Context(), w)
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteWebhook(id); err != nil {
		log.Printf("error deleting webhook: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	webhooks, err := s.db.ListWebhooks()
	if err != nil {
		log.Printf("error listing webhooks: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	templates.WebhookList(webhooks).Render(r.Context(), w)
}

func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(r.FormValue("webhook_id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid webhook_id", http.StatusBadRequest)
		return
	}

	webhook, err := s.db.GetWebhook(id)
	if err != nil {
		log.Printf("error getting webhook: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if webhook == nil {
		http.NotFound(w, r)
		return
	}

	webhookURL := strings.TrimSpace(r.FormValue("webhook_url"))
	if webhookURL == "" {
		http.Error(w, "webhook_url is required", http.StatusBadRequest)
		return
	}

	// Ensure the URL doesn't have a trailing slash
	webhookURL = strings.TrimSuffix(webhookURL, "/")

	// Build the target URL
	targetURL := webhookURL + webhook.Path

	body := r.FormValue("body")

	// Create the request
	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewReader([]byte(body))
	}

	req, err := http.NewRequest(webhook.HttpMethod, targetURL, bodyReader)
	if err != nil {
		log.Printf("error creating request: %v", err)
		http.Error(w, "error creating request", http.StatusInternalServerError)
		return
	}

	// Add Authorization header
	req.Header.Set("Authorization", "Bearer "+s.webhookToken)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("error executing webhook request: %v", err)
		templates.TestResult(0, "", "Error: "+err.Error()).Render(r.Context(), w)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading response body: %v", err)
		templates.TestResult(resp.StatusCode, "", "Error reading response: "+err.Error()).Render(r.Context(), w)
		return
	}

	templates.TestResult(resp.StatusCode, string(respBody), "").Render(r.Context(), w)
}

func (s *Server) handleListLogs(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(s.logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			templates.LogList(nil).Render(r.Context(), w)
			return
		}
		log.Printf("error reading logs dir: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	if len(files) > 50 {
		files = files[:50]
	}

	templates.LogList(files).Render(r.Context(), w)
}

func (s *Server) handleViewLog(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")

	if strings.Contains(file, "..") || strings.Contains(file, "/") {
		http.Error(w, "invalid file name", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(filepath.Join(s.logsDir, file))
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
