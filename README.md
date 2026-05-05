# ncore-tmdb

A Go service that bridges TMDB (The Movie Database) with NCore torrent tracker and qBittorrent. Fetch movie/TV metadata from TMDB, search for torrents on NCore, and download them directly via qBittorrent through a simple REST API.

## Features

- TMDB metadata fetching for movies and TV shows
- Automatic torrent search on NCore
- Direct torrent download via qBittorrent integration
- Docker and Kubernetes/Helm support
- Simple REST API

## Prerequisites

- TMDB API key ([get one here](https://www.themoviedb.org/settings/api))
- Access to NCore tracker
- qBittorrent instance

## Running with Docker Compose

1. Create a `.env` file:
```env
TMDB_API_KEY=your_tmdb_api_key
NCORE_USER=your_ncore_username
NCORE_PASS=your_ncore_password
```

2. Start the services:
```bash
docker compose up -d
```

This starts three containers:
- `ncore` - NCore API proxy (port 1001)
- `qbit` - qBittorrent (port 1002, downloads in `./tmp/qbit_downloads`)
- `ncore-tmdb` - Main service (port 8080)

## Running with Helm (Kubernetes)

1. Create the secrets:
```bash
kubectl create secret generic ncore-tmdb-secrets \
  --from-literal=TMDB_API_KEY=your_tmdb_api_key \
  --from-literal=NCORE_USER=your_ncore_username \
  --from-literal=NCORE_PASS=your_ncore_password \
  --from-literal=QBIT_USER=admin \
  --from-literal=QBIT_PASS=adminadmin
```

2. Install the chart:
```bash
helm install ncore-tmdb ./k8s/ncore-tmdb
```

3. Update values in `k8s/ncore-tmdb/values.yaml` to match your setup (hosts, ingress, etc.)

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/health` | Health check |
| `GET /api/movie/:tmdbID` | Get movie details and torrents |
| `GET /api/tv/:tmdbID` | Get TV show details and torrents |
| `GET /api/download/:id` | Download torrent file |
| `GET /api/qbit/download/:id` | Send torrent to qBittorrent |

## Local Development

```bash
# Install dependencies
go mod download

# Run with hot reload (requires air)
make dev

# Or build and run manually
make build
./bin/ncore-tmdb
```
