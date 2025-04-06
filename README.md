# Telemetry Tracker

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker Build Status](https://img.shields.io/badge/Docker-Build-blue)](Dockerfile)

A lightweight event processing system written in Go. This service accepts telemetry events via HTTP, stores them in a PostgreSQL database, and exposes basic operational metrics for OpenTelemetry scraping.

This project serves as the core application component for a larger infrastructure learning project involving Kubernetes (EKS), Terraform, Helm, CI/CD with GitHub Actions, and monitoring.

## Features

- **Event Ingestion:** Accepts JSON events via `POST /events`.
- **Database Storage:** Persists events to a PostgreSQL database.
- **Metrics Exposition:** Exposes Prometheus-compatible metrics at `/metrics`.
- **Health Check:** Provides a simple health endpoint at `/healthz`.
- **Structured Logging:** Outputs JSON logs for better observability.
- **Containerized:** Includes a `Dockerfile`.
- **Docker Compose:** Includes `docker-compose.yml` for easy local development setup.
- **Graceful Shutdown:** Handles termination signals for clean shutdown.

## Technology Stack

- **Language:** Go (1.23+)
- **Database:** PostgreSQL
- **HTTP Router:** [chi](https://github.com/go-chi/chi)
- **Logging:** Go standard library `slog`
- **Metrics:** [prometheus/client_golang](https://github.com/prometheus/client_golang)
- **Database Driver:** [pgx](https://github.com/jackc/pgx)
- **Containerization:** Docker, Docker Compose

## Prerequisites

Before you begin, ensure you have the following installed:

1.  **Go:** Version 1.23 or later. ([Download Go](https://golang.org/dl/))
2.  **Docker & Docker Compose:** Docker Compose is included with Docker Desktop. If using Linux, ensure you have `docker-compose` installed separately if needed. ([Download Docker](https://www.docker.com/products/docker-desktop/))
3.  **make (Optional):** For using Makefile shortcuts.
    - On macOS: Install Xcode Command Line Tools (`xcode-select --install`).
    - On Debian/Ubuntu: `sudo apt-get update && sudo apt-get install build-essential`.
    - On Fedora/CentOS: `sudo yum groupinstall "Development Tools"`.
4.  **curl:** Or any other HTTP client for testing endpoints. (Usually pre-installed on Linux/macOS).

## Getting Started

1.  **Clone the Repository:**

    ```bash
    git clone https://github.com/kakhavain/telemetry-tracker.git # <-- Use your repo URL
    cd telemetry-tracker
    ```

2.  **Install Go Dependencies:**

    ```bash
    go mod tidy
    ```

3.  **Prepare Configuration:**
    - Copy the environment variable template:
      ```bash
      cp .env.example .env
      ```
    - Edit the new `.env` file:
      - **Set `DB_PASSWORD`:** Change `mysecretpassword` to a secure password. This password will be used by Docker Compose to initialize the Postgres container.
      - **Verify `DB_HOST`:** Ensure `DB_HOST` is set to `postgres`. This is the service name defined in `docker-compose.yml` and allows the app container to find the database container.
      - Adjust `DB_USER`, `DB_NAME`, `DB_PORT`, `APP_PORT` only if necessary.

**Important:** The `.env` file contains sensitive information and **should NOT be committed to Git**. The `.gitignore` file is configured to ignore it.

## Running the Application (Docker Compose - Recommended)

The easiest way to run the application and its database locally is using Docker Compose. Make sure you have created and correctly configured your `.env` file first.

1.  **Start Services:**

    - **Using Make:**
      ```bash
      make compose-up
      ```
      _(This command runs `docker-compose up -d --build`)_
    - **Without Make:**
      ```bash
      docker-compose up -d --build
      ```      
      This will build the application's Docker image (if it's not already built or if the source code changed) and start both the`app`and`postgres` containers in the background. The app will wait for the database to be healthy before fully starting.

2.  **View Logs:**

    - **Using Make:**
      ```bash
      make compose-logs
      ```
    - **Without Make:**
      ```bash
      docker-compose logs -f app
      ```

3.  **Check Status:**

    - **Using Make:**
      ```bash
      make compose-ps
      ```
    - **Without Make:**
      ```bash
      docker-compose ps
      ```

4.  **Stop Services:** - **Using Make:**
    `bash
      make compose-down
      ` - **Without Make:**
    `bash
docker-compose down
`
    This stops and removes the containers but **preserves the database data** stored in the Docker volume (`postgres-data`).

Once running (`make compose-up`), the application will be available at `http://localhost:8080` (or the port specified by `APP_PORT` in your `.env` file).

## Running Locally (Without Docker Compose)

You can still run the Go application directly on your host, but you will need to:

1.  **Run PostgreSQL separately:** Use the `docker run...` command from previous instructions or install and run Postgres natively.
2.  **Set `DB_HOST` appropriately:** In your `.env` file or exported environment variables, set `DB_HOST` to `localhost` (or wherever your separately running Postgres is accessible).
3.  **Export Environment Variables:** Make sure all required variables from `.env` are exported into your shell session.
    ```bash
    export $(grep -v '^#' .env | xargs) # Simple way, use with caution
    make run
    # or
    # go run ./cmd/server/main.go
    ```

## Testing the Application

Ensure the application is running (e.g., via `make compose-up`).

1.  **Send an Event:**

    ```bash
    curl -X POST http://localhost:8080/events \
         -H "Content-Type: application/json" \
         -d '{ ... event payload ... }' # Use a valid JSON event
    ```

    - Expected: `{"status": "accepted"}` (HTTP 202). Check `make compose-logs`.

2.  **Check Health:**

    ```bash
    curl http://localhost:8080/healthz
    ```

    - Expected: `OK` (HTTP 200).

3.  **Check Metrics:**

    ```bash
    curl http://localhost:8080/metrics
    ```

    - Expected: Prometheus metrics text format.

4.  **Verify Database Storage:**
    - Connect to Postgres. If using the Compose setup, you can connect to `localhost:5432` with the user/password/db from your `.env` file using `psql` or a GUI tool.
    - Run: `SELECT * FROM events ORDER BY received_at DESC LIMIT 10;`
    - Expected: See your posted events.

## Project Structure

```
├── cmd/server/ # Main application entrypoint
├── internal/ # Internal application code
├── .env.example # Example environment variables file
├── .gitignore # Files/directories ignored by Git
├── Dockerfile # For building the app's Docker image
├── Makefile # Convenience commands (build, compose, etc.)
├── docker-compose.yml # Docker Compose configuration
├── go.mod # Go module definition
├── go.sum # Go module checksums
├── README.md
└── schemas/schema.sql # PostgreSQL database schema
```

## Next Steps

- Provision cloud infrastructure (VPC, EKS, RDS) using Terraform.
- Package the application using Helm.
- Set up CI/CD pipelines with GitHub Actions.
- Integrate with AWS Secrets Manager.
- Deploy Prometheus and Grafana.

## License

This project is licensed under the MIT License.
