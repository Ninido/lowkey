# LowKey Router Mode — Development Plan

## Problem

Local LLM servers (llama-server, mtplx) load one model and respond to all requests with that model. Harnesses like pi specify model names in their API calls, but there's no local routing layer to match model requests to the correct running instance. Users want different models for different tasks (fast 1B model for subagents, capable 70B model for main work) without manually managing multiple server instances.

## Goal

Add a `router` mode to LowKey that acts as an OpenAI-compatible API gateway. It accepts requests specifying model names, routes them to the correct backend instance, and spawns new instances on demand — all while applying LowKey's thermal management to each instance.

## Architecture

```
Clients (pi, IDE, scripts)
    ↓ OpenAI-compatible requests to :8000
LowKey Router
    ├── Instance Manager
    │   ├── llama-server (qwen-3b)     :8001  [throttled]
    │   ├── llama-server (llama-1b)    :8002  [throttled]
    │   └── [spawn/kill based on demand]
    ├── Memory Monitor
    └── Profile Loader (thermal, context size per model)
```

## Components to Build

### 1. Router Server (`pkg/router/router.go`)

- HTTP server listening on configured port (default :8000)
- Parse OpenAI-compatible JSON requests
- Extract `model` field from request
- Route to correct instance based on model name
- Proxy request/response, including streaming (SSE)
- Endpoints: `POST /chat/completions`, `POST /completions`, `GET /models`
- Also support `/v1/` prefix for compatibility

### 2. Instance Manager (`pkg/router/instance_manager.go`)

- Track running instances: `map[string]Instance{port, process, engine, lastUsed, thermalProfile}`
- On request for model X:
  - If instance exists and healthy → route
  - If instance exists but unhealthy → kill, respawn
  - If no instance → check memory, spawn new engine process on available port
- Kill idle instances after configurable timeout
- Health check: periodic HTTP ping to each instance

### 3. Memory Monitor (`pkg/router/memory.go`)

- Estimate model memory: GGUF file size + context overhead (~20% margin)
- Monitor available system RAM
  - Linux: read `/proc/meminfo`
  - macOS: `vm_stat64` via syscall
  - Windows: WMI `Win32_OperatingSystem`
- Decide: spawn new instance vs. wait vs. evict LRU instance
- User-configurable threshold: "keep X% of RAM free before loading another model"

### 4. CLI Entry Point (`main.go`)

- New mode: `lowkey router --config router.json`
- Interactive wizard path: "🚀 Start Multi-Model Router" option
- Same profile save/load mechanism as single-model setup

## Configuration

```json
{
  "mode": "router",
  "port": 8000,
  "memory_reserve_pct": 30,
  "idle_timeout_min": 10,
  "models": {
    "qwen-3b": {
      "engine": "llamacpp",
      "path": "/models/qwen2.5-3b-instruct-q4_k_m.gguf",
      "context_size": 8192,
      "thermal_profile": "quiet"
    },
    "llama-1b": {
      "engine": "llamacpp",
      "path": "/models/llama-3.2-1b-instruct-q4_k_m.gguf",
      "context_size": 4096,
      "thermal_profile": "balanced"
    }
  }
}
```

## Reuse from Existing LowKey Code

- `engine.Engine` interface — know how to build launch commands
- `engine.BuildCommand()` — construct proper args per engine
- `engine.DiscoverModels()` — find GGUF files for model discovery
- `osutil.OSThrottler` — apply thermal profiles to spawned instances
- `osutil.ThermalProfile` — quiet/balanced/eco/priority-only presets
- `profile.Profile` — save/load router configuration

## Implementation Order

1. **Router server skeleton** — HTTP server, request parsing, model name extraction
2. **Instance manager** — spawn/kill processes, track instances, health checks
3. **Memory monitor** — platform-specific RAM detection, model size estimation
4. **Thermal integration** — apply throttler to spawned instances
5. **CLI integration** — add router mode to main.go
6. **Profile persistence** — save/load router configurations
7. **Streaming support** — proxy SSE responses

## What NOT to Build (YAGNI)

- Swap matrix DSL (advanced concurrency rules)
- TTL-based auto-unload (use idle timeout instead)
- Web UI (LowKey already has TUI)
- Prometheus metrics, API keys, CORS configuration
- Support for non-LLM endpoints (audio, image generation)

## Success Criteria

- User runs `lowkey router --config router.json`
- pi harness points to `http://localhost:8000`
- Requests for "qwen-3b" go to qwen instance, "llama-1b" to llama instance
- Instances spawn on first request, idle instances die after timeout
- Thermal profiles apply to each instance (fans stay quiet)
- Memory pressure handled gracefully (wait or evict, don't OOM)
