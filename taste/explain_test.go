package taste

import (
	"context"
	"fmt"
	"strings"
	"testing"

	engine "github.com/oluoyefeso/termiflow-engine"
)

func TestExplain_GeneratesExplanation(t *testing.T) {
	llm := &testLLM{
		response: &engine.CompletionResponse{Content: "  This scored high because of Go interest.  "},
	}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}

	explanation, err := Explain(context.Background(), llm, profile, "Go 1.23 released", 0.9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if explanation != "This scored high because of Go interest." {
		t.Errorf("explanation = %q, expected trimmed response", explanation)
	}
}

func TestExplain_NilProfile(t *testing.T) {
	llm := &testLLM{}
	explanation, err := Explain(context.Background(), llm, nil, "content", 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(explanation, "No taste profile") {
		t.Errorf("expected canned message for nil profile, got: %q", explanation)
	}
	// Should not have called the LLM.
	if llm.lastReq.Messages != nil {
		t.Error("LLM should not have been called for nil profile")
	}
}

func TestExplain_EmptyAffinities(t *testing.T) {
	llm := &testLLM{}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{},
	}
	explanation, err := Explain(context.Background(), llm, profile, "content", 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(explanation, "No taste profile") {
		t.Errorf("expected canned message for empty affinities, got: %q", explanation)
	}
}

func TestExplain_ProviderError(t *testing.T) {
	wantErr := fmt.Errorf("LLM failed")
	llm := &testLLM{err: wantErr}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}

	_, err := Explain(context.Background(), llm, profile, "content", 0.5)
	if err != wantErr {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

func TestExplain_IncludesScoreInPrompt(t *testing.T) {
	llm := &testLLM{
		response: &engine.CompletionResponse{Content: "explanation"},
	}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}

	Explain(context.Background(), llm, profile, "Go article", 0.87)

	prompt := llm.lastReq.Messages[0].Content
	if !strings.Contains(prompt, "0.87") {
		t.Errorf("prompt should contain score 0.87, got: %s", prompt[:100])
	}
}

func TestExplain_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	llm := &testLLM{err: ctx.Err()}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}

	_, err := Explain(ctx, llm, profile, "content", 0.5)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
