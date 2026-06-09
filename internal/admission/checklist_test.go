package admission

import "testing"

func TestPredictionMissingFields(t *testing.T) {
	tests := []struct {
		name       string
		prediction map[string]any
		wantErr    string
	}{
		{
			name:       "missing claim",
			prediction: map[string]any{"expected_effect": "e", "falsification_method": "f", "horizon": "h"},
			wantErr:    "prediction.claim is required and must be non-empty string",
		},
		{
			name:       "empty claim",
			prediction: map[string]any{"claim": "", "expected_effect": "e", "falsification_method": "f", "horizon": "h"},
			wantErr:    "prediction.claim is required and must be non-empty string",
		},
		{
			name:       "missing expected_effect",
			prediction: map[string]any{"claim": "c", "falsification_method": "f", "horizon": "h"},
			wantErr:    "prediction.expected_effect is required and must be non-empty string",
		},
		{
			name:       "missing falsification_method",
			prediction: map[string]any{"claim": "c", "expected_effect": "e", "horizon": "h"},
			wantErr:    "prediction.falsification_method is required and must be non-empty string",
		},
		{
			name:       "missing horizon",
			prediction: map[string]any{"claim": "c", "expected_effect": "e", "falsification_method": "f"},
			wantErr:    "prediction.horizon is required",
		},
		{
			name:       "empty horizon",
			prediction: map[string]any{"claim": "c", "expected_effect": "e", "falsification_method": "f", "horizon": ""},
			wantErr:    "prediction.horizon is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := map[string]any{
				"schema_version": "1",
				"task_id":        "T",
				"tier":           "standard",
				"owner":          "a",
				"accountable":    "b",
				"done_checklist": map[string]any{"item": true, "prediction_declared": true},
				"prediction":     tt.prediction,
				"evidence": map[string]any{
					"files_changed":    []any{"f.go"},
					"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
				},
				"claim": map[string]any{
					"fix_status": "fixed",
					"summary":    "s",
					"evidence":   []any{"e"},
				},
				"verification": map[string]any{
					"status": "passed",
					"checks": []any{},
				},
				"admission":         map[string]any{"outcome": "success"},
				"acceptance_status": "accepted",
				"handoff":           map[string]any{"next_action": "n", "owner": "o"},
			}
			result := Run(doc, false, false)
			if result.Outcome != "failed" {
				t.Fatalf("expected failed, got %s", result.Outcome)
			}
			found := false
			for _, e := range result.Errors {
				if e == tt.wantErr {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error %q, got %v", tt.wantErr, result.Errors)
			}
		})
	}
}

func TestDoneChecklistEvidenceAttachedTrueNoEvidence(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"evidence_attached": true, "prediction_declared": true},
		"prediction": map[string]any{
			"claim": "c", "expected_effect": "e", "falsification_method": "f", "horizon": "h",
		},
		"evidence": map[string]any{
			"files_changed": []any{"f.go"},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "n", "owner": "o"},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if e == "done_checklist.evidence_attached is true but no command_evidence, verification_artifacts, or manual_rationale is present" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected evidence_attached mismatch error, got %v", result.Errors)
	}
}

func TestDoneChecklistReadWriteSetsDeclaredTrueMissingSets(t *testing.T) {
	tests := []struct {
		name    string
		state   map[string]any
		wantErr string
	}{
		{
			name:    "missing read_set",
			state:   map[string]any{"write_set": []any{"w"}},
			wantErr: "done_checklist.read_write_sets_declared is true but state.read_set is missing",
		},
		{
			name:    "missing write_set",
			state:   map[string]any{"read_set": []any{"r"}},
			wantErr: "done_checklist.read_write_sets_declared is true but state.write_set is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := map[string]any{
				"schema_version": "1",
				"task_id":        "T",
				"tier":           "standard",
				"owner":          "a",
				"accountable":    "b",
				"done_checklist": map[string]any{"read_write_sets_declared": true, "prediction_declared": true},
				"prediction": map[string]any{
					"claim": "c", "expected_effect": "e", "falsification_method": "f", "horizon": "h",
				},
				"evidence": map[string]any{
					"files_changed":    []any{"f.go"},
					"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
				},
				"claim": map[string]any{
					"fix_status": "fixed",
					"summary":    "s",
					"evidence":   []any{"e"},
				},
				"verification": map[string]any{
					"status": "passed",
					"checks": []any{},
				},
				"admission":         map[string]any{"outcome": "success"},
				"acceptance_status": "accepted",
				"handoff":           map[string]any{"next_action": "n", "owner": "o"},
			}
			if tt.state != nil {
				doc["state"] = tt.state
			}
			result := Run(doc, false, false)
			if result.Outcome != "failed" {
				t.Fatalf("expected failed, got %s", result.Outcome)
			}
			found := false
			for _, e := range result.Errors {
				if e == tt.wantErr {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error %q, got %v", tt.wantErr, result.Errors)
			}
		})
	}
}

