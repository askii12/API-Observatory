# ADR-0002: Use PostgreSQL as the initial source of truth

- Status: Accepted
- Date: 2026-11-19

## Context

The MVP needs durable monitor configuration, scheduled execution metadata, and historical probe results. Redis and a message broker may become useful later, but introduce additional operational dependencies.

## Decision

Use PostgreSQL for monitor configuration, result history, and initial due-work claiming. Defer Redis until a measurable requirement appears.

## Consequences

- Fewer moving parts in local development.
- Transactions and row locking can support safe worker claiming.
- High-volume time-series workloads may eventually require partitioning or a dedicated metrics store.
- Queue throughput and scheduling contention must be measured before deciding to introduce another system.
