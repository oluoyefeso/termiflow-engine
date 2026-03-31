# termiflow-engine — Goal State & Roadmap

Last updated: 2026-03-31

---

## What This Repo Is

An open-source Go library for AI-powered content curation pipelines.
The core intelligence layer behind Termiflow, but designed to be useful standalone
for anyone building content products (news aggregators, newsletters, research tools).

**GitHub:** https://github.com/oluoyefeso/termiflow-engine
**Visibility:** Public (MIT license)

---

## Goal State (the "done" picture)

The engine needs to power everything the landing page promises about personalization.
Today it handles: search -> score -> summarize -> tag -> curate.
Goal state adds: **learn -> adapt -> explain**.

### Architecture Diagram (Goal State)

```
                    CURRENT (v0.1.0)                    GOAL STATE (v0.2.0+)
                    ────────────────                    ──────────────────────

                    ┌──────────────┐                    ┌──────────────────────┐
                    │ SearchResult │                    │ SearchResult          │
                    └──────┬───────┘                    └──────┬───────────────┘
                           │                                   │
                    ┌──────▼───────┐                    ┌──────▼───────────────┐
                    │    Score     │                    │  Score + Personalize │
                    │  (LLM-based) │                    │  (LLM + user affinity│
                    └──────┬───────┘                    │   model combined)    │
                           │                            └──────┬───────────────┘
                    ┌──────▼───────┐                           │
                    │   Filter     │                    ┌──────▼───────────────┐
                    │ (threshold)  │                    │  Filter + Explain    │
                    └──────┬───────┘                    │  (threshold + MATCHED│
                           │                            │   annotation why)    │
                    ┌──────▼───────┐                    └──────┬───────────────┘
                    │  Summarize   │                           │
                    │  (LLM)       │                    ┌──────▼───────────────┐
                    └──────┬───────┐                    │  Summarize           │
                           │                            └──────┬───────────────┘
                    ┌──────▼───────┐                           │
                    │    Tag       │                    ┌──────▼───────────────┐
                    │  (LLM)       │                    │  Tag + Learn         │
                    └──────┬───────┘                    │  (update affinity    │
                           │                            │   model from result) │
                    ┌──────▼───────┐                    └──────┬───────────────┘
                    │   Curate     │                           │
                    │ (sort+return)│                    ┌──────▼───────────────┐
                    └──────────────┘                    │  Curate + Rank       │
                                                       │  (personal "For You" │
                                                       │   ranking across     │
                                                       │   all topics)        │
                                                       └──────────────────────┘
```

### New Interfaces Needed

```go
// AffinityStore — persists user reading patterns and topic affinity scores
type AffinityStore interface {
    RecordEvent(ctx context.Context, event AffinityEvent) error
    GetAffinity(ctx context.Context, userID string) (*AffinityProfile, error)
    UpdateAffinity(ctx context.Context, userID string, profile *AffinityProfile) error
}

// AffinityEvent — a user interaction that updates the model
type AffinityEvent struct {
    Type      string    // "read", "skip", "return", "save", "ask"
    ItemID    string
    TopicID   string
    Tags      []string
    DwellTime time.Duration  // how long they spent reading
    Timestamp time.Time
}

// AffinityProfile — the user's learned preferences
type AffinityProfile struct {
    UserID        string
    ProfileWeek   int                    // weeks since first event
    TopicScores   map[string]float64     // topic -> affinity (0.0-1.0)
    ConceptScores map[string]float64     // concept/tag -> affinity
    Accuracy      float64                // self-assessed accuracy %
    LastUpdated   time.Time
}

// Explainer — generates MATCHED annotations
type Explainer interface {
    Explain(ctx context.Context, item CuratedItem, profile *AffinityProfile) (string, error)
}

// BriefingGenerator — creates "since you were last here" summaries
type BriefingGenerator interface {
    Generate(ctx context.Context, items []CuratedItem, profile *AffinityProfile, lastSeen time.Time) (*Briefing, error)
}
```

---

## Current State (v0.1.0)

