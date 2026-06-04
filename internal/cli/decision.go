package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// decisionRecordSchemaVersion is the schema_version emitted by
// `xh decision record` for safe V1 decision records. The version is fixed
// for the first slice to keep the contract deterministic and
// parity-safe with schemas/decision-record.schema.json.
const decisionRecordSchemaVersion = "1"

// decisionRecordDefaultDir is the default directory under which
// `xh decision record` writes its output when the caller does not pass
// --output. The directory is created on demand for safe V1; subsequent
// slices may tighten this behavior.
const decisionRecordDefaultDir = "decisions"

// decisionRecordStatuses is the closed enum accepted by
// `xh decision record --status` for safe V1. The list mirrors the
// `status` enum declared in schemas/decision-record.schema.json.
var decisionRecordStatuses = []string{
	"proposed",
	"accepted",
	"superseded",
	"deprecated",
}

// decisionRecordSpec captures the structured-flag input to
// `xh decision record` before it is normalized into the decision record.
// The CLI accepts repeatable and comma-delimited values for the
// list-shaped fields so users can express them either way.
type decisionRecordSpec struct {
	ID            string
	Title         string
	Date          string
	Status        string
	Decision      string
	Rationale     string
	Context       string
	Consequences  string
	SupersededBy  string
	Tags          []string
	AffectedPaths []string
	Notes         string
}

// handleDecision is the entry point for `xh decision ...`. Safe V1 only
// exposes the `record` and `list` subcommands; `query`, `link`, and
// `affected` are intentionally deferred to follow-up slices.
func handleDecision(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "decision requires a subcommand: record, list")
		return ExitUsage
	}

	switch args[0] {
	case "record":
		return handleDecisionRecord(args[1:], stdout, stderr)
	case "list":
		return handleDecisionList(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown decision subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func handleDecisionRecord(args []string, stdout, stderr io.Writer) int {
	spec := decisionRecordSpec{
		Status: "proposed",
	}
	outputPath := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--id":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --id requires a value")
				return ExitUsage
			}
			spec.ID = args[i+1]
			i++
		case "--title":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --title requires a value")
				return ExitUsage
			}
			spec.Title = args[i+1]
			i++
		case "--date":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --date requires a value")
				return ExitUsage
			}
			spec.Date = args[i+1]
			i++
		case "--status":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --status requires a value")
				return ExitUsage
			}
			spec.Status = args[i+1]
			i++
		case "--decision":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --decision requires a value")
				return ExitUsage
			}
			spec.Decision = args[i+1]
			i++
		case "--rationale":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --rationale requires a value")
				return ExitUsage
			}
			spec.Rationale = args[i+1]
			i++
		case "--context":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --context requires a value")
				return ExitUsage
			}
			spec.Context = args[i+1]
			i++
		case "--consequence":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --consequence requires a value")
				return ExitUsage
			}
			spec.Consequences = args[i+1]
			i++
		case "--superseded-by":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --superseded-by requires a value")
				return ExitUsage
			}
			spec.SupersededBy = args[i+1]
			i++
		case "--tag":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --tag requires a value")
				return ExitUsage
			}
			spec.Tags = appendList(spec.Tags, args[i+1])
			i++
		case "--affected-path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --affected-path requires a value")
				return ExitUsage
			}
			spec.AffectedPaths = appendList(spec.AffectedPaths, args[i+1])
			i++
		case "--note":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --note requires a value")
				return ExitUsage
			}
			spec.Notes = args[i+1]
			i++
		case "--output":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --output requires a value")
				return ExitUsage
			}
			outputPath = args[i+1]
			i++
		case "--json":
			jsonMode = true
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh decision record --id <id> --decision <text> --rationale <text> [--title <text>] [--status proposed|accepted|superseded|deprecated] [--date <iso-date>] [--context <text>] [--consequence <text>] [--superseded-by <id>] [--tag <text> ...] [--affected-path <path> ...] [--note <text>] [--output <path>] [--json]")
			return ExitUsage
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", arg)
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", arg)
			return ExitUsage
		}
	}

	status, err := normalizeDecisionStatus(spec.Status)
	if err != nil {
		fmt.Fprintf(stderr, "error: --status %v\n", err)
		return ExitUsage
	}
	spec.Status = status

	record, err := buildDecisionRecord(spec)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitUsage
	}

	if outputPath == "" {
		outputPath = defaultDecisionOutputPath(record)
		if err := ensureDecisionOutputDir(outputPath, true); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
	} else {
		if err := ensureDecisionOutputDir(outputPath, false); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
	}

	var payload []byte
	if jsonMode || strings.EqualFold(filepath.Ext(outputPath), ".json") {
		payload, err = json.MarshalIndent(record, "", "  ")
	} else {
		payload, err = yaml.Marshal(record)
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}
	payload = append(payload, '\n')

	if err := os.WriteFile(outputPath, payload, 0644); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	fmt.Fprintf(stdout, "Decision record written: %s\n", outputPath)
	return ExitOK
}

func handleDecisionList(args []string, stdout, stderr io.Writer) int {
	dir := decisionRecordDefaultDir
	jsonMode := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --dir requires a value")
				return ExitUsage
			}
			dir = args[i+1]
			i++
		case "--json":
			jsonMode = true
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh decision list [--dir <path>] [--json]")
			return ExitUsage
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", arg)
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", arg)
			return ExitUsage
		}
	}

	records, err := listDecisionRecords(dir)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, err := json.MarshalIndent(map[string]any{
			"directory": dir,
			"count":     len(records),
			"records":   records,
		}, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
		fmt.Fprintln(stdout, string(data))
		return ExitOK
	}

	fmt.Fprintf(stdout, "Directory: %s\n", dir)
	fmt.Fprintf(stdout, "Count: %d\n", len(records))
	if len(records) == 0 {
		return ExitOK
	}
	fmt.Fprintln(stdout, "Records:")
	for _, rec := range records {
		fmt.Fprintf(stdout, "  - id=%s status=%s title=%q\n    decision: %s\n    path: %s\n",
			rec.ID,
			rec.Status,
			rec.Title,
			rec.Decision,
			rec.Path,
		)
	}
	return ExitOK
}

