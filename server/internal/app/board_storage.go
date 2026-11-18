package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const boardSessionsMaxAge = 24 * time.Hour

func saveBoardImages(sessionDir string, images [][]byte) error {
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("creating board session dir: %w", err)
	}

	for minute, png := range images {
		name := fmt.Sprintf("minute-%02d.png", minute)

		if err := os.WriteFile(
			filepath.Join(sessionDir, name),
			png,
			0o644,
		); err != nil {
			return fmt.Errorf("saving board minute %d: %w", minute, err)
		}
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
