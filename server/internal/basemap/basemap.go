package basemap

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	cacheMaxAge   = 6 * time.Hour
	imageMaxBytes = 8 << 20
)

type MapboxProvider struct {
	token    string
	cacheDir string
	client   *http.Client

	mu     sync.Mutex
	memory map[string][]byte
}

func NewMapboxProvider(token, cacheDir string) *MapboxProvider {
	return &MapboxProvider{
		token:    token,
		cacheDir: cacheDir,
		client:   &http.Client{Timeout: 15 * time.Second},
		memory:   make(map[string][]byte),
	}
}

func (p *MapboxProvider) FetchBasemap(
	ctx context.Context,
	lat float64,
	lon float64,
	zoom float64,
	width int,
	height int,
) (image.Image, error) {
	key := fmt.Sprintf(
		"mapbox-dark-%.5f-%.5f-%.1f-%dx%d.png",
		lat,
		lon,
		zoom,
		width,
		height,
	)

	url := fmt.Sprintf(
		"https://api.mapbox.com/styles/v1/mapbox/dark-v11/static/"+
			"%s,%s,%s/%dx%d?access_token=%s",
		formatCoord(lon),
		formatCoord(lat),
		strconv.FormatFloat(zoom, 'f', 1, 64),
		width,
		height,
		p.token,
	)

	data, err := p.load(ctx, key, url)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decoding basemap image: %w", err)
	}

	return img, nil
}

func (p *MapboxProvider) load(ctx context.Context, key, url string) ([]byte, error) {
	p.mu.Lock()
	cached, inMemory := p.memory[key]
	p.mu.Unlock()

	if inMemory {
		return cached, nil
	}

	path := filepath.Join(p.cacheDir, key)

	if info, err := os.Stat(path); err == nil {
		if data, err := os.ReadFile(path); err == nil {
			if time.Since(info.ModTime()) <= cacheMaxAge {
				p.cacheInMemory(key, data)
				return data, nil
			}
		}
	}

	data, err := fetch(ctx, p.client, url)
	if err != nil {
		if stale, readErr := os.ReadFile(path); readErr == nil {
			return stale, nil
		}

		return nil, err
	}

	if err := os.MkdirAll(p.cacheDir, 0o755); err != nil {
		return data, nil
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return data, nil
	}

	p.cacheInMemory(key, data)

	return data, nil
}

func (p *MapboxProvider) cacheInMemory(key string, data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.memory[key] = data
}

func fetch(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating basemap request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting basemap: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("basemap request returned status %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, imageMaxBytes))
	if err != nil {
		return nil, fmt.Errorf("reading basemap image: %w", err)
	}

	return data, nil
}

func formatCoord(value float64) string {
	return strconv.FormatFloat(value, 'f', 5, 64)
}
