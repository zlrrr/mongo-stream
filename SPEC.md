# mongo-stream — Software Design Document (SDD)

## 1. Overview

`mongo-stream` is a Go CLI tool that continuously streams randomly-generated documents into a MongoDB deployment (mongos or mongod). It is designed for load testing, benchmarking, and validating MongoDB configurations.

---

## 2. Goals

| Goal | Description |
|------|-------------|
| G-1 | Connect to any MongoDB endpoint with optional auth |
| G-2 | Write a specified number of documents, or run indefinitely |
| G-3 | Distribute writes across N databases × M collections |
| G-4 | Support distribution patterns: **uniform** (MVP), gaussian, long-tail (future) |
| G-5 | Surface real-time write throughput and latency metrics |
| G-6 | Provide actionable error logs for troubleshooting |

---

## 3. Non-Goals (MVP)

- Gaussian / long-tail distribution (deferred to v2)
- Read workloads
- Schema customisation via external files
- Replica set / sharded cluster awareness

---

## 4. Architecture

```
┌──────────────┐     ┌─────────────────┐     ┌──────────────────┐
│   CLI (cobra) │────▶│  Config / Flags │────▶│   Distributor    │
└──────────────┘     └─────────────────┘     │ (uniform / …)    │
                                              └────────┬─────────┘
                                                       │ target list
                                              ┌────────▼─────────┐
                                              │     Writer Pool   │
                                              │  (goroutine/db)   │
                                              └────────┬─────────┘
                                                       │ insertMany
                                              ┌────────▼─────────┐
                                              │  MongoDB Driver   │
                                              └──────────────────┘
                                                       │
                                              ┌────────▼─────────┐
                                              │  Progress Logger  │
                                              │  (ticker, zap)    │
                                              └──────────────────┘
```

### Package layout

```
mongo-stream/
├── main.go
├── cmd/
│   └── root.go           # cobra command, flag binding
├── internal/
│   ├── config/
│   │   └── config.go     # Config struct + validation
│   ├── connector/
│   │   └── mongo.go      # client factory + ping
│   ├── generator/
│   │   └── document.go   # random BSON document builder
│   ├── distributor/
│   │   ├── distributor.go # interface + registry
│   │   └── uniform.go    # uniform distribution
│   ├── writer/
│   │   └── writer.go     # orchestration, worker pool
│   └── logger/
│       └── logger.go     # zap-based structured logger
├── go.mod
└── SPEC.md
```

---

## 5. Data Model

Each generated document contains:

```json
{
  "_id":        "<ObjectID>",
  "seq":        12345,
  "ts":         "2024-01-01T00:00:00Z",
  "payload":    "<random 64-byte hex string>",
  "tags":       ["tag1", "tag2"],
  "value":      3.14
}
```

---

## 6. Distribution Interface

```go
type Target struct {
    DB         string
    Collection string
}

type Distributor interface {
    // Targets returns the full ordered list of write targets.
    Targets() []Target
    // Next returns the next Target for the given document index.
    Next(index int64) Target
}
```

### 6.1 Uniform (MVP)

Targets are enumerated as `db_0..db_{N-1}` × `col_0..col_{M-1}`.
`Next(i)` maps `i % (N×M)` → target slot (round-robin).

### 6.2 Gaussian (future)

Weight targets according to a discrete Gaussian over the flattened target index.

### 6.3 Long-tail (future)

Weight targets according to a power-law (Zipf) distribution.

---

## 7. CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--uri` | `mongodb://localhost:27017` | MongoDB connection URI |
| `--username` | `""` | Auth username |
| `--password` | `""` | Auth password |
| `--auth-source` | `admin` | Auth source database |
| `--dbs` | `10` | Number of databases |
| `--collections` | `20` | Collections per database |
| `--total` | `0` | Total docs to insert (0 = unlimited) |
| `--batch` | `100` | Insert batch size |
| `--concurrency` | `4` | Parallel writer goroutines |
| `--log-interval` | `5s` | Progress log interval |
| `--distribution` | `uniform` | Distribution mode |

---

## 8. Progress Logging

Every `--log-interval` seconds the logger prints:

```
[2024-01-01T00:00:05Z] docs=5000 rate=1000/s p99_latency=12ms errors=0
```

---

## 9. Error Handling

- Connection errors: fatal, print URI (without password) and exit 1
- Insert errors: logged with db/collection context; writer continues
- Context cancellation (Ctrl-C): graceful drain then exit 0

---

## 10. Implementation Milestones (Checkpoints)

| # | Milestone | Done when |
|---|-----------|-----------|
| M1 | Spec + project scaffold | `go build ./...` succeeds, branch pushed |
| M2 | Config + CLI | `--help` shows all flags; config validates correctly; unit tests pass |
| M3 | Connector | `connector.New()` returns error on bad URI; ping succeeds on live server; unit tests pass |
| M4 | Generator | `generator.New()` produces valid BSON docs; deterministic with fixed seed; unit tests pass |
| M5 | Uniform Distributor | `Next(i)` returns correct target; table-driven tests pass |
| M6 | Writer + Logger | End-to-end dry-run (mock client) writes correct doc counts; progress metrics logged; all tests pass |
| M7 | Integration smoke test | `go test ./... -tags integration` (skipped without MONGO_URI env) passes |
| M8 | MVP release | All unit tests green; binary builds for linux/amd64; committed + pushed |

---

## 11. Testing Strategy

- **Unit tests**: every package has `_test.go`; mocks via interfaces
- **Integration tests**: tagged `//go:build integration`; skipped in CI without `MONGO_URI`
- **Coverage target**: ≥ 70% line coverage for non-main packages
- **CI**: `go vet ./...` + `go test ./...` must pass before each commit

---

## 12. Technology Choices

| Concern | Library | Reason |
|---------|---------|--------|
| CLI | `github.com/spf13/cobra` | Industry standard |
| MongoDB | `go.mongodb.org/mongo-driver/v2` | Official driver |
| Logging | `go.uber.org/zap` | Structured, performant |
| Testing | stdlib `testing` | No extra dependency |
