package episode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createEpisode(t *testing.T, dir, episodeID, taskID, createdAt string, previousEpisodeID *string) {
	t.Helper()

	dataContent := []byte("hello " + episodeID)
	dataPath := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(dataPath, dataContent, 0644); err != nil {
		t.Fatal(err)
	}
	dataHash := sha256Hex(dataContent)

	event := map[string]interface{}{
		"event_id":   "E1_" + episodeID,
		"event_type": "verify_completed",
		"outcome":    "success",
	}
	event["event_hash"] = computeTraceHash(event)
	event["previous_hash"] = nil
	eventBytes, _ := json.Marshal(event)
	traceContent := append(eventBytes, '\n')
	tracePath := filepath.Join(dir, "trace.jsonl")
	if err := os.WriteFile(tracePath, traceContent, 0644); err != nil {
		t.Fatal(err)
	}
	traceHash := sha256Hex(traceContent)

	hashes := map[string]interface{}{
		"schema_version": "1",
		"files": []map[string]interface{}{
			{"path": "data.txt", "sha256": "sha256:" + dataHash, "size_bytes": len(dataContent)},
			{"path": "trace.jsonl", "sha256": "sha256:" + traceHash, "size_bytes": len(traceContent)},
		},
	}
	hashesBytes, _ := json.MarshalIndent(hashes, "", "  ")
	hashesPath := filepath.Join(dir, "hashes.json")
	if err := os.WriteFile(hashesPath, hashesBytes, 0644); err != nil {
		t.Fatal(err)
	}
	hashesHash := sha256Hex(hashesBytes)

	manifest := map[string]interface{}{
		"schema_version":      "1",
		"episode_id":          episodeID,
		"task_id":             taskID,
		"created_at":          createdAt,
		"x_harness_version":   "0.1.0",
		"previous_episode_id": previousEpisodeID,
		"git": map[string]interface{}{
			"base_sha": nil, "head_sha": nil,
			"dirty_before_verify": false, "dirty_after_verify": false,
		},
		"policy_hashes": map[string]interface{}{},
		"schema_hashes": map[string]interface{}{},
		"verdict": map[string]interface{}{
			"admission_outcome": "success", "acceptance_status": "accepted", "blocking_predicate": nil,
		},
		"mutation_guard": map[string]interface{}{
			"enabled": false, "violated": false, "unexpected_delta_count": 0,
		},
		"signing": map[string]interface{}{
			"mode": "unsigned", "signature_ref": nil,
		},
		"bundle_refs": map[string]interface{}{
			"raw": nil, "redacted": nil,
		},
		"admission_authority": false,
		"hashes_hash":         "sha256:" + hashesHash,
	}

	manifestBytes, _ := json.Marshal(manifest)
	manifestHash := sha256Hex(manifestBytes)
	manifest["manifest_hash"] = "sha256:" + manifestHash

	manifestOutBytes, _ := json.MarshalIndent(manifest, "", "  ")
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestOutBytes, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyEpisodeChain_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true for empty dir, got errors: %v", result.Errors)
	}
	if result.EpisodesChecked != 0 {
		t.Fatalf("expected 0 episodes checked, got %d", result.EpisodesChecked)
	}
	if len(result.EpisodeIDs) != 0 {
		t.Fatalf("expected 0 episode ids, got %v", result.EpisodeIDs)
	}
}

func TestVerifyEpisodeChain_MissingDirectory(t *testing.T) {
	result, err := VerifyEpisodeChain("test-task", "/nonexistent/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true for missing dir, got errors: %v", result.Errors)
	}
	if result.EpisodesChecked != 0 {
		t.Fatalf("expected 0 episodes checked, got %d", result.EpisodesChecked)
	}
}

func TestVerifyEpisodeChain_NoMatchingTaskID(t *testing.T) {
	tmpDir := t.TempDir()
	epDir := filepath.Join(tmpDir, "ep_2024-01-01T00-00-00Z_other-task")
	os.MkdirAll(epDir, 0755)
	createEpisode(t, epDir, "ep_2024-01-01T00-00-00Z_other-task", "other-task", "2024-01-01T00:00:00Z", nil)

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true when no episodes match, got errors: %v", result.Errors)
	}
	if result.EpisodesChecked != 0 {
		t.Fatalf("expected 0 episodes checked, got %d", result.EpisodesChecked)
	}
}

