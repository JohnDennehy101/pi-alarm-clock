package display

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/fogleman/gg"
)

type BusProgress struct {
	Route            string
	Destination      string
	StopNumber       string
	ProgressFraction float64
	PreviousFraction float64
	StopFractions    []float64
	StopsRemaining   int
	ExpectedArrival  time.Time
	ScheduledArrival time.Time
	IsRealtime       bool
}

const (
	progressTrackStartX = 100.0
	progressTrackEndX   = float64(boardWidth) - 100
	progressTrackY      = 470.0
)

func (r *BoardRenderer) drawBusProgress(
	dc *gg.Context,
	now time.Time,
	progress *BusProgress,
) {
	if progress == nil || progress.Route == "" {
		r.drawProgressUnavailable(dc)
		return
	}

	status := arrivalStatusOf(
		progress.ExpectedArrival,
		progress.ScheduledArrival,
		progress.IsRealtime,
	)
	markerColor := status.statusColor()

	dc.SetColor(markerColor)
	dc.DrawRoundedRectangle(100, 348, 130, 70, 10)
	dc.Fill()

	dc.SetFontFace(r.face(54))
	dc.SetColor(color.RGBA{R: 13, G: 15, B: 20, A: 255})
	dc.DrawStringAnchored(progress.Route, 165, 383, 0.5, 0.36)

	dc.SetFontFace(r.face(40))
	dc.SetColor(color.RGBA{R: 154, G: 164, B: 178, A: 255})
	dc.DrawString(fmt.Sprintf("to %s", progress.Destination), 250, 390)

	r.drawProgressTrack(dc, progress, markerColor)

	r.drawProgressReadout(dc, now, progress, status)
}

func (r *BoardRenderer) drawProgressTrack(
	dc *gg.Context,
	progress *BusProgress,
	markerColor color.Color,
) {
	x1, x2 := progressTrackStartX, progressTrackEndX
	width := x2 - x1

	fraction := clamp01(progress.ProgressFraction)
	previous := clamp01(progress.PreviousFraction)

	dc.SetColor(color.RGBA{R: 40, G: 46, B: 58, A: 255})
	dc.SetLineWidth(3)
	dc.DrawLine(x1, progressTrackY, x2, progressTrackY)
	dc.Stroke()

	dc.SetColor(markerColor)
	dc.SetLineWidth(4)
	dc.DrawLine(x1, progressTrackY, x1+width*fraction, progressTrackY)
	dc.Stroke()

	dc.SetFontFace(r.face(24))
	for _, stopFraction := range progress.StopFractions {
		stopFraction = clamp01(stopFraction)
		if x := x1 + width*stopFraction; x < x2-14 {
			dc.SetColor(color.RGBA{R: 120, G: 130, B: 150, A: 180})
			dc.DrawCircle(x, progressTrackY, 3)
			dc.Fill()
		}
	}

	dc.SetFontFace(r.face(24))
	dc.SetColor(color.RGBA{R: 130, G: 140, B: 155, A: 255})
	dc.DrawStringAnchored("route start", x1, progressTrackY+26, 0, 0.5)

	dc.SetColor(color.RGBA{R: 235, G: 240, B: 245, A: 255})
	dc.SetLineWidth(3)
	dc.DrawRectangle(x2-9, progressTrackY-9, 18, 18)
	dc.Stroke()

	dc.SetColor(color.RGBA{R: 20, G: 23, B: 30, A: 235})
	dc.DrawRoundedRectangle(x2-70, progressTrackY-58, 140, 34, 8)
	dc.Fill()

	dc.SetFontFace(r.face(24))
	dc.SetColor(color.RGBA{R: 235, G: 240, B: 245, A: 255})
	dc.DrawStringAnchored(
		fmt.Sprintf("Stop %s", progress.StopNumber),
		x2,
		progressTrackY-41,
		1,
		0.5,
	)

	dc.DrawStringAnchored("your stop", x2, progressTrackY+26, 1, 0.5)

	previousX := x1 + width*previous
	dc.SetColor(color.RGBA{R: 96, G: 165, B: 250, A: 90})
	dc.SetLineWidth(3)
	dc.DrawLine(previousX, progressTrackY-10, previousX, progressTrackY+10)
	dc.Stroke()

	currentX := x1 + width*fraction

	if previousX > 0 && previousX != currentX {
		dc.SetColor(color.RGBA{R: 96, G: 165, B: 250, A: 90})
		dc.SetLineWidth(2)
		dc.SetDash(6, 5)
		dc.DrawLine(previousX, progressTrackY, currentX, progressTrackY)
		dc.Stroke()
		dc.SetDash()
	}

	dc.SetColor(markerColor)
	dc.DrawCircle(currentX, progressTrackY, 9)
	dc.Fill()

	dc.SetColor(color.RGBA{R: 13, G: 15, B: 20, A: 255})
	dc.DrawCircle(currentX, progressTrackY, 3)
	dc.Fill()
}

