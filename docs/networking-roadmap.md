# Federated P2P Networking Roadmap

## 1) Current state audit (code + docs + usage)

### What exists today

- CLI entrypoint and commands:
  - `p2p-test start` launches a node and interactive shell (`send`, `add-peer`, `peers`, `exit`) in [`cmd/node/start.go`](/Users/dsteele/go/src/p2p-test/cmd/node/start.go)
  - `p2p-test pingTest` starts two nodes and sends one ping each in [`cmd/node/pingTest.go`](/Users/dsteele/go/src/p2p-test/cmd/node/pingTest.go)
- Core node runtime:
  - Node construction and transport wiring in [`node/node.go`](/Users/dsteele/go/src/p2p-test/node/node.go)
  - gRPC server and ping handler in [`node/service.go`](/Users/dsteele/go/src/p2p-test/node/service.go)
  - outbound ping fanout + peer discovery via metadata in [`node/ping_ops.go`](/Users/dsteele/go/src/p2p-test/node/ping_ops.go)
  - in-memory peer registry with status/last-seen in [`node/registry.go`](/Users/dsteele/go/src/p2p-test/node/registry.go)
- Transport abstraction:
  - `Transport` interface exists and gRPC implementation is isolated in [`node/transport.go`](/Users/dsteele/go/src/p2p-test/node/transport.go)
- Event stream for UI/logging:
  - async event bus in [`node/events.go`](/Users/dsteele/go/src/p2p-test/node/events.go)
  - shell event rendering and de-dup display behavior in [`cmd/node/start.go`](/Users/dsteele/go/src/p2p-test/cmd/node/start.go)
- Dependency status:
  - project builds with `go test ./...` and has no tests yet.

### How to use now

- Help:
  - `go run ./main.go --help`
- Interactive node:
  - `go run ./main.go start --address 127.0.0.1:10000 --listener-addresses 127.0.0.1:10001`
- Second node:
  - `go run ./main.go start --address 127.0.0.1:10001 --listener-addresses 127.0.0.1:10000`
- Shell commands:
  - `send <message>`
  - `add-peer <host:port>`
  - `peers`
  - `exit`

### Documentation status

- Root README is accurate for basic local usage and peer learning flow in [`Readme.md`](/Users/dsteele/go/src/p2p-test/Readme.md).
- Root README covers current local usage and node behavior in [`Readme.md`](/Users/dsteele/go/src/p2p-test/Readme.md).

## 2) Network readiness analysis (before blockchain features)

### Strengths

- Clear transport seam already exists (`Transport` interface).
- Peer learning works through metadata propagation.
- Node CLI UX is simple and usable for manual experiments.
- Event hooks already provide a place to attach metrics/tracing later.

### Gaps blocking “network layer is stable”

1. No automated tests for delivery guarantees, churn, reconnect, or partitions.
2. No message envelope with protocol version/TTL/hop count/signature.
3. No receive-path dedupe or replay defense in node runtime.
4. No persistent peer state or message cache; restarts lose operational context.
5. No backpressure controls (queue sizing, drop policies, worker limits).
6. No structured metrics/SLO instrumentation.
7. No transport failover policy or runtime backend switch yet.
8. `pingTest` is demo-only and does not assert outcomes.

### Immediate conclusion

Do not start consensus logic yet. First harden networking to measurable SLOs.

## 3) Re-evaluated architecture overview

### Design principle

Keep strict boundaries so transport experiments (gRPC vs `libp2p`) do not change higher-level protocol/state/consensus modules.

### Layered architecture

1. `transport`
- Purpose: peer connect/disconnect, send/receive bytes/messages, health signals.
- Implementations:
  - `grpc` adapter (existing baseline)
  - `libp2p` adapter (primary target)

2. `protocol`
- Purpose: canonical envelope + message codecs + validation.
- Owns: `msg_id`, `type`, `origin`, `ts`, `ttl`, `hop`, `sig`, `payload`.

3. `network runtime`
- Purpose: gossip rules, dedupe cache, retry policy, fanout, peer scoring.
- Owns operational policy, not consensus rules.

4. `state`
- Purpose: mempool, block store metadata, validator set, checkpoints.

5. `consensus`
- Purpose: propose/vote/commit state machine over validated messages.

### Why this is right for this repo

- Existing code already separates transport from node API.
- Existing event bus can evolve into observability hooks.
- Existing code can host multiple transport backends without leaking implementation details upward.

## 4) Networking tech overview (target)

### 4.1 Protocol envelope

Add a single envelope type used across all network messages:

- `version` (protocol compatibility)
- `id` (globally unique, for dedupe)
- `type` (`PING`, `PEER_ANNOUNCE`, `TX`, `BLOCK_PROPOSAL`, `VOTE`)
- `origin` (node ID/public key)
- `timestamp`
- `ttl` (decrement per hop)
- `hop_count`
- `payload`
- `signature` (optional initially, required for consensus messages)

Rules:
- Drop invalid version.
- Drop expired TTL.
- Drop duplicates by `(id)` within TTL window.
- Validate type-specific schema before processing.

