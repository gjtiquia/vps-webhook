# vps-webhook

a deadsimple server that runs bash scripts from webhooks

use cases
- github action on push -> call webhook -> runs redeploy script (eg. kill process, git pull, run server)
- uptime kuma notification -> call webhook -> runs notify script (eg. phone/sms/email/bot notification)

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

