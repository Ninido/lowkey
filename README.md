# LowKey (⚡)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/ninido/lowkey?style=flat&color=7D56F4)](https://github.com/ninido/lowkey/releases)
[![Homebrew](https://img.shields.io/badge/brew-tap-success)](https://github.com/ninido/homebrew-tap)
[![GitHub Pages](https://img.shields.io/badge/Website-Live-04B575)](https://ninido.github.io/lowkey)

**Silent, cool, and battery-friendly local LLM launcher.**

LowKey is a cross-platform interactive launcher and thermal duty-cycle orchestrator for local AI inference engines. It stops your laptop fans from screaming, keeps temperatures under 60°C, and prevents battery drain when hosting local models (`mtplx`, `llama.cpp`, `ollama`, `vLLM`, `MLX`, `LM Studio`).

🌐 **[Visit the Website & Interactive Playground](https://ninido.github.io/lowkey)**

---

## Quick Install

### Homebrew (macOS & Linux)
```bash
brew tap ninido/tap
brew install lowkey
```

### One-line curl installer (macOS & Linux)
```bash
curl -fsSL https://raw.githubusercontent.com/ninido/lowkey/main/install.sh | bash
```

### Go Install
```bash
go install github.com/ninido/lowkey@latest
```

### Pre-built Binaries
Grab signed binaries for macOS (Apple Silicon / Intel), Linux (x86_64 / arm64), and Windows from the [Releases page](https://github.com/ninido/lowkey/releases).

---

## Why LowKey?

When you run raw inference engines directly (`ollama run`, `llama-server`, `vllm`), they aggressively saturate 100% of GPU/CPU compute threads:
- 🔥 **Excessive Heat (90°C+):** Fans ramp to 100% RPM within minutes.
- 📉 **Thermal Throttling:** Clock speeds degrade by up to 35% after 5 minutes of continuous token stream.
- 🔋 **Battery Drain:** Background coding assistants (e.g. Copilot local models) deplete a full charge in an hour.
- 🐢 **Desktop Stutter:** System UI, browser tabs, and IDEs freeze.

**LowKey fixes this transparently:**
- 🔇 **Duty-Cycle Throttling:** Micro-pauses (e.g. 1200ms work / 300ms pause) give silicon heat-dissipation windows without disrupting stream readability.
- ⚙️ **OS Background Quality-of-Service (QoS):** Uses macOS `taskpolicy` background clamping, Linux `ionice`/`nice`, or Windows low priority classes to route models to efficiency cores.
- 📂 **Instant Profile Switching:** Save ports, speculation depth, thinking budget, and thermal presets to launch in 1 click.

---

## Architecture Overview

```
lowkey/
├── main.go                     # Interactive Charm TUI & workflow runner
├── go.mod / go.sum
├── bin/                        # Pre-compiled cross-platform binaries
│   ├── lowkey-darwin-arm64
│   ├── lowkey-linux-amd64
│   └── lowkey-windows-amd64.exe
├── pkg/
│   ├── osutil/                 # OS Throttling & Priority Abstraction
│   │   ├── throttler.go        # Common interface & thermal profiles
│   │   ├── throttler_darwin.go # macOS: taskpolicy, nice, SIGSTOP/SIGCONT, pmset
│   │   ├── throttler_linux.go  # Linux: ionice, nice, SIGSTOP/SIGCONT, sysfs power
│   │   └── throttler_windows.go# Windows: NtSuspendProcess, NtResumeProcess, PriorityClass, PowerStatus
│   │
│   ├── engine/                 # Pluggable Inference Engines
│   │   ├── engine.go           # Engine registry, scanner & ModelInfo interface
│   │   ├── mtplx.go            # MTPLX (Apple Silicon MTP server)
│   │   ├── llamacpp.go         # llama.cpp (llama-server)
│   │   ├── lmstudio.go         # LM Studio (lms CLI & local cache)
│   │   ├── omlx.go             # oMLX (Apple Silicon production server)
│   │   ├── ollama.go           # Ollama (ollama CLI & model list)
│   │   ├── vllm.go             # vLLM (CUDA / ROCm / CPU)
│   │   └── mlx.go              # MLX-LM (Apple MLX framework server)
│   │
│   └── profile/                # Setup Persistence
│       └── profile.go          # JSON profile save & load (~/.lowkey/profiles)
```

---

## Features

1. **Inference Engine Auto-Scanner**:
   - Detects all available runtimes in your environment (`mtplx`, `llama-server`, `lms`, `omlx`, `ollama`, `vllm`, `mlx_lm`).
   - Visually indicates auto-detected engines (`✅`) vs uninstalled engines (`⚠️`).
2. **Model Discovery**:
   - Inspects `~/.mtplx/models/`, `~/.cache/lm-studio/models/`, `~/.cache/huggingface/hub/`, `~/models/`, LM Studio CLI (`lms ls --json`), and Ollama local cache (`ollama list`).
   - Also supports custom path / repo entry.
3. **Setup Profile Persistence (JSON)**:
   - Save your preferred models, ports, throughput presets, thinking effort, speculation depth, and thermal profiles.
   - Start with **📂 Load Saved Setup** for instant 1-click execution or **✨ Create New Setup** for the step-by-step wizard.
   - Profiles are saved in `~/.lowkey/profiles/*.json`.
4. **Thermal & Power Duty-Cycle Throttler**:
   - **Quiet**: 1200ms work / 300ms pause (75–80% duty cycle, fans stay low).
   - **Balanced**: 1800ms work / 200ms pause (85–90% duty cycle).
   - **Eco / Battery**: 800ms work / 400ms pause (65% duty cycle, maximizes battery life).
   - **Priority-Only**: 100% duty cycle with OS-level background/idle process scheduling.
   - Automatic AC / Battery power source detection.

---

## Build Commands

### macOS (Apple Silicon & Intel)
```bash
go build -o lowkey main.go
```

### Linux (x86_64 / arm64)
```bash
GOOS=linux GOARCH=amd64 go build -o bin/lowkey-linux-amd64 main.go
GOOS=linux GOARCH=arm64 go build -o bin/lowkey-linux-arm64 main.go
```

### Windows (x86_64)
```bash
GOOS=windows GOARCH=amd64 go build -o bin/lowkey-windows-amd64.exe main.go
```
