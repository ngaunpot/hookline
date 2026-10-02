# hookline# Hookline

A webhook delivery service in Go. Your application hands it an event; it takes
responsibility for that event actually reaching every subscriber — with retries,
signed payloads and a full delivery log.

[![CI](https://github.com/ngaunpot/hookline/actions/workflows/ci.yml/badge.svg)](https://github.com/ngaunpot/hookline/actions/workflows/ci.yml)

> **Status: in active development.** The HTTP ingestion layer works and is
> tested. Persistence and the delivery worker are next — see the
> [roadmap](#roadmap) for what is and isn't built yet.

## The problem

When something happens in your system — a user signs up, a payment clears, an
identity check finishes — other systems need to know. Polling ("anything new?
anything new?") is wasteful and slow, so the standard answer is a webhook: you
POST to a URL the subscriber registered in advance.

Sending one POST is three lines of code. What makes this hard is that the
receiver is not always there:

- **Their server is down.** The POST fails. You have to try again later, backing
  off so you don't hammer a service that's already struggling.
- **Their server is slow.** If you deliver inline, your own signup endpoint
  blocks on someone else's infrastructure. Delivery has to be asynchronous.
- **Their response got lost.** They processed it, you didn't hear back, you
  retry — and they process the same payment twice. Receivers need a stable event
  ID to deduplicate against.
- **Anyone can POST to a public URL.** The receiver needs cryptographic proof
  the request came from you.
- **A customer says they never got the event.** Someone needs to be able to look
  up what was sent, when, and what came back.

Every company sending webhooks eventually builds this component. Hookline is
that component, standalone.

## How it works

```
                  ┌─────────────────────────────────────────┐
                  │              Hookline                   │
  your app        │                                         │        subscriber
  ────────────────┼──►  POST /events ──► store ──► queue    │
     fire &       │                                  │      │
     forget       │                                  ▼      │
   ◄──── 202 ─────┼───                      delivery worker ├──────►  POST /hook
                  │                                  │      │   signed payload
                  │                      retry w/ backoff   │   ◄──── 2xx = done
                  │                      on non-2xx         │
                  └─────────────────────────────────────────┘
```

Your application calls `POST /events` and gets `202 Accepted` immediately —
accepted, not yet delivered — then moves on. In the background, workers pick the
event up, find every subscriber registered for that event type, and POST the
payload with an HMAC signature header. A 2xx means done. Anything else is
retried with exponential backoff until it succeeds or is marked dead, and every
attempt is logged for later inspection and manual replay.

## Quick start

Requires Go 1.27 or newer.

```bash
git clone git@github.com:ngaunpot/hookline.git
cd hookline
go run ./cmd/hookline
```

The server listens on `:8080`.

```bash
# health check
curl http://localhost:8080/healthz

# publish an event
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{"type":"user.created","payload":{"id":42}}'
```

```json
{
  "id": "9b1f0c7e-3d4a-4f1b-8c2e-7a5d6e8f0a1b",
  "type": "user.created",
  "payload": { "id": 42 },
  "created_at": "2026-10-02T06:15:00Z"
}
```

## API

| Method | Path       | Description                          |
| ------ | ---------- | ------------------------------------ |
| GET    | `/healthz` | Liveness check. Returns `ok`.        |
| POST   | `/events`  | Publish an event. Returns `202`.     |

### `POST /events`

| Field     | Type   | Required | Description                                      |
| --------- | ------ | -------- | ------------------------------------------------ |
| `type`    | string | yes      | Event type, e.g. `user.created`. Routes delivery. |
| `payload` | object | yes      | Arbitrary JSON, forwarded to subscribers as-is.   |

| Status | Meaning                                              |
| ------ | ---------------------------------------------------- |
| 202    | Accepted and queued.                                 |
| 400    | Malformed JSON, unknown field, or body over 1 MiB.   |
| 422    | Valid JSON, but `type` or `payload` is missing.      |

Errors are returned as `{"error": "type is required"}`.

## Project layout

```
cmd/hookline/      process entry point — wiring and startup only
internal/api/      HTTP layer: routing, handlers, request validation
```

`internal/` is used rather than `pkg/` because nothing here is intended as a
public library; the compiler enforces that.

## Design decisions

**The handlers hang off a `Server` struct.** `NewServer()` builds the router and
registers routes in one place. This is the dependency injection point: when
Postgres and the worker pool arrive, they become fields on `Server` rather than
package-level globals, and tests can substitute fakes. `Server` implements
`http.Handler`, so it drops straight into `httptest` without binding a port.

**Payloads are `json.RawMessage`, not `map[string]any`.** A webhook gateway has
no business interpreting the payload — it forwards it. Keeping the raw bytes
means subscribers receive exactly what the publisher sent, which becomes a
correctness requirement in sprint 4: an HMAC signature is computed over bytes,
and unmarshalling then re-marshalling can reorder keys and invalidate it.

**Request and domain types are separate.** Clients decode into
`createEventRequest`, which carries only `type` and `payload`. If they decoded
straight into `Event`, a caller could set their own `id` or backdate
`created_at`. The separation is the standard defence against mass assignment.

**Request bodies are capped and strict.** `http.MaxBytesReader` caps bodies at
1 MiB so one request can't exhaust memory, and `DisallowUnknownFields` turns a
client's typo into a 400 instead of a silently ignored field.

**Tests are table-driven and use `httptest`.** Adding a case means adding a
struct to a slice, not writing another function. No ports are bound, so the
suite runs in milliseconds.

## Testing

```bash
go vet ./...
go test ./... -race -cover
```

## Roadmap

Built in two-week increments, each tagged as a release.

- [x] **v0.1** HTTP skeleton — routing, validation, error envelope, tests, CI
- [ ] **v0.2** Persistence — Postgres via pgx, migrations, subscription CRUD, docker-compose
- [ ] **v0.3** Delivery worker — goroutine pool, `context` cancellation, exponential backoff with jitter, dead-letter state
- [ ] **v0.4** Security — HMAC-SHA256 signed payloads, API keys, idempotency keys
- [ ] **v0.5** Observability — `slog` structured logging, Prometheus metrics, graceful shutdown
- [ ] **v0.6** Stats and replay — per-subscriber success rate and p95 latency, manual replay of dead events
- [ ] **v0.7** Hardening — integration tests with testcontainers-go, load test with k6, measured throughput
- [ ] **v1.0** Deployment — public demo instance, design write-up

## Licence

MIT. See [LICENSE](LICENSE).