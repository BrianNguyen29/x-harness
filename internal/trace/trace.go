package trace

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TraceEvent is a single trace event stored as a flexible map.
type TraceEvent map[string]interface{}

func (e TraceEvent) getString(key string) string {
	if v, ok := e[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func sha256String(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func computeEventHash(event TraceEvent, previousHash string) string {
	data, _ := json.Marshal(event)
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	delete(m, "previous_hash")
	delete(m, "event_hash")
	m["previous_hash"] = previousHash
	canonical, _ := json.Marshal(m)
	return sha256String(string(canonical))
}

// ReadTrace reads events from traceDir/events.jsonl.
func ReadTrace(traceDir string) ([]TraceEvent, error) {
	filePath := filepath.Join(traceDir, "events.jsonl")
	return ReadTraceFromFile(filePath)
}

// ReadTraceFromFile reads events from a specific JSONL file.
func ReadTraceFromFile(filePath string) ([]TraceEvent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var events []TraceEvent
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event TraceEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// ChainVerificationResult is the result of VerifyTraceChain.
type ChainVerificationResult struct {
	Valid              bool   `json:"valid"`
	EventsChecked      int    `json:"events_checked"`
	FirstBrokenIndex   *int   `json:"first_broken_index,omitempty"`
	FirstBrokenEventID string `json:"first_broken_event_id,omitempty"`
	ExpectedHash       string `json:"expected_hash,omitempty"`
	ActualHash         string `json:"actual_hash,omitempty"`
}

// VerifyTraceChain verifies the integrity of the trace hash chain.
func VerifyTraceChain(events []TraceEvent) ChainVerificationResult {
	if len(events) == 0 {
		return ChainVerificationResult{Valid: true, EventsChecked: 0}
	}

	for i := 0; i < len(events); i++ {
		event := events[i]
		var previousHash string
		if i > 0 {
			previousHash = events[i-1].getString("event_hash")
		}

		// Skip legacy events without event_hash.
		if event.getString("event_hash") == "" {
			continue
		}

		expectedHash := computeEventHash(event, previousHash)
		if event.getString("event_hash") != expectedHash {
			idx := i
			return ChainVerificationResult{
				Valid:              false,
				EventsChecked:      i + 1,
				FirstBrokenIndex:   &idx,
				FirstBrokenEventID: event.getString("event_id"),
				ExpectedHash:       expectedHash,
				ActualHash:         event.getString("event_hash"),
			}
		}

		// Verify previous_hash linkage for i > 0.
		if i > 0 {
			expectedPrev := events[i-1].getString("event_hash")
			actualPrev := event.getString("previous_hash")
			if actualPrev != expectedPrev {
				idx := i
				return ChainVerificationResult{
					Valid:              false,
					EventsChecked:      i + 1,
					FirstBrokenIndex:   &idx,
					FirstBrokenEventID: event.getString("event_id"),
					ExpectedHash:       expectedPrev,
					ActualHash:         actualPrev,
				}
			}
		}
	}

	return ChainVerificationResult{
		Valid:         true,
		EventsChecked: len(events),
	}
}
