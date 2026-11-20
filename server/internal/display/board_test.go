package display

import (
	"bytes"
	"context"
	"image"
	"testing"
	"time"
)

func TestRenderBusBoard(t *testing.T) {
	renderer, err := NewBoardRenderer(nil, nil)
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

	png, err := renderer.RenderBoard(context.Background(), now, BoardView{
		Panel:      PanelBus,
		Departures: departures,
	})
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
	renderer, err := NewBoardRenderer(nil, nil)
	if err != nil {
		t.Fatalf("NewBoardRenderer: %v", err)
	}

	if _, err := renderer.RenderBoard(
		context.Background(),
		time.Now(),
		BoardView{Panel: PanelBus},
	); err != nil {
		t.Fatalf("RenderBoard with no departures: %v", err)
	}
}

func TestRenderMapBoard(t *testing.T) {
	renderer, err := NewBoardRenderer(nil, nil)
	if err != nil {
		t.Fatalf("NewBoardRenderer: %v", err)
	}

	now := time.Date(2026, 3, 8, 8, 25, 0, 0, time.Local)

	board := &MapBoard{
		StopNumber:   "475",
		StopPosition: MapPoint{Lat: 53.3127, Lon: -6.2019},
		Trips: []MapTrip{
			{
				Route:         "66",
				Destination:   "Maynooth",
				ShapeID:       "shape-1",
				Position:      MapPoint{Lat: 53.3150, Lon: -6.2100},
				Previous:      MapPoint{Lat: 53.3149, Lon: -6.2098},
				MinutesToStop: 4,
				IsRealtime:    true,
				Polyline: []MapPoint{
					{Lat: 53.3150, Lon: -6.2100},
					{Lat: 53.3130, Lon: -6.2020},
				},
				Stops: []MapPoint{
					{Lat: 53.3150, Lon: -6.2100},
					{Lat: 53.3130, Lon: -6.2020},
				},
			},
		},
	}

	png, err := renderer.RenderBoard(context.Background(), now, BoardView{
		Panel: PanelBusMap,
		Map:   board,
	})
	if err != nil {
		t.Fatalf("RenderBoard map: %v", err)
	}

	decoded, _, err := image.DecodeConfig(bytes.NewReader(png))
	if err != nil {
		t.Fatalf("decoding rendered board: %v", err)
	}

	if decoded.Width != boardWidth || decoded.Height != boardHeight {
		t.Fatalf("board size = %dx%d, want %dx%d", decoded.Width, decoded.Height, boardWidth, boardHeight)
	}
}

func TestRenderMapBoardEmpty(t *testing.T) {
	renderer, err := NewBoardRenderer(nil, nil)
	if err != nil {
		t.Fatalf("NewBoardRenderer: %v", err)
	}

	board := &MapBoard{
		StopNumber:   "475",
		StopPosition: MapPoint{Lat: 53.3127, Lon: -6.2019},
	}

	if _, err := renderer.RenderBoard(
		context.Background(),
		time.Now(),
		BoardView{Panel: PanelBusMap, Map: board},
	); err != nil {
		t.Fatalf("RenderBoard with no trips: %v", err)
	}
}

func TestRenderProgressPanel(t *testing.T) {
	renderer, err := NewBoardRenderer(nil, nil)
	if err != nil {
		t.Fatalf("NewBoardRenderer: %v", err)
	}

	now := time.Date(2026, 3, 8, 8, 25, 0, 0, time.Local)

	png, err := renderer.RenderBoard(context.Background(), now, BoardView{
		Panel: PanelBusProgress,
		Progress: &BusProgress{
			Route:            "66",
			Destination:      "Maynooth",
			StopNumber:       "475",
			ProgressFraction: 0.62,
			PreviousFraction: 0.61,
			StopFractions:    []float64{0, 0.3, 0.5},
			StopsRemaining:   4,
			ExpectedArrival:  now.Add(9 * time.Minute),
			ScheduledArrival: now.Add(7 * time.Minute),
			IsRealtime:       true,
		},
	})
	if err != nil {
		t.Fatalf("RenderBoard progress: %v", err)
	}

	decoded, _, err := image.DecodeConfig(bytes.NewReader(png))
	if err != nil {
		t.Fatalf("decoding rendered board: %v", err)
	}

	if decoded.Width != boardWidth || decoded.Height != boardHeight {
		t.Fatalf("board size = %dx%d, want %dx%d", decoded.Width, decoded.Height, boardWidth, boardHeight)
	}
}

func TestProgressDueLabel(t *testing.T) {
	now := time.Date(2026, 3, 8, 8, 9, 0, 0, time.Local)

	// The label counts from the render time, not the wall clock.
	if got := progressDueLabel(
		now,
		now.Add(17*time.Minute),
	); got != "Due in 17 min" {
		t.Errorf("progressDueLabel = %q, want %q", got, "Due in 17 min")
	}

	if got := progressDueLabel(now, now); got != "Arriving now" {
		t.Errorf("progressDueLabel = %q, want %q", got, "Arriving now")
	}
}

func TestRenderProgressPanelUnavailable(t *testing.T) {
	renderer, err := NewBoardRenderer(nil, nil)
	if err != nil {
		t.Fatalf("NewBoardRenderer: %v", err)
	}

	if _, err := renderer.RenderBoard(
		context.Background(),
		time.Now(),
		BoardView{Panel: PanelBusProgress, Progress: &BusProgress{}},
	); err != nil {
		t.Fatalf("RenderBoard with no buses to track: %v", err)
	}
}