### 4.2 Peer and connection management

- Keep `PeerRegistry`, add:
  - score
  - failure count
  - last RTT
  - cooldown-until
- Add retry policy with exponential backoff + jitter.
- Add outbound worker pool and bounded queue.
- Add periodic peer reconciliation and liveness probes.

### 4.3 Gossip policy

- Deterministic fanout (`k` peers per message type).
- Prioritize healthy peers.
- Controlled rebroadcast budget.
- Message-level retry only for critical types.

### 4.4 Observability + SLOs

Metrics:
- `messages_sent_total`, `messages_recv_total`, `messages_dropped_total`
- `message_delivery_latency_ms`
- `peer_connect_total`, `peer_disconnect_total`, `peer_reconnect_total`
- `duplicate_messages_total`
- `queue_depth`

Initial SLO targets (local/dev):
- >= 99% delivery under 5-node 10-min soak
- P95 point-to-point message latency < 500ms on localhost
- reconnect within <= 5s after single-node restart
- zero panics/deadlocks under churn test

### 4.5 Transport abstraction target interface

Refine current transport interface toward message-oriented operations:

- `Start(localConfig) error`
- `Stop(ctx) error`
- `Send(ctx, peerID, envelope) error`
- `Broadcast(ctx, envelope) error`
- `Subscribe(handler)`
- `Peers() []PeerInfo`
- `Connect(peerAddr) error`

Keep this independent of gRPC/libp2p specifics.

## 5) Migration and build-out plan (work items)

## Phase 0: Baseline and guardrails

W0.1 Add `docs/` architecture docs and ADR for transport boundary.
W0.2 Add Make targets/scripts for smoke and soak tests.
W0.3 Add CI job running `go test ./...` and `go vet ./...`.

Definition of done:
- Docs and commands are reproducible by a new contributor.

## Phase 1: Stabilize current gRPC network path

W1.1 Add protocol envelope package and type validation.
W1.2 Integrate receive-path dedupe cache + TTL drop behavior.
W1.3 Add peer health scoring and reconnect backoff.
W1.4 Add structured logging fields (`peer`, `msg_id`, `type`, `latency_ms`, `result`).
W1.5 Add integration tests for:
- 3-node broadcast convergence
- duplicate suppression
- restart and rejoin
- temporary partition and healing

Definition of done:
- SLOs met in local soak test.
- No regressions in CLI behavior.

## Phase 2: Introduce pluggable transport backend switch (primary: libp2p)

W2.1 Add runtime config flag `--transport=grpc|libp2p`.
W2.2 Implement `libp2p` transport adapter behind same interface.
W2.3 Add contract tests shared by both backends.
W2.4 Capture backend comparison report (latency, reconnect, delivery).
Definition of done:
- Same high-level network tests pass against `grpc` and `libp2p`.

## Phase 3: Add transaction gossip (first blockchain-like workload)

W3.1 Define `Tx` schema and deterministic validation.
W3.2 Add mempool with size limits and dedupe.
W3.3 Gossip valid tx envelopes and apply anti-replay policy.
W3.4 Persist mempool snapshots and restore on startup.

Definition of done:
- 3+ nodes converge on near-identical mempool under churn.

## Phase 4: Consensus skeleton

W4.1 Add validator-set config and node identity keys.
W4.2 Add round state machine primitives (`height`, `round`, `step`).
W4.3 Add wire messages: `PROPOSE`, `PREVOTE`, `PRECOMMIT`, `COMMIT`.
W4.4 Add timeout and round-change mechanism.

Definition of done:
- Simulated single-proposer happy-path commit works locally.

## Phase 5: Fault tolerance and hardening

W5.1 Byzantine-ish test cases (equivocation, stale replay, late votes).
W5.2 Crash/restart recovery tests.
W5.3 Performance profiling and queue tuning.
W5.4 Security pass (signature verification mandatory on consensus paths).
W5.5 If running across home/cloud networks, add NAT traversal diagnostics (STUN) and connection strategy tuning.

Definition of done:
- Project is stable enough for iterative consensus experiments.

## 6) Supporting tools decisions

- `go-libp2p`: primary future network transport stack.
- `pion/stun`: use only when moving beyond localhost/LAN and validating NAT behavior.
- `mkcert`: preferred local TLS bootstrap for dev and integration tests.
- `certstrap`: optional if you want your own CA workflow and cert lifecycle scripts.
- `hashicorp/raft`: optional alternate track for crash-fault-tolerant replicated state machine experiments; not the default path for blockchain-style Byzantine consensus.
- `traefik`: optional deployment edge/proxy tool; not part of core P2P protocol.

## 7) Suggested execution order for next 1-2 weeks

1. W1.1 envelope package
2. W1.2 dedupe + TTL in receive path
3. W1.4 structured logs
4. W1.5 first 3-node integration tests
5. W1.3 reconnect/health scoring
6. W2.1 runtime transport switch scaffolding
7. W2.2 libp2p transport adapter skeleton

This order yields the highest confidence in network correctness before any consensus code.
