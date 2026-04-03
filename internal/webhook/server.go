package webhook

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vps-webhook/internal/db"
)

type Server struct {
	db      *db.DB
	logsDir string
	token   string
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
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

func NewServer(database *db.DB, logsDir string, token string) *Server {
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatalf("failed to create logs directory: %v", err)
	}
	return &Server{db: database, logsDir: logsDir, token: token}
}

func (s *Server) Handler() http.Handler {
	return s.loggingMiddleware(http.HandlerFunc(s.handleWebhook))
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("%s %s - %d (%v) from %s",
			r.Method,
			r.URL.Path,
			rw.statusCode,
			duration,
			r.RemoteAddr,
		)
	})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// 1. Check Bearer token authentication
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	providedToken := strings.TrimPrefix(authHeader, "Bearer ")
	if subtle.ConstantTimeCompare([]byte(providedToken), []byte(s.token)) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Lookup webhook in DB
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

	// 3. Check HTTP method
	if r.Method != webhook.HttpMethod {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 4. Limit body size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

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
	log.Printf("running script %s with log %s in goroutine", webhook.ScriptPath, logPath)

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
