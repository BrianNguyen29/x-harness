package federation

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/evidence"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
)

var (
	admissionOutcomes = map[string]bool{
		"success": true, "failed": true, "blocked": true,
		"skipped": true, "timeout": true, "error": true,
	}
	acceptanceStatuses = map[string]bool{
		"accepted": true, "withheld": true,
	}
)

// FederationPolicy mirrors the TypeScript FederationPolicy interface.
type FederationPolicy struct {
	Version    int `yaml:"version"`
	Federation struct {
		Enabled          bool     `yaml:"enabled"`
		DefaultEnabled   bool     `yaml:"default_enabled"`
		RequireOptIn     bool     `yaml:"require_opt_in"`
		RequireRedaction bool     `yaml:"require_redaction"`
		TenantBoundary   string   `yaml:"tenant_boundary"`
		RetentionDays    int      `yaml:"retention_days"`
		DataSent         []string `yaml:"data_sent"`
		DataNeverSent    []string `yaml:"data_never_sent"`
		Import           struct {
			DefaultDryRun    bool `yaml:"default_dry_run"`
			AffectsAdmission bool `yaml:"affects_admission"`
		} `yaml:"import"`
	} `yaml:"federation"`
}

// BenchmarkMetrics holds optional benchmark values.
type BenchmarkMetrics struct {
	FalseAcceptCount            *int     `json:"false_accept_count,omitempty"`
	AdversarialFalseAcceptCount *int     `json:"adversarial_false_accept_count,omitempty"`
	FalseRejectCount            *int     `json:"false_reject_count,omitempty"`
	RuntimeMs                   *float64 `json:"runtime_ms,omitempty"`
}

// Signal mirrors the signal object in the pattern schema.
type Signal struct {
	PredicateHash    *string `json:"predicate_hash"`
	PredicatePresent bool    `json:"predicate_present"`
	AdmissionOutcome *string `json:"admission_outcome,omitempty"`
	AcceptanceStatus *string `json:"acceptance_status,omitempty"`
	EvidenceLayer    string  `json:"evidence_layer"`
}

// Redaction mirrors the redaction object in the pattern schema.
type Redaction struct {
	Mode                   string `json:"mode"`
	RedactedRequired       bool   `json:"redacted_required"`
	RawContentIncluded     bool   `json:"raw_content_included"`
	SecretScanReplacements int    `json:"secret_scan_replacements"`
}

// FederationPattern mirrors the TypeScript FederationPattern interface.
type FederationPattern struct {
	SchemaVersion        string           `json:"schema_version"`
	PatternID            string           `json:"pattern_id"`
	TenantHash           string           `json:"tenant_hash"`
	SourceHash           string           `json:"source_hash"`
	PatternClass         string           `json:"pattern_class"`
	Signal               Signal           `json:"signal"`
	EvidenceKind         string           `json:"evidence_kind"`
	ComponentHashes      []string         `json:"component_hashes"`
	BenchmarkMetrics     *BenchmarkMetrics `json:"benchmark_metrics"`
	CreatedAt            string           `json:"created_at"`
	RetentionExpiresAt   string           `json:"retention_expires_at"`
	Redaction            Redaction        `json:"redaction"`
	AdmissionAuthority   bool             `json:"admission_authority"`
}

// FederationExportResult mirrors the TypeScript export result.
type FederationExportResult struct {
	OK                bool   `json:"ok"`
	OutPath           string `json:"out_path"`
	RecordCount       int    `json:"record_count"`
	PolicyEnabled     bool   `json:"policy_enabled"`
	OptIn             bool   `json:"opt_in"`
	Redacted          bool   `json:"redacted"`
	TenantHash        string `json:"tenant_hash"`
	SourceHash        string `json:"source_hash"`
	AdmissionAuthority bool  `json:"admission_authority"`
}

// FederationImportResult mirrors the TypeScript import result.
type FederationImportResult struct {
	OK                bool     `json:"ok"`
	DryRun            bool     `json:"dry_run"`
	Target            string   `json:"target"`
	PlannedCount      int      `json:"planned_count"`
	WrittenCount      int      `json:"written_count"`
	Errors            []string `json:"errors"`
	AdmissionAuthority bool    `json:"admission_authority"`
}

// ValidationResult is returned by ValidateFederationPatternFile.
type ValidationResult struct {
	OK       bool                `json:"ok"`
	Patterns []FederationPattern `json:"patterns"`
	Errors   []string            `json:"errors"`
}