func TestVerifyEpisodeChain_SingleEpisode(t *testing.T) {
	tmpDir := t.TempDir()
	epDir := filepath.Join(tmpDir, "ep_2024-01-01T00-00-00Z_test-task")
	os.MkdirAll(epDir, 0755)
	createEpisode(t, epDir, "ep_2024-01-01T00-00-00Z_test-task", "test-task", "2024-01-01T00:00:00Z", nil)

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got errors: %v", result.Errors)
	}
	if result.EpisodesChecked != 1 {
		t.Fatalf("expected 1 episode checked, got %d", result.EpisodesChecked)
	}
	if len(result.EpisodeIDs) != 1 || result.EpisodeIDs[0] != "ep_2024-01-01T00-00-00Z_test-task" {
		t.Fatalf("unexpected episode ids: %v", result.EpisodeIDs)
	}
}

func TestVerifyEpisodeChain_ValidChain(t *testing.T) {
	tmpDir := t.TempDir()

	ep1 := filepath.Join(tmpDir, "ep_2024-01-01T00-00-00Z_test-task")
	os.MkdirAll(ep1, 0755)
	prev1 := "ep_2024-01-01T00-00-00Z_test-task"
	createEpisode(t, ep1, prev1, "test-task", "2024-01-01T00:00:00Z", nil)

	ep2 := filepath.Join(tmpDir, "ep_2024-01-02T00-00-00Z_test-task")
	os.MkdirAll(ep2, 0755)
	createEpisode(t, ep2, "ep_2024-01-02T00-00-00Z_test-task", "test-task", "2024-01-02T00:00:00Z", &prev1)

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got errors: %v", result.Errors)
	}
	if result.EpisodesChecked != 2 {
		t.Fatalf("expected 2 episodes checked, got %d", result.EpisodesChecked)
	}
	if len(result.EpisodeIDs) != 2 {
		t.Fatalf("expected 2 episode ids, got %v", result.EpisodeIDs)
	}
	if result.EpisodeIDs[0] != prev1 {
		t.Fatalf("expected first episode %s, got %s", prev1, result.EpisodeIDs[0])
	}
	if result.EpisodeIDs[1] != "ep_2024-01-02T00-00-00Z_test-task" {
		t.Fatalf("expected second episode ep_2024-01-02T00-00-00Z_test-task, got %s", result.EpisodeIDs[1])
	}
}

func TestVerifyEpisodeChain_MissingPreviousEpisode(t *testing.T) {
	tmpDir := t.TempDir()

	ep1 := filepath.Join(tmpDir, "ep_2024-01-01T00-00-00Z_test-task")
	os.MkdirAll(ep1, 0755)
	createEpisode(t, ep1, "ep_2024-01-01T00-00-00Z_test-task", "test-task", "2024-01-01T00:00:00Z", nil)

	ep2 := filepath.Join(tmpDir, "ep_2024-01-02T00-00-00Z_test-task")
	os.MkdirAll(ep2, 0755)
	missingPrev := "ep_nonexistent"
	createEpisode(t, ep2, "ep_2024-01-02T00-00-00Z_test-task", "test-task", "2024-01-02T00:00:00Z", &missingPrev)

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "missing previous episode ep_nonexistent") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing previous episode error, got %v", result.Errors)
	}
}

func TestVerifyEpisodeChain_WrongSequentialOrder(t *testing.T) {
	tmpDir := t.TempDir()

	ep1 := filepath.Join(tmpDir, "ep_2024-01-01T00-00-00Z_test-task")
	os.MkdirAll(ep1, 0755)
	prev1 := "ep_2024-01-01T00-00-00Z_test-task"
	createEpisode(t, ep1, prev1, "test-task", "2024-01-01T00:00:00Z", nil)

	ep2 := filepath.Join(tmpDir, "ep_2024-01-02T00-00-00Z_test-task")
	os.MkdirAll(ep2, 0755)
	wrongPrev := "ep_wrong"
	createEpisode(t, ep2, "ep_2024-01-02T00-00-00Z_test-task", "test-task", "2024-01-02T00:00:00Z", &wrongPrev)

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "previous_episode_id expected") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected previous_episode_id mismatch error, got %v", result.Errors)
	}
}

