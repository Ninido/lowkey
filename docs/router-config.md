# Multi-Model Router Config

LowKey's multi-model router lets you run multiple local LLMs behind a single OpenAI-compatible API endpoint. The router intelligently launches, manages, and shuts down model instances based on request patterns.

## Quick Start

Create a router config file and start the router:

```bash
# Start the router with your config
lowkey  # Choose "Start Multi-Model Router" and provide your config path
```

Or create a config interactively:

```bash
lowkey  # Choose "Create Router Config File"
```

## Config File Format

Router config is a JSON file. Example:

```json
{
  "host": "127.0.0.1",
  "port": 8000,
  "memory_reserve_pct": 10,
  "idle_timeout_min": 10,
  "models": {
    "fast-small": {
      "engine": "ollama",
      "path": "gemma:2b",
      "context_size": 8192,
      "thermal_profile": "quiet"
    },
    "coder-27b": {
      "engine": "llamacpp",
      "path": "/Users/ninido/models/Qwen3.8-27B-TurboFCFusion.gguf",
      "context_size": 16384,
      "thermal_profile": "balanced"
    }
  }
}
```

## Config Fields

### Top-level fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `host` | string | No | `127.0.0.1` | Bind address. Use `127.0.0.1` for localhost-only (secure). Use `0.0.0.0` to allow external access. |
| `port` | int | No | `8000` | Port to listen on |
| `memory_reserve_pct` | int | No | `10` | Percentage of system memory to keep reserved for the OS (prevents swapping when multiple models are loaded) |
| `idle_timeout_min` | int | No | `10` | Minutes of inactivity before the router shuts down an idle model instance to free memory |
| `models` | object | Yes | — | Map of model names to their configurations (see below) |

### Model configuration (per-model)

Each entry in the `models` object defines a model that can be requested via the API:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `engine` | string | Yes | — | Inference engine to use. Options: `ollama`, `mtplx`, `llamacpp`, `lmstudio`, `vllm`, `mlx`, `omlx` |
| `path` | string | Yes | — | Model path or identifier (engine-specific). For ollama: model name like `gemma:2b`. For others: file path or HuggingFace repo |
| `context_size` | int | No | `8192` | Context window size in tokens |
| `thermal_profile` | string | No | `quiet` | Thermal/power profile: `quiet`, `balanced`, `eco`, or `priority` |
| `fallback` | array | No | — | Ordered list of model names to try if this model can't be loaded (e.g., memory constraints). Only tried after eviction fails. |

## Example: Model with fallback

```json
{
  "port": 8000,
  "models": {
    "large-70b": {
      "engine": "mtplx",
      "path": "/path/to/70b-model",
      "fallback": ["coder-27b", "fast-small"]
    },
    "coder-27b": {
      "engine": "llamacpp",
      "path": "/path/to/27b-model.gguf"
    },
    "fast-small": {
      "engine": "ollama",
      "path": "gemma:2b"
    }
  }
}
```

If `large-70b` can't be loaded (e.g., not enough memory), the router tries `coder-27b`, then `fast-small`.

## Example: Two models, different engines

```json
{
  "port": 8000,
  "idle_timeout_min": 5,
  "models": {
    "subagent-fast": {
      "engine": "ollama",
      "path": "gemma:2b",
      "context_size": 4096,
      "thermal_profile": "quiet"
    },
    "main-work": {
      "engine": "llamacpp",
      "path": "/models/Mixtral-8x7B-Q5_K_M.gguf",
      "context_size": 32768,
      "thermal_profile": "balanced"
    }
  }
}
```

## Using the router API

Once the router is running, use it like any OpenAI API endpoint:

```bash
# List available models
curl http://localhost:8000/v1/models

# Chat completion
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "small-fast",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

The router will automatically:
- Launch the requested model if not already running
- Manage memory between multiple running models
- Shut down idle models after the timeout period

## Security note

The router binds to `127.0.0.1` by default, meaning only your machine can access it. To allow external access (e.g., from other devices on your network), set `host` to `0.0.0.0` or your specific IP address. Be aware this exposes the API to your network.
