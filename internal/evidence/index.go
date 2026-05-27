package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/loader"
)

// IndexResult represents index building outcome.
type IndexResult struct {
	OK            bool         `json:"ok"`
	SchemaVersion string       `json:"schema_version"`
	TaskID        string       `json:"task_id"`
	CreatedAt     string       `json:"created_at"`
	EntryCount    int          `json:"entry_count"`
	IndexHash     string       `json:"index_hash"`
	Entries       []IndexEntry `json:"entries"`
	OutPath       string       `json:"out_path"`
	RedactedDir   string       `json:"redacted_dir,omitempty"`
	Warnings      []string     `json:"warnings,omitempty"`
	Errors        []string     `json:"errors,omitempty"`
}

// IndexEntry represents a single evidence index entry.
type IndexEntry struct {
	SchemaVersion      string         `json:"schema_version"`
	TaskID             string         `json:"task_id"`
	EvidenceID         string         `json:"evidence_id"`
	Layer              string         `json:"layer"`
	Kind               string         `json:"kind"`
	Path               string         `json:"path"`
	SourcePath         string         `json:"source_path,omitempty"`
	SHA256             string         `json:"sha256"`
	SizeBytes          int64          `json:"size_bytes"`
	Redacted           bool           `json:"redacted"`
	Redaction          *RedactionInfo `json:"redaction,omitempty"`
	CreatedAt          string         `json:"created_at"`
	AdmissionAuthority bool           `json:"admission_authority"`
}

// RedactionInfo holds redaction metadata for an entry.
type RedactionInfo struct {
	Mode         string   `json:"mode"`
	Patterns     []string `json:"patterns"`
	Replacements int      `json:"replacements"`
}

// IndexOptions configures the BuildIndex behavior.
type IndexOptions struct {
	Episode     string
	Card        string
	TaskID      string
	Out         string
	Redact      bool
	RedactedDir string
}

