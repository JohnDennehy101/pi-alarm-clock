package display

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/fogleman/gg"
)

type BasemapProvider interface {
	FetchBasemap(
		ctx context.Context,
		lat float64,
		lon float64,
		zoom float64,
		width int,
		height int,
	) (image.Image, error)
}

type MapPoint struct {
	Lat float64
	Lon float64
}

type MapTrip struct {
	Route            string
	Destination      string
	ShapeID          string
	Position         MapPoint
	Previous         MapPoint
	MinutesToStop    int
	IsRealtime       bool
	Arrival          time.Time
	ScheduledArrival time.Time
	Polyline         []MapPoint
	Stops            []MapPoint
}

type MapBoard struct {
	StopNumber   string
	StopPosition MapPoint
	Trips        []MapTrip
}

type mapProjection struct {
	zoom         float64
	centerLat    float64
	lon          float64
	centerX      float64
	centerY      float64
	worldPixels  float64
	pixelPadding float64
}

const (
	mapMinZoom         = 12.0
	mapMaxZoom         = 17.0
	mapZoomStep        = 0.5
	badgeHeight        = 44
	badgeMinWidth      = 58
	badgeHStep         = float64(badgeHeight) + 14
	badgeMaxAttempts   = 5
	badgeStandoff      = 12.0
	minutesLabelOffset = 18.0
)

func newMapProjection(
	center MapPoint,
	points []MapPoint,
	width float64,
	height float64,
) *mapProjection {
	zoom := 15.0
	if len(points) > 0 {
		for candidate := mapMaxZoom; candidate >= mapMinZoom; candidate -= mapZoomStep {
			zoom = candidate

			if fitsMapPoints(center, points, candidate, width, height) {
				break
			}
		}
	}

	zoom = quantizeMapZoom(zoom)
	projection := &mapProjection{
		zoom:        zoom,
		centerLat:   center.Lat,
		lon:         center.Lon,
		worldPixels: worldPixelsAt(zoom),
	}

	projection.centerX, projection.centerY = projection.mercator(
		center.Lat,
		center.Lon,
	)

	return projection
}

func quantizeMapZoom(zoom float64) float64 {
	return math.Round(zoom/mapZoomStep) * mapZoomStep
}

func worldPixelsAt(zoom float64) float64 {
	return 256 * math.Pow(2, zoom)
}

func fitsMapPoints(
	center MapPoint,
	points []MapPoint,
	zoom float64,
	width float64,
	height float64,
) bool {
	world := worldPixelsAt(zoom)

	cx, cy := mercatorAt(center.Lat, center.Lon, world)

	maxDX := math.Min(width/2-70, 460)
	maxDY := height/2 - 70

	for _, point := range points {
		px, py := mercatorAt(point.Lat, point.Lon, world)

		if math.Abs(px-cx) > maxDX || math.Abs(py-cy) > maxDY {
			return false
		}
	}

	return true
}

func mercatorAt(lat, lon float64, world float64) (float64, float64) {
	x := (lon + 180) / 360 * world

	latRad := lat * math.Pi / 180
	y := (1 - math.Log(math.Tan(latRad)+1/math.Cos(latRad))/math.Pi) / 2 * world

	return x, y
}

func (p *mapProjection) mercator(lat, lon float64) (float64, float64) {
	return mercatorAt(lat, lon, p.worldPixels)
}

func (p *mapProjection) project(point MapPoint) (float64, float64) {
	x, y := p.mercator(point.Lat, point.Lon)

	return boardWidth/2 + (x - p.centerX), float64(boardHeight)/2 + (y - p.centerY)
}

func (r *BoardRenderer) drawBusMap(
	ctx context.Context,
	dc *gg.Context,
	board *MapBoard,
) {
	if board == nil {
		r.drawMapUnavailable(dc, "no bus map data")
		return
	}

	points := make([]MapPoint, 0, len(board.Trips)*2)
	for _, trip := range board.Trips {
		points = append(points, trip.Position, trip.Previous)
	}

	projection := newMapProjection(
		board.StopPosition,
		points,
		boardWidth,
		boardHeight,
	)

	if !r.drawBasemap(ctx, dc, board, projection) {
		drawMapGridBackdrop(dc)
	}

	r.drawMapRoutes(dc, board, projection)
	r.drawMapStops(dc, board, projection)

	stopX, stopY := projection.project(board.StopPosition)
	avoid := r.mapStopRects(dc, board, stopX, stopY)

	r.drawMapBuses(dc, board.Trips, projection, avoid)
	r.drawMapStopMarker(dc, board, stopX, stopY)

	dc.SetFontFace(r.face(20))
	dc.SetColor(color.RGBA{R: 130, G: 140, B: 155, A: 180})
	dc.DrawStringAnchored(
		"© Mapbox · © OpenStreetMap",
		boardWidth-12,
		boardHeight-10,
		1,
		1,
	)
}

