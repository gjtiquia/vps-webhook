---
name: Webhook Security Hardening
overview: "Add three security features to the webhook server: global Bearer token authentication, per-webhook HTTP method restriction, and request body size limiting."
todos:
  - id: auth-token
    content: Add WEBHOOK_TOKEN env var loading in cmd/webhook/main.go, Bearer token check in webhook handler, .env.example, and .gitignore update
    status: pending
  - id: method-restriction
    content: Add http_method column to DB, update CRUD operations, add method check in webhook handler, update admin forms/templates
    status: pending
  - id: body-size-limit
    content: Add http.MaxBytesReader(w, r.Body, 1<<20) in webhook handler before io.ReadAll
    status: pending
  - id: templ-generate
    content: Run templ generate to regenerate Go files from updated .templ templates
    status: pending
  - id: build-verify
    content: Run go build ./... to verify everything compiles
    status: pending
isProject: false
---

# Webhook Security Hardening

Three changes to the public-facing webhook server: auth token, method restriction, body size limit.

## 1. Global Bearer Token Authentication

Load a `WEBHOOK_TOKEN` environment variable at startup. Every incoming request must include `Authorization: Bearer <token>` matching it. Use `crypto/subtle.ConstantTimeCompare` to avoid timing attacks.

**Files:**
- [cmd/webhook/main.go](cmd/webhook/main.go) -- read `WEBHOOK_TOKEN` from env via `os.Getenv`, pass to `NewServer`. Fatal if empty (require it to be set).
- [internal/webhook/server.go](internal/webhook/server.go) -- add `token string` field to `Server` struct. At the top of `handleWebhook`, extract Bearer token from `Authorization` header, compare with `subtle.ConstantTimeCompare`. Return `401 Unauthorized` on mismatch.
- `.env.example` (new) -- document the expected var: `WEBHOOK_TOKEN=your-secret-token-here`
- [.gitignore](.gitignore) -- add `.env`

## 2. Per-Webhook HTTP Method Restriction

Add an `http_method` column to the `webhooks` table so each webhook specifies which HTTP method it accepts (e.g. `POST`). Default to `POST`.

**Files:**
- [internal/db/db.go](internal/db/db.go):
  - Add `HttpMethod string` to `Webhook` struct
  - Add migration: `ALTER TABLE webhooks ADD COLUMN http_method TEXT NOT NULL DEFAULT 'POST'` (wrapped in a "column exists" check since SQLite doesn't support `IF NOT EXISTS` for columns)
  - Update all `SELECT` queries to include `http_method`
  - Update `CreateWebhook` and `UpdateWebhook` signatures to accept `httpMethod`
- [internal/webhook/server.go](internal/webhook/server.go) -- after fetching the webhook, check `r.Method != webhook.HttpMethod` and return `405 Method Not Allowed`
- [internal/admin/server.go](internal/admin/server.go) -- read `http_method` form value in `handleCreateWebhook` and `handleUpdateWebhook`, pass through to DB calls
- [internal/admin/templates/index.templ](internal/admin/templates/index.templ):
  - Add a `<select>` dropdown for HTTP method in the "Add Webhook" form (options: GET, POST, PUT, DELETE)
  - Add "Method" column to the webhooks table display
  - Add method dropdown in `WebhookEditRow`

## 3. Request Body Size Limit

Wrap `r.Body` with `http.MaxBytesReader` before reading, capped at 1MB.

**Files:**
- [internal/webhook/server.go](internal/webhook/server.go) -- add `r.Body = http.MaxBytesReader(w, r.Body, 1<<20)` before the `io.ReadAll` call. The existing error handling on `io.ReadAll` will catch the `MaxBytesError` and return 400.

## Request handling order in `handleWebhook`

After all changes, the handler checks in this order:

```
1. Auth token   --> 401 if missing/wrong
2. DB lookup    --> 404 if no match
3. Method check --> 405 if wrong method
4. Size-limited body read --> 400 if too large
5. Log + execute script
```

Auth is checked first (before DB lookup) so unauthenticated requests are rejected immediately without touching the database.
