package frozen

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/schema"
)

var includePaths = []string{
	"README.md",
	"AGENTS.md",
	"X_HARNESS.md",
	"CHANGELOG.md",
	"LICENSE",
	"docs",
	"schemas",
	"policies",
	"templates",
	"adapters",
	"components/registry.yaml",
	"examples/golden",
	"examples/adversarial",
	"tools/experimental/evolve",
}

// FrozenManifestFile represents a single file in the manifest.
type FrozenManifestFile struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
	Size   int    `json:"size"`
}

// FrozenManifest represents the bundle manifest.
type FrozenManifest struct {
	SchemaVersion        string              `json:"schema_version"`
	BundleID             string              `json:"bundle_id"`
	XHarnessVersion      string              `json:"x_harness_version"`
	CreatedAt            string              `json:"created_at"`
	SourceCommit         string              `json:"source_commit"`
	MaturityLevel        string              `json:"maturity_level"`
	Benchmark            map[string]any      `json:"benchmark"`
	IncludedComponents   []string            `json:"included_components"`
	Files                []FrozenManifestFile `json:"files"`
	Signing              map[string]any      `json:"signing"`
}

// FrozenVerifyResult is the result of verify.
type FrozenVerifyResult struct {
	OK          bool           `json:"ok"`
	BundlePath  string         `json:"bundle_path"`
	Manifest    *FrozenManifest `json:"manifest"`
	FileCount   int            `json:"file_count"`
	Errors      []string       `json:"errors"`
}

// FrozenImportResult is the result of import.
type FrozenImportResult struct {
	OK        bool     `json:"ok"`
	DryRun    bool     `json:"dry_run"`
	Target    string   `json:"target"`
	Planned   []string `json:"planned"`
	Written   []string `json:"written"`
	Skipped   []string `json:"skipped"`
	Conflicts []string `json:"conflicts"`
	Errors    []string `json:"errors"`
}

// FrozenExportResult is the result of export.
type FrozenExportResult struct {
	OK        bool           `json:"ok"`
	Out       string         `json:"out"`
	Manifest  *FrozenManifest `json:"manifest"`
	FileCount int            `json:"file_count"`
}

// BundleEntry represents a single entry in the bundle.
type BundleEntry struct {
	Path string
	Data []byte
}

func sha256Buffer(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func assertSafeArchivePath(relativePath string) error {
	normalized := filepath.ToSlash(filepath.Clean(relativePath))
	if strings.HasPrefix(normalized, "../") || normalized == ".." || filepath.IsAbs(normalized) || normalized != relativePath {
		return fmt.Errorf("unsafe archive path: %s", relativePath)
	}
	return nil
}

func collectFiles(root, relativePath string) ([]string, error) {
	full := filepath.Join(root, relativePath)
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return []string{filepath.ToSlash(relativePath)}, nil
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		child := filepath.Join(relativePath, entry.Name())
		childFiles, err := collectFiles(root, child)
		if err != nil {
			return nil, err
		}
		files = append(files, childFiles...)
	}
	sort.Strings(files)
	return files, nil
}

func gitCommit(root string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func packageVersion(root string) string {
	pkgPath := filepath.Join(root, "packages", "cli", "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return "unknown"
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "unknown"
	}
	if pkg.Version != "" {
		return pkg.Version
	}
	return "unknown"
}

func componentIds(root string) ([]string, error) {
	registryPath := filepath.Join(root, "components", "registry.yaml")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, nil
	}
	re := regexp.MustCompile(`^\s*-\s+id:\s+(.+)$`)
	lines := strings.Split(string(data), "\n")
	var ids []string
	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) > 1 {
			ids = append(ids, strings.TrimSpace(m[1]))
		}
	}
	return ids, nil
}

func buildManifest(root string, files []FrozenManifestFile) (*FrozenManifest, error) {
	now := time.Now().UTC()
	bundleID := fmt.Sprintf("xh_frozen_%s", now.Format("20060102150405"))
	components, err := componentIds(root)
	if err != nil {
		return nil, err
	}
	return &FrozenManifest{
		SchemaVersion:   "1",
		BundleID:        bundleID,
		XHarnessVersion: packageVersion(root),
		CreatedAt:       now.Format(time.RFC3339Nano),
		SourceCommit:    gitCommit(root),
		MaturityLevel:   "H2",
		Benchmark: map[string]any{
			"false_accept_count":           0,
			"adversarial_false_accept_count": 0,
			"episode_packaging_success_rate": nil,
		},
		IncludedComponents: components,
		Files:              files,
		Signing: map[string]any{
			"mode": "unsigned",
		},
	}, nil
}

func validateManifest(manifest *FrozenManifest, root string) ([]string, error) {
	schemaPath := filepath.Join(root, "schemas", "frozen-manifest.schema.json")
	v, err := schema.Compile(schemaPath)
	if err != nil {
		return nil, err
	}
	// jsonschema/v6 requires JSON-compatible input, not a Go struct pointer.
	data, err := json.Marshal(manifest)
	if err != nil {
		return []string{err.Error()}, nil
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return []string{err.Error()}, nil
	}
	if err := v.Validate(doc); err != nil {
		return []string{err.Error()}, nil
	}
	return nil, nil
}

