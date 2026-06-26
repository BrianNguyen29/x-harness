package admission

import (
	"encoding/json"
	"testing"
)

func FuzzRun(f *testing.F) {
	f.Add([]byte(`{"schema_version":"1","task_id":"t","tier":"light","owner":"o","accountable":"a","claim":{"fix_status":"fixed"},"verification":{"status":"passed"},"admission":{"outcome":"success"},"acceptance_status":"accepted","handoff":{"next_action":"n","owner":"u"}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"tier":"deep","stale_ground":true}`))
	f.Add([]byte(`{"tier":"standard","claim":{"fix_status":"fixed","summary":"s"},"evidence":{"files_changed":["f"],"command_evidence":[{"command":"go test","exit_code":0}]},"verification":{"status":"passed","checks":[]},"admission":{"outcome":"success"},"acceptance_status":"accepted","handoff":{"next_action":"n","owner":"o"}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Skip()
		}
		_ = Run(doc, AdmissionOptions{})
	})
}
