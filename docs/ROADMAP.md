# Roadmap

## M0 — Foundation
- [ ] Confirm repository baseline and Go version
- [ ] Create repository structure
- [ ] Add config loading and `.env.example`
- [ ] Add API entry point and `/healthz`
- [ ] Add Docker Compose PostgreSQL
- [ ] Add Makefile and CI
- [ ] Add initial architecture and product docs

## M1 — Monitor management
- [ ] Define domain entities and validation
- [ ] Add PostgreSQL migrations
- [ ] Implement repository
- [ ] Implement create/list/get/update/delete use cases
- [ ] Add HTTP handlers and OpenAPI contract
- [ ] Add unit and integration tests

## M2 — Probe execution
- [ ] Implement safe HTTP probe runner
- [ ] Add timeout, redirect, DNS/IP protections
- [ ] Persist probe results
- [ ] Add worker process and bounded concurrency
- [ ] Add due-monitor claiming and crash recovery
- [ ] Add tests using local test servers

## M3 — Reliability and observability
- [ ] Structured logging and request IDs
- [ ] Prometheus metrics
- [ ] OpenTelemetry traces
- [ ] Graceful shutdown
- [ ] Load tests and profiling

## M4 — Alerts
- [ ] Alert rules
- [ ] Incident lifecycle
- [ ] Notification abstraction
- [ ] Webhook delivery and retry policy

## M5 — User experience
- [ ] React + TypeScript dashboard
- [ ] Availability and latency charts
- [ ] Monitor detail and history views
- [ ] Authentication and RBAC

## M6 — Advanced capabilities
- [ ] Multi-location probes
- [ ] SLO and error-budget views
- [ ] Anomaly detection
- [ ] AI-assisted incident summaries