func TestDoneChecklistPredictionDeclaredMismatch(t *testing.T) {
	t.Run("true_but_missing", func(t *testing.T) {
		doc := map[string]any{
			"schema_version": "1",
			"task_id":        "T",
			"tier":           "standard",
			"owner":          "a",
			"accountable":    "b",
			"done_checklist": map[string]any{"prediction_declared": true},
			"prediction":     map[string]any{},
			"evidence": map[string]any{
				"files_changed":    []any{"f.go"},
				"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
			},
			"claim": map[string]any{
				"fix_status": "fixed",
				"summary":    "s",
				"evidence":   []any{"e"},
			},
			"verification": map[string]any{
				"status": "passed",
				"checks": []any{},
			},
			"admission":         map[string]any{"outcome": "success"},
			"acceptance_status": "accepted",
			"handoff":           map[string]any{"next_action": "n", "owner": "o"},
		}
		result := Run(doc, false, false)
		if result.Outcome != "failed" {
			t.Fatalf("expected failed, got %s", result.Outcome)
		}
		found := false
		for _, e := range result.Errors {
			if e == "done_checklist.prediction_declared is true but prediction is missing" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected prediction_declared mismatch error, got %v", result.Errors)
		}
		if result.BlockingPredicate != "done_checklist_prediction_mismatch" {
			t.Fatalf("expected blocking predicate done_checklist_prediction_mismatch, got %s", result.BlockingPredicate)
		}
	})
}

func TestDoneChecklistEvidenceAttachedFalseButPresent(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"evidence_attached": false, "prediction_declared": true},
		"prediction": map[string]any{
			"claim": "c", "expected_effect": "e", "falsification_method": "f", "horizon": "h",
		},
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "n", "owner": "o"},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if e == "done_checklist.evidence_attached is false but evidence is present" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected evidence_attached false mismatch error, got %v", result.Errors)
	}
}

func TestDoneChecklistReadWriteSetsDeclaredFalseButPresent(t *testing.T) {
	tests := []struct {
		name    string
		state   map[string]any
		wantErr string
	}{
		{
			name:    "read_set_present",
			state:   map[string]any{"read_set": []any{"r"}},
			wantErr: "done_checklist.read_write_sets_declared is false but state read/write sets are present",
		},
		{
			name:    "write_set_present",
			state:   map[string]any{"write_set": []any{"w"}},
			wantErr: "done_checklist.read_write_sets_declared is false but state read/write sets are present",
		},
		{
			name:    "both_present",
			state:   map[string]any{"read_set": []any{"r"}, "write_set": []any{"w"}},
			wantErr: "done_checklist.read_write_sets_declared is false but state read/write sets are present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := map[string]any{
				"schema_version": "1",
				"task_id":        "T",
				"tier":           "standard",
				"owner":          "a",
				"accountable":    "b",
				"done_checklist": map[string]any{"read_write_sets_declared": false, "prediction_declared": true},
				"prediction": map[string]any{
					"claim": "c", "expected_effect": "e", "falsification_method": "f", "horizon": "h",
				},
				"evidence": map[string]any{
					"files_changed":    []any{"f.go"},
					"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
				},
				"claim": map[string]any{
					"fix_status": "fixed",
					"summary":    "s",
					"evidence":   []any{"e"},
				},
				"verification": map[string]any{
					"status": "passed",
					"checks": []any{},
				},
				"admission":         map[string]any{"outcome": "success"},
				"acceptance_status": "accepted",
				"handoff":           map[string]any{"next_action": "n", "owner": "o"},
			}
			if tt.state != nil {
				doc["state"] = tt.state
			}
			result := Run(doc, false, false)
			if result.Outcome != "failed" {
				t.Fatalf("expected failed, got %s", result.Outcome)
			}
			found := false
			for _, e := range result.Errors {
				if e == tt.wantErr {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error %q, got %v", tt.wantErr, result.Errors)
			}
		})
	}
}