// buildDecisionRecord converts a structured spec into a map matching
// schemas/decision-record.schema.json (safe V1). Required fields:
// schema_version, id, decision, rationale. All other fields are emitted
// only when explicitly set so the YAML/JSON output is compact and
// round-trip safe.
func buildDecisionRecord(spec decisionRecordSpec) (map[string]any, error) {
	if strings.TrimSpace(spec.ID) == "" {
		return nil, fmt.Errorf("--id is required")
	}
	if strings.TrimSpace(spec.Decision) == "" {
		return nil, fmt.Errorf("--decision is required")
	}
	if strings.TrimSpace(spec.Rationale) == "" {
		return nil, fmt.Errorf("--rationale is required")
	}

	record := map[string]any{
		"schema_version": decisionRecordSchemaVersion,
		"id":             strings.TrimSpace(spec.ID),
		"decision":       strings.TrimSpace(spec.Decision),
		"rationale":      strings.TrimSpace(spec.Rationale),
	}
	if v := strings.TrimSpace(spec.Title); v != "" {
		record["title"] = v
	}
	if v := strings.TrimSpace(spec.Date); v != "" {
		record["date"] = v
	} else {
		// Default to today's date (UTC) so the record carries a stable
		// authoring timestamp without requiring the caller to pass one.
		record["date"] = time.Now().UTC().Format("2006-01-02")
	}
	if v := strings.TrimSpace(spec.Status); v != "" {
		record["status"] = v
	}
	if v := strings.TrimSpace(spec.Context); v != "" {
		record["context"] = v
	}
	if v := strings.TrimSpace(spec.Consequences); v != "" {
		record["consequences"] = v
	}
	if v := strings.TrimSpace(spec.SupersededBy); v != "" {
		record["superseded_by"] = v
	}
	if len(spec.Tags) > 0 {
		record["tags"] = toAnySlice(spec.Tags)
	}
	if len(spec.AffectedPaths) > 0 {
		record["affected_paths"] = toAnySlice(spec.AffectedPaths)
	}
	if v := strings.TrimSpace(spec.Notes); v != "" {
		record["notes"] = v
	}
	return record, nil
}

func normalizeDecisionStatus(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "proposed", nil
	}
	for _, candidate := range decisionRecordStatuses {
		if candidate == trimmed {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("expected one of %s, got %q", strings.Join(decisionRecordStatuses, ", "), raw)
}

// defaultDecisionOutputPath returns the canonical on-disk path for a
// freshly built decision record when the caller did not pass --output.
// Safe V1 uses decisions/<id>.yaml (YAML is preferred because the
// product_intent precedent already established the YAML output
// convention and the package reuses gopkg.in/yaml.v3).
func defaultDecisionOutputPath(record map[string]any) string {
	id := asString(record["id"])
	return filepath.Join(decisionRecordDefaultDir, id+".yaml")
}

// ensureDecisionOutputDir enforces the safe V1 parent-directory policy
// for the decision record output path. When allowCreate is true (the
// default decisions/<id>.yaml path), a missing parent is created on
// demand. When allowCreate is false (an explicit --output path), the
// parent must already exist; the CLI refuses to silently create
// arbitrary intermediate directories because that would mask typos and
// stray-pipeline writes. The behavior mirrors `writeIntentContractOutput`
// in intake.go for the strict path and the implicit-default behavior of
// the product intent slice for the lenient path.
func ensureDecisionOutputDir(path string, allowCreate bool) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	parent := filepath.Dir(abs)
	if _, err := os.Stat(parent); err == nil {
		return nil
	} else if !allowCreate {
		return fmt.Errorf("parent directory does not exist: %s", parent)
	}
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	return nil
}

// decisionListEntry is a compact projection of a stored decision record
// used by `xh decision list`. It carries the original on-disk path so
// callers can locate the full record without re-parsing the directory.
type decisionListEntry struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Title    string `json:"title"`
	Decision string `json:"decision"`
	Path     string `json:"path"`
}

// listDecisionRecords scans dir for decision-record files (.yaml,
// .yml, .json), loads the minimum fields needed for the list view, and
// returns the entries sorted by id for deterministic output. A missing
// directory is reported as an empty result, not an error, so that the
// slice is safe to call against a fresh workspace.
func listDecisionRecords(dir string) ([]decisionListEntry, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return []decisionListEntry{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", abs)
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}

	out := make([]decisionListEntry, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}
		full := filepath.Join(abs, name)
		var doc map[string]any
		if err := readDecisionFile(full, &doc); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out = append(out, decisionListEntry{
			ID:       asString(doc["id"]),
			Status:   asString(doc["status"]),
			Title:    asString(doc["title"]),
			Decision: asString(doc["decision"]),
			Path:     full,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// readDecisionFile loads a YAML or JSON decision record into a generic
// map. The function uses yaml.Unmarshal for both formats because
// gopkg.in/yaml.v3 transparently parses JSON-compatible input. This
// keeps the list path dependency-free without forcing a hard format
// detection decision on the caller.
func readDecisionFile(path string, out *map[string]any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

// asString coerces an arbitrary map value to a string. Used by the list
// view where missing fields are valid (the spec only requires
// schema_version, id, decision, rationale).
func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
