# mongo-stream

用 Go 编写的高性能命令行工具，可向 MongoDB 部署持续写入随机生成的文档，适用于负载测试、基准测试和 Schema 验证。

[English Documentation](README.md)

---

## 功能特性

- 支持连接 **mongos**（分片集群）或 **mongod**，可选用户名/密码认证
- 支持写入**指定数量**的文档，或**持续无限**写入（`--total 0`）
- 可配置写入的**数据库 × 集合**数量（默认：10 个数据库 × 每个 20 个集合）
- **均匀分布**（MVP）：轮询写入所有目标，更多分布模式即将推出
- **实时进度日志**：按可配置间隔输出写入速率（条/秒）和错误数
- **结构化错误日志**：包含完整上下文信息（数据库、集合、序号、错误信息）
- **优雅退出**：按 `Ctrl-C` 后等待进行中的批次完成再退出

---

## 安装

### 下载预编译二进制

从 [Releases](../../releases) 页面下载对应平台的最新版本：

| 平台 | 文件 |
|------|------|
| Linux (amd64) | `mongo-stream_linux_amd64.tar.gz` |
| Windows (amd64) | `mongo-stream_windows_amd64.zip` |

### 从源码编译

**前置条件**：Go 1.21+

```bash
git clone https://github.com/zlrrr/mongo-stream.git
cd mongo-stream
GONOSUMDB='*' GOFLAGS='-mod=mod' go build -o mongo-stream .
```

---

## 快速开始

```bash
# 向本地 MongoDB 写入 10,000 条文档（无需认证）
./mongo-stream --total 10000

# 带认证的无限写入，自定义并发数
./mongo-stream \
  --uri mongodb://host:27017 \
  --username admin --password secret \
  --total 0 \
  --concurrency 8 \
  --log-interval 3s
```

---

## 分布规律预览

写入开始前，`mongo-stream` 会打印分布表格，让你清楚了解数据将写入哪些目标：

```
INFO  distribution plan   mode=uniform databases=10 collections_per_db=20 total_targets=200
INFO  --- distribution table ---
INFO  db assignment       database=db_0  collections=20  share_pct=10.00
INFO  db assignment       database=db_1  collections=20  share_pct=10.00
...
INFO  --------------------------
```

---

## 进度日志

每隔 `--log-interval` 秒（默认 `5s`）输出一次进度：

```
INFO  progress  docs_total=5000  rate_per_sec=987.23  errors=0
INFO  progress  docs_total=10000 rate_per_sec=1024.10 errors=0
INFO  run complete  docs_inserted=10000  errors=0
```

---

## 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--uri` | `mongodb://localhost:27017` | MongoDB 连接 URI |
| `--username` | _（空）_ | 认证用户名 |
| `--password` | _（空）_ | 认证密码 |
| `--auth-source` | `admin` | 认证数据库 |
| `--dbs` | `10` | 写入的数据库数量 |
| `--collections` | `20` | 每个数据库的集合数量 |
| `--total` | `0` | 写入的文档总数（`0` = 无限） |
| `--batch` | `100` | 每次 `insertMany` 的批量大小 |
| `--concurrency` | `4` | 并行写入协程数 |
| `--log-interval` | `5s` | 进度日志输出间隔（如 `2s`、`1m`） |
| `--distribution` | `uniform` | 分布模式：`uniform` \| `gaussian` \| `longtail` |

---

## 文档结构

每条生成的文档包含以下字段：

```json
{
  "_id":     "<ObjectID>",
  "seq":     12345,
  "ts":      "2024-01-01T00:00:00Z",
  "payload": "a3f9...（64 位十六进制字符串）",
  "tags":    ["alpha", "gamma"],
  "value":   712.34
}
```

---

## 分布模式

| 模式 | 状态 | 说明 |
|------|------|------|
| `uniform` | ✅ 已可用 | 轮询——每个目标均等写入 |
| `gaussian` | 🔜 规划中 | 正态分布（高斯分布）加权写入 |
| `longtail` | 🔜 规划中 | 幂律（Zipf）加权写入，模拟长尾分布 |

---

## 架构

```
CLI (cobra)
  └─▶ 配置/参数解析
        └─▶ 分布器（uniform / …）
              └─▶ 写入协程池（N 个 goroutine）
                    └─▶ MongoDB Driver（insertMany）
                              │
                    进度日志（zap，定时器）
```

---

## 开发指南

```bash
# 运行所有单元测试
GONOSUMDB='*' GOFLAGS='-mod=mod' go test ./...

# 代码检查
go vet ./...

# 跨平台编译
GOOS=linux   GOARCH=amd64 go build -o dist/mongo-stream_linux_amd64  .
GOOS=windows GOARCH=amd64 go build -o dist/mongo-stream_windows_amd64.exe .
```

---

## 开源协议

[MIT](LICENSE)
