package engine

import "context"

// LLMProvider is the interface users implement for their LLM backend.
// Implementations are responsible for retry logic and rate-limit handling.
// The engine will not retry failed Complete calls.
type LLMProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

// StreamProvider is an optional interface for LLM backends that support streaming.
// If a provider implements StreamProvider, Ask() will use streaming when available.
type StreamProvider interface {
	LLMProvider
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}

// SearchProvider is the interface users implement for their search backend.
type SearchProvider interface {
	Search(ctx context.Context, req SearchRequest) ([]SearchResult, error)
}
