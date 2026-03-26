package engine

import (
	"context"
	"fmt"
	"strings"
)

// Summarize generates a 2-3 sentence summary of content for a topic subscriber.
// The content parameter should be plain text (not HTML), truncated to ~2000 chars by the caller.
// The engine does not perform HTML stripping or length truncation.
func Summarize(ctx context.Context, llm LLMProvider, topic, title, content string) (string, error) {
	prompt := fmt.Sprintf(DefaultSummaryPrompt, topic, title, content)

	resp, err := llm.Complete(ctx, CompletionRequest{
		Messages:    []Message{{Role: "user", Content: prompt}},
		MaxTokens:   200,
		Temperature: 0.5,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}