// BuildIndex builds an evidence index from an episode directory or completion card.
func BuildIndex(opts IndexOptions) (*IndexResult, error) {
	if opts.Episode == "" && opts.Card == "" {
		return nil, fmt.Errorf("--episode or --card is required")
	}

	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	var entries []IndexEntry
	var warnings []string
	taskID := opts.TaskID

	if opts.Card != "" {
		cardPath, err := filepath.Abs(opts.Card)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(cardPath); err != nil {
			return nil, fmt.Errorf("completion card not found: %s", cardPath)
		}
		cardInfo, err := loadCardTask(cardPath)
		if err != nil {
			return nil, err
		}
		if taskID == "" {
			taskID = cardInfo.TaskID
		}
		if taskID == "" {
			return nil, fmt.Errorf("task id is required when card has no task_id")
		}

		root := filepath.Dir(cardPath)
		rel := relativeTo(root, cardPath)
		if rel == "" {
			rel = filepath.Base(cardPath)
		}
		hash, size, err := hashFile(cardPath)
		if err != nil {
			return nil, err
		}
		entries = append(entries, IndexEntry{
			SchemaVersion:      "1",
			TaskID:             taskID,
			EvidenceID:         evidenceID([]string{taskID, "raw", "completion_card", rel}),
			Layer:              "raw",
			Kind:               "completion_card",
			Path:               rel,
			SHA256:             hash,
			SizeBytes:          size,
			Redacted:           false,
			CreatedAt:          createdAt,
			AdmissionAuthority: false,
		})

		for i, item := range cardInfo.CommandEvidenceEntries {
			entries = append(entries, makeVirtualEntry(taskID, fmt.Sprintf("completion-card.yaml#/evidence/command_evidence/%d", i), "command_evidence", createdAt, item))
		}
		for i, item := range cardInfo.VerificationArtifactEntries {
			entries = append(entries, makeVirtualEntry(taskID, fmt.Sprintf("completion-card.yaml#/evidence/verification_artifacts/%d", i), "verification_artifact", createdAt, item))
		}
	}

	if opts.Episode != "" {
		episodeDir, err := filepath.Abs(opts.Episode)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(episodeDir); err != nil {
			return nil, fmt.Errorf("episode directory not found: %s", episodeDir)
		}

		if taskID == "" {
			cardPath := filepath.Join(episodeDir, "completion-card.yaml")
			if _, err := os.Stat(cardPath); err == nil {
				cardInfo, err := loadCardTask(cardPath)
				if err == nil && cardInfo.TaskID != "" {
					taskID = cardInfo.TaskID
				}
			}
		}
		if taskID == "" {
			manifestPath := filepath.Join(episodeDir, "manifest.json")
			if data, err := os.ReadFile(manifestPath); err == nil {
				var manifest map[string]any
				if err := json.Unmarshal(data, &manifest); err == nil {
					if id, ok := manifest["task_id"].(string); ok && id != "" {
						taskID = id
					}
				}
			}
		}
		if taskID == "" {
			return nil, fmt.Errorf("--task-id is required for this episode")
		}

		files, err := collectFiles(episodeDir)
		if err != nil {
			return nil, err
		}

		var redactedDir string
		if opts.Redact {
			if opts.RedactedDir != "" {
				redactedDir, _ = filepath.Abs(opts.RedactedDir)
			} else {
				redactedDir = filepath.Join(episodeDir, "evidence", "redacted", taskID)
			}
		}

		for _, filePath := range files {
			kind := inferKind(filePath)
			rel := relativeTo(episodeDir, filePath)
			hash, size, err := hashFile(filePath)
			if err != nil {
				return nil, err
			}
			rawEntry := IndexEntry{
				SchemaVersion:      "1",
				TaskID:             taskID,
				EvidenceID:         evidenceID([]string{taskID, "raw", kind, rel}),
				Layer:              "raw",
				Kind:               kind,
				Path:               rel,
				SHA256:             hash,
				SizeBytes:          size,
				Redacted:           false,
				CreatedAt:          createdAt,
				AdmissionAuthority: false,
			}
			entries = append(entries, rawEntry)

			if !opts.Redact {
				continue
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, err
			}
			if !IsTextFile(filePath, content) {
				warnings = append(warnings, fmt.Sprintf("skipped binary redaction for %s", rel))
				continue
			}

			redactedText, patterns, replacements := RedactText(string(content))
			if replacements == 0 {
				continue
			}

			relFromEpisode, _ := filepath.Rel(episodeDir, filePath)
			redactedPath := filepath.Join(redactedDir, relFromEpisode)
			if err := os.MkdirAll(filepath.Dir(redactedPath), 0755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(redactedPath, []byte(redactedText), 0644); err != nil {
				return nil, err
			}

			redactedHash, redactedSize, err := hashFile(redactedPath)
			if err != nil {
				return nil, err
			}
			entries = append(entries, IndexEntry{
				SchemaVersion:      "1",
				TaskID:             taskID,
				EvidenceID:         evidenceID([]string{taskID, "redacted", kind, rel}),
				Layer:              "redacted",
				Kind:               kind,
				Path:               relativeTo(episodeDir, redactedPath),
				SourcePath:         rel,
				SHA256:             redactedHash,
				SizeBytes:          redactedSize,
				Redacted:           true,
				Redaction: &RedactionInfo{
					Mode:         "secret-redaction",
					Patterns:     patterns,
					Replacements: replacements,
				},
				CreatedAt:          createdAt,
				AdmissionAuthority: false,
			})
		}
	}

	if taskID == "" {
		return nil, fmt.Errorf("task id could not be inferred")
	}

	entries = sortEntries(entries)
	indexHash := hashEntries(entries)

	outPath := ""
	if opts.Out != "" {
		outPath, _ = filepath.Abs(opts.Out)
	} else {
		if opts.Episode != "" {
			outPath = filepath.Join(opts.Episode, "evidence", "index.jsonl")
		} else {
			outPath = filepath.Join(filepath.Dir(opts.Card), "evidence", "index.jsonl")
		}
	}

	// Validate before writing
	var mapEntries []map[string]any
	for _, e := range entries {
		b, _ := json.Marshal(e)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		mapEntries = append(mapEntries, m)
	}
	ok, errs := ValidateEntries(mapEntries)
	if !ok {
		return &IndexResult{
			OK:            false,
			SchemaVersion: "1",
			TaskID:        taskID,
			CreatedAt:     createdAt,
			EntryCount:    len(entries),
			IndexHash:     indexHash,
			Entries:       entries,
			OutPath:       relativeToCwd(outPath),
			Warnings:      warnings,
			Errors:        errs,
		}, fmt.Errorf("evidence index validation failed")
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return nil, err
	}
	if err := writeJsonl(outPath, entries); err != nil {
		return nil, err
	}

	result := &IndexResult{
		OK:            true,
		SchemaVersion: "1",
		TaskID:        taskID,
		CreatedAt:     createdAt,
		EntryCount:    len(entries),
		IndexHash:     indexHash,
		Entries:       entries,
		OutPath:       relativeToCwd(outPath),
		Warnings:      warnings,
	}

	if opts.Redact {
		if opts.RedactedDir != "" {
			result.RedactedDir = relativeToCwd(opts.RedactedDir)
		} else if opts.Episode != "" {
			result.RedactedDir = relativeToCwd(filepath.Join(opts.Episode, "evidence", "redacted", taskID))
		} else {
			result.RedactedDir = relativeToCwd(filepath.Join(filepath.Dir(opts.Card), "evidence", "redacted", taskID))
		}
	}

	return result, nil
}

type cardInfo struct {
	TaskID                        string
	CommandEvidenceEntries        []any
	VerificationArtifactEntries   []any
}

func loadCardTask(cardPath string) (*cardInfo, error) {
	var card map[string]any
	if err := loader.LoadDocument(cardPath, &card); err != nil {
		return nil, err
	}
	info := &cardInfo{}
	if id, ok := card["task_id"].(string); ok {
		info.TaskID = id
	}
	if evidence, ok := card["evidence"].(map[string]any); ok {
		if ce, ok := evidence["command_evidence"].([]any); ok {
			info.CommandEvidenceEntries = ce
		}
		if va, ok := evidence["verification_artifacts"].([]any); ok {
			info.VerificationArtifactEntries = va
		}
	}
	return info, nil
}

func makeVirtualEntry(taskID, path, kind, createdAt string, value any) IndexEntry {
	serialized := stableStringify(value)
	hash := sha256String(serialized)
	return IndexEntry{
		SchemaVersion:      "1",
		TaskID:             taskID,
		EvidenceID:         evidenceID([]string{taskID, "raw", kind, path}),
		Layer:              "raw",
		Kind:               kind,
		Path:               path,
		SHA256:             hash,
		SizeBytes:          int64(len(serialized)),
		Redacted:           false,
		CreatedAt:          createdAt,
		AdmissionAuthority: false,
	}
}

func stableStringify(value any) string {
	if value == nil {
		return "null"
	}
	switch v := value.(type) {
	case string:
		return jsonString(v)
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
			parts[i] = jsonString(k) + ":" + stableStringify(v[k])
		}
		return "{" + strings.Join(parts, ",") + "}"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func evidenceID(parts []string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, ":")))
	return hex.EncodeToString(h[:])[:16]
}