### What's Done
- Complete curation pipeline: search -> score -> summarize -> tag -> curate
- Provider-agnostic: `LLMProvider` and `SearchProvider` interfaces
- Optional `StreamProvider` for streaming
- Configurable: concurrency, thresholds, scoring weights, custom prompts
- Mock package for deterministic testing
- 3 runnable examples, comprehensive tests, benchmarks
- Zero external dependencies (just stdlib)

### What's Not Done
- No personalization / affinity tracking
- No concept matching (uses keyword matching)
- No MATCHED explanation generation
- No briefing generation
- No concrete provider adapters (users implement interfaces themselves)
- No pipeline hooks/middleware system

---

## Gap Analysis

```
CURRENT                                 GOAL
─────────────────────────────────────────────────────
Static scoring (LLM only)       -->    Personalized scoring (LLM + affinity)
No user model                   -->    AffinityStore + AffinityProfile
Keyword matching                -->    Concept matching via LLM
No explanations                 -->    MATCHED annotations (Explainer)
No briefing                     -->    Morning briefing (BriefingGenerator)
No provider adapters            -->    Anthropic, OpenAI, Tavily adapters
No middleware                   -->    Pipeline hooks (post v0.2.0)
```

---

## Phased Roadmap

### Phase 1: Provider Adapters
**Goal:** Make the engine immediately usable without writing your own provider.
**Priority:** P2 (makes adoption easier, unblocks external contributors).

| Task | Effort | Depends On |
|------|--------|------------|
| `providers/anthropic/` module (separate go.mod) | S | Nothing |
| `providers/openai/` module | S | Nothing |
| `providers/tavily/` module | S | Nothing |
| Built-in retry (429/503) in each adapter | S | Adapters |
| Lift ~280 lines from CLI provider code | S | Nothing |
| Tag releases: `providers/anthropic/v0.1.0` etc. | S | Above |

### Phase 2: Personalization Engine
**Goal:** The core differentiator. This is the critical path for the landing page vision.
**Priority:** P1 (blocks CLI personalization features).

| Task | Effort | Depends On |
|------|--------|------------|
| Define `AffinityStore` interface + types | S | Nothing |
| Implement `AffinityProfile` with topic + concept scores | M | Nothing |
| `sqlite` reference implementation of `AffinityStore` | M | Interface defined |
| Personalized scoring: combine LLM score with affinity weight | M | AffinityProfile |
| Temporal learning: score evolution over weeks | M | Personalized scoring |
| Accuracy calculation algorithm | S | AffinityProfile |

### Phase 3: Intelligence Features
**Goal:** Concept matching, explanations, briefings.

| Task | Effort | Depends On |
|------|--------|------------|
| Concept matching (LLM-based, not keyword) | M | Phase 2 |
| `Explainer` interface + LLM implementation | M | Concept matching |
| MATCHED annotation generation | S | Explainer |
| `BriefingGenerator` interface + implementation | M | AffinityProfile |
| "For You" cross-topic ranking algorithm | M | Personalized scoring |
| Examples: `examples/personalization` | S | All above |

### Phase 4: Ecosystem
**Goal:** External developer experience.

| Task | Effort | Depends On |
|------|--------|------------|
| Pipeline hooks/middleware system | M | Real user feedback |
| Cross-repo release automation (-> CLI auto-bump) | S | Stable releases |
| Benchmarks for personalization overhead | S | Phase 2 |
| Documentation site or detailed godoc | M | All above |

---

## Contributor Quick Reference

### Dev Setup
```bash
git clone https://github.com/oluoyefeso/termiflow-engine.git
cd termiflow-engine
go test ./...
go test -bench=. ./...
go run ./examples/basic
```

### Key Files
```
engine.go              # Core Curator type + Curate() method
types.go               # All shared types (SearchResult, CuratedItem, etc.)
config.go              # Functional options (WithMaxConcurrency, etc.)
scoring.go             # Relevance scoring logic
mock/                  # Mock providers for testing
examples/              # Runnable examples
providers/             # (future) Concrete provider adapters
```

### Design Principles
1. **Zero dependencies** in the root module (stdlib only)
2. **Provider adapters** are separate Go modules with their own go.mod
3. **Interfaces first** — define the contract, then implement
4. **Mock everything** — the mock package must cover all new interfaces
5. **No CLI concerns** — this is a library, not an application
