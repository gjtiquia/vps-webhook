package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"vps-webhook/internal/db"
)

type Server struct {
	db      *db.DB
	logsDir string
}

type RequestLog struct {
	Timestamp   string              `json:"timestamp"`
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	Path        string              `json:"path"`
	Query       map[string][]string `json:"query"`
	Headers     map[string][]string `json:"headers"`
	Body        string              `json:"body"`
	RemoteAddr  string              `json:"remote_addr"`
	WebhookID   int64               `json:"webhook_id"`
	WebhookPath string              `json:"webhook_path"`
}

func NewServer(database *db.DB, logsDir string) *Server {
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatalf("failed to create logs directory: %v", err)
	}
	return &Server{db: database, logsDir: logsDir}
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.handleWebhook)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	webhook, err := s.db.GetWebhookByPath(r.URL.Path)
	if err != nil {
		log.Printf("error looking up webhook for path %s: %v", r.URL.Path, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if webhook == nil {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading body: %v", err)
		http.Error(w, "error reading body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	now := time.Now().UTC()
	reqLog := RequestLog{
		Timestamp:   now.Format(time.RFC3339),
		Method:      r.Method,
		URL:         r.URL.String(),
		Path:        r.URL.Path,
		Query:       r.URL.Query(),
		Headers:     r.Header,
		Body:        string(body),
		RemoteAddr:  r.RemoteAddr,
		WebhookID:   webhook.ID,
		WebhookPath: webhook.Path,
	}

	logFilename := now.Format("2006-01-02-15-04-05") + "-request.json"
	logPath := filepath.Join(s.logsDir, logFilename)

	logData, err := json.MarshalIndent(reqLog, "", "  ")
	if err != nil {
		log.Printf("error marshaling request log: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(logPath, logData, 0644); err != nil {
		log.Printf("error writing log file: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("webhook matched: path=%s script=%s log=%s", webhook.Path, webhook.ScriptPath, logPath)

	absLogPath, _ := filepath.Abs(logPath)
	absScriptPath, _ := filepath.Abs(webhook.ScriptPath)

	go func() {
		cmd := exec.Command("bash", absScriptPath, absLogPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("script %s failed: %v", webhook.ScriptPath, err)
		} else {
			log.Printf("script %s completed successfully", webhook.ScriptPath)
		}
	}()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "webhook received\n")
}
