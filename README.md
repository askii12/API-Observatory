# API Observatory

API Observatory is a self-hosted API observability platform for monitoring HTTP endpoints, measuring availability and latency, investigating failures, and notifying teams when service health changes.

> Current stage: project foundation. The repository is intentionally structured as a modular monolith with separately runnable API and worker processes. Avoid premature microservices.

## Product goals

- Register HTTP endpoints to monitor.
- Execute scheduled probes and record status, latency, and response metadata.
- Expose health and metrics endpoints for the platform itself.
- Provide an API for querying monitors and their latest results.
- Add alerting, dashboards, distributed tracing, and anomaly detection incrementally.

## Non-goals for the first milestone

- Full distributed tracing backend.
- AI-generated incident diagnosis.
- Kubernetes deployment.
- Multi-region probe network.
- Complex billing, teams, and RBAC.

## Technology choices

| Area | Choice | Initial role |
|---|---|---|
| Backend | Go | API and probe worker |
| Database | PostgreSQL | Monitors and probe results |
| Cache / queue | Redis | Deferred; introduce when needed |
| Metrics | Prometheus | Platform metrics and scrape target |
| Instrumentation | OpenTelemetry | Traces and metrics instrumentation |
| Frontend | React + TypeScript | Later milestone |
| Local environment | Docker Compose | PostgreSQL and supporting services |
| CI | GitHub Actions | Formatting, tests, build |

## Repository layout

```text
api-observatory/
├── cmd/
│   ├── api/                 # HTTP API entry point
│   └── worker/              # Scheduled probe worker entry point
├── internal/
│   ├── config/              # Environment/config loading
│   ├── domain/              # Core entities and domain errors
│   ├── monitor/              # Monitor use cases and interfaces
│   ├── probe/                # HTTP probe execution
│   ├── results/              # Probe result use cases
│   ├── storage/postgres/     # PostgreSQL adapters
│   └── transport/http/       # HTTP handlers, routing, middleware
├── migrations/               # SQL migrations
├── deploy/
│   ├── docker/               # Dockerfiles
│   └── prometheus/            # Prometheus configuration
├── web/                      # Future React application
├── docs/
│   ├── adr/                  # Architecture Decision Records
│   ├── api/                  # API contracts
│   ├── product/              # Product requirements
│   └── runbooks/             # Operational procedures
├── scripts/                  # Developer scripts
├── .github/workflows/        # CI workflows
├── .env.example
├── .gitignore
├── docker-compose.yml
├── Makefile
└── go.mod
```

## Quick start

Requirements: Go (version declared in `go.mod`), Git, Docker Desktop with Compose.

1. Copy `.env.example` to `.env` and review values.
2. Start dependencies: `docker compose up -d postgres`.
3. Run the API: `go run ./cmd/api`.
4. Check `http://localhost:8080/healthz`.

## Development commands

```bash
make fmt       # format Go code
make test      # run tests
make vet       # run go vet
make run-api   # run API
make infra-up  # start local infrastructure
make infra-down
```

## Engineering principles

1. Keep domain logic independent from HTTP and database implementations.
2. Prefer explicit interfaces at boundaries; do not create interfaces without a consumer or testing need.
3. Use context propagation and bounded timeouts for I/O.
4. Store timestamps in UTC.
5. Make probe execution safe: restrict targets to valid HTTP(S) URLs, set timeouts, and prevent access to loopback, private, link-local, and cloud metadata addresses in production.
6. Add migrations for schema changes.
7. Every meaningful feature includes tests and documentation.

## Milestones

- M0 — Foundation: repository structure, configuration, health endpoint, local PostgreSQL.
- M1 — Monitor management: create/list/get/update/delete monitors.
- M2 — Probe execution: HTTP checks, latency measurement, result persistence.
- M3 — Scheduling and worker reliability.
- M4 — Alert rules and notification channels.
- M5 — Dashboard and historical analytics.
- M6 — OpenTelemetry, advanced diagnostics, and anomaly detection.

## Security note

A URL-monitoring service can become an SSRF proxy if arbitrary targets are fetched without controls. The probe subsystem must validate destinations, resolve DNS safely, re-check resolved IPs, limit redirects, and block private/reserved ranges. Treat this as a release blocker before exposing the service to untrusted users.
