package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type Webhook struct {
	ID         int64
	Path       string
	ScriptPath string
	Active     bool
	HttpMethod string
}

func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return d, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) migrate() error {
	_, err := d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			script_path TEXT NOT NULL,
			active BOOLEAN NOT NULL DEFAULT 1
		);
	`)
	if err != nil {
		return err
	}

	// Add http_method column if it doesn't exist (SQLite doesn't support IF NOT EXISTS for columns)
	var columnCount int
	err = d.conn.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('webhooks') WHERE name = 'http_method'
	`).Scan(&columnCount)
	if err != nil {
		return err
	}

	if columnCount == 0 {
		_, err = d.conn.Exec(`ALTER TABLE webhooks ADD COLUMN http_method TEXT NOT NULL DEFAULT 'POST'`)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DB) CreateWebhook(path, scriptPath, httpMethod string) (*Webhook, error) {
	if httpMethod == "" {
		httpMethod = "POST"
	}
	result, err := d.conn.Exec(
		"INSERT INTO webhooks (path, script_path, http_method) VALUES (?, ?, ?)",
		path, scriptPath, httpMethod,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Webhook{ID: id, Path: path, ScriptPath: scriptPath, Active: true, HttpMethod: httpMethod}, nil
}

func (d *DB) ListWebhooks() ([]Webhook, error) {
	rows, err := d.conn.Query("SELECT id, path, script_path, active, http_method FROM webhooks ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var w Webhook
		if err := rows.Scan(&w.ID, &w.Path, &w.ScriptPath, &w.Active, &w.HttpMethod); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

func (d *DB) GetWebhookByPath(path string) (*Webhook, error) {
	var w Webhook
	err := d.conn.QueryRow(
		"SELECT id, path, script_path, active, http_method FROM webhooks WHERE path = ? AND active = 1",
		path,
	).Scan(&w.ID, &w.Path, &w.ScriptPath, &w.Active, &w.HttpMethod)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (d *DB) GetWebhook(id int64) (*Webhook, error) {
	var w Webhook
	err := d.conn.QueryRow(
		"SELECT id, path, script_path, active, http_method FROM webhooks WHERE id = ?",
		id,
	).Scan(&w.ID, &w.Path, &w.ScriptPath, &w.Active, &w.HttpMethod)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (d *DB) UpdateWebhook(id int64, path, scriptPath string, active bool, httpMethod string) error {
	if httpMethod == "" {
		httpMethod = "POST"
	}
	_, err := d.conn.Exec(
		"UPDATE webhooks SET path = ?, script_path = ?, active = ?, http_method = ? WHERE id = ?",
		path, scriptPath, active, httpMethod, id,
	)
	return err
}

func (d *DB) DeleteWebhook(id int64) error {
	_, err := d.conn.Exec("DELETE FROM webhooks WHERE id = ?", id)
	return err
}
