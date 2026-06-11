package episode

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/evidence"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	"github.com/BrianNguyen29/x-harness/internal/trace"
)

// EpisodeValidationResult represents validation outcome.
type EpisodeValidationResult struct {
	OK        bool             `json:"ok"`
	EpisodeID *string          `json:"episode_id"`
	TaskID    *string          `json:"task_id"`
	Errors    []string         `json:"errors"`
	Warnings  []string         `json:"warnings"`
	Manifest  *EpisodeManifest `json:"manifest,omitempty"`
	FileCount int              `json:"file_count"`
}

// EpisodeManifest represents manifest.json structure.
type EpisodeManifest struct {
	SchemaVersion      string                 `json:"schema_version"`
	EpisodeID          string                 `json:"episode_id"`
	TaskID             string                 `json:"task_id"`
	CreatedAt          string                 `json:"created_at"`
	XHarnessVersion    string                 `json:"x_harness_version"`
	PreviousEpisodeID  *string                `json:"previous_episode_id"`
	Git                map[string]interface{} `json:"git"`
	PolicyHashes       map[string]interface{} `json:"policy_hashes"`
	SchemaHashes       map[string]interface{} `json:"schema_hashes"`
	Verdict            map[string]interface{} `json:"verdict"`
	MutationGuard      map[string]interface{} `json:"mutation_guard"`
	Signing            map[string]interface{} `json:"signing"`
	BundleRefs         map[string]interface{} `json:"bundle_refs"`
	AdmissionAuthority bool                   `json:"admission_authority"`
	HashesHash         string                 `json:"hashes_hash"`
	ManifestHash       string                 `json:"manifest_hash"`
}

// EpisodeFileHash represents a single file hash entry.
type EpisodeFileHash struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	SizeBytes int    `json:"size_bytes"`
}

// EpisodeHashes represents hashes.json structure.
type EpisodeHashes struct {
	SchemaVersion string            `json:"schema_version"`
	Files         []EpisodeFileHash `json:"files"`
}

// InspectEpisode validates an episode directory or bundle.
func InspectEpisode(path string) (*EpisodeValidationResult, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		resolved = path
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("episode not found: %s", resolved)
	}

	if info.IsDir() {
		return validateEpisodeDirectory(resolved)
	}

	if strings.HasSuffix(resolved, ".tar.gz") || strings.HasSuffix(resolved, ".tgz") {
		tempDir, err := os.MkdirTemp(os.TempDir(), "xh-episode-")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tempDir)

		if err := extractTarball(resolved, tempDir); err != nil {
			return nil, fmt.Errorf("failed to extract tarball: %w", err)
		}

		episodeDir, err := findEpisodeDir(tempDir)
		if err != nil {
			return nil, err
		}

		return validateEpisodeDirectory(episodeDir)
	}

	return nil, fmt.Errorf("episode path must be a directory or .tar.gz bundle")
}

func extractTarball(tarPath, destDir string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target, err := safeTarTarget(destDir, header.Name)
		if err != nil {
			return fmt.Errorf("tarball contains unsafe path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func safeTarTarget(destDir, name string) (string, error) {
	if strings.TrimSpace(name) == "" || filepath.IsAbs(name) {
		return "", fmt.Errorf("unsafe tar path")
	}
	cleanDest, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(cleanDest, name))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(cleanDest, target)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return target, nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes destination")
	}
	return target, nil
}

