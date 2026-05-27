package episode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EpisodeChainResult represents chain validation outcome.
type EpisodeChainResult struct {
	OK              bool     `json:"ok"`
	TaskID          string   `json:"task_id"`
	EpisodesChecked int      `json:"episodes_checked"`
	Errors          []string `json:"errors"`
	EpisodeIDs      []string `json:"episode_ids"`
}

// episodeEntry holds a parsed manifest with its directory.
type episodeEntry struct {
	Dir      string
	Manifest EpisodeManifest
}

// VerifyEpisodeChain validates episode chain integrity for a task.
func VerifyEpisodeChain(taskID, episodesDir string) (*EpisodeChainResult, error) {
	result := &EpisodeChainResult{
		OK:         true,
		TaskID:     taskID,
		Errors:     []string{},
		EpisodeIDs: []string{},
	}

	// 1. Scan episodes directory for ep_* subdirectories
	entries, err := os.ReadDir(episodesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("failed to read episodes directory: %w", err)
	}

	var episodes []episodeEntry
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "ep_") {
			continue
		}
		dir := filepath.Join(episodesDir, entry.Name())
		manifestPath := filepath.Join(dir, "manifest.json")
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var manifest EpisodeManifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			continue
		}
		if manifest.TaskID == taskID {
			episodes = append(episodes, episodeEntry{Dir: dir, Manifest: manifest})
		}
	}

	// 3. If no episodes match, return empty chain (ok: true)
	if len(episodes) == 0 {
		return result, nil
	}

	// 4. Sort episodes by created_at
	sort.Slice(episodes, func(i, j int) bool {
		return episodes[i].Manifest.CreatedAt < episodes[j].Manifest.CreatedAt
	})

	idSet := make(map[string]bool)
	for _, ep := range episodes {
		idSet[ep.Manifest.EpisodeID] = true
	}

	// 5. For each episode: call validateEpisodeDirectory, collect errors
	for _, ep := range episodes {
		validation, err := validateEpisodeDirectory(ep.Dir)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: validation error: %v", ep.Manifest.EpisodeID, err))
		} else if !validation.OK {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", ep.Manifest.EpisodeID, strings.Join(validation.Errors, "; ")))
		}
		result.EpisodeIDs = append(result.EpisodeIDs, ep.Manifest.EpisodeID)

		// 6. Chain linkage validation: previous_episode_id must exist in set
		if ep.Manifest.PreviousEpisodeID != nil && *ep.Manifest.PreviousEpisodeID != "" {
			if !idSet[*ep.Manifest.PreviousEpisodeID] {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: missing previous episode %s", ep.Manifest.EpisodeID, *ep.Manifest.PreviousEpisodeID))
			}
		}
	}

	// 6b. Sequential ordering: episodes[i].previous_episode_id must equal episodes[i-1].episode_id
	for i := 1; i < len(episodes); i++ {
		expectedPrevious := episodes[i-1].Manifest.EpisodeID
		actualPrevious := episodes[i].Manifest.PreviousEpisodeID
		if actualPrevious == nil || *actualPrevious != expectedPrevious {
			var got string
			if actualPrevious == nil {
				got = "<nil>"
			} else {
				got = *actualPrevious
			}
			result.Errors = append(result.Errors, fmt.Sprintf("%s: previous_episode_id expected %s, got %s", episodes[i].Manifest.EpisodeID, expectedPrevious, got))
		}
	}

	result.EpisodesChecked = len(episodes)
	result.OK = len(result.Errors) == 0
	return result, nil
}