func resolvePath(root, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, p)
}

// LoadFederationPolicy loads the federation policy from the given root.
func LoadFederationPolicy(root, policyPath string) (*FederationPolicy, error) {
	resolved := resolvePath(root, policyPath)
	if policyPath == "" {
		resolved = filepath.Join(root, "policies", "federation.yaml")
	}
	var policy FederationPolicy
	if err := loader.LoadDocument(resolved, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// ReadFederationPatterns reads patterns from a JSONL file or JSON envelope.
func ReadFederationPatterns(filePath string) ([]FederationPattern, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return []FederationPattern{}, nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var envelope map[string]any
		if err := json.Unmarshal(data, &envelope); err == nil {
			if patternsRaw, ok := envelope["patterns"]; ok {
				if patternsArr, ok := patternsRaw.([]any); ok {
					patterns := make([]FederationPattern, 0, len(patternsArr))
					for _, p := range patternsArr {
						b, _ := json.Marshal(p)
						var fp FederationPattern
						if err := json.Unmarshal(b, &fp); err != nil {
							return nil, err
						}
						patterns = append(patterns, fp)
					}
					return patterns, nil
				}
			}
			var fp FederationPattern
			if err := json.Unmarshal(data, &fp); err != nil {
				return nil, err
			}
			return []FederationPattern{fp}, nil
		}
	}
	patterns := []FederationPattern{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var fp FederationPattern
		if err := json.Unmarshal([]byte(line), &fp); err != nil {
			return nil, err
		}
		patterns = append(patterns, fp)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return patterns, nil
}

// ValidateFederationPatternFile validates patterns against the schema and performs secret scanning.
func ValidateFederationPatternFile(filePath string) (*ValidationResult, error) {
	patterns, err := ReadFederationPatterns(filePath)
	if err != nil {
		return &ValidationResult{OK: false, Patterns: []FederationPattern{}, Errors: []string{err.Error()}}, nil
	}
	root, err := repo.FindRoot("")
	if err != nil {
		return &ValidationResult{OK: false, Patterns: patterns, Errors: []string{fmt.Sprintf("cannot find repository root: %v", err)}}, nil
	}
	schemaPath := assets.NewLocator(root).Schema("federation-pattern.schema.json")
	validator, err := schema.Compile(schemaPath)
	if err != nil {
		return &ValidationResult{OK: false, Patterns: patterns, Errors: []string{fmt.Sprintf("cannot compile schema: %v", err)}}, nil
	}
	var errors []string
	for _, pattern := range patterns {
		m := structToMap(pattern)
		if err := validator.Validate(m); err != nil {
			errors = append(errors, fmt.Sprintf("%v", err))
		}
		_, _, replacements := evidence.RedactText(stableStringify(pattern))
		if replacements > 0 {
			errors = append(errors, fmt.Sprintf("secret-like value detected in %s", pattern.PatternID))
		}
	}
	return &ValidationResult{OK: len(errors) == 0, Patterns: patterns, Errors: errors}, nil
}

// BuildInput configures pattern building.
type BuildInput struct {
	Root                string
	IndexPath           string
	Tenant              string
	Source              string
	BenchmarkReportPath string
	PolicyPath          string
	Now                 string
}

// BuildResult holds built patterns and hashes.
type BuildResult struct {
	Patterns   []FederationPattern
	Policy     *FederationPolicy
	TenantHash string
	SourceHash string
}

// BuildFederationPatterns builds patterns from an evidence index.
func BuildFederationPatterns(input BuildInput) (*BuildResult, error) {
	root, err := filepath.Abs(input.Root)
	if err != nil {
		return nil, err
	}
	policy, err := LoadFederationPolicy(root, input.PolicyPath)
	if err != nil {
		return nil, err
	}

	entries, err := evidence.ReadIndex(resolvePath(root, input.IndexPath))
	if err != nil {
		return nil, err
	}
	ok, errs := evidence.ValidateEntries(entries)
	if !ok {
		return nil, fmt.Errorf("evidence index validation failed: %s", strings.Join(errs, "; "))
	}

	tenantHash := scopedHash(input.Tenant, "tenant")
	sourceHash := scopedHash(input.Tenant, input.Source)
	createdAt := input.Now
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	benchmarkMetrics := loadBenchmarkMetrics(input.BenchmarkReportPath)

	candidates := filterCandidates(entries)

	patterns := make([]FederationPattern, 0, len(candidates))
	for _, entry := range candidates {
		outcome, _ := getCanonicalSignalMetadata(entry, "admission_outcome")
		acceptance, _ := getCanonicalSignalMetadata(entry, "acceptance_status")
		predicate, _ := entry["predicate"].(string)

		patternClass := "observation"
		if isFailureSignal(predicate, outcome, acceptance) {
			patternClass = "failure"
		}

		sig := Signal{
			PredicateHash:    ptrOrNil(scopedHash(input.Tenant, predicate)),
			PredicatePresent: predicate != "",
			EvidenceLayer:    stringField(entry, "layer"),
		}
		if outcome != "" {
			sig.AdmissionOutcome = &outcome
		}
		if acceptance != "" {
			sig.AcceptanceStatus = &acceptance
		}
		if predicate == "" {
			sig.PredicateHash = nil
		}

		basePattern := FederationPattern{
			SchemaVersion:      "1",
			PatternID:          scopedHash(input.Tenant, stableStringify(map[string]any{
				"evidence_id": stringField(entry, "evidence_id"),
				"kind":        stringField(entry, "kind"),
				"predicate":   predicate,
				"outcome":     outcome,
				"acceptance":  acceptance,
			})),
			TenantHash:         tenantHash,
			SourceHash:         sourceHash,
			PatternClass:       patternClass,
			Signal:             sig,
			EvidenceKind:       stringField(entry, "kind"),
			ComponentHashes:    hashStrings(componentHints(entry), input.Tenant),
			BenchmarkMetrics:   benchmarkMetrics,
			CreatedAt:          createdAt,
			RetentionExpiresAt: retentionExpiry(createdAt, policy.Federation.RetentionDays),
			AdmissionAuthority: false,
		}

		_, _, replacements := evidence.RedactText(stableStringify(basePattern))
		basePattern.Redaction = Redaction{
			Mode:                   "anonymized-pattern",
			RedactedRequired:       true,
			RawContentIncluded:     false,
			SecretScanReplacements: replacements,
		}
		patterns = append(patterns, basePattern)
	}

	if err := validatePatterns(patterns); err != nil {
		return nil, err
	}

	return &BuildResult{
		Patterns:   patterns,
		Policy:     policy,
		TenantHash: tenantHash,
		SourceHash: sourceHash,
	}, nil
}

// ExportFederationPatterns exports patterns after enforcing opt-in/redacted policy.
func ExportFederationPatterns(root, indexPath, outPath, tenant, source string, optIn, redacted bool, benchmarkReportPath, policyPath string) (*FederationExportResult, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	policy, err := LoadFederationPolicy(absRoot, policyPath)
	if err != nil {
		return nil, err
	}
	if policy.Federation.RequireOptIn && !optIn {
		return nil, fmt.Errorf("federation export requires explicit --opt-in")
	}
	if policy.Federation.RequireRedaction && !redacted {
		return nil, fmt.Errorf("federation export requires --redacted")
	}
	if strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("federation export requires a non-empty --tenant")
	}

	result, err := BuildFederationPatterns(BuildInput{
		Root:                absRoot,
		IndexPath:           indexPath,
		Tenant:              tenant,
		Source:              source,
		BenchmarkReportPath: benchmarkReportPath,
		PolicyPath:          policyPath,
	})
	if err != nil {
		return nil, err
	}

	absOut := resolvePath(absRoot, outPath)
	if err := os.MkdirAll(filepath.Dir(absOut), 0755); err != nil {
		return nil, err
	}
	if err := writeJsonl(absOut, result.Patterns); err != nil {
		return nil, err
	}

	return &FederationExportResult{
		OK:                 true,
		OutPath:            absOut,
		RecordCount:        len(result.Patterns),
		PolicyEnabled:      policy.Federation.Enabled,
		OptIn:              true,
		Redacted:           true,
		TenantHash:         result.TenantHash,
		SourceHash:         result.SourceHash,
		AdmissionAuthority: false,
	}, nil
}

// ImportFederationPatterns validates and optionally stores patterns.
func ImportFederationPatterns(root, patternsPath, targetPath string, dryRun, merge, force bool) (*FederationImportResult, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	target := resolvePath(absRoot, targetPath)

	rootPrefix := absRoot + string(filepath.Separator)
	if target != absRoot && !strings.HasPrefix(target, rootPrefix) {
		return &FederationImportResult{
			OK: false, DryRun: dryRun, Target: target,
			PlannedCount: 0, WrittenCount: 0,
			Errors: []string{"federation import target must stay inside --root"},
			AdmissionAuthority: false,
		}, nil
	}

	validation, err := ValidateFederationPatternFile(patternsPath)
	if err != nil {
		return &FederationImportResult{
			OK: false, DryRun: dryRun, Target: target,
			PlannedCount: 0, WrittenCount: 0,
			Errors: []string{err.Error()},
			AdmissionAuthority: false,
		}, nil
	}
	if !validation.OK {
		return &FederationImportResult{
			OK: false, DryRun: dryRun, Target: target,
			PlannedCount: 0, WrittenCount: 0,
			Errors: validation.Errors,
			AdmissionAuthority: false,
		}, nil
	}

	if dryRun {
		return &FederationImportResult{
			OK: true, DryRun: true, Target: target,
			PlannedCount: len(validation.Patterns), WrittenCount: 0,
			Errors: []string{},
			AdmissionAuthority: false,
		}, nil
	}

	if _, err := os.Stat(target); err == nil && !merge && !force {
		return &FederationImportResult{
			OK: false, DryRun: false, Target: target,
			PlannedCount: len(validation.Patterns), WrittenCount: 0,
			Errors: []string{"target exists; use --merge or --force"},
			AdmissionAuthority: false,
		}, nil
	}

	patterns := validation.Patterns
	if merge {
		if _, err := os.Stat(target); err == nil {
			existing, err := ReadFederationPatterns(target)
			if err != nil {
				return &FederationImportResult{
					OK: false, DryRun: false, Target: target,
					PlannedCount: len(validation.Patterns), WrittenCount: 0,
					Errors: []string{err.Error()},
					AdmissionAuthority: false,
				}, nil
			}
			byID := make(map[string]FederationPattern, len(existing))
			for _, p := range existing {
				byID[p.PatternID] = p
			}
			for _, p := range validation.Patterns {
				byID[p.PatternID] = p
			}
			patterns = make([]FederationPattern, 0, len(byID))
			for _, p := range byID {
				patterns = append(patterns, p)
			}
			sort.Slice(patterns, func(i, j int) bool {
				return patterns[i].PatternID < patterns[j].PatternID
			})
		}
	}

	if err := validatePatterns(patterns); err != nil {
		return &FederationImportResult{
			OK: false, DryRun: false, Target: target,
			PlannedCount: len(validation.Patterns), WrittenCount: 0,
			Errors: []string{err.Error()},
			AdmissionAuthority: false,
		}, nil
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return nil, err
	}
	if err := writeJsonl(target, patterns); err != nil {
		return nil, err
	}

	return &FederationImportResult{
		OK: true, DryRun: false, Target: target,
		PlannedCount: len(validation.Patterns), WrittenCount: len(patterns),
		Errors: []string{},
		AdmissionAuthority: false,
	}, nil
}

// Helper functions

func scopedHash(tenant, value string) string {
	h := sha256.Sum256([]byte(tenant + ":" + value))
	return hex.EncodeToString(h[:])
}

func stableStringify(value any) string {
	if value == nil {
		return "null"
	}
	switch v := value.(type) {
	case string:
		b, _ := json.Marshal(v)
		return string(b)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = stableStringify(item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = stableStringify(k) + ":" + stableStringify(v[k])
		}
		return "{" + strings.Join(parts, ",") + "}"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func stringField(entry map[string]any, key string) string {
	if v, ok := entry[key].(string); ok {
		return v
	}
	return ""
}

func getCanonicalSignalMetadata(entry map[string]any, key string) (string, error) {
	meta, ok := entry["metadata"].(map[string]any)
	if !ok {
		return "", nil
	}
	val, ok := meta[key].(string)
	if !ok {
		return "", nil
	}
	var allowed map[string]bool
	if key == "admission_outcome" {
		allowed = admissionOutcomes
	} else if key == "acceptance_status" {
		allowed = acceptanceStatuses
	} else {
		return "", nil
	}
	if allowed[val] {
		return val, nil
	}
	evidenceID := stringField(entry, "evidence_id")
	return "", fmt.Errorf("invalid federation %s metadata for %s: %s", key, evidenceID, val)
}

func componentHints(entry map[string]any) []string {
	meta, ok := entry["metadata"].(map[string]any)
	if ok {
		if ids, ok := meta["component_ids"].([]any); ok {
			hints := []string{}
			for _, id := range ids {
				if s, ok := id.(string); ok {
					hints = append(hints, s)
				}
			}
			if len(hints) > 0 {
				return hints
			}
		}
	}
	path := stringField(entry, "path")
	if path != "" {
		parts := strings.Split(path, "/")
		if len(parts) > 0 && parts[0] != "" {
			return []string{parts[0]}
		}
	}
	return []string{}
}

func isFailureSignal(predicate, outcome, acceptance string) bool {
	if outcome != "" && outcome != "success" {
		return true
	}
	if acceptance == "withheld" {
		return true
	}
	if predicate != "" {
		matched, _ := regexp.MatchString(`(?i)(blocked|failed|withheld|missing|error|timeout|false_accept)`, predicate)
		return matched
	}
	return false
}

func retentionExpiry(createdAt string, days int) string {
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		// Fallback: try parsing with nano
		t, _ = time.Parse(time.RFC3339Nano, createdAt)
	}
	t = t.AddDate(0, 0, days)
	return t.Format(time.RFC3339Nano)
}

func filterCandidates(entries []map[string]any) []map[string]any {
	candidates := []map[string]any{}
	for _, entry := range entries {
		outcome, _ := getCanonicalSignalMetadata(entry, "admission_outcome")
		acceptance, _ := getCanonicalSignalMetadata(entry, "acceptance_status")
		predicate := stringField(entry, "predicate")
		if predicate != "" || outcome != "" || acceptance != "" {
			candidates = append(candidates, entry)
		}
	}
	return candidates
}

func hashStrings(values []string, tenant string) []string {
	hashed := make([]string, len(values))
	for i, v := range values {
		hashed[i] = scopedHash(tenant, v)
	}
	sort.Strings(hashed)
	return hashed
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func validatePatterns(patterns []FederationPattern) error {
	root, err := repo.FindRoot("")
	if err != nil {
		return fmt.Errorf("cannot find repository root: %w", err)
	}
	schemaPath := assets.NewLocator(root).Schema("federation-pattern.schema.json")
	validator, err := schema.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("cannot compile schema: %w", err)
	}
	var errors []string
	for _, pattern := range patterns {
		m := structToMap(pattern)
		if err := validator.Validate(m); err != nil {
			errors = append(errors, fmt.Sprintf("%v", err))
		}
		_, _, replacements := evidence.RedactText(stableStringify(pattern))
		if replacements > 0 {
			errors = append(errors, fmt.Sprintf("secret-like value detected in %s", pattern.PatternID))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("federation pattern validation failed: %s", strings.Join(errors, "; "))
	}
	return nil
}

func writeJsonl(path string, patterns []FederationPattern) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, p := range patterns {
		if err := enc.Encode(p); err != nil {
			return err
		}
	}
	return nil
}

func structToMap(v any) map[string]any {
	b, _ := json.Marshal(v)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func loadBenchmarkMetrics(path string) *BenchmarkMetrics {
	if path == "" {
		return nil
	}
	var report any
	if err := loader.LoadDocument(path, &report); err != nil {
		return nil
	}
	return extractMetrics(report)
}

func extractMetrics(report any) *BenchmarkMetrics {
	if report == nil {
		return nil
	}
	m, ok := report.(map[string]any)
	if !ok {
		return nil
	}
	metricsRaw, ok := m["metrics"]
	if !ok || metricsRaw == nil {
		return nil
	}
	metrics, ok := metricsRaw.(map[string]any)
	if !ok {
		return nil
	}
	var out BenchmarkMetrics
	hasAny := false
	if v, ok := toInt(metrics["false_accept_count"]); ok {
		out.FalseAcceptCount = &v
		hasAny = true
	}
	if v, ok := toInt(metrics["adversarial_false_accept_count"]); ok {
		out.AdversarialFalseAcceptCount = &v
		hasAny = true
	}
	if v, ok := toInt(metrics["false_reject_count"]); ok {
		out.FalseRejectCount = &v
		hasAny = true
	}
	if v, ok := toFloat64(metrics["runtime_ms"]); ok {
		out.RuntimeMs = &v
		hasAny = true
	}
	if !hasAny {
		return nil
	}
	return &out
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
