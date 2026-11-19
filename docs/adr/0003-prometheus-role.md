# ADR-0003: Use Prometheus for platform metrics, PostgreSQL for product history

- Status: Accepted
- Date: 2026-11-19

## Context

The product needs both operational metrics about itself and user-facing history for individual monitored endpoints.

## Decision

Prometheus scrapes API/worker operational metrics. PostgreSQL stores individual probe outcomes and serves monitor history in the MVP.

## Consequences

- Clear separation between platform telemetry and product data.
- No need to force per-monitor history into Prometheus labels, which could create high cardinality.
- Long-term retention and aggregation for probe history will need deliberate database design.