func createTarGz(entries []BundleEntry) ([]byte, error) {
	var buf strings.Builder
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, entry := range entries {
		header := &tar.Header{
			Name: entry.Path,
			Mode: 0644,
			Size: int64(len(entry.Data)),
		}
		if err := tw.WriteHeader(header); err != nil {
			return nil, err
		}
		if _, err := tw.Write(entry.Data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func readTarGz(data []byte) ([]BundleEntry, error) {
	gr, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	var entries []BundleEntry
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != 0 {
			continue
		}
		data := make([]byte, header.Size)
		_, err = io.ReadFull(tr, data)
		if err != nil {
			return nil, err
		}
		entries = append(entries, BundleEntry{Path: header.Name, Data: data})
	}
	return entries, nil
}

func checksumsFromEntry(entry *BundleEntry) map[string]string {
	checksums := make(map[string]string)
	if entry == nil {
		return checksums
	}
	lines := strings.Split(string(entry.Data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) == 2 {
			checksums[parts[1]] = parts[0]
		}
	}
	return checksums
}

// ExportFrozenBundle exports a frozen bundle.
func ExportFrozenBundle(root, out string) (*FrozenExportResult, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return nil, err
	}

	var relativeFiles []string
	for _, item := range includePaths {
		files, err := collectFiles(root, item)
		if err != nil {
			return nil, err
		}
		relativeFiles = append(relativeFiles, files...)
	}
	sort.Strings(relativeFiles)

	var payloadEntries []BundleEntry
	var manifestFiles []FrozenManifestFile
	for _, relativePath := range relativeFiles {
		if err := assertSafeArchivePath(relativePath); err != nil {
			return nil, err
		}
		data, err := os.ReadFile(filepath.Join(root, relativePath))
		if err != nil {
			return nil, err
		}
		manifestFiles = append(manifestFiles, FrozenManifestFile{
			Path:   relativePath,
			Sha256: sha256Buffer(data),
			Size:   len(data),
		})
		payloadEntries = append(payloadEntries, BundleEntry{Path: relativePath, Data: data})
	}

	manifest, err := buildManifest(root, manifestFiles)
	if err != nil {
		return nil, err
	}
	manifestErrors, err := validateManifest(manifest, root)
	if err != nil {
		return nil, err
	}
	if len(manifestErrors) > 0 {
		return nil, fmt.Errorf("frozen manifest validation failed: %s", strings.Join(manifestErrors, "; "))
	}

	checksums := make([]string, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		checksums = append(checksums, fmt.Sprintf("%s  %s", file.Sha256, file.Path))
	}
	version := map[string]any{
		"x_harness_version": manifest.XHarnessVersion,
		"source_commit":     manifest.SourceCommit,
		"created_at":        manifest.CreatedAt,
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	versionJSON, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return nil, err
	}

	entries := []BundleEntry{
		{Path: "manifest.json", Data: manifestJSON},
		{Path: "checksums.sha256", Data: []byte(strings.Join(checksums, "\n") + "\n")},
		{Path: "version.json", Data: versionJSON},
	}
	entries = append(entries, payloadEntries...)

	bundleData, err := createTarGz(entries)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(out, bundleData, 0644); err != nil {
		return nil, err
	}

	return &FrozenExportResult{
		OK:        true,
		Out:       out,
		Manifest:  manifest,
		FileCount: len(manifest.Files),
	}, nil
}

func readFrozenArchive(bundlePath string) ([]BundleEntry, *FrozenManifest, []BundleEntry, []string, error) {
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	entries, err := readTarGz(data)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var errors []string
	var manifestEntry *BundleEntry
	var checksumEntry *BundleEntry
	var payload []BundleEntry
	for i := range entries {
		switch entries[i].Path {
		case "manifest.json":
			manifestEntry = &entries[i]
		case "checksums.sha256":
			checksumEntry = &entries[i]
		case "version.json":
			// metadata, skip
		default:
			payload = append(payload, entries[i])
		}
	}
	if manifestEntry == nil {
		return nil, nil, nil, nil, fmt.Errorf("frozen bundle missing manifest.json")
	}
	var manifest FrozenManifest
	if err := json.Unmarshal(manifestEntry.Data, &manifest); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("invalid manifest.json: %w", err)
	}
	schemaPath := filepath.Join(".", "schemas", "frozen-manifest.schema.json")
	if _, statErr := os.Stat(schemaPath); statErr == nil {
		manifestErrors, err := validateManifest(&manifest, ".")
		if err != nil {
			return nil, nil, nil, nil, err
		}
		errors = append(errors, manifestErrors...)
	}
	checksums := checksumsFromEntry(checksumEntry)

	manifestPaths := make(map[string]struct{})
	checksumPaths := make(map[string]struct{})
	payloadByPath := make(map[string]BundleEntry)

	for _, file := range manifest.Files {
		if err := assertSafeArchivePath(file.Path); err != nil {
			errors = append(errors, err.Error())
			continue
		}
		if _, exists := manifestPaths[file.Path]; exists {
			errors = append(errors, fmt.Sprintf("duplicate manifest file path: %s", file.Path))
			continue
		}
		manifestPaths[file.Path] = struct{}{}
	}

	for _, entry := range payload {
		if err := assertSafeArchivePath(entry.Path); err != nil {
			errors = append(errors, err.Error())
			continue
		}
		if _, exists := payloadByPath[entry.Path]; exists {
			errors = append(errors, fmt.Sprintf("duplicate payload file path: %s", entry.Path))
			continue
		}
		payloadByPath[entry.Path] = entry
		if _, ok := manifestPaths[entry.Path]; !ok {
			errors = append(errors, fmt.Sprintf("payload file not declared in manifest: %s", entry.Path))
		}
	}

	for path := range checksums {
		checksumPaths[path] = struct{}{}
		if _, ok := manifestPaths[path]; !ok {
			errors = append(errors, fmt.Sprintf("checksums.sha256 path not declared in manifest: %s", path))
		}
	}

	var verifiedPayload []BundleEntry
	for _, file := range manifest.Files {
		entry, ok := payloadByPath[file.Path]
		if !ok {
			errors = append(errors, fmt.Sprintf("manifest file missing from bundle: %s", file.Path))
			continue
		}
		actual := sha256Buffer(entry.Data)
		if actual != file.Sha256 {
			errors = append(errors, fmt.Sprintf("checksum mismatch for %s", file.Path))
		}
		if checksums[file.Path] != file.Sha256 {
			errors = append(errors, fmt.Sprintf("checksums.sha256 mismatch for %s", file.Path))
		}
		verifiedPayload = append(verifiedPayload, entry)
	}

	return entries, &manifest, verifiedPayload, errors, nil
}

