# Lintasan Go — Full Feature Parity Plan

> Goal: Surpass Lintasan Node and 9router in features.
> 25+ features, 6 phases, estimated 2-3 weeks.

## Phase 1: Resilience Core (3 subagents, parallel)

### Task 1.1: Retry + Circuit Breaker packages
- Create: `internal/retry/retry.go` — exponential backoff with configurable max retries, base delay, jitter
- Create: `internal/circuit/circuit.go` — 3-state breaker (CLOSED→OPEN→HALF_OPEN), configurable threshold/cooldown
- Both are pure Go packages with zero file conflicts

### Task 1.2: Rate Limiter + Token Counter
- Create: `internal/ratelimit/ratelimit.go` — sliding window per-key/per-IP rate limiter
- Modify: `internal/server/proxy.go` — count tokens from stream/response into request_logs

### Task 1.3: Fallback Chain + Wire All
- Create: `internal/fallback/fallback.go` — model+connection fallback chains from settings
- Modify: `internal/server/proxy.go` — integrate retry, circuit breaker, rate limiter, fallback into doUpstream
- Build + test on port 20181

## Phase 2-6: To be executed after Phase 1
(See full roadmap in audit above)
