# termiflow-engine

A Go library for AI-powered content curation pipelines: **search -> score -> summarize -> tag -> curate**.

Built for developers creating news aggregators, newsletter tools, research assistants, or any content product that needs intelligent curation.

## Install

```bash
go get github.com/oluoyefeso/termiflow-engine
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    engine "github.com/oluoyefeso/termiflow-engine"
    "github.com/oluoyefeso/termiflow-engine/mock"
)

func main() {
    curator := engine.NewCurator(mock.LLM())

    results := []engine.SearchResult{
        {Title: "Go 1.24 Released", URL: "https://go.dev/blog", Content: "..."},
    }

    items, _ := curator.Curate(context.Background(), "go programming", results)
    for _, item := range items {
        fmt.Printf("%s (score: %.2f)\n", item.Title, item.RelevanceScore)
    }
}
```

## How It Works

```
SearchResult[]
    │
    ▼
  Score          LLM scores each result for topic relevance (0.0-1.0)
    │
    ▼
  Filter         Drop items below relevance threshold
    │
    ▼
  Summarize      LLM generates 2-3 sentence digest per item
    │
    ▼
  Tag            LLM assigns technical keywords
    │
    ▼
  Curate         Sort by weighted relevance + recency, return
```

## Bring Your Own Provider

The engine is provider-agnostic. Implement two interfaces:

```go
type LLMProvider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

type SearchProvider interface {
    Search(ctx context.Context, req SearchRequest) ([]SearchResult, error)
}
```

Optional streaming:

```go
type StreamProvider interface {
    LLMProvider
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}
```

### Provider Adapters (coming soon)

Pre-built adapters as separate modules (zero impact on root module dependencies):

```bash
go get github.com/oluoyefeso/termiflow-engine/providers/anthropic
go get github.com/oluoyefeso/termiflow-engine/providers/openai
go get github.com/oluoyefeso/termiflow-engine/providers/tavily
```

## Configuration

```go
curator := engine.NewCurator(llm,
    engine.WithMaxConcurrency(10),        // concurrent LLM calls (default: 5)
    engine.WithRelevanceThreshold(0.3),   // minimum score to include (default: 0.5)
    engine.WithScoringWeight(0.6, 0.4),   // relevance vs recency weight (default: 0.7, 0.3)
    engine.WithScoringPrompt(myPrompt),   // custom scoring prompt template
)
```

## Testing

Use the `mock` package for deterministic testing without API keys:

```go
import "github.com/oluoyefeso/termiflow-engine/mock"

llm := mock.LLM(mock.WithScore(0.9))
search := mock.Search()
```

## Roadmap

See [GOAL-STATE.md](GOAL-STATE.md) for the full roadmap. Key upcoming features:

- **Provider adapters** — Anthropic, OpenAI, Tavily (separate Go modules)
- **Personalization engine** — topic affinity tracking, concept matching, temporal learning
- **MATCHED annotations** — explain WHY an item was relevant
- **Briefing generator** — "since you were last here" summaries
- **Pipeline hooks** — inject custom logic at any stage

## Examples

| Example | Description |
|---------|-------------|
| `examples/basic` | Minimal curation pipeline |
| `examples/custom-scoring` | Custom scoring prompt and config |
| `examples/ask` | Question-answering with sources |

Run any example:
```bash
go run ./examples/basic
```

## Contributing

### Dev Setup
```bash
git clone https://github.com/oluoyefeso/termiflow-engine.git
cd termiflow-engine
go test ./...
go test -bench=. ./...
```

### Design Principles
1. **Zero dependencies** in the root module
2. **Provider adapters** are separate Go modules (`providers/*/go.mod`)
3. **Interfaces first** — define the contract, then implement
4. **Mock everything** — `mock/` must cover all interfaces
5. **Library, not application** — no CLI concerns, no config files, no SQLite

### Before Pushing
```bash
go test ./...
go vet ./...
```

## Related Repos

| Repo | Description |
|------|-------------|
| [termiflow](https://github.com/oluoyefeso/termiflow) | CLI + TUI that uses this library |
| [termiflow.com](https://termiflow.com) | Marketing website |

## License

MIT
