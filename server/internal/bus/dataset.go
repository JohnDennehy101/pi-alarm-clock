package bus

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	datasetURL    = "https://www.transportforireland.ie/transitData/Data/GTFS_All.zip"
	datasetName   = "GTFS.zip"
	datasetMaxAge = 7 * 24 * time.Hour
)

type DatasetManager struct {
	dataDir string
	client  *http.Client
}

func NewDatasetManager(dataDir string) *DatasetManager {
	return &DatasetManager{
		dataDir: dataDir,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (m *DatasetManager) Ensure(ctx context.Context) (string, error) {
	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return "", fmt.Errorf("create bus data directory: %w", err)
	}

	datasetPath := filepath.Join(m.dataDir, datasetName)

	if m.isUpToDate(datasetPath) {
		return datasetPath, nil
	}

	if err := m.download(ctx, datasetPath); err != nil {
		if validateDataset(datasetPath) == nil {
			return datasetPath, fmt.Errorf(
				"refresh GTFS dataset; using existing cache: %w",
				err,
			)
		}

		return "", err
	}

	return datasetPath, nil
}

func (m *DatasetManager) isUpToDate(datasetPath string) bool {
	info, err := os.Stat(datasetPath)
	if err != nil {
		return false
	}

	age := time.Since(info.ModTime())
	if age < 0 || age >= datasetMaxAge {
		return false
	}

	return validateDataset(datasetPath) == nil
}

func (m *DatasetManager) download(
	ctx context.Context,
	destination string,
) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		datasetURL,
		nil,
	)
	if err != nil {
		return fmt.Errorf("create GTFS request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("download GTFS dataset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"GTFS download returned status %s",
			resp.Status,
		)
	}

	tempFile, err := os.CreateTemp(m.dataDir, "gtfs-*.zip")
	if err != nil {
		return fmt.Errorf("create temporary GTFS file: %w", err)
	}

	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		tempFile.Close()
		return fmt.Errorf("save GTFS dataset: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close GTFS dataset: %w", err)
	}

	if err := validateDataset(tempPath); err != nil {
		return fmt.Errorf("validate downloaded GTFS dataset: %w", err)
	}

	if err := os.Rename(tempPath, destination); err != nil {
		return fmt.Errorf("replace GTFS dataset: %w", err)
	}

	return nil
}

func validateDataset(filename string) error {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return fmt.Errorf("open GTFS archive: %w", err)
	}
	defer reader.Close()

	required := map[string]bool{
		"agency.txt":     false,
		"routes.txt":     false,
		"trips.txt":      false,
		"stops.txt":      false,
		"stop_times.txt": false,
	}

	hasCalendar := false

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		name := path.Base(file.Name)

		if _, ok := required[name]; ok {
			required[name] = true
		}

		if name == "calendar.txt" || name == "calendar_dates.txt" {
			hasCalendar = true
		}
	}

	for name, found := range required {
		if !found {
			return fmt.Errorf("missing required GTFS file: %s", name)
		}
	}

	if !hasCalendar {
		return fmt.Errorf("missing calendar.txt and calendar_dates.txt")
	}

	return nil
}
