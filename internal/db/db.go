//go:generate sqlc generate -f ../../sqlc.yaml

package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"vps-webhook/internal/db/sqlc"
)

type DB struct {
	conn   *sql.DB
	queries *sqlc.Queries
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

	d := &DB{
		conn:    conn,
		queries: sqlc.New(conn),
	}
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
	ctx := context.Background()
	result, err := d.queries.CreateWebhook(ctx, sqlc.CreateWebhookParams{
		Path:       path,
		ScriptPath: scriptPath,
		HttpMethod: httpMethod,
	})
	if err != nil {
		return nil, err
	}
	return &Webhook{
		ID:         result.ID,
		Path:       result.Path,
		ScriptPath: result.ScriptPath,
		Active:     result.Active,
		HttpMethod: result.HttpMethod,
	}, nil
}

func (d *DB) ListWebhooks() ([]Webhook, error) {
	ctx := context.Background()
	results, err := d.queries.ListWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	webhooks := make([]Webhook, len(results))
	for i, r := range results {
		webhooks[i] = Webhook{
			ID:         r.ID,
			Path:       r.Path,
			ScriptPath: r.ScriptPath,
			Active:     r.Active,
			HttpMethod: r.HttpMethod,
		}
	}
	return webhooks, nil
}

func (d *DB) GetWebhookByPath(path string) (*Webhook, error) {
	ctx := context.Background()
	result, err := d.queries.GetWebhookByPath(ctx, path)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Webhook{
		ID:         result.ID,
		Path:       result.Path,
		ScriptPath: result.ScriptPath,
		Active:     result.Active,
		HttpMethod: result.HttpMethod,
	}, nil
}

func (d *DB) GetWebhook(id int64) (*Webhook, error) {
	ctx := context.Background()
	result, err := d.queries.GetWebhook(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Webhook{
		ID:         result.ID,
		Path:       result.Path,
		ScriptPath: result.ScriptPath,
		Active:     result.Active,
		HttpMethod: result.HttpMethod,
	}, nil
}

func (d *DB) UpdateWebhook(id int64, path, scriptPath string, active bool, httpMethod string) error {
	if httpMethod == "" {
		httpMethod = "POST"
	}
	ctx := context.Background()
	return d.queries.UpdateWebhook(ctx, sqlc.UpdateWebhookParams{
		Path:       path,
		ScriptPath: scriptPath,
		Active:     active,
		HttpMethod: httpMethod,
		ID:         id,
	})
}

func (d *DB) DeleteWebhook(id int64) error {
	ctx := context.Background()
	return d.queries.DeleteWebhook(ctx, id)
}