func hashFile(path string) (string, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), int64(len(data)), nil
}

func sha256String(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func hashEntries(entries []IndexEntry) string {
	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = stableStringify(entryToMap(e))
	}
	return sha256String(strings.Join(parts, "\n"))
}

func entryToMap(e IndexEntry) map[string]any {
	b, _ := json.Marshal(e)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func sortEntries(entries []IndexEntry) []IndexEntry {
	sort.Slice(entries, func(i, j int) bool {
		a := fmt.Sprintf("%s:%s:%s:%s", entries[i].TaskID, entries[i].Layer, entries[i].Kind, entries[i].Path)
		b := fmt.Sprintf("%s:%s:%s:%s", entries[j].TaskID, entries[j].Layer, entries[j].Kind, entries[j].Path)
		return a < b
	})
	return entries
}

func collectFiles(rootDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func inferKind(filePath string) string {
	name := strings.ToLower(filepath.Base(filePath))
	if name == "completion-card.yaml" || name == "completion-card.json" {
		return "completion_card"
	}
	if strings.HasSuffix(name, ".jsonl") && strings.Contains(name, "trace") {
		return "trace_event"
	}
	if strings.Contains(name, "stdout") || strings.Contains(name, "stderr") || strings.Contains(name, "test") || strings.Contains(name, "verify") {
		return "verification_artifact"
	}
	return "episode_file"
}

func relativeTo(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}

func relativeToCwd(target string) string {
	cwd, _ := os.Getwd()
	return relativeTo(cwd, target)
}

func writeJsonl(path string, entries []IndexEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return nil
}

// ValidateIndex validates built entries against the evidence-index schema.
func ValidateIndex(entries []IndexEntry) (bool, []string) {
	var mapEntries []map[string]any
	for _, e := range entries {
		b, _ := json.Marshal(e)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		mapEntries = append(mapEntries, m)
	}
	return ValidateEntries(mapEntries)
}
