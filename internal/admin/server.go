package admin

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"vps-webhook/internal/admin/templates"
	"vps-webhook/internal/db"
)

type Server struct {
	db      *db.DB
	logsDir string
}

func NewServer(database *db.DB, logsDir string) *Server {
	return &Server{db: database, logsDir: logsDir}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("POST /webhooks", s.handleCreateWebhook)
	mux.HandleFunc("GET /webhooks/{id}/edit", s.handleEditWebhookForm)
	mux.HandleFunc("GET /webhooks/{id}", s.handleGetWebhookRow)
	mux.HandleFunc("PUT /webhooks/{id}", s.handleUpdateWebhook)
	mux.HandleFunc("DELETE /webhooks/{id}", s.handleDeleteWebhook)
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

	if path == "" || scriptPath == "" {
		http.Error(w, "path and script_path are required", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if _, err := s.db.CreateWebhook(path, scriptPath); err != nil {
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

	if path == "" || scriptPath == "" {
		http.Error(w, "path and script_path are required", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	active := activeStr == "on" || activeStr == "true" || activeStr == "1"

	if err := s.db.UpdateWebhook(id, path, scriptPath, active); err != nil {
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