func (r *BoardRenderer) drawProgressReadout(
	dc *gg.Context,
	now time.Time,
	progress *BusProgress,
	status arrivalStatus,
) {
	due := progressDueLabel(now, progress.ExpectedArrival)

	dc.SetFontFace(r.face(60))
	dc.SetColor(color.RGBA{R: 235, G: 240, B: 245, A: 255})
	dc.DrawString(due, 100, 590)

	dc.SetFontFace(r.face(34))
	dc.SetColor(color.RGBA{R: 154, G: 164, B: 178, A: 255})
	dc.DrawString(
		fmt.Sprintf(
			"Scheduled %s · Expected %s",
			progress.ScheduledArrival.Format("15:04"),
			progress.ExpectedArrival.Format("15:04"),
		),
		100,
		636,
	)

	delayText := "on schedule"
	if !progress.ExpectedArrival.Equal(progress.ScheduledArrival) {
		delay := progress.ExpectedArrival.Sub(progress.ScheduledArrival)
		if delay > 0 {
			delayText = fmt.Sprintf("%d min late", int(
				delay.Minutes()+0.5,
			))
		} else {
			delayText = fmt.Sprintf("%d min early", int(
				(-delay).Minutes()+0.5,
			))
		}
	}

	chipText := delayText
	if status == arrivalUnknown {
		chipText = delayText + " (scheduled)"
	}

	dc.SetFontFace(r.face(28))

	chipWidth, _ := dc.MeasureString(chipText)
	chipWidth += 28

	chipColor := status.statusColor()
	if status == arrivalUnknown {
		chipColor = color.RGBA{R: 40, G: 46, B: 58, A: 255}
	}

	dc.SetColor(chipColor)
	dc.DrawRoundedRectangle(
		float64(boardWidth)-100-chipWidth,
		560,
		chipWidth,
		40,
		8,
	)
	dc.Fill()

	dc.SetColor(color.RGBA{R: 235, G: 240, B: 245, A: 255})
	dc.DrawStringAnchored(
		chipText,
		float64(boardWidth)-100-chipWidth/2,
		580,
		0.5,
		0.36,
	)

	dc.SetFontFace(r.face(28))
	dc.SetColor(color.RGBA{R: 130, G: 140, B: 155, A: 255})
	remaining := fmt.Sprintf(
		"%d stops to Stop %s",
		progress.StopsRemaining,
		progress.StopNumber,
	)
	if progress.StopsRemaining == 1 {
		remaining = fmt.Sprintf(
			"next stop is Stop %s",
			progress.StopNumber,
		)
	} else if progress.StopsRemaining == 0 {
		remaining = fmt.Sprintf("at Stop %s", progress.StopNumber)
	}

	dc.DrawStringAnchored(remaining, float64(boardWidth)-100, 636, 1, 0)
}

func (r *BoardRenderer) drawProgressUnavailable(dc *gg.Context) {
	dc.SetFontFace(r.face(40))
	dc.SetColor(color.RGBA{R: 130, G: 140, B: 155, A: 255})
	dc.DrawStringAnchored(
		"No buses to track",
		boardWidth/2,
		480,
		0.5,
		0.5,
	)

	dc.SetFontFace(r.face(26))
	dc.SetColor(color.RGBA{R: 90, G: 98, B: 112, A: 255})
	dc.DrawStringAnchored(
		"no approaching buses within the tracking window",
		boardWidth/2,
		530,
		0.5,
		0.5,
	)
}

func progressDueLabel(now, expected time.Time) string {
	minutes := int(math.Ceil(expected.Sub(now).Minutes()))
	if minutes > 0 {
		return fmt.Sprintf("Due in %d min", minutes)
	}

	return "Arriving now"
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}

	if value > 1 {
		return 1
	}

	return value
}