func (r *BoardRenderer) drawBasemap(
	ctx context.Context,
	dc *gg.Context,
	board *MapBoard,
	projection *mapProjection,
) bool {
	if r.basemap == nil {
		return false
	}

	basemap, err := r.basemap.FetchBasemap(
		ctx,
		board.StopPosition.Lat,
		board.StopPosition.Lon,
		projection.zoom,
		boardWidth,
		boardHeight,
	)
	if err != nil || basemap == nil {
		return false
	}

	dc.DrawImage(basemap, 0, 0)

	return true
}

func drawMapGridBackdrop(dc *gg.Context) {
	dc.SetColor(color.RGBA{R: 13, G: 15, B: 20, A: 255})
	dc.Clear()

	dc.SetColor(color.RGBA{R: 30, G: 34, B: 44, A: 255})
	dc.SetLineWidth(1)

	for x := 0; x < boardWidth; x += 64 {
		dc.DrawLine(float64(x), 330, float64(x), boardHeight)
		dc.Stroke()
	}

	for y := 320; y < boardHeight; y += 64 {
		dc.DrawLine(0, float64(y), boardWidth, float64(y))
		dc.Stroke()
	}
}

func (r *BoardRenderer) drawMapRoutes(
	dc *gg.Context,
	board *MapBoard,
	projection *mapProjection,
) {
	drawn := make(map[string]bool)

	for _, trip := range board.Trips {
		if drawn[trip.ShapeID] || len(trip.Polyline) < 2 {
			continue
		}

		drawn[trip.ShapeID] = true

		dc.SetColor(color.RGBA{R: 70, G: 78, B: 95, A: 200})
		dc.SetLineWidth(4)

		first := true
		for _, point := range trip.Polyline {
			x, y := projection.project(point)
			if first {
				dc.MoveTo(x, y)
				first = false
				continue
			}

			dc.LineTo(x, y)
		}

		dc.Stroke()
	}
}

func (r *BoardRenderer) drawMapStops(
	dc *gg.Context,
	board *MapBoard,
	projection *mapProjection,
) {
	drawn := make(map[string]bool)

	for _, trip := range board.Trips {
		for _, stop := range trip.Stops {
			key := fmt.Sprintf("%.5f,%.5f", stop.Lat, stop.Lon)
			if drawn[key] {
				continue
			}

			drawn[key] = true

			x, y := projection.project(stop)

			dc.SetColor(color.RGBA{R: 120, G: 130, B: 150, A: 160})
			dc.DrawCircle(x, y, 3)
			dc.Fill()
		}
	}
}

func (r *BoardRenderer) mapStopRects(
	dc *gg.Context,
	board *MapBoard,
	x float64,
	y float64,
) []mapRect {
	dc.SetFontFace(r.face(30))
	textWidth, _ := dc.MeasureString(board.StopNumber)

	chipWidth := textWidth + 18
	chipHeight := 40.0
	chipTop := y - 26 - chipHeight

	return []mapRect{
		{x0: x - 14, y0: y - 14, x1: x + 14, y1: y + 14},
		{x0: x - chipWidth/2, y0: chipTop, x1: x + chipWidth/2, y1: chipTop + chipHeight},
	}
}

func (r *BoardRenderer) drawMapStopMarker(
	dc *gg.Context,
	board *MapBoard,
	x float64,
	y float64,
) {
	dc.SetColor(color.RGBA{R: 235, G: 240, B: 245, A: 255})
	dc.SetLineWidth(4)
	dc.DrawRectangle(x-9, y-9, 18, 18)
	dc.Stroke()

	dc.SetColor(color.RGBA{R: 13, G: 15, B: 20, A: 200})
	dc.DrawRectangle(x-7, y-7, 14, 14)
	dc.Fill()

	dc.SetFontFace(r.face(30))
	textWidth, _ := dc.MeasureString(board.StopNumber)

	chipWidth := textWidth + 18
	chipHeight := 40.0
	chipTop := y - 26 - chipHeight

	dc.SetColor(color.RGBA{R: 20, G: 23, B: 30, A: 235})
	dc.DrawRoundedRectangle(
		x-chipWidth/2,
		chipTop,
		chipWidth,
		chipHeight,
		8,
	)
	dc.Fill()

	dc.SetColor(color.RGBA{R: 235, G: 240, B: 245, A: 255})
	dc.DrawStringAnchored(board.StopNumber, x, chipTop+chipHeight/2, 0.5, 0.42)
}

