package display

import (
	"bytes"
	"context"
	"image"
	"testing"
	"time"
)

func TestRenderBusBoard(t *testing.T) {
	renderer, err := NewBoardRenderer(nil)
	if err != nil {
		t.Fatalf("NewBoardRenderer: %v", err)
	}

	now := time.Date(2026, 3, 8, 8, 25, 0, 0, time.Local)

	departures := []BusDeparture{
		{
			Route:        "145",
			Destination:  "Heuston Station via Extremely Long Destination Avenue",
			ExpectedTime: now.Add(2 * time.Minute),
			MinutesAway:  2,
			IsRealtime:   true,
		},
		{
			Route:        "39A",
			Destination:  "Ongar",
			ExpectedTime: now.Add(11 * time.Minute),
			MinutesAway:  11,
			IsRealtime:   false,
		},
	}

	png, err := renderer.RenderBoard(context.Background(), now, PanelBus, departures)
	if err != nil {
		t.Fatalf("RenderBoard: %v", err)
	}

	decoded, _, err := image.DecodeConfig(bytes.NewReader(png))
	if err != nil {
		t.Fatalf("decoding rendered board: %v", err)
	}

	if decoded.Width != boardWidth || decoded.Height != boardHeight {
		t.Fatalf("board size = %dx%d, want %dx%d", decoded.Width, decoded.Height, boardWidth, boardHeight)
	}
}

func TestRenderBusBoardEmpty(t *testing.T) {
	renderer, err := NewBoardRenderer(nil)
	if err != nil {
		t.Fatalf("NewBoardRenderer: %v", err)
	}

	if _, err := renderer.RenderBoard(
		context.Background(),
		time.Now(),
		PanelBus,
		nil,
	); err != nil {
		t.Fatalf("RenderBoard with no departures: %v", err)
	}
}
