package boundary

import (
	"os"
	"path/filepath"
	"strings"
)

// collectCandidateFiles walks dir and returns the candidate source files
// for boundary checking. The walk mirrors the contract package's
// conventions:
//
//   - Skip binary files by extension.
//   - Skip files larger than 1 MB.
//   - Skip generated/vendor directories (best-effort heuristics).
//
// The candidate file set is the union of files with a known source
// extension. Hidden files (starting with `.`) are skipped except for
// `.github` workflows, which are kept because CI YAML is sometimes
// reviewed against boundary rules; for V1, workflows are out of scope
// and the walk is permissive.
func collectCandidateFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			// Skip common noise directories at any depth.
			if name == "node_modules" || name == "vendor" || name == ".git" || name == "dist" || name == "build" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".go":
			// supported
		default:
			return nil
		}
		// Skip files larger than 1MB.
		if info.Size() > 1024*1024 {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
