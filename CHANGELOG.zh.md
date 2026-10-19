# 更新日志

本项目的所有重要变更将记录在此文件中。

## [未发布]

### 新增功能（行为随机化反风控）

- **UA / Version / Locale 随机池**（`deepseek/client.go`）
  - `NewClient` 启动时从池中随机选取：
    - **版本**：`2.5.0` / `2.4.5` / `2.4.0` / `2.3.5` / `2.3.0`
    - **区域**：`zh_CN` / `en_US` / `zh_TW` / `ja_JP` / `ko_KR`
    - **User-Agent**：7 种常见组合，覆盖 Windows / Mac / Linux + Chrome / Firefox / Edge
  - 同一账号每次重启服务后自动选用不同指纹，降低设备指纹相似度。

- **请求前随机抖动**（`deepseek/client.go` + `handlers/handler.go`）
  - 在发送 `/chat/completion` 请求前插入 `REQUEST_JITTER_MIN ~ REQUEST_JITTER_MAX` 的随机等待（默认 200ms ~ 800ms）。
  - 模拟人类思考间隔，避免固定节奏的机械请求模式。

- **POW 求解模拟延迟**（`deepseek/pow.go`）
  - `Solve(minDelay, maxDelay)` 在真实哈希运算前增加 `POW_JITTER_MIN ~ POW_JITTER_MAX` 随机等待（默认 30ms ~ 150ms）。
  - 浏览器解 POW 需约 30–150ms，Go 实现毫秒级完成；此参数让响应时间更贴近真实浏览器行为。

- **新建会话随机延迟**（`handlers/handler.go`）
  - `CreateSession` 调用前增加 `SESSION_CREATE_JITTER` 随机等待（默认 200ms）。
  - 防止批量创建会话时呈现机器批量操作特征。

### 新增环境变量配置（续）

| 变量名 | 默认值 | 说明 |
|---|---|---|
| `ACCOUNT_QPM` | `30` | 每账号每秒滑动窗口内最大请求数 |
| `MAX_CONCURRENT_PER_ACCOUNT` | `5` | 每账号同时进行的最大并发请求数 |
| `RETRY_MAX_ATTEMPTS` | `3` | 遇到 429 最多重试次数 |
| `RETRY_BASE_DELAY` | `2s` | 指数退避基础延迟（实际等待：2s、4s、8s…） |
| `REQUEST_JITTER_MIN` | `200ms` | 每次请求前的最小随机等待（毫秒） |
| `REQUEST_JITTER_MAX` | `800ms` | 每次请求前的最大随机等待（毫秒） |
| `POW_JITTER_MIN` | `30ms` | POW 求解器最小模拟延迟（毫秒） |
| `POW_JITTER_MAX` | `150ms` | POW 求解器最大模拟延迟（毫秒） |
| `SESSION_CREATE_JITTER` | `200ms` | 新建会话前的随机等待（毫秒） |
| `DEVICE_ID_ROTATE_HOURS` | `0` | Device ID 定期轮换间隔（小时，0=不轮换） |

### 启动日志增强

启动时打印所有防风控参数，方便运维确认配置是否生效：

```
  account qpm              : 30
  max concurrent/account   : 5
  retry max attempts       : 3 (base 2s)
  request jitter           : 200ms ~ 800ms
  pow jitter               : 30ms ~ 150ms
  session create jitter    : 200ms
```

### 代码变更

- `deepseek/client.go`：新增 `clientVersions` / `clientLocales` / `userAgents` 随机池；`NewClient` 接受 `powJitterMin/Max` 参数。
- `deepseek/pow.go`：`Solve` 方法增加 `minDelay/maxDelay` 参数，求解前执行随机延迟。
- `handlers/handler.go`：`doCompletionWithRetry` 在 `CreateSession` 和 `Completion` 调用前插入随机抖动；`NewPool` 透传 jitter 参数。
- `config/config.go`：新增 6 个反风控字段及 `parseDurationEnv` 解析函数。
- `main.go`：启动日志增加新参数输出。

### 优化动机

上述改动与上一轮限流/重试功能配合，形成完整的反风控层：

1. **指纹多样性**：UA/Version/Locale 随机化避免所有账号共享同一设备指纹。
2. **节奏拟人**：请求抖动 + POW 延迟让请求间隔和计算耗时接近真实浏览器行为。
3. **会话分散**：新建会话随机延迟防止批量创建会话的模式识别。
4. **账号隔离**：限流器 + 并发信号量 + 429 重试轮换，确保单账号异常不影响其他账号。