func TestVerifyEpisodeChain_NilPreviousForSecond(t *testing.T) {
	tmpDir := t.TempDir()

	ep1 := filepath.Join(tmpDir, "ep_2024-01-01T00-00-00Z_test-task")
	os.MkdirAll(ep1, 0755)
	createEpisode(t, ep1, "ep_2024-01-01T00-00-00Z_test-task", "test-task", "2024-01-01T00:00:00Z", nil)

	ep2 := filepath.Join(tmpDir, "ep_2024-01-02T00-00-00Z_test-task")
	os.MkdirAll(ep2, 0755)
	createEpisode(t, ep2, "ep_2024-01-02T00-00-00Z_test-task", "test-task", "2024-01-02T00:00:00Z", nil)

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "previous_episode_id expected ep_2024-01-01T00-00-00Z_test-task, got <nil>") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected nil previous_episode_id error, got %v", result.Errors)
	}
}

func TestVerifyEpisodeChain_SortedByCreatedAt(t *testing.T) {
	tmpDir := t.TempDir()

	ep1 := filepath.Join(tmpDir, "ep_2024-01-02T00-00-00Z_test-task")
	os.MkdirAll(ep1, 0755)
	prev1 := "ep_2024-01-01T00-00-00Z_test-task"
	createEpisode(t, ep1, "ep_2024-01-02T00-00-00Z_test-task", "test-task", "2024-01-02T00:00:00Z", &prev1)

	ep2 := filepath.Join(tmpDir, "ep_2024-01-01T00-00-00Z_test-task")
	os.MkdirAll(ep2, 0755)
	createEpisode(t, ep2, prev1, "test-task", "2024-01-01T00:00:00Z", nil)

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got errors: %v", result.Errors)
	}
	if len(result.EpisodeIDs) != 2 {
		t.Fatalf("expected 2 episode ids, got %v", result.EpisodeIDs)
	}
	if result.EpisodeIDs[0] != prev1 {
		t.Fatalf("expected first episode %s after sort, got %s", prev1, result.EpisodeIDs[0])
	}
	if result.EpisodeIDs[1] != "ep_2024-01-02T00-00-00Z_test-task" {
		t.Fatalf("expected second episode ep_2024-01-02T00-00-00Z_test-task, got %s", result.EpisodeIDs[1])
	}
}

func TestVerifyEpisodeChain_InvalidEpisodeDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	epDir := filepath.Join(tmpDir, "ep_2024-01-01T00-00-00Z_test-task")
	os.MkdirAll(epDir, 0755)
	// Create an invalid episode (missing hashes.json)
	manifest := map[string]interface{}{
		"schema_version":      "1",
		"episode_id":          "ep_2024-01-01T00-00-00Z_test-task",
		"task_id":             "test-task",
		"created_at":          "2024-01-01T00:00:00Z",
		"x_harness_version":   "0.1.0",
		"previous_episode_id": nil,
		"git": map[string]interface{}{
			"base_sha": nil, "head_sha": nil,
			"dirty_before_verify": false, "dirty_after_verify": false,
		},
		"policy_hashes": map[string]interface{}{},
		"schema_hashes": map[string]interface{}{},
		"verdict": map[string]interface{}{
			"admission_outcome": "success", "acceptance_status": "accepted", "blocking_predicate": nil,
		},
		"mutation_guard": map[string]interface{}{
			"enabled": false, "violated": false, "unexpected_delta_count": 0,
		},
		"signing": map[string]interface{}{
			"mode": "unsigned", "signature_ref": nil,
		},
		"bundle_refs": map[string]interface{}{
			"raw": nil, "redacted": nil,
		},
		"admission_authority": false,
		"hashes_hash":         "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}
	manifestBytes, _ := json.Marshal(manifest)
	manifestHash := sha256Hex(manifestBytes)
	manifest["manifest_hash"] = "sha256:" + manifestHash
	manifestOut, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(filepath.Join(epDir, "manifest.json"), manifestOut, 0644)
	os.WriteFile(filepath.Join(epDir, "trace.jsonl"), []byte("{}\n"), 0644)

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "hashes.json not found") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected hashes.json not found error, got %v", result.Errors)
	}
}

func TestVerifyEpisodeChain_NonEpDirectoryIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	otherDir := filepath.Join(tmpDir, "other_dir")
	os.MkdirAll(otherDir, 0755)
	os.WriteFile(filepath.Join(otherDir, "manifest.json"), []byte("{}"), 0644)

	result, err := VerifyEpisodeChain("test-task", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got errors: %v", result.Errors)
	}
	if result.EpisodesChecked != 0 {
		t.Fatalf("expected 0 episodes checked, got %d", result.EpisodesChecked)
	}
}
