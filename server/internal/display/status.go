package display

import (
	"image/color"
	"time"
)

type arrivalStatus int

const (
	arrivalUnknown arrivalStatus = iota
	arrivalOnTime
	arrivalEarly
	arrivalLate
)

const statusThreshold = time.Minute

func arrivalStatusOf(
	expected time.Time,
	scheduled time.Time,
	realtime bool,
) arrivalStatus {
	if !realtime {
		return arrivalUnknown
	}

	delay := expected.Sub(scheduled)

	switch {
	case delay >= statusThreshold:
		return arrivalLate
	case delay <= -statusThreshold:
		return arrivalEarly
	default:
		return arrivalOnTime
	}
}

func withAlpha(c color.RGBA, alpha uint8) color.RGBA {
	c.A = alpha

	return c
}

func (s arrivalStatus) statusColor() color.RGBA {
	switch s {
	case arrivalOnTime:
		return color.RGBA{R: 74, G: 222, B: 128, A: 255}
	case arrivalEarly:
		return color.RGBA{R: 74, G: 222, B: 128, A: 255}
	case arrivalLate:
		return color.RGBA{R: 239, G: 68, B: 68, A: 255}
	default:
		return color.RGBA{R: 235, G: 240, B: 245, A: 255}
	}
}
