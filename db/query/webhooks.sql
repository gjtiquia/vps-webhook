-- name: CreateWebhook :one
INSERT INTO webhooks (path, script_path, http_method)
VALUES (?, ?, ?)
RETURNING id, path, script_path, active, http_method;

-- name: ListWebhooks :many
SELECT id, path, script_path, active, http_method
FROM webhooks
ORDER BY id;

-- name: GetWebhookByPath :one
SELECT id, path, script_path, active, http_method
FROM webhooks
WHERE path = ? AND active = 1;

-- name: GetWebhook :one
SELECT id, path, script_path, active, http_method
FROM webhooks
WHERE id = ?;

-- name: UpdateWebhook :exec
UPDATE webhooks
SET path = ?, script_path = ?, active = ?, http_method = ?
WHERE id = ?;

-- name: DeleteWebhook :exec
DELETE FROM webhooks
WHERE id = ?;