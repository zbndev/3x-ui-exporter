# 3x-ui-exporter

Prometheus exporter for [3X-UI](https://github.com/MHSanaei/3x-ui) panel. Collects client traffic, online status, server health, and node metrics via native panel REST API.

## Metrics

All metrics are prefixed with `xui_`.

**Client**
- Upload/download bytes, traffic quota, expiry time per client (with `email`, `inbound_remark`, `protocol` labels)
- Online status (`1`/`0`) for every client

**Inbound**
- Aggregate upload/download/total per inbound

**Server**
- CPU, memory, swap, disk, network I/O, load averages, uptime, TCP connections, Xray running state

**Nodes**
- Status, latency, CPU/memory usage, uptime, client/online/depleted counts per node

## Requirements

- 3X-UI panel with API token (Settings → Security → API Token)
- Bearer token authentication

## Installation

### Docker (recommended)

```bash
mv .env.example .env
docker compose up -d
```

### Binary

Download from [Releases](https://github.com/zbndev/3x-ui-exporter/releases) or build from source:

```bash
go build -o 3x-ui-exporter .
```

## Configuration

Environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BASE_URL` | Yes | — | Panel URL (e.g. `https://example.com:2053`) |
| `TOKEN` | Yes | — | Bearer API token |
| `EXPORTER_PORT` | No | `9847` | Metrics server port |
| `SCRAPE_TIMEOUT` | No | `10` | API request timeout in seconds |

## Usage

### Run

```bash
export BASE_URL=https://your-panel.example.com
export TOKEN=your-api-token
./3x-ui-exporter
```

### Endpoints

- `GET /metrics` — Prometheus scrape endpoint
- `GET /health` — Health check (`200 ok` / `503 unhealthy`)

### Prometheus config

```yaml
scrape_configs:
  - job_name: "3x-ui"
    static_configs:
      - targets: ["localhost:9847"]
```

## License

MIT
