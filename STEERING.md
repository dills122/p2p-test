# Project Steering

## Mission

Build a federated, resilient peer-to-peer networking foundation in Go that can safely support blockchain-style message propagation and consensus experiments.

## Current Direction

- Primary transport target is `go-libp2p`.
- Existing gRPC path remains the baseline and test oracle until libp2p parity is achieved.
- Consensus work starts only after networking SLOs and failure-mode tests pass.

## Engineering Priorities

1. Networking correctness over feature velocity.
2. Security controls by default for all external network paths.
3. Deterministic behavior and reproducible tests.
4. Clear modular boundaries between transport, protocol, runtime, state, and consensus.
5. Small, reviewable increments with measurable acceptance criteria.

## Go Quality Standards

- Keep packages cohesive and interfaces small.
- Prefer composition over deep inheritance-like patterns.
- Return explicit errors with context (`fmt.Errorf("...: %w", err)`).
- Use `context.Context` for all networked and cancellable operations.
- Avoid hidden global state except carefully scoped config/log singletons.
- Add table-driven tests for protocol logic and edge cases.

## Networking and P2P Standards

- Validate all inbound data before processing.
- Enforce message envelope versioning, TTL, and dedupe.
- Bound queues, retries, and goroutine fanout to avoid resource exhaustion.
- Use exponential backoff with jitter for reconnects.
- Track peer health score and cooldown misbehaving peers.
- Define and monitor delivery, latency, and reconnect SLOs.

## Security Standards

- No plaintext secrets in code, logs, or fixtures.
- Authenticate peer identity before accepting privileged message types.
- Sign and verify consensus-critical messages.
- Sanitize logs to avoid leaking private keys or sensitive payload content.
- Keep dependency updates regular and review transitive risk.

## Definition of Network-Ready

- Integration tests cover node churn, duplicate messages, temporary partitions, and restart recovery.
- Local soak tests meet agreed SLO thresholds.
- No deadlocks, panic loops, or unbounded memory growth in churn scenarios.

