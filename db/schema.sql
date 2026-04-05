-- Schema for vps-webhook SQLite database

CREATE TABLE IF NOT EXISTS webhooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    script_path TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT 1,
    http_method TEXT NOT NULL DEFAULT 'POST'
);