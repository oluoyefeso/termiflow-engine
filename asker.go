package engine

import (
	"context"
	"fmt"
)

// Ask performs a search and generates an answer using the LLM.
// If searchProvider is nil, answers using the LLM alone without sources.
// If the LLM provider implements StreamProvider, callers can use AskStream instead.
func Ask(ctx context.Context, question string, llm LLMProvider, searchProvider SearchProvider, maxSources int) (*AskResult, error) {
	var sources []SearchResult
	if searchProvider != nil && maxSources > 0 {
		results, err := searchProvider.Search(ctx, SearchRequest{
			Query:      question,
			MaxResults: maxSources,
			TimeRange:  "week",
		})
		if err == nil {
			sources = results
		}
	}

	prompt := buildPromptWithSources(question, sources)

	resp, err := llm.Complete(ctx, CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: DefaultAskSystemPrompt},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2048,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, err
	}

	return &AskResult{
		Answer:  resp.Content,
		Sources: sources,
	}, nil
}

// AskStream is like Ask but returns a streaming channel if the provider supports it.
// Falls back to Ask() if the provider does not implement StreamProvider.
func AskStream(ctx context.Context, question string, llm LLMProvider, searchProvider SearchProvider, maxSources int) (<-chan StreamChunk, []SearchResult, error) {
	sp, ok := llm.(StreamProvider)
	if !ok {
		// Fallback: use non-streaming Ask and wrap result in a single chunk
		result, err := Ask(ctx, question, llm, searchProvider, maxSources)
		if err != nil {
			return nil, nil, err
		}
		ch := make(chan StreamChunk, 1)
		ch <- StreamChunk{Content: result.Answer, Done: true}
		close(ch)
		return ch, result.Sources, nil
	}

	var sources []SearchResult
	if searchProvider != nil && maxSources > 0 {
		results, err := searchProvider.Search(ctx, SearchRequest{
			Query:      question,
			MaxResults: maxSources,
			TimeRange:  "week",
		})
		if err == nil {
			sources = results
		}
	}

	prompt := buildPromptWithSources(question, sources)

	ch, err := sp.Stream(ctx, CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: DefaultAskSystemPrompt},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2048,
		Temperature: 0.7,
		Stream:      true,
	})
	if err != nil {
		return nil, nil, err
	}

	return ch, sources, nil
}

func buildPromptWithSources(question string, sources []SearchResult) string {
	prompt := ""

	if len(sources) > 0 {
		prompt += "Use the following sources to inform your answer:\n\n"
		for i, src := range sources {
			// FIX: use fmt.Sprintf instead of rune arithmetic (breaks for 10+ sources)
			prompt += fmt.Sprintf("Source %d: %s\n", i+1, src.Title)
			prompt += "URL: " + src.URL + "\n"
			if src.Snippet != "" {
				prompt += "Content: " + src.Snippet + "\n"
			}
			prompt += "\n"
		}
		prompt += "---\n\n"
	}

	prompt += "Question: " + question
	return prompt
}
