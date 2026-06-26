package intake

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupFuzzPolicy(t *testing.T, root string) {
	t.Helper()
	policyPath := filepath.Join(root, "policies")
	if err := os.MkdirAll(policyPath, 0755); err != nil {
		t.Fatal(err)
	}
	content := `version: 1
intake_labels:
  tiny:
    runtime_tier: light
    signals:
      - comment_only
  normal:
    runtime_tier: standard
    signals:
      - routine_implementation
  high_risk:
    runtime_tier: deep
    signals:
      - auth
high_risk_signals:
  auth:
    description: Authentication changes
    examples:
      - login
runtime_tier_confirmation:
  tiers: [light, standard, deep]
  note: Tiers remain light, standard, deep.
`
	if err := os.WriteFile(filepath.Join(policyPath, "intake.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func FuzzExplainCardIntake(f *testing.F) {
	f.Add([]byte(`{"tier":"light","claim":{"summary":"fix"}}`))
	f.Add([]byte(`{"tier":"standard","intake":{"classification":"normal","mapped_tier":"standard"}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"tier":"deep","claim":{"summary":"auth update"},"evidence":{"files_changed":["src/auth.go"]},"governance":{"approval_status":"approved"}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var card map[string]any
		if err := json.Unmarshal(data, &card); err != nil {
			t.Skip()
		}
		tmpDir := t.TempDir()
		setupFuzzPolicy(t, tmpDir)
		policy, err := LoadIntakePolicy(tmpDir)
		if err != nil {
			t.Fatalf("load policy: %v", err)
		}
		_ = ExplainCardIntake(card, policy)
	})
}
