# mongo-stream

A high-performance CLI tool written in Go that streams randomly-generated documents into a MongoDB deployment for load testing, benchmarking, and schema validation.

[中文文档](README_zh.md)

---

## Features

- Connect to **mongos** (sharded cluster) or **mongod** with optional username/password authentication
- Write a **fixed number** of documents or run **indefinitely** (`--total 0`)
- Distribute writes across configurable **databases × collections** (default: 10 DBs × 20 collections)
- **Three distribution modes**: `uniform` (round-robin), `gaussian` (bell-curve hot centre), `longtail` (Zipf power-law hot spot)
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

**Requirements**: Go 1.24+

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
| `--webui` | `false` | Start a web monitoring UI |
| `--webui-port` | `8080` | HTTP port for the web UI (requires `--webui`) |
| `--webui-bind` | `""` (all interfaces) | Bind address: `""` / `0.0.0.0` = LAN+local, `127.0.0.1` = localhost only |

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

All three modes are available today. Select one with `--distribution <mode>`.

---

### `uniform` — Round-Robin

Every target (database × collection pair) receives exactly the same number of
writes. The target is selected by:

```
target_index = document_index mod total_targets
```

**When to use**: baseline benchmarks, shard-balance verification, any scenario
that requires perfectly equal data spread.

**Reference**: standard modular round-robin scheduling, widely described in
operating-systems literature (e.g. Silberschatz et al., *Operating System
Concepts*, §5.3).

---

### `gaussian` — Bell-Curve (Normal Distribution)

Targets are arranged in a ranked list. The **middle** target receives the most
writes; targets closer to either end receive exponentially fewer writes,
following the probability density function of the normal distribution:

```
weight(i) = exp( -0.5 × ((i − μ) / σ)² )

  μ = (N − 1) / 2        (centre of the target list)
  σ = N / 6              (±3σ spans the full list, capturing ~99.7 % of mass)
```

The weight array is normalised into a cumulative distribution function (CDF).
For each document, the index is hashed with **splitmix64** to produce a
deterministic value `p ∈ [0, 1)`, which is mapped to a target via binary
search on the CDF.

**Concrete example** (N = 200 targets, default 10 DBs × 20 collections):

| Rank from centre | Relative write share |
|-----------------|---------------------|
| Centre (rank 0) | 1.000 (baseline) |
| ±33 targets | ~0.135 |
| ±67 targets (edge) | ~0.011 |

The centre collection attracts roughly **90× more writes** than the outermost
collections.

**When to use**: simulating workloads with a "warm" mid-range of IDs (e.g.
time-series data where recent documents are hot), or testing index performance
under non-uniform access.

**References**:
- Abramowitz, M. & Stegun, I. A. (1964). *Handbook of Mathematical Functions*, §26.2 — Normal distribution CDF tables.
- Press, W. H. et al. (2007). *Numerical Recipes*, 3rd ed., §7.3 — Transformation methods for non-uniform random deviates.
- Lehmer, D. H. (1951). Mathematical methods in large-scale computing units. *Proc. 2nd Symp. on Large-Scale Digital Calculating Machinery*, pp. 141–146 — splitmix64 lineage (linear congruential generators).
- Steele, G. & Vigna, S. (2021). *Computationally easy, spectrally good multipliers for congruential pseudorandom number generators*. Software: Practice and Experience, 52(2). — splitmix64 constants used in `indexToProb`.

---

### `longtail` — Power-Law / Zipf Distribution

A small number of **"hot" targets** (low rank) receive disproportionately more
writes than the rest, following Zipf's law:

```
weight(i) = 1 / (i + 1)^s        (i = 0, 1, 2, …, N−1)

  s = skew exponent (default 1.07)
```

Like the Gaussian mode, weights are normalised to a CDF and each document index
is hashed to select a target. The skew exponent `s` controls how aggressive
the hot-spot is:

| Skew `s` | Effect |
|----------|--------|
| `0.5` | Mild skew — approaches uniform as s → 0 |
| `1.07` *(default)* | Realistic e-commerce / cache workload |
| `2.0` | Aggressive hot-spot; top-3 collections dominate |

**Concrete example** (N = 200 targets, default skew 1.07):

| Rank | Relative weight | Approximate share |
|------|----------------|-------------------|
| 0 (hottest) | 1.000 | ~1.2 % |
| 9 | ~0.085 | ~0.10 % |
| Top-10 combined | — | ~40 % of all writes |
| Bottom-100 combined | — | ~15 % of all writes |

**When to use**: simulating realistic application workloads (web traffic,
cache-miss patterns, social-media hot posts), testing hot-shard detection,
or reproducing the "80/20 rule" in MongoDB collections.

**References**:
- Zipf, G. K. (1949). *Human Behavior and the Principle of Least Effort*. Addison-Wesley. — original empirical observation of power-law rank–frequency relationships.
- Adamic, L. A. & Huberman, B. A. (2002). Zipf's law and the internet. *Glottometrics*, 3, 143–150. — web-traffic validation of Zipf's law.
- Gray, J. et al. (1994). Quickly generating billion-record synthetic databases. *Proc. ACM SIGMOD*, pp. 243–252. — Zipf data generation for database benchmarks (TPC-C lineage).
- Breslau, L. et al. (1999). Web caching and Zipf-like distributions: evidence and implications. *Proc. IEEE INFOCOM*, vol. 1, pp. 126–134. — empirical skew exponents in production caches (s ≈ 0.7–1.2).

---

### Implementation Notes (all modes)

| Property | Detail |
|----------|--------|
| **Goroutine safety** | CDF tables are read-only after construction; `Next()` has no mutable shared state |
| **Determinism** | `indexToProb` uses splitmix64 — same index always selects the same target |
| **Target selection** | Binary search on pre-built CDF → O(log N) per call |
| **Hash function** | splitmix64 (Steele & Vigna 2021); passes BigCrush statistical tests |

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
