# Telemetry Tracker

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker Build Status](https://img.shields.io/badge/Docker-Build-blue)](Dockerfile)

A lightweight event processing system written in Go. This service accepts telemetry events via HTTP, stores them in a PostgreSQL database, and exposes basic operational metrics for OpenTelemetry scraping.

This project serves as the core application component for a larger infrastructure learning project involving Kubernetes (EKS), Terraform, Helm, CI/CD with GitHub Actions, and monitoring.

---

## Features

- **Event Ingestion:** Accepts JSON events via `POST /events`.
- **Database Storage:** Persists events to a PostgreSQL database.
- **Metrics Exposition:** Exposes Prometheus-compatible metrics at `/metrics`.
- **Health Check:** Provides a simple health endpoint at `/healthz`.
- **Structured Logging:** Outputs JSON logs for better observability.
- **Containerized:** Includes a `Dockerfile`.
- **Docker Compose:** Includes `docker-compose.yml` for local dev.
- **Graceful Shutdown:** Handles termination signals for clean shutdown.

---

## Technology Stack

- **Language:** Go (1.23+)
- **HTTP Router:** [chi](https://github.com/go-chi/chi)
- **Database:** PostgreSQL via [pgx](https://github.com/jackc/pgx)
- **Logging:** Go standard library [`slog`](https://pkg.go.dev/log/slog)
- **Metrics & Tracing:** [OpenTelemetry](https://opentelemetry.io/), Prometheus, Grafana Tempo
- **Log Aggregation:** OTEL Logs + Grafana Loki
- **Collector:** [OpenTelemetry Collector Contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib)
- **Visualization:** [Grafana](https://grafana.com/)
- **Deployment:** Docker, Docker Compose, Helmfile, Kubernetes

---

## Prerequisites

Install the following tools:

- [Go (1.23+)](https://golang.org/dl/)
- [Docker + Docker Compose](https://www.docker.com/products/docker-desktop)
- `make` (optional)
- `curl`
- `helm`, `helmfile`, `kubectl`, `kind` (for Kubernetes deployments)

---

## Getting Started

```bash
git clone https://github.com/kakhavain/telemetry-tracker.git
cd telemetry-tracker
```

```bash
go mod tidy
```

```bash
cp .env.example .env
# Edit .env to configure DB_PASSWORD, DB_HOST, etc.
```

---

## Running with Docker Compose (Local Dev)

```bash
make compose-up
```

```bash
make compose-logs
```

```bash
make compose-ps
```

```bash
make compose-down
```

Access app at: `http://localhost:8080`

---

## Running Without Docker Compose

1. Start a local Postgres instance (via Docker or native).
2. Set `DB_HOST` to `localhost` in `.env`.
3. Export your environment variables:

```bash
export $(grep -v '^#' .env | xargs)
```

4. Run the app:

```bash
make run
# or
go run ./cmd/server/main.go
```

---

## Running with Helmfile (Kubernetes Deployment)

This is separate from Docker Compose and intended for local Kind or remote clusters.

### Start Kind Cluster

```bash
kind create cluster --name telemetry-tracker --config kind-multinode.yaml
```

### Load Image into Kind

```bash
make docker-build
```

```bash
kind load docker-image telemetry-tracker:latest --name telemetry-tracker
```

### Deploy with Helmfile

```bash
helmfile sync
```

### Tear Down

```bash
helmfile destroy
```

### Debugging & Logs

```bash
kubectl get pods -n default
kubectl logs deployment/telemetry-tracker-telemetry-tracker -n default
kubectl port-forward svc/telemetry-tracker-telemetry-tracker 8080:8080
kubectl port-forward svc/grafana 3000:80
```

---

## Testing

```bash
curl -X POST http://localhost:8080/events \
     -H "Content-Type: application/json" \
     -d '{ "type": "test", "message": "hello world" }'
```

```bash
curl http://localhost:8080/healthz
```

---

## Verify DB Events

```sql
SELECT * FROM events ORDER BY received_at DESC LIMIT 10;
```

---

## Useful Commands

```bash
helmfile sync
```

```bash
helmfile destroy
```

```bash
kubectl get pods -n default
```

```bash
kubectl delete pvc --all -n default
```

```bash
kubectl get svc -A
```

```bash
kubectl port-forward svc/telemetry-tracker-telemetry-tracker 8080:8080
```

```bash
kubectl port-forward svc/grafana 3000:80
```

```bash
kubectl logs deployment/telemetry-tracker-telemetry-tracker -n default
```

```bash
kind create cluster --name telemetry-tracker --config kind-multinode.yaml
```

```bash
kind load docker-image telemetry-tracker:latest --name telemetry-tracker
```

```bash
kubectl rollout restart deployment telemetry-tracker-telemetry-tracker
```

---

## License

This project is licensed under the MIT License.
