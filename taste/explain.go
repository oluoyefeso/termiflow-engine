package taste

import (
	"context"
	"fmt"
	"strings"

	engine "github.com/oluoyefeso/termiflow-engine"
)

const explainPrompt = `Given this user's taste profile and the content below, explain in 2-3 sentences why this content received a relevance score of %.2f.

%s

CONTENT:
%s

Explain how the user's demonstrated interests and the content's topics led to this score.`

// Explain generates a natural-language explanation of why content scored
// the way it did given the user's taste profile.
//
// If profile is nil or has no affinities, returns a canned message without
// making an LLM call. Designed for on-demand single-item use, not batch.
func Explain(ctx context.Context, llm engine.LLMProvider, profile *TasteProfile, content string, score float64) (string, error) {
	if profile == nil || len(profile.Affinities) == 0 {
		return "No taste profile available; score is based on topic relevance alone.", nil
	}

	tasteCtx := buildTasteContext(profile)
	prompt := fmt.Sprintf(explainPrompt, score, tasteCtx, content)

	resp, err := llm.Complete(ctx, engine.CompletionRequest{
		Messages:    []engine.Message{{Role: "user", Content: prompt}},
		MaxTokens:   300,
		Temperature: 0.7,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}
