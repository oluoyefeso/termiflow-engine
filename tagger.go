package engine

import (
	"context"
	"fmt"
	"strings"
)

// ExtractTags extracts 2-4 relevant technical tags from content.
// Returns lowercase tags with leading # symbols stripped.
func ExtractTags(ctx context.Context, llm LLMProvider, title, content string) ([]string, error) {
	prompt := fmt.Sprintf(DefaultTagPrompt, title, content)

	resp, err := llm.Complete(ctx, CompletionRequest{
		Messages:    []Message{{Role: "user", Content: prompt}},
		MaxTokens:   50,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, err
	}

	tagStr := strings.TrimSpace(resp.Content)
	tags := strings.Split(tagStr, ",")

	var clean []string
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		tag = strings.TrimPrefix(tag, "#")
		if tag != "" {
			clean = append(clean, tag)
		}
	}

	return clean, nil
}
