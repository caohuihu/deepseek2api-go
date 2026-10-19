# 更新日志

本项目的所有重要变更将记录在此文件中。

## [未发布]

### 新增功能

- **账号级请求限流**（`handlers/limiter.go`）
  - 基于 `deviceID` 的每账号令牌桶 QPM（每分钟请求数）限流器。
  - 每账号独立的 in-flight 并发信号量，防止突发流量打爆上游。
  - 滑动分钟窗口：时间窗口到期后令牌自动 replenish。
  - 若运行时调高 `MAX_CONCURRENT_PER_ACCOUNT`，信号量容量会动态扩容。

- **429 指数退避重试 + 账号轮换**（`handlers/handler.go`）
  - 遇到 `429 Too Many Requests` 时，按指数退避等待（`baseDelay × 2^(attempt-1)`，最长 5 分钟）。
  - 每次重试切换到轮询池中的下一个账号，并创建全新的 DeepSeek 会话。
  - 非 429 错误直接返回，不进行重试。

- **新增环境变量配置**（`config/config.go`）
  | 变量名 | 默认值 | 说明 |
  |---|---|---|
  | `ACCOUNT_QPM` | `30` | 每账号每秒滑动窗口内最大请求数 |
  | `MAX_CONCURRENT_PER_ACCOUNT` | `5` | 每账号同时进行的最大并发请求数 |
  | `RETRY_MAX_ATTEMPTS` | `3` | 遇到 429 最多重试次数 |
  | `RETRY_BASE_DELAY` | `2s` | 指数退避基础延迟（实际等待：2s、4s、8s…） |

- **启动日志增强**（`main.go`）
  - 启动时打印 `account qpm`、`max concurrent/account`、`retry max attempts`、`retry base delay` 四项参数。

### 变更

- `config.Config` 结构体新增上述四个字段。
- `handlers.Handler` 新增 `*AccountLimiter` 字段，由 `cfg.AccountQPM` 初始化。
- `ChatCompletion` 在处理请求前获取限流令牌和并发槽位；请求结束时通过 `defer` 释放。
- `doCompletionWithRetry` 统一封装 session 创建逻辑，使重试到新账号时能干净地新建会话。

### 优化动机

上述改动旨在降低被 DeepSeek 网页端风控系统检测到的风险：
- **类人请求节奏**：令牌桶限流模拟正常用户的使用频率（约 30 请求/分钟/账号）。
- **并发封顶**：防止所有账号同时发起请求造成瞬时峰值。
- **优雅降级**：遇到限流时自动轮换账号并退避，而非直接报错放弃。
