package taste

import (
	"context"
	"fmt"
	"strings"
	"testing"

	engine "github.com/oluoyefeso/termiflow-engine"
)

// testLLM is a mock LLMProvider that captures requests for inspection.
type testLLM struct {
	lastReq  engine.CompletionRequest
	response *engine.CompletionResponse
	err      error
}

func (t *testLLM) Complete(_ context.Context, req engine.CompletionRequest) (*engine.CompletionResponse, error) {
	t.lastReq = req
	if t.err != nil {
		return nil, t.err
	}
	if t.response != nil {
		return t.response, nil
	}
	return &engine.CompletionResponse{Content: "0.8"}, nil
}

func TestPersonalizeScorer_ColdStart(t *testing.T) {
	inner := &testLLM{}
	wrapped := PersonalizeScorer(inner, nil)
	if wrapped != inner {
		t.Error("nil profile should return original provider unchanged")
	}
}

func TestPersonalizeScorer_EmptyAffinities(t *testing.T) {
	inner := &testLLM{}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{},
	}
	wrapped := PersonalizeScorer(inner, profile)
	if wrapped != inner {
		t.Error("empty affinities should return original provider unchanged")
	}
}

func TestPersonalizeScorer_InjectsTasteContext(t *testing.T) {
	inner := &testLLM{}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8, "rust": 0.5},
	}
	wrapped := PersonalizeScorer(inner, profile)

	req := engine.CompletionRequest{
		Messages: []engine.Message{
			{Role: "user", Content: "Rate this content."},
		},
		MaxTokens: 10,
	}
	wrapped.Complete(context.Background(), req)

	if !strings.HasPrefix(inner.lastReq.Messages[0].Content, "TASTE CONTEXT") {
		t.Errorf("expected TASTE CONTEXT prefix, got: %s", inner.lastReq.Messages[0].Content[:50])
	}
}

func TestPersonalizeScorer_SystemMessageInjection(t *testing.T) {
	inner := &testLLM{}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}
	wrapped := PersonalizeScorer(inner, profile)

	req := engine.CompletionRequest{
		Messages: []engine.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "What is Go?"},
		},
	}
	wrapped.Complete(context.Background(), req)

	// Taste context should be in the system message.
	if !strings.HasPrefix(inner.lastReq.Messages[0].Content, "TASTE CONTEXT") {
		t.Error("expected TASTE CONTEXT in system message")
	}
	// User message should be unchanged.
	if inner.lastReq.Messages[1].Content != "What is Go?" {
		t.Errorf("user message was modified: %q", inner.lastReq.Messages[1].Content)
	}
}

func TestPersonalizeScorer_UserMessageInjection(t *testing.T) {
	inner := &testLLM{}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}
	wrapped := PersonalizeScorer(inner, profile)

	req := engine.CompletionRequest{
		Messages: []engine.Message{
			{Role: "user", Content: "Rate this."},
		},
	}
	wrapped.Complete(context.Background(), req)

	// Taste context should be prepended to user message.
	if !strings.Contains(inner.lastReq.Messages[0].Content, "TASTE CONTEXT") {
		t.Error("expected TASTE CONTEXT in user message")
	}
	if !strings.Contains(inner.lastReq.Messages[0].Content, "Rate this.") {
		t.Error("expected original content preserved")
	}
}

func TestPersonalizeScorer_DoesNotMutateOriginalRequest(t *testing.T) {
	inner := &testLLM{}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}
	wrapped := PersonalizeScorer(inner, profile)

	originalContent := "Rate this content."
	msgs := []engine.Message{
		{Role: "user", Content: originalContent},
	}
	req := engine.CompletionRequest{Messages: msgs}
	wrapped.Complete(context.Background(), req)

	// Original messages slice should be untouched.
	if msgs[0].Content != originalContent {
		t.Errorf("original message mutated: %q, want %q", msgs[0].Content, originalContent)
	}
}

func TestPersonalizeScorer_Top20Affinities(t *testing.T) {
	inner := &testLLM{}
	affinities := make(map[string]float64)
	for i := 0; i < 30; i++ {
		affinities[fmt.Sprintf("topic%02d", i)] = float64(i+1) / 30.0
	}
	profile := &TasteProfile{
		Version:    1,
		Affinities: affinities,
	}
	wrapped := PersonalizeScorer(inner, profile)

	req := engine.CompletionRequest{
		Messages: []engine.Message{{Role: "user", Content: "test"}},
	}
	wrapped.Complete(context.Background(), req)

	// Count how many topics appear in the taste context.
	content := inner.lastReq.Messages[0].Content
	count := 0
	for i := 0; i < 30; i++ {
		if strings.Contains(content, fmt.Sprintf("topic%02d", i)) {
			count++
		}
	}
	if count > 20 {
		t.Errorf("found %d topics in context, want at most 20", count)
	}
}

func TestPersonalizeScorer_Bucketing(t *testing.T) {
	inner := &testLLM{}
	profile := &TasteProfile{
		Version: 1,
		Affinities: map[string]float64{
			"strong":   0.8,
			"moderate": 0.4,
			"disliked": -0.5,
			"weak":     0.1, // too weak, should not appear
		},
	}
	wrapped := PersonalizeScorer(inner, profile)

	req := engine.CompletionRequest{
		Messages: []engine.Message{{Role: "user", Content: "test"}},
	}
	wrapped.Complete(context.Background(), req)

	content := inner.lastReq.Messages[0].Content
	if !strings.Contains(content, "Strong interests") {
		t.Error("expected 'Strong interests' bucket")
	}
	if !strings.Contains(content, "strong (0.80)") {
		t.Error("expected 'strong' in strong bucket")
	}
	if !strings.Contains(content, "Moderate interests") {
		t.Error("expected 'Moderate interests' bucket")
	}
	if !strings.Contains(content, "Disliked topics") {
		t.Error("expected 'Disliked topics' bucket")
	}
	// Weak topics (< 0.3 positive) should not appear in any bucket.
	if strings.Contains(content, "weak (0.10)") {
		t.Error("weak topic should not appear in taste context")
	}
}

func TestPersonalizeScorer_PassesThroughResponse(t *testing.T) {
	inner := &testLLM{
		response: &engine.CompletionResponse{Content: "0.95"},
	}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}
	wrapped := PersonalizeScorer(inner, profile)

	resp, err := wrapped.Complete(context.Background(), engine.CompletionRequest{
		Messages: []engine.Message{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "0.95" {
		t.Errorf("response content = %q, want %q", resp.Content, "0.95")
	}
}

func TestPersonalizeScorer_PassesThroughError(t *testing.T) {
	wantErr := fmt.Errorf("provider error")
	inner := &testLLM{err: wantErr}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}
	wrapped := PersonalizeScorer(inner, profile)

	_, err := wrapped.Complete(context.Background(), engine.CompletionRequest{
		Messages: []engine.Message{{Role: "user", Content: "test"}},
	})
	if err != wantErr {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

func TestPersonalizeScorer_EmptyMessages(t *testing.T) {
	inner := &testLLM{
		response: &engine.CompletionResponse{Content: "ok"},
	}
	profile := &TasteProfile{
		Version:    1,
		Affinities: map[string]float64{"go": 0.8},
	}
	wrapped := PersonalizeScorer(inner, profile)

	// Empty messages: should pass through without panic.
	resp, err := wrapped.Complete(context.Background(), engine.CompletionRequest{
		Messages: []engine.Message{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("response content = %q, want %q", resp.Content, "ok")
	}
}
