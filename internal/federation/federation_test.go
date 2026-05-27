package federation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeValidEvidenceEntry() map[string]any {
	return map[string]any{
		"schema_version":      "1",
		"task_id":             "task-001",
		"evidence_id":         "ev-001",
		"layer":               "raw",
		"kind":                "other",
		"path":                "test.txt",
		"sha256":              "0000000000000000000000000000000000000000000000000000000000000000",
		"size_bytes":          0,
		"redacted":            false,
		"created_at":          "2024-01-01T00:00:00Z",
		"admission_authority": false,
	}
}

func writeEvidenceIndex(t *testing.T, dir string, entries []map[string]any) string {
	t.Helper()
	path := filepath.Join(dir, "evidence", "index.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, e := range entries {
		b, _ := json.Marshal(e)
		lines = append(lines, string(b))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writePolicy(t *testing.T, dir string, content string) {
	t.Helper()
	policyDir := filepath.Join(dir, "policies")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "federation.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFederationPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: true
  require_redaction: true
  tenant_boundary: required
  retention_days: 30
  data_sent:
    - anonymized_failure_predicates
  data_never_sent:
    - raw_source_code
  import:
    default_dry_run: true
    affects_admission: false
`)
	policy, err := LoadFederationPolicy(tmpDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Version != 1 {
		t.Fatalf("expected version 1, got %d", policy.Version)
	}
	if !policy.Federation.RequireOptIn {
		t.Fatal("expected require_opt_in")
	}
}

func TestBuildFederationPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: true
  require_redaction: true
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "test-failure"
	entry["metadata"] = map[string]any{
		"admission_outcome": "failed",
		"acceptance_status": "withheld",
	}
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	result, err := BuildFederationPatterns(BuildInput{
		Root:      tmpDir,
		IndexPath: "evidence/index.jsonl",
		Tenant:    "tenant-a",
		Source:    "local",
		Now:       "2024-06-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(result.Patterns))
	}
	p := result.Patterns[0]
	if p.SchemaVersion != "1" {
		t.Fatalf("expected schema_version 1, got %s", p.SchemaVersion)
	}
	if p.PatternClass != "failure" {
		t.Fatalf("expected failure class, got %s", p.PatternClass)
	}
	if p.TenantHash != scopedHash("tenant-a", "tenant") {
		t.Fatalf("unexpected tenant hash")
	}
	if p.Signal.PredicateHash == nil || *p.Signal.PredicateHash != scopedHash("tenant-a", "test-failure") {
		t.Fatalf("unexpected predicate hash")
	}
	if p.Signal.EvidenceLayer != "raw" {
		t.Fatalf("expected layer raw, got %s", p.Signal.EvidenceLayer)
	}
	if p.EvidenceKind != "other" {
		t.Fatalf("expected kind other, got %s", p.EvidenceKind)
	}
	if len(p.ComponentHashes) == 0 {
		t.Fatal("expected component hashes")
	}
	if p.Redaction.Mode != "anonymized-pattern" {
		t.Fatalf("unexpected redaction mode: %s", p.Redaction.Mode)
	}
	if p.AdmissionAuthority != false {
		t.Fatal("expected admission_authority false")
	}
}

func TestBuildFederationPatternsObservationClass(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: false
  require_redaction: false
  tenant_boundary: required
  retention_days: 7
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "success-predicate"
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	result, err := BuildFederationPatterns(BuildInput{
		Root:      tmpDir,
		IndexPath: "evidence/index.jsonl",
		Tenant:    "tenant-b",
		Source:    "src",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(result.Patterns))
	}
	if result.Patterns[0].PatternClass != "observation" {
		t.Fatalf("expected observation class, got %s", result.Patterns[0].PatternClass)
	}
}

func TestExportFederationPatternsMissingOptIn(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: true
  require_redaction: true
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "fail"
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	_, err := ExportFederationPatterns(tmpDir, "evidence/index.jsonl", "out.jsonl", "tenant", "local", false, true, "", "")
	if err == nil {
		t.Fatal("expected error for missing opt-in")
	}
	if !strings.Contains(err.Error(), "--opt-in") {
		t.Fatalf("expected opt-in error, got: %v", err)
	}
}

func TestExportFederationPatternsMissingRedacted(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: true
  require_redaction: true
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "fail"
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	_, err := ExportFederationPatterns(tmpDir, "evidence/index.jsonl", "out.jsonl", "tenant", "local", true, false, "", "")
	if err == nil {
		t.Fatal("expected error for missing redacted")
	}
	if !strings.Contains(err.Error(), "--redacted") {
		t.Fatalf("expected redacted error, got: %v", err)
	}
}

func TestExportFederationPatternsSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: false
  require_redaction: false
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "fail"
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	result, err := ExportFederationPatterns(tmpDir, "evidence/index.jsonl", "out.jsonl", "tenant", "local", false, false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatal("expected ok")
	}
	if result.RecordCount != 1 {
		t.Fatalf("expected 1 record, got %d", result.RecordCount)
	}
	if _, err := os.Stat(result.OutPath); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}

func TestValidateFederationPatternFileValid(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: false
  require_redaction: false
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "fail"
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	exportResult, err := ExportFederationPatterns(tmpDir, "evidence/index.jsonl", "out.jsonl", "tenant", "local", false, false, "", "")
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	validation, err := ValidateFederationPatternFile(exportResult.OutPath)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if !validation.OK {
		t.Fatalf("expected valid patterns, got errors: %v", validation.Errors)
	}
	if len(validation.Patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(validation.Patterns))
	}
}

func TestValidateFederationPatternFileInvalidRecord(t *testing.T) {
	tmpDir := t.TempDir()
	badPattern := map[string]any{
		"schema_version": "1",
		"pattern_id":     "bad-id",
	}
	b, _ := json.Marshal(badPattern)
	path := filepath.Join(tmpDir, "bad.jsonl")
	if err := os.WriteFile(path, []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	validation, err := ValidateFederationPatternFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validation.OK {
		t.Fatal("expected validation to fail")
	}
	if len(validation.Errors) == 0 {
		t.Fatal("expected errors")
	}
}

func TestValidateFederationPatternFileSecretScan(t *testing.T) {
	tmpDir := t.TempDir()
	// pattern with secret-like content in a field that gets stringified
	pattern := FederationPattern{
		SchemaVersion:    "1",
		PatternID:        strings.Repeat("0", 64),
		TenantHash:       strings.Repeat("0", 64),
		SourceHash:       strings.Repeat("0", 64),
		PatternClass:     "failure",
		Signal:           Signal{PredicatePresent: false, EvidenceLayer: "raw"},
		EvidenceKind:     "other",
		ComponentHashes:  []string{},
		BenchmarkMetrics: nil,
		CreatedAt:        "2024-01-01T00:00:00Z",
		RetentionExpiresAt: "2024-02-01T00:00:00Z",
		Redaction: Redaction{
			Mode:                   "anonymized-pattern",
			RedactedRequired:       true,
			RawContentIncluded:     false,
			SecretScanReplacements: 0,
		},
		AdmissionAuthority: false,
	}
	// Inject a secret into a field
	pattern.PatternID = "ghp_" + strings.Repeat("a", 36)
	b, _ := json.Marshal(pattern)
	path := filepath.Join(tmpDir, "secret.jsonl")
	if err := os.WriteFile(path, []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	validation, err := ValidateFederationPatternFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validation.OK {
		t.Fatal("expected validation to fail due to secret scan")
	}
	foundSecret := false
	for _, e := range validation.Errors {
		if strings.Contains(e, "secret-like value detected") {
			foundSecret = true
			break
		}
	}
	if !foundSecret {
		t.Fatalf("expected secret scan error, got: %v", validation.Errors)
	}
}

func TestImportFederationPatternsDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: false
  require_redaction: false
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "fail"
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	exportResult, err := ExportFederationPatterns(tmpDir, "evidence/index.jsonl", "out.jsonl", "tenant", "local", false, false, "", "")
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	result, err := ImportFederationPatterns(tmpDir, exportResult.OutPath, ".x-harness/federation/imported.jsonl", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok, got errors: %v", result.Errors)
	}
	if !result.DryRun {
		t.Fatal("expected dry-run")
	}
	if result.PlannedCount != 1 {
		t.Fatalf("expected planned_count 1, got %d", result.PlannedCount)
	}
}

func TestImportFederationPatternsWrite(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: false
  require_redaction: false
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "fail"
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	exportResult, err := ExportFederationPatterns(tmpDir, "evidence/index.jsonl", "out.jsonl", "tenant", "local", false, false, "", "")
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	target := filepath.Join(tmpDir, ".x-harness", "federation", "imported.jsonl")
	result, err := ImportFederationPatterns(tmpDir, exportResult.OutPath, target, false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok, got errors: %v", result.Errors)
	}
	if result.WrittenCount != 1 {
		t.Fatalf("expected written_count 1, got %d", result.WrittenCount)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected target file to exist: %v", err)
	}
}

func TestImportFederationPatternsConflict(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: false
  require_redaction: false
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "fail"
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	exportResult, err := ExportFederationPatterns(tmpDir, "evidence/index.jsonl", "out.jsonl", "tenant", "local", false, false, "", "")
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	target := filepath.Join(tmpDir, ".x-harness", "federation", "imported.jsonl")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ImportFederationPatterns(tmpDir, exportResult.OutPath, target, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected conflict error")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "target exists") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected target exists error, got: %v", result.Errors)
	}
}

func TestImportFederationPatternsMerge(t *testing.T) {
	tmpDir := t.TempDir()
	writePolicy(t, tmpDir, `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: false
  require_redaction: false
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`)
	entry := makeValidEvidenceEntry()
	entry["predicate"] = "fail"
	writeEvidenceIndex(t, tmpDir, []map[string]any{entry})

	exportResult, err := ExportFederationPatterns(tmpDir, "evidence/index.jsonl", "out.jsonl", "tenant", "local", false, false, "", "")
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	target := filepath.Join(tmpDir, ".x-harness", "federation", "imported.jsonl")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	// Write an existing pattern with a different pattern_id
	existing := FederationPattern{
		SchemaVersion:      "1",
		PatternID:          strings.Repeat("a", 64),
		TenantHash:         strings.Repeat("0", 64),
		SourceHash:         strings.Repeat("0", 64),
		PatternClass:       "observation",
		Signal:             Signal{PredicatePresent: false, EvidenceLayer: "raw"},
		EvidenceKind:       "other",
		ComponentHashes:    []string{},
		CreatedAt:          "2024-01-01T00:00:00Z",
		RetentionExpiresAt: "2024-02-01T00:00:00Z",
		Redaction: Redaction{
			Mode:                   "anonymized-pattern",
			RedactedRequired:       true,
			RawContentIncluded:     false,
			SecretScanReplacements: 0,
		},
		AdmissionAuthority: false,
	}
	b, _ := json.Marshal(existing)
	if err := os.WriteFile(target, []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ImportFederationPatterns(tmpDir, exportResult.OutPath, target, false, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok, got errors: %v", result.Errors)
	}
	if result.WrittenCount != 2 {
		t.Fatalf("expected written_count 2 after merge, got %d", result.WrittenCount)
	}
}

func TestReadFederationPatternsJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	p := FederationPattern{
		SchemaVersion:      "1",
		PatternID:          strings.Repeat("0", 64),
		TenantHash:         strings.Repeat("0", 64),
		SourceHash:         strings.Repeat("0", 64),
		PatternClass:       "failure",
		Signal:             Signal{PredicatePresent: false, EvidenceLayer: "raw"},
		EvidenceKind:       "other",
		ComponentHashes:    []string{},
		CreatedAt:          "2024-01-01T00:00:00Z",
		RetentionExpiresAt: "2024-02-01T00:00:00Z",
		Redaction: Redaction{
			Mode:                   "anonymized-pattern",
			RedactedRequired:       true,
			RawContentIncluded:     false,
			SecretScanReplacements: 0,
		},
		AdmissionAuthority: false,
	}
	b, _ := json.Marshal(p)
	path := filepath.Join(tmpDir, "patterns.jsonl")
	if err := os.WriteFile(path, []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	patterns, err := ReadFederationPatterns(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
}

func TestReadFederationPatternsEnvelope(t *testing.T) {
	tmpDir := t.TempDir()
	p := FederationPattern{
		SchemaVersion:      "1",
		PatternID:          strings.Repeat("0", 64),
		TenantHash:         strings.Repeat("0", 64),
		SourceHash:         strings.Repeat("0", 64),
		PatternClass:       "failure",
		Signal:             Signal{PredicatePresent: false, EvidenceLayer: "raw"},
		EvidenceKind:       "other",
		ComponentHashes:    []string{},
		CreatedAt:          "2024-01-01T00:00:00Z",
		RetentionExpiresAt: "2024-02-01T00:00:00Z",
		Redaction: Redaction{
			Mode:                   "anonymized-pattern",
			RedactedRequired:       true,
			RawContentIncluded:     false,
			SecretScanReplacements: 0,
		},
		AdmissionAuthority: false,
	}
	envelope := map[string]any{
		"schema_version": "1",
		"created_at":     "2024-01-01T00:00:00Z",
		"record_count":   1,
		"patterns":       []FederationPattern{p},
		"admission_authority": false,
	}
	b, _ := json.Marshal(envelope)
	path := filepath.Join(tmpDir, "patterns.json")
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}
	patterns, err := ReadFederationPatterns(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
}

func TestScopedHashDeterministic(t *testing.T) {
	h1 := scopedHash("tenant-a", "value-x")
	h2 := scopedHash("tenant-a", "value-x")
	if h1 != h2 {
		t.Fatal("expected deterministic hash")
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64 char hex, got %d", len(h1))
	}
}
