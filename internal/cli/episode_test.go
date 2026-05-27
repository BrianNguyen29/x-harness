package cli

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

func computeTraceHashForTest(event map[string]interface{}) string {
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

func createValidEpisodeForCLITest(t *testing.T, dir string) {
	t.Helper()

	dataContent := []byte("hello world")
	dataPath := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(dataPath, dataContent, 0644); err != nil {
		t.Fatal(err)
	}
	dataHash := sha256Hex(dataContent)

	event := map[string]interface{}{
		"event_id":   "E1",
		"event_type": "verify_completed",
		"outcome":    "success",
	}
	event["event_hash"] = computeTraceHashForTest(event)
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
		"hashes_hash":         "sha256:" + hashesHash,
	}
	manifestBytes, _ := json.Marshal(manifest)
	manifestHash := sha256Hex(manifestBytes)
	manifest["manifest_hash"] = "sha256:" + manifestHash
	manifestOut, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifestOut, 0644); err != nil {
		t.Fatal(err)
	}
}

func createTarballForCLITest(t *testing.T, sourceDir, destPath string) {
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

func TestEpisodeInspectValidDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisodeForCLITest(t, tmpDir)

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"episode", "inspect", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# x-harness Episode Inspect") {
		t.Fatalf("expected header, got:\n%s", out)
	}
	if !strings.Contains(out, "- ok: true") {
		t.Fatalf("expected ok=true, got:\n%s", out)
	}
	if !strings.Contains(out, "- episode_id: ep_") {
		t.Fatalf("expected episode_id, got:\n%s", out)
	}
	if !strings.Contains(out, "- task_id: test-task") {
		t.Fatalf("expected task_id, got:\n%s", out)
	}
}

func TestEpisodeInspectValidDirectoryJSON(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisodeForCLITest(t, tmpDir)

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"episode", "inspect", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v. output:\n%s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true in JSON, got %v", result["ok"])
	}
	if result["episode_id"] != "ep_2024-01-01T00-00-00Z_test-task" {
		t.Fatalf("unexpected episode_id: %v", result["episode_id"])
	}
	if result["task_id"] != "test-task" {
		t.Fatalf("unexpected task_id: %v", result["task_id"])
	}
	if result["file_count"] != float64(2) {
		t.Fatalf("unexpected file_count: %v", result["file_count"])
	}
}

func TestEpisodeInspectValidTarball(t *testing.T) {
	sourceDir := t.TempDir()
	createValidEpisodeForCLITest(t, sourceDir)

	tarPath := filepath.Join(t.TempDir(), "episode.tar.gz")
	createTarballForCLITest(t, sourceDir, tarPath)

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"episode", "inspect", tarPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "- ok: true") {
		t.Fatalf("expected ok=true, got:\n%s", stdout.String())
	}
}

func TestEpisodeInspectMissingPath(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"episode", "inspect"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "requires a path argument") {
		t.Fatalf("expected path argument error, got:\n%s", stderr.String())
	}
}

func TestEpisodeInspectMissingManifest(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "hashes.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "trace.jsonl"), []byte("{}\n"), 0644)

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"episode", "inspect", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "manifest.json not found") {
		t.Fatalf("expected manifest.json not found, got:\n%s", stdout.String())
	}
}

func TestEpisodeInspectHashMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	createValidEpisodeForCLITest(t, tmpDir)
	os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("tampered"), 0644)

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"episode", "inspect", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "hash mismatch for data.txt") {
		t.Fatalf("expected hash mismatch error, got:\n%s", stdout.String())
	}
}

func TestEpisodeInspectUnknownFlag(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"episode", "inspect", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got:\n%s", stderr.String())
	}
}

func TestEpisodeInspectUnknownSubcommand(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"episode", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown episode subcommand") {
		t.Fatalf("expected unknown subcommand error, got:\n%s", stderr.String())
	}
}

func TestEpisodeInspectMissingSubcommand(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"episode"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got:\n%s", stderr.String())
	}
}
