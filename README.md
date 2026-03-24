# mongo-stream

A high-performance CLI tool written in Go that streams randomly-generated documents into a MongoDB deployment for load testing, benchmarking, and schema validation.

[中文文档](README_zh.md)

---

## Features

- Connect to **mongos** (sharded cluster) or **mongod** with optional username/password authentication
- Write a **fixed number** of documents or run **indefinitely** (`--total 0`)
- Distribute writes across configurable **databases × collections** (default: 10 DBs × 20 collections)
- **Uniform distribution** (MVP): round-robin across all targets — more patterns coming soon
- **Real-time progress logging**: write rate (docs/s) and error count at a configurable interval
- **Structured error logging** with full context (database, collection, sequence, error message)
- **Graceful shutdown** on `Ctrl-C` — drains in-flight batches before exiting

---

## Installation

### Pre-built binaries

Download the latest release for your platform from the [Releases](../../releases) page:

| Platform | File |
|----------|------|
| Linux (amd64) | `mongo-stream_linux_amd64.tar.gz` |
| Windows (amd64) | `mongo-stream_windows_amd64.zip` |

### Build from source

**Requirements**: Go 1.21+

```bash
git clone https://github.com/zlrrr/mongo-stream.git
cd mongo-stream
GONOSUMDB='*' GOFLAGS='-mod=mod' go build -o mongo-stream .
```

---

## Quick Start

```bash
# Write 10,000 documents to a local MongoDB (no auth)
./mongo-stream --total 10000

# Unlimited write with auth, custom concurrency
./mongo-stream \
  --uri mongodb://host:27017 \
  --username admin --password secret \
  --total 0 \
  --concurrency 8 \
  --log-interval 3s
```

---

## Distribution Preview

Before writing begins, `mongo-stream` prints a distribution table so you know exactly where data will land:

```
INFO  distribution plan   mode=uniform databases=10 collections_per_db=20 total_targets=200
INFO  --- distribution table ---
INFO  db assignment       database=db_0  collections=20  share_pct=10.00
INFO  db assignment       database=db_1  collections=20  share_pct=10.00
...
INFO  --------------------------
```

---

## Progress Logging

Every `--log-interval` seconds (default `5s`):

```
INFO  progress  docs_total=5000  rate_per_sec=987.23  errors=0
INFO  progress  docs_total=10000 rate_per_sec=1024.10 errors=0
INFO  run complete  docs_inserted=10000  errors=0
```

---

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--uri` | `mongodb://localhost:27017` | MongoDB connection URI |
| `--username` | _(empty)_ | Auth username |
| `--password` | _(empty)_ | Auth password |
| `--auth-source` | `admin` | Auth source database |
| `--dbs` | `10` | Number of databases to write to |
| `--collections` | `20` | Collections per database |
| `--total` | `0` | Total documents to insert (`0` = unlimited) |
| `--batch` | `100` | Batch size per `insertMany` call |
| `--concurrency` | `4` | Number of parallel writer goroutines |
| `--log-interval` | `5s` | How often to print progress (e.g. `2s`, `1m`) |
| `--distribution` | `uniform` | Distribution mode: `uniform` \| `gaussian` \| `longtail` |

---

## Document Schema

Each generated document has the following fields:

```json
{
  "_id":     "<ObjectID>",
  "seq":     12345,
  "ts":      "2024-01-01T00:00:00Z",
  "payload": "a3f9...  (64 hex chars)",
  "tags":    ["alpha", "gamma"],
  "value":   712.34
}
```

---

## Distribution Modes

| Mode | Status | Description |
|------|--------|-------------|
| `uniform` | ✅ Available | Round-robin — equal writes to every target |
| `gaussian` | 🔜 Planned | Bell-curve weighted write distribution |
| `longtail` | 🔜 Planned | Power-law (Zipf) weighted distribution |

---

## Architecture

```
CLI (cobra)
  └─▶ Config / Flags
        └─▶ Distributor (uniform / …)
              └─▶ Writer Pool (N goroutines)
                    └─▶ MongoDB Driver (insertMany)
                              │
                    Progress Logger (zap, ticker)
```

---

## Development

```bash
# Run all unit tests
GONOSUMDB='*' GOFLAGS='-mod=mod' go test ./...

# Lint
go vet ./...

# Build for all platforms
GOOS=linux  GOARCH=amd64 go build -o dist/mongo-stream_linux_amd64  .
GOOS=windows GOARCH=amd64 go build -o dist/mongo-stream_windows_amd64.exe .
```

---

## License

[MIT](LICENSE)
