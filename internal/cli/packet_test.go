package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func makeTestCard(t *testing.T, dir string, taskId string) string {
	t.Helper()
	card := map[string]any{
		"schema_version":    "1",
		"task_id":           taskId,
		"tier":              "light",
		"owner":             "test-owner",
		"accountable":       "test-accountable",
		"claim":             map[string]any{"fix_status": "fixed", "summary": "Test"},
		"verification":      map[string]any{"status": "passed", "checks": []any{}},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "done", "owner": "user"},
	}
	cardPath := filepath.Join(dir, "completion-card.yaml")
	data, err := yaml.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cardPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	return cardPath
}

func TestPacketCreate(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := makeTestCard(t, tmpDir, "TASK-CREATE-001")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "packet created:") {
		t.Fatalf("expected 'packet created:', got:\n%s", out)
	}
	if !strings.Contains(out, "packet_id: packet-") {
		t.Fatalf("expected packet_id, got:\n%s", out)
	}
	if !strings.Contains(out, "task_id: TASK-CREATE-001") {
		t.Fatalf("expected task_id, got:\n%s", out)
	}
	if !strings.Contains(out, "previous_packet_id: null") {
		t.Fatalf("expected previous_packet_id null, got:\n%s", out)
	}
}

func TestPacketCreateChains(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := makeTestCard(t, tmpDir, "TASK-CHAIN-001")

	var stdout1 bytes.Buffer
	var stderr1 bytes.Buffer
	code := Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &stdout1, &stderr1)
	if code != ExitOK {
		t.Fatalf("first create failed: %d. stderr: %s", code, stderr1.String())
	}
	firstId := ""
	for _, line := range strings.Split(stdout1.String(), "\n") {
		if strings.HasPrefix(line, "packet_id: ") {
			firstId = strings.TrimPrefix(line, "packet_id: ")
		}
	}
	if firstId == "" {
		t.Fatal("expected first packet_id")
	}

	var stdout2 bytes.Buffer
	var stderr2 bytes.Buffer
	code = Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &stdout2, &stderr2)
	if code != ExitOK {
		t.Fatalf("second create failed: %d. stderr: %s", code, stderr2.String())
	}
	out2 := stdout2.String()
	if !strings.Contains(out2, "previous_packet_id: "+firstId) {
		t.Fatalf("expected chain to first packet, got:\n%s", out2)
	}
}

func TestPacketCreateCardNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"packet", "create", "--card", filepath.Join(tmpDir, "missing.yaml"), "--packets-dir", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Fatalf("expected not found error, got:\n%s", stderr.String())
	}
}

func TestPacketVerifyChainValid(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := makeTestCard(t, tmpDir, "TASK-VC-001")

	var sb bytes.Buffer
	var se bytes.Buffer
	code := Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("first create failed")
	}
	code = Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("second create failed")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = Run([]string{"packet", "verify-chain", "--task-id", "TASK-VC-001", "--packets-dir", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "chain valid:") {
		t.Fatalf("expected chain valid, got:\n%s", out)
	}
	if !strings.Contains(out, "2 packet(s)") {
		t.Fatalf("expected 2 packets, got:\n%s", out)
	}
}

func TestPacketVerifyChainTampered(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := makeTestCard(t, tmpDir, "TASK-TAMPER-001")

	var sb bytes.Buffer
	var se bytes.Buffer
	code := Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("create failed")
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	var packetFile string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "packet-") && strings.HasSuffix(e.Name(), ".yaml") {
			packetFile = filepath.Join(tmpDir, e.Name())
			break
		}
	}
	if packetFile == "" {
		t.Fatal("expected packet file")
	}

	data, err := os.ReadFile(packetFile)
	if err != nil {
		t.Fatal(err)
	}
	var packet map[string]any
	if err := yaml.Unmarshal(data, &packet); err != nil {
		t.Fatal(err)
	}
	packet["payload_hash"] = "tampered"
	outData, err := yaml.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packetFile, outData, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = Run([]string{"packet", "verify-chain", "--task-id", "TASK-TAMPER-001", "--packets-dir", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "chain broken:") {
		t.Fatalf("expected chain broken, got:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "payload hash mismatch") {
		t.Fatalf("expected payload hash mismatch, got:\n%s", stderr.String())
	}
}

func TestPacketVerifyChainTaskNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := makeTestCard(t, tmpDir, "TASK-NF-001")

	var sb bytes.Buffer
	var se bytes.Buffer
	code := Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("create failed")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = Run([]string{"packet", "verify-chain", "--task-id", "UNKNOWN", "--packets-dir", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "No packets found") {
		t.Fatalf("expected No packets found, got:\n%s", stderr.String())
	}
}

func TestPacketVerifyChainMissingTaskID(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"packet", "verify-chain", "--packets-dir", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--task-id is required") {
		t.Fatalf("expected missing task-id error, got:\n%s", stderr.String())
	}
}

func TestPacketUnsupportedSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"packet", "unsupported"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown packet subcommand") {
		t.Fatalf("expected unknown subcommand error, got:\n%s", stderr.String())
	}
}

func TestPacketCreateUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"packet", "create", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got:\n%s", stderr.String())
	}
}

func TestPacketCreateMissingFlagValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"packet", "create", "--card"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--card requires a value") {
		t.Fatalf("expected missing value error, got:\n%s", stderr.String())
	}
}

func TestPacketVerifyChainUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"packet", "verify-chain", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got:\n%s", stderr.String())
	}
}

func TestPacketVerifyChainMissingParent(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := makeTestCard(t, tmpDir, "TASK-MP-001")

	var sb bytes.Buffer
	var se bytes.Buffer
	code := Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("create failed")
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	var packetFile string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "packet-") && strings.HasSuffix(e.Name(), ".yaml") {
			packetFile = filepath.Join(tmpDir, e.Name())
			break
		}
	}
	if packetFile == "" {
		t.Fatal("expected packet file")
	}

	data, err := os.ReadFile(packetFile)
	if err != nil {
		t.Fatal(err)
	}
	var packet map[string]any
	if err := yaml.Unmarshal(data, &packet); err != nil {
		t.Fatal(err)
	}
	packet["previous_packet_id"] = "nonexistent-parent"
	outData, err := yaml.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packetFile, outData, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = Run([]string{"packet", "verify-chain", "--task-id", "TASK-MP-001", "--packets-dir", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "missing parent") {
		t.Fatalf("expected missing parent error, got:\n%s", stderr.String())
	}
}

func TestPacketVerifyChainCycle(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := makeTestCard(t, tmpDir, "TASK-CYCLE-001")

	var sb bytes.Buffer
	var se bytes.Buffer
	code := Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("first create failed")
	}
	code = Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("second create failed")
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	var secondFile string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "packet-") && strings.HasSuffix(e.Name(), ".yaml") {
			if secondFile != "" {
				secondFile = filepath.Join(tmpDir, e.Name())
			} else {
				secondFile = filepath.Join(tmpDir, e.Name())
			}
		}
	}
	if secondFile == "" {
		t.Fatal("expected packet files")
	}

	// Find the later file by mtime or just rewrite both and target the one with non-null previous_packet_id
	var targetFile string
	for _, e := range entries {
		fpath := filepath.Join(tmpDir, e.Name())
		data, err := os.ReadFile(fpath)
		if err != nil {
			continue
		}
		var p map[string]any
		if err := yaml.Unmarshal(data, &p); err != nil {
			continue
		}
		if p["previous_packet_id"] != nil {
			targetFile = fpath
			break
		}
	}
	if targetFile == "" {
		t.Fatal("expected packet with previous_packet_id")
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err := yaml.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	p["previous_packet_id"] = p["packet_id"]
	outData, err := yaml.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetFile, outData, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = Run([]string{"packet", "verify-chain", "--task-id", "TASK-CYCLE-001", "--packets-dir", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "cycle") {
		t.Fatalf("expected cycle error, got:\n%s", stderr.String())
	}
}

func TestPacketVerifyChainFork(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := makeTestCard(t, tmpDir, "TASK-FORK-001")

	var sb bytes.Buffer
	var se bytes.Buffer
	code := Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("first create failed")
	}
	code = Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("second create failed")
	}
	code = Run([]string{"packet", "create", "--card", cardPath, "--packets-dir", tmpDir}, &sb, &se)
	if code != ExitOK {
		t.Fatal("third create failed")
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	var firstId string
	var thirdFile string
	for _, e := range entries {
		fpath := filepath.Join(tmpDir, e.Name())
		data, err := os.ReadFile(fpath)
		if err != nil {
			continue
		}
		var p map[string]any
		if err := yaml.Unmarshal(data, &p); err != nil {
			continue
		}
		if firstId == "" {
			firstId, _ = p["packet_id"].(string)
		}
		if p["previous_packet_id"] != nil {
			thirdFile = fpath
		}
	}
	if firstId == "" || thirdFile == "" {
		t.Fatal("expected first and third packets")
	}

	data, err := os.ReadFile(thirdFile)
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err := yaml.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	p["previous_packet_id"] = firstId
	outData, err := yaml.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(thirdFile, outData, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = Run([]string{"packet", "verify-chain", "--task-id", "TASK-FORK-001", "--packets-dir", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "fork") {
		t.Fatalf("expected fork error, got:\n%s", stderr.String())
	}
}

func TestPacketHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"packet", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}
