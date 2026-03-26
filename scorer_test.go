package engine

import (
	"context"
	"fmt"
	"testing"
)

func TestScoreRelevance_ValidScore(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "0.85"}, nil
		},
	}

	score, err := ScoreRelevance(context.Background(), llm, "go", "Go 1.23", "new features")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if score != 0.85 {
		t.Errorf("score = %f, want 0.85", score)
	}
}

func TestScoreRelevance_NonNumeric(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "I cannot rate this"}, nil
		},
	}

	score, err := ScoreRelevance(context.Background(), llm, "go", "Go 1.23", "new features")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if score != 0.5 {
		t.Errorf("non-numeric should default to 0.5, got %f", score)
	}
}

func TestScoreRelevance_ClampsOutOfRange(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     float64
	}{
		{"negative", "-0.5", 0.0},
		{"above one", "1.5", 1.0},
		{"zero", "0.0", 0.0},
		{"one", "1.0", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &testLLM{
				completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
					return &CompletionResponse{Content: tt.response}, nil
				},
			}

			score, err := ScoreRelevance(context.Background(), llm, "test", "title", "snippet")
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if score != tt.want {
				t.Errorf("score = %f, want %f", score, tt.want)
			}
		})
	}
}

func TestScoreRelevance_ProviderError(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return nil, fmt.Errorf("provider down")
		},
	}

	_, err := ScoreRelevance(context.Background(), llm, "test", "title", "snippet")
	if err == nil {
		t.Fatal("expected error from provider failure")
	}
}

func TestScoreRelevance_EmptyInputs(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "0.5"}, nil
		},
	}

	score, err := ScoreRelevance(context.Background(), llm, "", "", "")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if score != 0.5 {
		t.Errorf("score = %f, want 0.5", score)
	}
}
