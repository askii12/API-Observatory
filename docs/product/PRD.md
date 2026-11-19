# Product Requirements Document — API Observatory

- Status: Draft v0.1
- Owner: Project team
- Last updated: 2026-11-19

## 1. Summary

API Observatory is a platform for monitoring HTTP APIs from configured probes. It helps developers answer:
- Is my endpoint reachable?
- How long does it take to respond?
- When did failures begin?
- Is the issue isolated or recurring?

The first release is a single-instance deployment for individual developers and small engineering teams.

## 2. Problem

Developers often discover API outages from users rather than from monitoring. Lightweight projects may not have a simple, self-hostable place to register endpoints, inspect availability history, and receive actionable alerts.

## 3. Target users

- Backend developers maintaining several APIs.
- Small engineering teams operating internal or public HTTP services.
- Students and portfolio reviewers evaluating practical backend engineering.

## 4. MVP scope

### Must have
- Register, edit, list, view, and delete an HTTP monitor.
- Validate monitor URL and polling interval.
- Execute an HTTP probe with a strict timeout.
- Record status code, success/failure, latency, timestamp, and concise error category.
- Retrieve latest status and historical results.
- Provide `/healthz` and `/readyz` for platform health.
- Provide structured logs and basic Prometheus metrics.
- Run locally with documented commands and Docker Compose.

### Should have
- Worker process separated from API process.
- Retry policy for transient infrastructure errors, without hiding actual failed checks.
- Pagination for historical results.
- OpenAPI contract.

### Later
- Alert rules (threshold, consecutive failures, recovery).
- Email/Telegram/webhook notifications.
- React dashboard.
- Team accounts and RBAC.
- Distributed probe locations.
- AI-assisted incident summaries.

## 5. Out of scope for MVP

- Browser synthetic monitoring.
- Full packet capture.
- Full-featured APM replacement.
- Automatic remediation.
- Billing and SaaS subscription management.

## 6. Core user journeys

### Create a monitor
1. User submits a name, URL, and interval.
2. API validates the request and persists the monitor.
3. API returns the monitor ID and configuration.

### Inspect monitor health
1. User requests a monitor.
2. API returns configuration and latest known result.
3. User requests history with pagination.

### Execute a probe
1. Worker obtains due monitors.
2. Probe runner performs a bounded HTTP request.
3. Result is normalized and persisted.
4. Metrics are updated.

## 7. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-01 | Create an HTTP monitor | Must |
| FR-02 | List monitors with pagination | Must |
| FR-03 | Retrieve a monitor by ID | Must |
| FR-04 | Update monitor configuration | Must |
| FR-05 | Delete a monitor | Must |
| FR-06 | Execute HTTP probe with timeout | Must |
| FR-07 | Persist probe result | Must |
| FR-08 | Query result history | Must |
| FR-09 | Expose platform health endpoints | Must |
| FR-10 | Expose platform Prometheus metrics | Should |

## 8. Non-functional requirements

- Reliability: probe execution must have a configurable timeout and bounded resource use.
- Security: prevent SSRF, validate schemes, restrict redirects, and avoid leaking secrets in logs.
- Performance: paginated list/history endpoints; database indexes for monitor/time queries.
- Maintainability: domain logic separated from transport and persistence.
- Observability: structured logs, request IDs, platform metrics, and OpenTelemetry instrumentation.
- Portability: local development on Windows, macOS, and Linux through Go and Docker Compose.

## 9. Initial API surface

| Method | Path | Purpose |
|---|---|---|
| GET | `/healthz` | Liveness |
| GET | `/readyz` | Readiness (checks required dependencies) |
| GET | `/api/v1/monitors` | List monitors |
| POST | `/api/v1/monitors` | Create monitor |
| GET | `/api/v1/monitors/{id}` | Get monitor |
| PATCH | `/api/v1/monitors/{id}` | Update monitor |
| DELETE | `/api/v1/monitors/{id}` | Delete monitor |
| GET | `/api/v1/monitors/{id}/results` | Result history |
| GET | `/metrics` | Prometheus scrape endpoint |

## 10. Success criteria for M0

- A fresh clone can be started using documented steps.
- `go test ./...` passes.
- API starts with environment-based configuration.
- `/healthz` returns HTTP 200.
- `/readyz` accurately reports dependency readiness once PostgreSQL integration is added.
- CI runs formatting checks, tests, and `go vet`.

## 11. Risks

- SSRF through user-controlled monitor URLs.
- Probe scheduling drift or duplicate execution.
- Unbounded result growth.
- False alerts from transient network failures.
- Overengineering before validating the core monitoring workflow.

## 12. Open questions

- Which notification channels are required first?
- What default polling interval should be supported?
- Should the first release support only public endpoints or trusted private networks too? (Security implications are significant.)
