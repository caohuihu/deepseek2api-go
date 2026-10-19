# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- **Per-account rate limiting** (`handlers/limiter.go`)
  - Token-bucket QPM (requests per minute) limiter per DeepSeek account, keyed by `deviceID`.
  - In-flight concurrency semaphore per account to prevent burst flooding.
  - Sliding-minute window: tokens auto-refill when the minute rolls over.
  - Semaphore capacity grows dynamically if `MAX_CONCURRENT_PER_ACCOUNT` is raised at runtime.

- **429 exponential-backoff retry with account rotation** (`handlers/handler.go`)
  - On a `429 Too Many Requests` response, the handler waits with exponential backoff (`baseDelay * 2^(attempt-1)`, capped at 5 min).
  - Each retry switches to the next account in the round-robin pool and starts a fresh DeepSeek session.
  - Non-429 errors are surfaced immediately without retry.

- **New environment variables** (`config/config.go`)
  | Variable | Default | Description |
  |---|---|---|
  | `ACCOUNT_QPM` | `30` | Max requests per account per sliding minute |
  | `MAX_CONCURRENT_PER_ACCOUNT` | `5` | Max in-flight completions per account simultaneously |
  | `RETRY_MAX_ATTEMPTS` | `3` | Maximum 429 retry attempts before giving up |
  | `RETRY_BASE_DELAY` | `2s` | Base delay for exponential backoff (actual waits: 2s, 4s, 8s…) |

- **Startup log lines** (`main.go`)
  - Prints `account qpm`, `max concurrent/account`, `retry max attempts`, and `retry base delay` on launch.

### Changed

- `config.Config` struct extended with the four new fields above.
- `handlers.Handler` now holds an `*AccountLimiter` initialised from `cfg.AccountQPM`.
- `ChatCompletion` acquires a rate-limit token and a concurrency slot before invoking the upstream completion; both are released via `defer`.
- `doCompletionWithRetry` centralises session creation so that a retry on a new account can start a clean session.

### Why

These changes reduce the risk of DeepSeek web anti-fraud detection:
- Human-like request pacing (token bucket ≈ 30 req/min per account).
- Bounded concurrency prevents all accounts from hammering the upstream simultaneously.
- Graceful 429 recovery rotates load across accounts instead of failing hard.
