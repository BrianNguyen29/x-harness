package episode

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func computeTraceHash(event map[string]interface{}) string {
	data, _ := json.Marshal(event)
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	delete(m, "previous_hash")
	delete(m, "event_hash")
	m["previous_hash"] = ""
	canonical, _ := json.Marshal(m)
	sum := sha256.Sum256(canonical)
	return fmt.Sprintf("%x", sum)
}

func createValidEpisode(t *testing.T, dir string) {
	t.Helper()

	// Create data.txt
	dataContent := []byte("hello world")
	dataPath := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(dataPath, dataContent, 0644); err != nil {
		t.Fatal(err)
	}
	dataHash := sha256Hex(dataContent)

	// Create trace.jsonl with a valid trace event
	event := map[string]interface{}{
		"event_id":   "E1",
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

	// Create evidence-index.jsonl
	evidenceEntry := map[string]interface{}{
		"schema_version":      "1",
		"task_id":             "test-task",
		"evidence_id":         "ev-001",
		"layer":               "raw",
		"kind":                "other",
		"path":                "data.txt",
		"sha256":              dataHash,
		"size_bytes":          len(dataContent),
		"redacted":            false,
		"created_at":          "2024-01-01T00:00:00Z",
		"admission_authority": false,
	}
	evidenceBytes, _ := json.Marshal(evidenceEntry)
	evidenceContent := append(evidenceBytes, '\n')
	evidencePath := filepath.Join(dir, "evidence-index.jsonl")
	if err := os.WriteFile(evidencePath, evidenceContent, 0644); err != nil {
		t.Fatal(err)
	}
	evidenceHash := sha256Hex(evidenceContent)

	// Create hashes.json
	hashes := map[string]interface{}{
		"schema_version": "1",
		"files": []map[string]interface{}{
			{"path": "data.txt", "sha256": "sha256:" + dataHash, "size_bytes": len(dataContent)},
			{"path": "trace.jsonl", "sha256": "sha256:" + traceHash, "size_bytes": len(traceContent)},
			{"path": "evidence-index.jsonl", "sha256": "sha256:" + evidenceHash, "size_bytes": len(evidenceContent)},
		},
	}
	hashesBytes, _ := json.MarshalIndent(hashes, "", "  ")
	hashesPath := filepath.Join(dir, "hashes.json")
	if err := os.WriteFile(hashesPath, hashesBytes, 0644); err != nil {
		t.Fatal(err)
	}
	hashesHash := sha256Hex(hashesBytes)

	// Create manifest.json
	manifest := map[string]interface{}{
		"schema_version":      "1",
		"episode_id":          "ep_2024-01-01T00-00-00Z_test-task",
		"task_id":             "test-task",
		"created_at":          "2024-01-01T00:00:00Z",
		"x_harness_version":   "0.1.0",
		"previous_episode_id": nil,
		"git": map[string]interface{}{
			"base_sha":            nil,
			"head_sha":            nil,
			"dirty_before_verify": false,
			"dirty_after_verify":  false,
		},
		"policy_hashes": map[string]interface{}{},
		"schema_hashes": map[string]interface{}{},
		"verdict": map[string]interface{}{
			"admission_outcome":  "success",
			"acceptance_status":  "accepted",
			"blocking_predicate": nil,
		},
		"mutation_guard": map[string]interface{}{
			"enabled":               false,
			"violated":              false,
			"unexpected_delta_count": 0,
		},
		"signing": map[string]interface{}{
			"mode":          "unsigned",
			"signature_ref": nil,
		},
		"bundle_refs": map[string]interface{}{
			"raw":      nil,
			"redacted": nil,
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

func createTarball(t *testing.T, sourceDir, destPath string) {
	t.Helper()
	file, err := os.Create(destPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			tw.WriteHeader(&tar.Header{
				Name:     rel + "/",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			})
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		tw.WriteHeader(&tar.Header{
			Name: rel,
			Mode: 0644,
			Size: int64(len(data)),
		})
		tw.Write(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInspectEpisode_ValidDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisode(t, tmpDir)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got errors: %v", result.Errors)
	}
	if result.EpisodeID == nil || *result.EpisodeID != "ep_2024-01-01T00-00-00Z_test-task" {
		t.Fatalf("unexpected episode_id: %v", result.EpisodeID)
	}
	if result.TaskID == nil || *result.TaskID != "test-task" {
		t.Fatalf("unexpected task_id: %v", result.TaskID)
	}
	if result.FileCount != 3 {
		t.Fatalf("expected file_count=3, got %d", result.FileCount)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
}

func TestInspectEpisode_ValidTarball(t *testing.T) {
	sourceDir := t.TempDir()
	createValidEpisode(t, sourceDir)

	tarPath := filepath.Join(t.TempDir(), "episode.tar.gz")
	createTarball(t, sourceDir, tarPath)

	result, err := InspectEpisode(tarPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got errors: %v", result.Errors)
	}
}

func TestInspectEpisode_MissingManifest(t *testing.T) {
	tmpDir := t.TempDir()
	// Only create hashes.json and trace.jsonl
	os.WriteFile(filepath.Join(tmpDir, "hashes.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "trace.jsonl"), []byte("{}\n"), 0644)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	if len(result.Errors) == 0 || !strings.Contains(result.Errors[0], "manifest.json not found") {
		t.Fatalf("expected manifest.json not found error, got %v", result.Errors)
	}
}

func TestInspectEpisode_MissingHashes(t *testing.T) {
	tmpDir := t.TempDir()
	// Create manifest.json without hashes_hash reference issues
	manifest := map[string]interface{}{
		"schema_version":      "1",
		"episode_id":          "ep_test",
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
	os.WriteFile(filepath.Join(tmpDir, "manifest.json"), manifestOut, 0644)
	os.WriteFile(filepath.Join(tmpDir, "trace.jsonl"), []byte("{}\n"), 0644)

	result, err := InspectEpisode(tmpDir)
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

func TestInspectEpisode_MissingTrace(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisode(t, tmpDir)
	os.Remove(filepath.Join(tmpDir, "trace.jsonl"))
	// Update hashes.json to remove trace.jsonl
	hashesData, _ := os.ReadFile(filepath.Join(tmpDir, "hashes.json"))
	var hashes map[string]interface{}
	json.Unmarshal(hashesData, &hashes)
	files := hashes["files"].([]interface{})
	var newFiles []interface{}
	for _, f := range files {
		fileMap := f.(map[string]interface{})
		if fileMap["path"] != "trace.jsonl" {
			newFiles = append(newFiles, f)
		}
	}
	hashes["files"] = newFiles
	hashesBytes, _ := json.MarshalIndent(hashes, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "hashes.json"), hashesBytes, 0644)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "trace.jsonl not found") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected trace.jsonl not found error, got %v", result.Errors)
	}
}

func TestInspectEpisode_MissingEvidenceIndexWarning(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisode(t, tmpDir)
	os.Remove(filepath.Join(tmpDir, "evidence-index.jsonl"))
	// Update hashes.json to remove evidence-index.jsonl
	hashesData, _ := os.ReadFile(filepath.Join(tmpDir, "hashes.json"))
	var hashes map[string]interface{}
	json.Unmarshal(hashesData, &hashes)
	files := hashes["files"].([]interface{})
	var newFiles []interface{}
	for _, f := range files {
		fileMap := f.(map[string]interface{})
		if fileMap["path"] != "evidence-index.jsonl" {
			newFiles = append(newFiles, f)
		}
	}
	hashes["files"] = newFiles
	hashesBytes, _ := json.MarshalIndent(hashes, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "hashes.json"), hashesBytes, 0644)

	// Update manifest hashes_hash
	newHashesHash := sha256Hex(hashesBytes)
	manifestData, _ := os.ReadFile(filepath.Join(tmpDir, "manifest.json"))
	var manifest map[string]interface{}
	json.Unmarshal(manifestData, &manifest)
	manifest["hashes_hash"] = "sha256:" + newHashesHash
	delete(manifest, "manifest_hash")
	manifestBytes, _ := json.Marshal(manifest)
	manifestHash := sha256Hex(manifestBytes)
	manifest["manifest_hash"] = "sha256:" + manifestHash
	manifestOut, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "manifest.json"), manifestOut, 0644)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got errors: %v", result.Errors)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "evidence-index.jsonl not found") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected evidence-index.jsonl warning, got %v", result.Warnings)
	}
}

func TestInspectEpisode_HashMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisode(t, tmpDir)
	// Tamper with data.txt
	os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("tampered"), 0644)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "hash mismatch for data.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected hash mismatch error, got %v", result.Errors)
	}
}

