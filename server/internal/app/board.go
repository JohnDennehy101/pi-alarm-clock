package app

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
)

type BoardService struct {
	mu sync.Mutex

	renderer   *display.BoardRenderer
	boardsRoot string
	baseURL    string
}

type BoardProvider interface {
	BoardURLs(ctx context.Context, start time.Time, minutes int) ([]string, error)
}

func NewBoardService(boardsRoot string) *BoardService {
	return &BoardService{
		boardsRoot: boardsRoot,
	}
}

func (b *BoardService) setRenderer(renderer *display.BoardRenderer) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.renderer = renderer
}

func (b *BoardService) setBaseURL(url string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.baseURL = url
}

func (b *BoardService) BoardURLs(
	ctx context.Context,
	start time.Time,
	minutes int,
) ([]string, error) {
	if minutes < 1 {
		minutes = 1
	}

	b.mu.Lock()
	renderer := b.renderer
	baseURL := b.baseURL
	boardsRoot := b.boardsRoot
	b.mu.Unlock()

	if renderer == nil {
		return nil, fmt.Errorf("board renderer is not configured")
	}

	if baseURL == "" {
		return nil, fmt.Errorf("board URL is not configured")
	}

	images, err := renderBatch(ctx, renderer, start, minutes)
	if err != nil {
		return nil, err
	}

	session := start.Format("20060102T1504")
	sessionDir := filepath.Join(boardsRoot, session)

	if err := saveBoardImages(sessionDir, images); err != nil {
		return nil, err
	}

	urls := make([]string, 0, minutes)

	for minute := range images {
		name := fmt.Sprintf("minute-%02d.png", minute)
		urls = append(urls, fmt.Sprintf("%s/%s/%s", baseURL, session, name))
	}

	pruneOldSessions(boardsRoot)

	return urls, nil
}

func (b *BoardService) Handler() http.Handler {
	b.mu.Lock()
	renderer := b.renderer
	boardsRoot := b.boardsRoot
	b.mu.Unlock()

	return display.Handler(renderer, boardsRoot)
}
