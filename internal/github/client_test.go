// internal/github/client_test.go
package github

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-token")
	if client == nil {
		t.Error("expected non-nil client")
	}
}

func TestParseEventsJSON(t *testing.T) {
	jsonData := `[{
		"type": "PushEvent",
		"repo": {"name": "user/repo"},
		"payload": {"commits": [{"sha": "abc123", "message": "test commit"}]},
		"created_at": "2024-12-30T10:00:00Z"
	}]`

	events, err := parseEvents([]byte(jsonData))
	if err != nil {
		t.Fatalf("failed to parse events: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != "PushEvent" {
		t.Errorf("expected PushEvent, got %s", events[0].Type)
	}
}
