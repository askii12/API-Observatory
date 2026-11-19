# ADR-0001: Start with a modular monolith

- Status: Accepted
- Date: 2026-11-19

## Context

API Observatory needs an API process and background probe execution. Starting with multiple independently deployed microservices would add networking, deployment, observability, and consistency costs before product boundaries are validated.

## Decision

Use one Go codebase with clear internal modules and two executable entry points:
- `cmd/api`
- `cmd/worker`

Both processes share domain and adapter packages. They may be deployed separately.

## Consequences

### Positive
- Simple local development and deployment.
- Shared types and business rules.
- Clear module boundaries without distributed-system overhead.
- Worker can scale independently by running more replicas once job claiming is safe.

### Negative
- Modules share a release lifecycle.
- Boundaries require discipline and code review.
- A shared database can become a coupling point.

## Revisit when

A component needs independent scaling, release cadence, ownership, or fault isolation that cannot be achieved by separate processes.
