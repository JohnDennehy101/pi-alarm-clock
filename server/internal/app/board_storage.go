package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const boardSessionsMaxAge = 24 * time.Hour

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
		pruneSessionEntry(boardsRoot, entry, cutoff)
	}
}

func pruneSessionEntry(boardsRoot string, entry os.DirEntry, cutoff time.Time) {
	if !entry.IsDir() {
		return
	}

	info, err := entry.Info()
	if err != nil {
		return
	}

	if !info.ModTime().Before(cutoff) {
		return
	}

	if err := os.RemoveAll(filepath.Join(boardsRoot, entry.Name())); err != nil {
		log.Printf(
			"Boards: pruning old session %s: %v",
			entry.Name(),
			err,
		)
	}
}
