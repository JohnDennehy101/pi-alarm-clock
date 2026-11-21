package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const boardSessionsMaxAge = 24 * time.Hour

// saveBoardImage writes a board frame atomically: the cast device
// must never observe a half-written image.
func saveBoardImage(sessionDir string, name string, png []byte) error {
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("creating board session dir: %w", err)
	}

	tmpPath := filepath.Join(sessionDir, name+".tmp")

	if err := os.WriteFile(tmpPath, png, 0o644); err != nil {
		return fmt.Errorf("saving board frame %s: %w", name, err)
	}

	if err := os.Rename(tmpPath, filepath.Join(sessionDir, name)); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("saving board frame %s: %w", name, err)
	}

	return nil
}

func pruneOldSessions(boardsRoot string) {
	entries, err := os.ReadDir(boardsRoot)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-boardSessionsMaxAge)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.RemoveAll(filepath.Join(boardsRoot, entry.Name())); err != nil {
				log.Printf(
					"Boards: pruning old session %s: %v",
					entry.Name(),
					err,
				)
			}
		}
	}
}
