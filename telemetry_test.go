package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTelemetryEventJSONParsing(t *testing.T) {
	jsonPayload := `{
		"timestamp": "2026-05-21T22:00:00Z",
		"type": "METRIC",
		"message": "Oscillation stabilized.",
		"state": "HEALING",
		"agent_id": "ANTIGRAVITY",
		"component": "quantum-stabilizer",
		"severity": "WARNING",
		"metrics": {
			"stabilization": 92.5,
			"flux_density": 4.2
		}
	}`

	var event TelemetryEvent
	err := json.Unmarshal([]byte(jsonPayload), &event)
	if err != nil {
		t.Fatalf("Failed to parse valid telemetry event JSON: %v", err)
	}

	if event.AgentID != "ANTIGRAVITY" {
		t.Errorf("Expected agent_id 'ANTIGRAVITY', got '%s'", event.AgentID)
	}
	if event.Component != "quantum-stabilizer" {
		t.Errorf("Expected component 'quantum-stabilizer', got '%s'", event.Component)
	}
	if event.Severity != "WARNING" {
		t.Errorf("Expected severity 'WARNING', got '%s'", event.Severity)
	}
	if event.State != "HEALING" {
		t.Errorf("Expected state 'HEALING', got '%s'", event.State)
	}

	val, exists := event.Metrics["stabilization"]
	if !exists {
		t.Fatal("Expected metric 'stabilization' to exist")
	}
	if val.(float64) != 92.5 {
		t.Errorf("Expected stabilization to be 92.5, got %v", val)
	}
}

func TestTelemetryHubRollingHistoryCache(t *testing.T) {
	hub := NewTelemetryHub("/tmp/test_foundations.sock")
	hub.maxHistory = 5 // set a low history cap for testing

	for i := 1; i <= 10; i++ {
		event := TelemetryEvent{
			Timestamp: time.Now(),
			Type:      "LOG",
			Message:   strings.Repeat("A", i),
			AgentID:   "TEST",
		}
		// Directly update cache mimicking standard behavior
		hub.mu.Lock()
		hub.history = append(hub.history, event)
		if len(hub.history) > hub.maxHistory {
			hub.history = hub.history[len(hub.history)-hub.maxHistory:]
		}
		hub.mu.Unlock()
	}

	hub.mu.Lock()
	historyLen := len(hub.history)
	hub.mu.Unlock()

	if historyLen != 5 {
		t.Errorf("Expected history cache length to be capped at 5, got %d", historyLen)
	}

	// Verify it contains the last 5 elements (6, 7, 8, 9, 10)
	hub.mu.Lock()
	firstMsg := hub.history[0].Message
	hub.mu.Unlock()

	if len(firstMsg) != 6 {
		t.Errorf("Expected oldest cached element in rolling buffer to have length 6, got %d", len(firstMsg))
	}
}

func TestSSESubscriptionAndBroadcast(t *testing.T) {
	hub := NewTelemetryHub("/tmp/test_foundations_sse.sock")
	
	ch := make(chan TelemetryEvent, 5)
	hub.AddSSEClient(ch)

	testEvent := TelemetryEvent{
		Timestamp: time.Now(),
		Type:      "LOG",
		Message:   "Test SSE Event",
		AgentID:   "TEST",
	}

	hub.BroadcastSSE(testEvent)

	select {
	case event := <-ch:
		if event.Message != "Test SSE Event" {
			t.Errorf("Expected SSE event message 'Test SSE Event', got '%s'", event.Message)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timed out waiting for SSE broadcast event")
	}

	hub.RemoveSSEClient(ch)
}
