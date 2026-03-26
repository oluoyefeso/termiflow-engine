package engine

import "time"

// FeedItem represents a curated content item produced by the engine.
// This type intentionally omits database fields (ID, SubscriptionID, IsRead, FetchedAt)
// — those belong in the application layer. Use mapping functions to convert between
// engine FeedItems and your persistence model.
type FeedItem struct {
	Title          string
	Summary        string
	Content        string
	SourceName     string
	SourceURL      string
	PublishedAt    *time.Time
	RelevanceScore float64
	Tags           []string
	Error          error // non-nil if this item had a per-item failure during curation
}

// SearchResult represents a single result from a search provider.
type SearchResult struct {
	Title       string
	URL         string
	Snippet     string
	Content     string
	PublishedAt time.Time
	Source      string
}

// SearchRequest describes a search query.
type SearchRequest struct {
	Query      string
	MaxResults int
	TimeRange  string // "day", "week", "month", "year"
}

// CompletionRequest is the input to an LLM provider.
type CompletionRequest struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	Stream      bool
}

// CompletionResponse is the output from an LLM provider.
type CompletionResponse struct {
	Content      string
	FinishReason string
	Usage        Usage
}

// Message represents a single message in an LLM conversation.
type Message struct {
	Role    string
	Content string
}

// Usage tracks token consumption for an LLM call.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// StreamChunk represents a single chunk from a streaming LLM response.
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// AskResult contains the answer and sources from a question-answering call.
type AskResult struct {
	Answer  string
	Sources []SearchResult
}
