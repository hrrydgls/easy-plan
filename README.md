# Easy Plan

A self-hosted calendar for tagging days and counting them. Define your own
kinds of days (💼 Work, 🏖️ Holiday, 🎌 National holiday…), tag any day
with one or more of them, and Easy Plan totals them per month or per year.

- Accounts with name, email and password (no email verification yet)
- As many calendars per person as you like, each starting fresh
- Per-calendar events, each counted monthly, yearly, or not at all
- Month view with ISO week numbers, swipe or arrow keys to change month
- Light and dark mode

One Go binary (API + embedded web UI) backed by SQLite. Nothing else to
install on the server.

## Install on a server

```bash
curl -fsSL https://raw.githubusercontent.com/hrrydgls/easy-plan/main/install/install.sh | bash
```

The installer downloads the `easyplan` binary, creates an unprivileged
`easyplan` system user, and installs+starts a systemd service on port
`9092`. Open `http://<your-server-ip>:9092/register` and create your account.

Options (pass after `bash -s --`):

```bash
curl -fsSL .../install.sh | bash -s -- --port 8080 --close-registration
```

| Option                 | What it does                                          |
| ---------------------- | ----------------------------------------------------- |
| `--port <n>`           | Port to listen on (default `9092`)                    |
| `--version <tag>`      | Install a specific release (default: latest)          |
| `--close-registration` | Stop new sign-ups (do this after creating your account) |
| `--binary <path>`      | Install a binary you built yourself                   |

Re-running the installer upgrades the binary and keeps your data.

```bash
systemctl status easyplan     # check it's running
journalctl -u easyplan -f     # tail logs
```

Settings live in `/etc/easyplan/easyplan.env`, data in
`/var/lib/easyplan/easyplan.db` (back up this one file).

### HTTPS

Put Easy Plan behind a reverse proxy such as Caddy and set
`EASYPLAN_SECURE_COOKIES=true` in the env file:

```
plan.example.com {
	reverse_proxy 127.0.0.1:9092
}
```

## Project layout

```
server/   Go backend: SQLite store, auth, JSON API, and the embedded
          frontend build (server/internal/web/dist)
web/      SvelteKit + Tailwind frontend (built as a static SPA)
install/  install.sh, the one-command installer
```

## Development

Requires Go 1.24+, Node 22+, and pnpm.

```bash
make dev-server   # API on :9092, data in ./.dev-data
make dev-web      # UI on http://localhost:5173 (proxies /api to :9092)
```

Build everything into one binary, the same way a release does:

```bash
make build        # builds web/, embeds it, builds ./easyplan
./easyplan        # http://localhost:9092
```

| Flag               | Env var                   | Default  |
| ------------------ | ------------------------- | -------- |
| `--port`           | `EASYPLAN_PORT`           | `9092`   |
| `--host`           | `EASYPLAN_HOST`           | all      |
| `--data-dir`       | `EASYPLAN_DATA_DIR`       | `./data` |
| `--registration`   | `EASYPLAN_REGISTRATION`   | `true`   |
| `--secure-cookies` | `EASYPLAN_SECURE_COOKIES` | `false`  |

### API

All under `/api`, JSON in and out, session cookie auth.

| Method | Path                                  | Body                                  |
| ------ | ------------------------------------- | ------------------------------------- |
| POST   | `/auth/register`                      | `{name, email, password}`             |
| POST   | `/auth/login`                         | `{email, password}`                   |
| POST   | `/auth/logout`                        | `{}`                                  |
| GET    | `/me`                                 |                                       |
| GET    | `/calendars`                          |                                       |
| POST   | `/calendars`                          | `{name, starter}`                     |
| GET    | `/calendars/{id}`                     | returns events and tagged days        |
| PATCH  | `/calendars/{id}`                     | `{name}`                              |
| DELETE | `/calendars/{id}`                     |                                       |
| PUT    | `/calendars/{id}/event-types`         | `[{id, name, emoji, count}]` (full list; `id: 0` creates) |
| PUT    | `/calendars/{id}/days/{YYYY-MM-DD}`   | `{eventTypeIds: [..]}`                |

## Publishing releases

`install.sh` downloads `easyplan-linux-amd64` / `easyplan-linux-arm64`
from this repo's latest GitHub release.
[release.yml](.github/workflows/release.yml) builds and publishes both
whenever a `v*.*.*` tag is pushed:

```bash
git tag v0.1.0
git push origin v0.1.0
```

[ci.yml](.github/workflows/ci.yml) type-checks and builds the frontend
and vets/builds the backend on every push and PR to `main`.

If your GitHub repo isn't `hrrydgls/easy-plan`, change `REPO` at the top
of `install/install.sh` and the URL above.

## Notes on the design

- Sessions are stored in SQLite (only a hash of each token), so restarts
  don't log anyone out. They last 30 days.
- Passwords are hashed with PBKDF2-SHA256 (600,000 rounds) from Go's
  standard library.
- Login and registration are rate limited per IP (10 a minute).
- The SQLite driver ([ncruces/go-sqlite3](https://github.com/ncruces/go-sqlite3))
  is pure Go, so binaries are built with `CGO_ENABLED=0` and run on any
  Linux distro.