// VerifyFrozenBundle verifies a frozen bundle.
func VerifyFrozenBundle(bundlePath string) (*FrozenVerifyResult, error) {
	resolved, err := filepath.Abs(bundlePath)
	if err != nil {
		return &FrozenVerifyResult{
			OK:         false,
			BundlePath: bundlePath,
			Manifest:   nil,
			FileCount:  0,
			Errors:     []string{err.Error()},
		}, nil
	}
	_, manifest, payload, errors, err := readFrozenArchive(resolved)
	if err != nil {
		return &FrozenVerifyResult{
			OK:         false,
			BundlePath: resolved,
			Manifest:   nil,
			FileCount:  0,
			Errors:     []string{err.Error()},
		}, nil
	}
	return &FrozenVerifyResult{
		OK:         len(errors) == 0,
		BundlePath: resolved,
		Manifest:   manifest,
		FileCount:  len(payload),
		Errors:     errors,
	}, nil
}

// ImportFrozenBundle imports a frozen bundle.
func ImportFrozenBundle(bundlePath, target string, dryRun, merge, force bool) (*FrozenImportResult, error) {
	resolvedBundle, err := filepath.Abs(bundlePath)
	if err != nil {
		return nil, err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return nil, err
	}

	_, _, payload, errors, err := readFrozenArchive(resolvedBundle)
	if err != nil {
		return &FrozenImportResult{
			OK:      false,
			DryRun:  dryRun,
			Target:  target,
			Errors:  []string{err.Error()},
		}, nil
	}

	planned := make([]string, 0)
	written := make([]string, 0)
	skipped := make([]string, 0)
	conflicts := make([]string, 0)
	resultErrors := make([]string, 0, len(errors))
	resultErrors = append(resultErrors, errors...)

	for _, entry := range payload {
		if err := assertSafeArchivePath(entry.Path); err != nil {
			resultErrors = append(resultErrors, err.Error())
			continue
		}
		dest := filepath.Join(target, filepath.FromSlash(entry.Path))
		destClean := filepath.Clean(dest)
		targetClean := filepath.Clean(target)
		rel, err := filepath.Rel(targetClean, destClean)
		if err != nil || strings.HasPrefix(rel, "..") {
			resultErrors = append(resultErrors, fmt.Sprintf("unsafe import path: %s", entry.Path))
			continue
		}
		planned = append(planned, entry.Path)
		exists := false
		if _, err := os.Stat(dest); err == nil {
			exists = true
		}
		if exists && merge {
			skipped = append(skipped, entry.Path)
			continue
		}
		if exists && !force && !dryRun {
			conflicts = append(conflicts, entry.Path)
			continue
		}
		if !dryRun {
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				resultErrors = append(resultErrors, err.Error())
				continue
			}
			if err := os.WriteFile(dest, entry.Data, 0644); err != nil {
				resultErrors = append(resultErrors, err.Error())
				continue
			}
			written = append(written, entry.Path)
		}
	}

	if len(conflicts) > 0 {
		resultErrors = append(resultErrors, "protected files already exist; use --merge or --force")
	}

	return &FrozenImportResult{
		OK:        len(resultErrors) == 0,
		DryRun:    dryRun,
		Target:    target,
		Planned:   planned,
		Written:   written,
		Skipped:   skipped,
		Conflicts: conflicts,
		Errors:    resultErrors,
	}, nil
}
