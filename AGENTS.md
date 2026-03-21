# Agent Guidelines

## Scope

These rules apply to any automated or human-assisted coding agent modifying this repository.

## Primary Objectives

1. Preserve network correctness and safety.
2. Keep transport-agnostic architecture intact.
3. Improve test coverage alongside behavior changes.
4. Avoid regressions in CLI usability and node operability.

## Required Workflow

1. Read relevant files before editing.
2. Make the smallest viable change for the stated objective.
3. Add or update tests when behavior changes.
4. Run validation commands before finishing:
- `go test ./...`
- `go vet ./...`
5. Summarize risk, assumptions, and follow-up work in the final update.

## Architecture Constraints

- Do not couple consensus logic directly to a specific transport implementation.
- Keep message schema/versioning in protocol-focused code, not transport adapters.
- Keep peer management policy in runtime/registry layers.
- Preserve clear boundaries between:
- transport
- protocol
- network runtime
- state
- consensus

## Networking Safety Rules

- Never trust inbound payloads; validate size, format, and required fields.
- Enforce message TTL and duplicate suppression.
- Use bounded channels/queues and explicit backpressure handling.
- Use context deadlines/timeouts for all network calls.
- Apply retry with exponential backoff and jitter, not tight loops.
- Avoid unbounded goroutine creation in message handlers.

## Security Rules

- Do not log secrets, private keys, or full sensitive payloads.
- Require identity verification for privileged operations.
- Sign and verify consensus-critical messages.
- Prefer secure defaults; any insecure mode must be explicit and documented.

## Code Style and Maintainability

- Follow idiomatic Go naming and package structure.
- Keep functions focused; extract helpers instead of deeply nested conditionals.
- Wrap errors with actionable context.
- Add brief comments only where logic is non-obvious.
- Avoid speculative abstractions that are not used.

## Test Expectations

- Add table-driven unit tests for protocol and validation code.
- Add integration tests for multi-node scenarios:
- broadcast convergence
- reconnect after peer restart
- duplicate suppression
- temporary network partition recovery
- Favor deterministic tests over timing-sensitive flaky flows.

## Change Management

- Never rewrite unrelated files during focused tasks.
- Call out any tradeoffs that reduce security/reliability, even temporarily.
- If a change needs a migration path, include rollout steps in docs.

