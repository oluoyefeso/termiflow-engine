package taste

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	engine "github.com/oluoyefeso/termiflow-engine"
)

const (
	maxAffinities      = 20
	strongThreshold    = 0.6
	moderateThreshold  = 0.3
)

// PersonalizeScorer wraps an engine.LLMProvider to inject taste profile
// context into scoring prompts. The returned provider can be passed to
// engine.NewCurator() as a drop-in replacement.
//
// If profile is nil or has no affinities (cold start), returns the original
// provider unchanged with zero overhead.
func PersonalizeScorer(llm engine.LLMProvider, profile *TasteProfile) engine.LLMProvider {
	if profile == nil || len(profile.Affinities) == 0 {
		return llm
	}
	return &personalizedLLM{inner: llm, profile: copyProfile(profile)}
}

type personalizedLLM struct {
	inner   engine.LLMProvider
	profile *TasteProfile
}

func (p *personalizedLLM) Complete(ctx context.Context, req engine.CompletionRequest) (*engine.CompletionResponse, error) {
	tasteCtx := buildTasteContext(p.profile)
	if tasteCtx == "" {
		return p.inner.Complete(ctx, req)
	}

	// Clone the messages slice to avoid mutating the caller's data.
	msgs := make([]engine.Message, len(req.Messages))
	copy(msgs, req.Messages)

	if len(msgs) == 0 {
		return p.inner.Complete(ctx, req)
	}

	// Inject taste context: prepend to first message (system or user).
	msgs[0].Content = tasteCtx + "\n\n" + msgs[0].Content

	modified := req
	modified.Messages = msgs
	return p.inner.Complete(ctx, modified)
}

type affinityEntry struct {
	topic  string
	weight float64
}

func buildTasteContext(profile *TasteProfile) string {
	if len(profile.Affinities) == 0 {
		return ""
	}

	// Collect and sort by absolute weight descending.
	entries := make([]affinityEntry, 0, len(profile.Affinities))
	for topic, weight := range profile.Affinities {
		entries = append(entries, affinityEntry{topic, weight})
	}
	sort.Slice(entries, func(i, j int) bool {
		return math.Abs(entries[i].weight) > math.Abs(entries[j].weight)
	})

	// Take top N.
	if len(entries) > maxAffinities {
		entries = entries[:maxAffinities]
	}

	// Bucket into strong, moderate, disliked.
	var strong, moderate, disliked []string
	for _, e := range entries {
		label := fmt.Sprintf("%s (%.2f)", e.topic, e.weight)
		absW := math.Abs(e.weight)
		if e.weight < 0 {
			disliked = append(disliked, label)
		} else if absW >= strongThreshold {
			strong = append(strong, label)
		} else if absW >= moderateThreshold {
			moderate = append(moderate, label)
		}
		// Positive values below 0.3 are too weak to include.
	}

	var b strings.Builder
	b.WriteString("TASTE CONTEXT (user preferences):\n")

	if len(strong) > 0 {
		b.WriteString("Strong interests (|w| >= 0.6): ")
		b.WriteString(strings.Join(strong, ", "))
		b.WriteString("\n")
	}
	if len(moderate) > 0 {
		b.WriteString("Moderate interests (|w| >= 0.3): ")
		b.WriteString(strings.Join(moderate, ", "))
		b.WriteString("\n")
	}
	if len(disliked) > 0 {
		b.WriteString("Disliked topics (negative weight): ")
		b.WriteString(strings.Join(disliked, ", "))
		b.WriteString("\n")
	}

	b.WriteString("\nWhen scoring content relevance, factor in how well this content aligns with the\n")
	b.WriteString("user's demonstrated interests. Increase scores for strong matches, decrease for\n")
	b.WriteString("topics the user has shown disinterest in.")

	return b.String()
}