type mapRect struct {
	x0, y0, x1, y1 float64
}

func (r mapRect) overlaps(other mapRect) bool {
	return r.x0 < other.x1 && r.x1 > other.x0 &&
		r.y0 < other.y1 && r.y1 > other.y0
}

func (r *BoardRenderer) drawMapBuses(
	dc *gg.Context,
	trips []MapTrip,
	projection *mapProjection,
	avoid []mapRect,
) {
	routeFace := r.face(34)

	placed := append([]mapRect(nil), avoid...)

	for _, trip := range trips {
		prevX, prevY := projection.project(trip.Previous)
		x, y := projection.project(trip.Position)

		status := arrivalStatusOf(
			trip.Arrival,
			trip.ScheduledArrival,
			trip.IsRealtime,
		)
		markerColor := status.statusColor()
		ghostColor := withAlpha(markerColor, 90)

		if x != prevX || y != prevY {
			dc.SetColor(ghostColor)
			dc.SetLineWidth(2)
			dc.SetDash(6, 5)
			dc.DrawLine(prevX, prevY, x, y)
			dc.Stroke()
			dc.SetDash()

			dc.SetColor(ghostColor)
			dc.DrawCircle(prevX, prevY, 7)
			dc.Fill()
		}

		dc.SetColor(markerColor)
		dc.DrawCircle(x, y, 5)
		dc.Fill()

		dc.SetFontFace(routeFace)
		textWidth, _ := dc.MeasureString(trip.Route)

		badgeWidth := math.Max(
			badgeMinWidth,
			textWidth+22,
		)

		var chosenRect mapRect

		for attempt := 0; attempt < badgeMaxAttempts; attempt++ {
			offset := 0.0
			if attempt > 0 {
				offset = float64((attempt+1)/2) * badgeHStep
				if attempt%2 == 0 {
					offset = -offset
				}
			}

			candidate := mapRect{
				x0: x - badgeWidth/2,
				y0: y + offset - badgeHeight/2 - badgeStandoff,
				x1: x + badgeWidth/2,
				y1: y + offset + badgeHeight/2 - badgeStandoff,
			}

			free := true
			for _, taken := range placed {
				if candidate.overlaps(taken) {
					free = false
					break
				}
			}

			if free {
				chosenRect = candidate
				break
			}

			if attempt == badgeMaxAttempts-1 {
				chosenRect = candidate
			}
		}

		placed = append(placed, chosenRect)

		badgeX := (chosenRect.x0 + chosenRect.x1) / 2
		badgeY := (chosenRect.y0 + chosenRect.y1) / 2

		leaderY := chosenRect.y1
		if badgeY < y {
			leaderY = chosenRect.y0
		}

		if leaderY != y {
			dc.SetColor(color.RGBA{R: 154, G: 164, B: 178, A: 200})
			dc.SetLineWidth(2)
			dc.DrawLine(x, y, badgeX, leaderY)
			dc.Stroke()
		}

		dc.SetColor(markerColor)
		dc.DrawRoundedRectangle(
			chosenRect.x0,
			chosenRect.y0,
			badgeWidth,
			badgeHeight,
			7,
		)
		dc.Fill()

		dc.SetFontFace(routeFace)
		dc.SetColor(color.RGBA{R: 13, G: 15, B: 20, A: 255})
		dc.DrawStringAnchored(trip.Route, badgeX, badgeY, 0.5, 0.46)

		label := "due"
		if trip.MinutesToStop > 0 {
			label = fmt.Sprintf("%d min", trip.MinutesToStop)
		}

		dc.SetFontFace(r.face(26))
		dc.SetColor(color.RGBA{R: 154, G: 164, B: 178, A: 255})

		if badgeY < y {
			dc.DrawStringAnchored(
				label,
				badgeX,
				chosenRect.y0-minutesLabelOffset,
				0.5,
				0.5,
			)
		} else {
			dc.DrawStringAnchored(
				label,
				badgeX,
				chosenRect.y1+minutesLabelOffset,
				0.5,
				0.5,
			)
		}
	}
}

func (r *BoardRenderer) drawMapUnavailable(
	dc *gg.Context,
	reason string,
) {
	dc.SetFontFace(r.face(40))
	dc.SetColor(color.RGBA{R: 130, G: 140, B: 155, A: 255})
	dc.DrawStringAnchored(
		"Bus map unavailable",
		boardWidth/2,
		480,
		0.5,
		0.5,
	)

	dc.SetFontFace(r.face(26))
	dc.SetColor(color.RGBA{R: 90, G: 98, B: 112, A: 255})
	dc.DrawStringAnchored(reason, boardWidth/2, 530, 0.5, 0.5)
}
