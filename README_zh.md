# mongo-stream

用 Go 编写的高性能命令行工具，可向 MongoDB 部署持续写入随机生成的文档，适用于负载测试、基准测试和 Schema 验证。

[English Documentation](README.md)

---

## 功能特性

- 支持连接 **mongos**（分片集群）或 **mongod**，可选用户名/密码认证
- 支持写入**指定数量**的文档，或**持续无限**写入（`--total 0`）
- 可配置写入的**数据库 × 集合**数量（默认：10 个数据库 × 每个 20 个集合）
- **三种分布模式**：`uniform`（轮询均匀）、`gaussian`（正态分布热点居中）、`longtail`（Zipf 幂律热点集中）
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

**前置条件**：Go 1.24+

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
| `--webui` | `false` | 启动 Web 监控界面 |
| `--webui-port` | `8080` | Web 界面的 HTTP 端口（需同时指定 `--webui`） |

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

三种模式均已可用，通过 `--distribution <mode>` 选择。

---

### `uniform` — 轮询均匀分布

每个目标（数据库 × 集合）接收完全相同数量的写入，目标选择公式为：

```
目标下标 = 文档序号 mod 目标总数
```

**适用场景**：基准测试基线、分片均衡验证、需要完全均等数据分布的任何场景。

**参考文献**：标准取模轮询调度，广泛见于操作系统教材（如 Silberschatz 等，*操作系统概念*，§5.3）。

---

### `gaussian` — 正态分布（高斯分布）

所有目标按顺序排列，**中间位置**的目标接收最多写入，越靠近两端的目标写入量以指数速度递减，权重公式为标准正态概率密度函数：

```
weight(i) = exp( -0.5 × ((i − μ) / σ)² )

  μ = (N − 1) / 2      （目标列表中心）
  σ = N / 6            （±3σ 覆盖完整列表，约 99.7% 的概率质量落在范围内）
```

权重数组归一化为累积分布函数（CDF）。每条文档的序号经 **splitmix64** 哈希后得到确定性值 `p ∈ [0, 1)`，再通过 CDF 的二分查找映射到目标。

**具体示例**（N = 200 个目标，默认 10 DB × 20 集合）：

| 距中心的排名偏移 | 相对写入权重 |
|----------------|------------|
| 中心（偏移 0） | 1.000（基准） |
| ±33 个目标 | ~0.135 |
| ±67 个目标（边缘） | ~0.011 |

中心集合的写入量约为边缘集合的 **90 倍**。

**适用场景**：模拟具有"温热"中间区间的工作负载（如时序数据中近期文档访问频繁）、测试非均匀访问下的索引性能。

**参考文献**：
- Abramowitz, M. & Stegun, I. A. (1964). *Handbook of Mathematical Functions*（数学函数手册），§26.2 — 正态分布 CDF 表。
- Press, W. H. 等 (2007). *Numerical Recipes*（数值计算方法），第 3 版，§7.3 — 非均匀随机偏差的变换方法。
- Lehmer, D. H. (1951). Mathematical methods in large-scale computing units. *Proc. 2nd Symp. on Large-Scale Digital Calculating Machinery*，pp. 141–146 — splitmix64 的线性同余发生器前身。
- Steele, G. & Vigna, S. (2021). Computationally easy, spectrally good multipliers for congruential pseudorandom number generators. *Software: Practice and Experience*, 52(2). — `indexToProb` 所用 splitmix64 常量的来源。

---

### `longtail` — 幂律分布（Zipf 分布）

少量排名靠前的**"热点"目标**接收远多于其他目标的写入，符合 Zipf 定律：

```
weight(i) = 1 / (i + 1)^s        （i = 0, 1, 2, …, N−1）

  s = 偏斜指数（默认 1.07）
```

与高斯模式相同，权重归一化为 CDF，每条文档序号经哈希后选取目标。偏斜指数 `s` 控制热点强度：

| 偏斜 `s` | 效果 |
|---------|------|
| `0.5` | 轻度偏斜，趋向均匀（s → 0 时完全均匀） |
| `1.07` *（默认）* | 模拟电商 / 缓存的真实工作负载 |
| `2.0` | 强热点，前 3 个集合占主导 |

**具体示例**（N = 200 个目标，默认偏斜 1.07）：

| 排名 | 相对权重 | 近似写入占比 |
|-----|---------|------------|
| 0（最热） | 1.000 | ~1.2% |
| 9 | ~0.085 | ~0.10% |
| 前 10 合计 | — | ~40% 总写入 |
| 后 100 合计 | — | ~15% 总写入 |

**适用场景**：模拟真实应用工作负载（Web 流量、缓存缺失、社交媒体热帖）、测试 MongoDB 热分片检测、复现集合级别的"二八定律"。

**参考文献**：
- Zipf, G. K. (1949). *Human Behavior and the Principle of Least Effort*（人类行为与最小努力原则）. Addison-Wesley. — 幂律排名-频率关系的原始实证观察。
- Adamic, L. A. & Huberman, B. A. (2002). Zipf's law and the internet. *Glottometrics*, 3, 143–150. — 互联网流量对 Zipf 定律的验证。
- Gray, J. 等 (1994). Quickly generating billion-record synthetic databases. *Proc. ACM SIGMOD*，pp. 243–252. — 数据库基准测试（TPC-C）中的 Zipf 数据生成方法。
- Breslau, L. 等 (1999). Web caching and Zipf-like distributions: evidence and implications. *Proc. IEEE INFOCOM*, vol. 1, pp. 126–134. — 生产缓存中实测偏斜指数（s ≈ 0.7–1.2）。

---

### 实现说明（三种模式通用）

| 属性 | 说明 |
|------|------|
| **Goroutine 安全** | CDF 表在构建后只读，`Next()` 无任何可变共享状态 |
| **确定性** | `indexToProb` 使用 splitmix64，相同序号始终选择相同目标 |
| **目标选择** | 预构建 CDF 上的二分查找，每次调用 O(log N) |
| **哈希函数** | splitmix64（Steele & Vigna 2021），通过 BigCrush 统计测试 |

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
