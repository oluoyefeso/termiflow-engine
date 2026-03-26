package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// ScoreRelevance evaluates how relevant content is to a topic, returning a score from 0.0 to 1.0.
// On parse error (e.g., LLM returns non-numeric text), defaults to 0.5.
// On provider error, returns the error.
func ScoreRelevance(ctx context.Context, llm LLMProvider, topic, title, snippet string) (float64, error) {
	return scoreRelevanceWithPrompt(ctx, llm, DefaultScoringPrompt, topic, title, snippet)
}

func scoreRelevanceWithPrompt(ctx context.Context, llm LLMProvider, prompt, topic, title, snippet string) (float64, error) {
	formatted := fmt.Sprintf(prompt, topic, title, snippet)

	resp, err := llm.Complete(ctx, CompletionRequest{
		Messages:    []Message{{Role: "user", Content: formatted}},
		MaxTokens:   10,
		Temperature: 0.1,
	})
	if err != nil {
		return 0, err
	}

	score, err := strconv.ParseFloat(strings.TrimSpace(resp.Content), 64)
	if err != nil {
		return 0.5, nil
	}

	// Clamp to [0.0, 1.0]
	if score < 0 {
		score = 0
	} else if score > 1 {
		score = 1
	}

	return score, nil
}
