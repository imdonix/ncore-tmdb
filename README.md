# ncore-tmdb

A Go service that bridges TMDB (The Movie Database) with NCore torrent tracker and qBittorrent. Includes a **mobile-friendly NCore search client** (React + shadcn) embedded in the binary, plus a TMDB reverse-proxy with an improved torrent widget.

## Features

- **NCore web client** — search, filter, paginate, and open torrent details
- **TMDB proxy** — browse movies on a proxied TMDB site; header/nav switches between clients
- **Torrent widget** on TMDB movie pages — clicking a torrent opens the NCore client (no instant download)
- **qBittorrent** integration from the torrent detail page
- Frontend assets **embedded** into the final Go binary (`go:embed`)
- Docker and Kubernetes/Helm support

## Prerequisites

- TMDB API key ([get one here](https://www.themoviedb.org/settings/api))
- Access to NCore tracker (via [ncore-go](https://github.com/imdonix/ncore-go) REST API / `imdonix/ncore` image)
- qBittorrent instance
- [Bun](https://bun.sh) (to build frontends)
- Go 1.25+

## Running with Docker Compose

1. Create a `.env` file:
```env
TMDB_API_KEY=your_tmdb_api_key
NCORE_USER=your_ncore_username
NCORE_PASS=your_ncore_password
```

2. Start the services:
```bash
docker compose up -d --build
```

This starts three containers:
- `ncore` — NCore API (port 1001)
- `qbit` — qBittorrent (port 1002)
- `ncore-tmdb` — main service (port 8080)

## Local Development

```bash
# Single command: bun install + build SPA/widget + embed into Go binary
make

# Run
./bin/ncore-tmdb
# or
make run

# Hot-reload Go (rebuilds frontends first)
make dev
```

### Frontend only (Vite via bun)

```bash
cd webapp && bun run dev      # proxies /api → :8080
cd widget && bun run build
```

## App routes

| Path | Description |
|------|-------------|
| `/` | Proxied TMDB site (unchanged asset handling) |
| `/ncore` | NCore search client |
| `/ncore/torrent/:id` | Torrent detail + qBit / .torrent download |
| `/widget/*` | Embedded widget assets |
| `/api/*` | REST API |

TMDB pages get an **NCore** button in the header linking to `/ncore`. The NCore app header links back to TMDB (`/`).

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/health` | Health check |
| `GET /api/movie/:tmdbID` | Movie details + torrents (for widget) |
| `GET /api/tv/:tmdbID` | TV details + torrents |
| `GET /api/download/:id` | Download torrent file (cached DB row) |
| `GET /api/qbit/download/:id` | Send torrent to qBittorrent (cached) |
| `POST /api/ncore/search` | Search nCore (ncore-go API) |
| `GET /api/ncore/torrent/:id` | Torrent details |
| `GET /api/ncore/download/:id` | Download .torrent by id |
| `POST /api/ncore/qbit/:id` | Send to qBittorrent by id |
| `GET /api/ncore/types` | Category list for UI |
| `GET /api/ncore/recommended` | Recommended torrents |
| `GET /api/ncore/activity` | Hit & Run activity |

### Search body example

```json
{
  "pattern": "inception",
  "type": "hd_hun",
  "where": "name",
  "sort_by": "seeders",
  "sort_order": "DESC",
  "page": 1
}
```

## Build layout

```
webapp/                 # React + Vite + Tailwind + shadcn SPA
widget/                 # React widget injected into TMDB pages
widget/public/ncore.png # Provider icon for the widget
internal/static/webapp  # Built SPA (embedded by Go)
internal/static/widget  # Built widget + snippet.html (embedded)
```

`make` (or `make build`) runs bun install/build for both frontends, then `go build` with embedded assets. The Dockerfile multi-stage build does the same with `oven/bun`.

## Helm (Kubernetes)

```bash
kubectl create secret generic ncore-tmdb-secrets \
  --from-literal=TMDB_API_KEY=your_tmdb_api_key \
  --from-literal=NCORE_USER=your_ncore_username \
  --from-literal=NCORE_PASS=your_ncore_password \
  --from-literal=QBIT_USER=admin \
  --from-literal=QBIT_PASS=adminadmin

helm install ncore-tmdb ./k8s/ncore-tmdb
```
