package llm

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:11434", "llama3.2")
	if client == nil {
		t.Error("expected non-nil client")
	}
}

func TestGeneratePrompt(t *testing.T) {
	prompt := GenerateDigestPrompt(DigestData{
		TotalCommits:   100,
		TotalRepos:     5,
		TotalStars:     50,
		TopLanguages:   []string{"Go", "Rust", "Python"},
		TrendingRepos:  []string{"ollama/ollama", "astral-sh/ruff"},
		MostActiveUser: "torvalds",
	})

	if prompt == "" {
		t.Error("expected non-empty prompt")
	}

	if len(prompt) < 100 {
		t.Error("prompt seems too short")
	}
}
