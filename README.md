# termiflow-engine

A Go library for AI-powered content curation pipelines: **search → score → summarize → tag → curate**.

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

    items, _ := curator.Curate(context.Background(), "go programming", results)
    for _, item := range items {
        fmt.Printf("%s (score: %.2f)\n", item.Title, item.RelevanceScore)
    }
}
```

## How It Works

1. **Score** each search result for relevance to a topic (0.0-1.0) using your LLM
2. **Filter** items below the relevance threshold
3. **Summarize** high-relevance items into 2-3 sentence digests
4. **Tag** each item with relevant technical keywords
5. **Sort** by a weighted combination of relevance and recency (exponential time-decay)

## Bring Your Own Provider

The engine uses two interfaces — implement them for your LLM and search backend:

```go
type LLMProvider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

type SearchProvider interface {
    Search(ctx context.Context, req SearchRequest) ([]SearchResult, error)
}
```

Optional streaming support:

```go
type StreamProvider interface {
    LLMProvider
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}
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

## Examples

- `examples/basic` — minimal curation pipeline
- `examples/custom-scoring` — custom scoring prompt and config
- `examples/ask` — question-answering with sources

Run any example: `go run ./examples/basic`

## License

MIT