func TestInspectEpisode_OrphanFile(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisode(t, tmpDir)
	// Add an extra file not in hashes.json
	os.WriteFile(filepath.Join(tmpDir, "orphan.txt"), []byte("orphan"), 0644)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "unhashed episode file: orphan.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected orphan file error, got %v", result.Errors)
	}
}

func TestInspectEpisode_TraceChainBroken(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisode(t, tmpDir)
	// Tamper with trace event hash
	traceData, _ := os.ReadFile(filepath.Join(tmpDir, "trace.jsonl"))
	var event map[string]interface{}
	json.Unmarshal(traceData, &event)
	event["event_hash"] = "tampered"
	eventBytes, _ := json.Marshal(event)
	os.WriteFile(filepath.Join(tmpDir, "trace.jsonl"), append(eventBytes, '\n'), 0644)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "trace chain broken") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected trace chain broken error, got %v", result.Errors)
	}
}

func TestInspectEpisode_ManifestHashMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisode(t, tmpDir)
	// Tamper with manifest.json content but keep old manifest_hash
	manifestData, _ := os.ReadFile(filepath.Join(tmpDir, "manifest.json"))
	var manifest map[string]interface{}
	json.Unmarshal(manifestData, &manifest)
	manifest["task_id"] = "tampered-task"
	manifestOut, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "manifest.json"), manifestOut, 0644)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "manifest_hash mismatch") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected manifest_hash mismatch error, got %v", result.Errors)
	}
}

