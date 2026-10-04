# Council Memo: LowKey Router Mode Development Plan

## Question and Scope

Should LowKey build a model routing layer into its codebase rather than wrapping llama-swap (an existing Go-based model router)? Does the development plan (docs/plans/router-mode.md) make sense architecturally?

**Scope:** Architecture decision, component design, reuse strategy, implementation priorities.
**Non-goals:** Implementation code review, testing strategy, timeline.

## Recommendation

**Approve with changes.** Build the router into LowKey's codebase, not as a wrapper around llama-swap.

**Rationale:** Both advisors verified that LowKey already owns the core infrastructure needed — the `engine.Engine` interface with `BuildCommand()` for spawning instances, `osutil.OSThrottler` for per-process thermal management, and `profile.Profile` for configuration persistence. Wrapping llama-swap would add a dependency layer without gaining anything LowKey doesn't already provide, and would make thermal management harder to compose with the routing layer.

## Accepted Feedback

**From both advisors:**
- Move SSE streaming proxying earlier in implementation priority (step 4-5, not step 7). Chat completions without streaming is incomplete for real-world use.
- Specify dynamic port allocation strategy (use `net.Listen(":0")` to find available ports, track in instance map).
- Skip swap matrix DSL for MVP. Simple model name → instance routing is sufficient.
- Add health check with 5s timeout to avoid request-blocking.

**From reviewer:**
- Document Windows WMI memory detection can be slow (seconds) — use async or cache.
- Consider memory estimation as a range, not fixed value (varies by context length, batch size, KV cache growth).

## Rejected Feedback

**From reviewer:**
- "llama-swap adds unnecessary complexity" — accepted as a risk but not a blocker. llama-swap is complex; LowKey's approach is simpler for this use case.
- "YAGNI decision to skip API keys and CORS" — accepted. These can be added later when needed.

## Owner Decisions

- HTTP library: Go standard library `net/http` with json (no external deps)
- Port allocation: Dynamic via `net.Listen(":0")`, tracked in instance map
- Swap matrix DSL: Skip, simple routing for MVP
- Streaming: Move SSE proxying to step 4-5 in priority
- Health check: 5s timeout, non-blocking
- Memory pressure: Soft limit (use swap/accept performance degradation)
- Concurrency: Separate process per model instance (llama-server handles concurrent requests natively)

## Risks

- Streaming SSE proxying is technically complex (chunk handling, connection pooling, error recovery)
- Memory estimation from GGUF file size is coarse (unverified assumption)
- Concurrency model for multiple simultaneous requests to same model instance not detailed
- Windows WMI memory detection can be slow

## Evidence

- `pkg/engine/engine.go` — `Engine` interface with `BuildCommand()` exists
- `pkg/osutil/throttler.go` — `OSThrottler` interface supports per-process thermal management
- `pkg/profile/profile.go` — Profile system uses `engine.LaunchConfig`, extendable for router mode
- llama-swap README — router-only, no thermal management

## Confidence

**Medium.** Based on plan review and code inspection by two independent advisors, not implementation testing. The architecture decision (build vs compose) is well-supported; implementation risks remain unverified until code is written.

## What Would Change the Decision

- If benchmarking shows llama-swap has significantly better request throughput or memory efficiency for the same workload
- If streaming SSE proxy proves unworkable without a dedicated reverse proxy (nginx/envoy)
- If thermal throttling per-instance doesn't compose well under concurrent high-load

## Roster, Passes, and Context Modes

- **Roster:** `oracle` (forked context), `reviewer` (default context)
- **Passes:** 1 (Pass 1 completed; both advisors agreed on recommendation and key risks)
- **Fallbacks:** None used (both advisors available)
- **Workflow ID:** 8eabe3c6-8d71-4da1-9a7c-5614199dc88e
- **Advisor run IDs:** oracle=373c77a8-6858-4e1d-8d68-68c6496ef67a, reviewer=fc8dd44b-80cb-41f4-9cfa-4034deaa206b

No Pass 2 was needed because both advisors converged on the same recommendation (approve with changes) and the same key risks. The only remaining open items were owner decisions, which were made during the council by the supervisor.
