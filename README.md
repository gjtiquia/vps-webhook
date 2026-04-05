# vps-webhook

a deadsimple server that runs bash scripts from webhooks

use cases
- github action on push -> call webhook -> runs redeploy script (eg. kill process, git pull, run server)
- uptime kuma notification -> call webhook -> runs notify script (eg. phone/sms/email/bot notification)

## quick start

```bash
# generate sqlc code (run after modifying db/query/*.sql files)
sqlc generate

# generate templ files
templ generate

# start webhook server (exposed to web via reverse proxy)
go run ./cmd/webhook/main.go -port 9000

# start admin dashboard (accessible via Tailscale or localhost only)
go run ./cmd/admin/main.go -port 9001
```

1. Open the admin dashboard at `http://localhost:9001`
2. Add a webhook with a path (e.g. `/deploy`) and a script path (e.g. `./scripts/example.sh`)
3. Send a request to the webhook server: `curl -X POST http://localhost:9000/deploy -d '{"ref":"main"}'`
4. The script runs with the request log JSON file path as the first argument

## flags

Both servers accept the same flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `9000` (webhook) / `9001` (admin) | Server port |
| `-db` | `./db.sqlite` | Path to SQLite database |
| `-logs` | `./logs` | Directory to store request JSON logs |

## how it works

**webhook server** (`cmd/webhook/main.go`)
- listens for HTTP requests on all paths
- looks up the request path in the SQLite database
- if a match is found, saves the full request as a JSON file in `./logs/`
- executes the configured bash script, passing the log file path as `$1`
- the script can read the JSON to access headers, body, query params, etc.

**admin dashboard** (`cmd/admin/main.go`)
- web UI to manage webhooks (add/delete) and view recent logs
- built with Go, templ, Tailwind CSS (CDN), and HTMX

## project structure

```
cmd/
  webhook/main.go     # webhook server entry point
  admin/main.go       # admin dashboard entry point
internal/
  db/db.go            # SQLite database layer
  webhook/server.go   # webhook HTTP handler
  admin/
    server.go         # admin HTTP handlers
    templates/         # templ templates (layout, index)
scripts/
  example.sh          # example webhook script
logs/                  # request JSON logs (gitignored)
```

## tech stack ramblings

will be two go servers

one go server will be the webhook server, listening for requests

the other go server will be the admin dashboard

both servers will run on different ports

webhook server will be exposed to the web via reverse proxy on a VPS (eg. Caddy)

admin dashboard will only be accessible via something like Tailscale

webhook server implementation notes
- listens to request
- on receive request, check if path matches any in sqlite db
- ignore if none matches
- if have path match, save request in "./logs/" named as "yyyy-mm-dd-hh-mm-ss-request.json" UTC time file
- calls bash script specified in sqlite, sqlite saves the bash script file path, and passes the request.json file path as an arg
- the request.json should contain everything the request has, eg. the url, headers, body, query params
- the bash script will then interpret the request.json however it likes (eg. perhaps running a python/go/js script to read the json and perform various actions)

admin dashboard implementation notes
- tech stack: go, templ, tailwind, htmx, sqlc
- db is "./db.sqlite" by default
- styling preference: minimal, bg-stone-900 bg, text-stone-50 text, no font size difference unless necessary, just bold with "# " prefix for h1, "## " for h2, and so on