func TestInspectEpisode_HashesHashMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisode(t, tmpDir)
	// Tamper with hashes.json but keep old hashes_hash in manifest
	hashesData, _ := os.ReadFile(filepath.Join(tmpDir, "hashes.json"))
	var hashes map[string]interface{}
	json.Unmarshal(hashesData, &hashes)
	files := hashes["files"].([]interface{})
	if len(files) > 0 {
		fileMap := files[0].(map[string]interface{})
		fileMap["sha256"] = "sha256:tampered"
	}
	hashesBytes, _ := json.MarshalIndent(hashes, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "hashes.json"), hashesBytes, 0644)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "hashes_hash mismatch") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected hashes_hash mismatch error, got %v", result.Errors)
	}
}

func TestInspectEpisode_InvalidSchema(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisode(t, tmpDir)
	// Make manifest invalid by removing required field
	manifestData, _ := os.ReadFile(filepath.Join(tmpDir, "manifest.json"))
	var manifest map[string]interface{}
	json.Unmarshal(manifestData, &manifest)
	delete(manifest, "task_id")
	// Recompute manifest_hash without task_id
	delete(manifest, "manifest_hash")
	manifestBytes, _ := json.Marshal(manifest)
	manifestHash := sha256Hex(manifestBytes)
	manifest["manifest_hash"] = "sha256:" + manifestHash
	manifestOut, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "manifest.json"), manifestOut, 0644)

	result, err := InspectEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "schema validation") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected schema validation error, got %v", result.Errors)
	}
}