func findEpisodeDir(root string) (string, error) {
	var manifestPath string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "manifest.json" && !info.IsDir() {
			manifestPath = path
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if manifestPath == "" {
		return "", fmt.Errorf("episode bundle does not contain manifest.json")
	}
	return filepath.Dir(manifestPath), nil
}

func validateEpisodeDirectory(dir string) (*EpisodeValidationResult, error) {
	result := &EpisodeValidationResult{OK: true, Errors: []string{}, Warnings: []string{}}
	var errors []string
	var warnings []string

	// 1. Validate manifest.json
	manifestPath := filepath.Join(dir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		result.OK = false
		result.EpisodeID = nil
		result.TaskID = nil
		result.Errors = []string{"manifest.json not found"}
		result.Warnings = []string{}
		result.FileCount = 0
		return result, nil
	}

	var manifestMap map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifestMap); err != nil {
		result.OK = false
		result.Errors = []string{fmt.Sprintf("manifest.json parse error: %v", err)}
		result.Warnings = []string{}
		result.FileCount = 0
		return result, nil
	}

	// Schema validation
	root, err := repo.FindRoot("")
	if err != nil {
		return nil, fmt.Errorf("cannot find repository root: %w", err)
	}
	schemaPath := assets.NewLocator(root).Schema("episode-manifest.schema.json")
	validator, err := schema.Compile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("cannot compile schema: %w", err)
	}
	if err := validator.Validate(manifestMap); err != nil {
		errors = append(errors, fmt.Sprintf("schema validation: %v", err))
	}

	// Manifest hash verification
	manifestHashValue, _ := manifestMap["manifest_hash"].(string)
	delete(manifestMap, "manifest_hash")
	canonicalBytes, err := json.Marshal(manifestMap)
	if err != nil {
		return nil, fmt.Errorf("cannot canonicalize manifest: %w", err)
	}
	expectedManifestHash := fmt.Sprintf("sha256:%x", sha256.Sum256(canonicalBytes))
	if manifestHashValue != expectedManifestHash {
		errors = append(errors, fmt.Sprintf("manifest_hash mismatch: expected %s, got %s", expectedManifestHash, manifestHashValue))
	}

	// Restore manifest_hash for manifest output
	manifestMap["manifest_hash"] = manifestHashValue

	var manifest EpisodeManifest
	manifestBytes, _ := json.Marshal(manifestMap)
	_ = json.Unmarshal(manifestBytes, &manifest)

	episodeID := manifest.EpisodeID
	taskID := manifest.TaskID

	// 2. Validate hashes.json
	hashesPath := filepath.Join(dir, "hashes.json")
	hashesData, err := os.ReadFile(hashesPath)
	if err != nil {
		errors = append(errors, "hashes.json not found")
	} else {
		var hashes EpisodeHashes
		if err := json.Unmarshal(hashesData, &hashes); err != nil {
			errors = append(errors, fmt.Sprintf("hashes.json parse error: %v", err))
		} else {
			// Verify hashes_hash
			hashesHashValue := manifest.HashesHash
			actualHashesHash := fmt.Sprintf("sha256:%x", sha256.Sum256(hashesData))
			if hashesHashValue != actualHashesHash {
				errors = append(errors, fmt.Sprintf("hashes_hash mismatch: expected %s, got %s", hashesHashValue, actualHashesHash))
			}

			// Validate each file entry
			seen := make(map[string]bool)
			declared := make(map[string]bool)
			for _, file := range hashes.Files {
				if !isSafeEpisodeRelativePath(file.Path) {
					errors = append(errors, fmt.Sprintf("unsafe hashed file path: %s", file.Path))
					continue
				}
				if seen[file.Path] {
					errors = append(errors, fmt.Sprintf("duplicate hashed file path: %s", file.Path))
					continue
				}
				seen[file.Path] = true
				declared[file.Path] = true

				fullPath := filepath.Join(dir, file.Path)
				fileData, err := os.ReadFile(fullPath)
				if err != nil {
					errors = append(errors, fmt.Sprintf("hashed file missing: %s", file.Path))
					continue
				}
				actualHash := fmt.Sprintf("sha256:%x", sha256.Sum256(fileData))
				if actualHash != file.SHA256 {
					errors = append(errors, fmt.Sprintf("hash mismatch for %s: expected %s, got %s", file.Path, file.SHA256, actualHash))
				}
			}

			// Check for orphan files
			actualFiles, err := collectFiles(dir)
			if err != nil {
				return nil, fmt.Errorf("failed to collect files: %w", err)
			}
			for _, file := range actualFiles {
				rel := relativeTo(dir, file)
				if rel == "hashes.json" || rel == "manifest.json" {
					continue
				}
				if !declared[rel] {
					errors = append(errors, fmt.Sprintf("unhashed episode file: %s", rel))
				}
			}

			result.FileCount = len(hashes.Files)
		}
	}

	// 3. Validate trace.jsonl
	tracePath := filepath.Join(dir, "trace.jsonl")
	events, err := trace.ReadTraceFromFile(tracePath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("trace.jsonl read error: %v", err))
	} else if events == nil {
		errors = append(errors, "trace.jsonl not found")
	} else {
		traceResult := trace.VerifyTraceChain(events)
		if !traceResult.Valid {
			firstBroken := 0
			if traceResult.FirstBrokenIndex != nil {
				firstBroken = *traceResult.FirstBrokenIndex
			}
			errors = append(errors, fmt.Sprintf("trace chain broken at index %d", firstBroken))
		}
	}

	// 4. Validate evidence-index.jsonl (optional)
	evidenceIndexPath := filepath.Join(dir, "evidence-index.jsonl")
	if _, err := os.Stat(evidenceIndexPath); err == nil {
		ok, evidenceErrors, _, err := evidence.ValidateIndexFile(evidenceIndexPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("evidence index read error: %v", err))
		} else if !ok {
			errors = append(errors, fmt.Sprintf("evidence index invalid: %s", strings.Join(evidenceErrors, "; ")))
		}
	} else {
		warnings = append(warnings, "evidence-index.jsonl not found")
	}

	result.OK = len(errors) == 0
	result.EpisodeID = &episodeID
	result.TaskID = &taskID
	if errors == nil {
		result.Errors = []string{}
	} else {
		result.Errors = errors
	}
	if warnings == nil {
		result.Warnings = []string{}
	} else {
		result.Warnings = warnings
	}
	result.Manifest = &manifest

	return result, nil
}

func isSafeEpisodeRelativePath(filePath string) bool {
	if filePath == "" || filepath.IsAbs(filePath) {
		return false
	}
	normalized := filepath.ToSlash(filepath.Clean(filePath))
	if normalized == "." {
		return false
	}
	if strings.HasPrefix(normalized, "../") {
		return false
	}
	if strings.Contains(normalized, "/../") {
		return false
	}
	return normalized == filepath.ToSlash(filePath)
}

func collectFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
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

func relativeTo(root, filePath string) string {
	rel, err := filepath.Rel(root, filePath)
	if err != nil {
		return filePath
	}
	return filepath.ToSlash(rel)
}
