# LowKey Router Mode — Development Prompt

Build a `router` mode into LowKey (Go) that acts as an OpenAI-compatible API gateway for local LLM models.

**Current LowKey:** Interactive CLI wizard to pick engine (llama-server, mtplx, ollama, etc.), model, and thermal profile. Spawns ONE inference process with duty-cycle throttling.

**Add:** A router that accepts requests on one port, inspects the `model` field, and routes to the correct backend instance. Spawns new instances on demand, kills idle ones.

**Architecture:**
- `pkg/router/router.go` — HTTP server, OpenAI-compatible endpoint parsing, model name extraction, request proxying
- `pkg/router/instance_manager.go` — spawn/kill/tracked engine processes, health checks, idle timeout
- `pkg/router/memory.go` — platform-specific RAM detection (Linux /proc/meminfo, macOS vm_stat64, Windows WMI), model size estimation from GGUF file size

**Reuse existing LowKey code:**
- `engine.Engine` interface and `engine.BuildCommand()` for spawning each model instance
- `osutil.OSThrottler` for per-instance thermal management
- `profile.Profile` for saving router configurations

**Configuration:** JSON file mapping model names to engine paths, with per-model thermal profiles and context sizes. Router listens on one port (default :8000).

**Key requirements:**
1. Parse incoming OpenAI-compatible requests, extract model name
2. Route to correct instance; spawn if not running; kill if idle
3. Apply thermal profiles to each instance (core LowKey value)
4. Handle memory: spawn new instance only if RAM allows, otherwise wait or evict LRU
5. Support streaming responses (SSE proxying)

**What to skip (YAGNI):** swap matrix DSL, TTL auto-unload, web UI, prometheus metrics, API keys, non-LLM endpoints.

**Implementation order:** Router skeleton → instance manager → memory monitor → thermal integration → CLI entry → profile persistence → streaming.
