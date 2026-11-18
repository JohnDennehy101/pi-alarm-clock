package display

import (
	"math"

	"github.com/fogleman/gg"
)

const (
	iconSun    = "#F5B841"
	iconMoon   = "#CBD5E1"
	iconCloud  = "#94A3B8"
	iconPrecip = "#60A5FA"
	iconSnow   = "#E5EAF2"
	iconBolt   = "#FCD34D"

	boardBackgroundHex = "#0D0F14"
)

type weatherKind int

const (
	kindClear weatherKind = iota
	kindPartly
	kindCloudy
	kindFog
	kindRain
	kindSnow
	kindThunder
)

func kindForCode(code int) weatherKind {
	switch code {
	case 0, 1:
		return kindClear
	case 2:
		return kindPartly
	case 3:
		return kindCloudy
	case 45, 48:
		return kindFog
	case 51, 53, 55, 56, 57,
		61, 63, 65, 66, 67,
		80, 81, 82:
		return kindRain
	case 71, 73, 75, 77, 85, 86:
		return kindSnow
	case 95, 96, 99:
		return kindThunder
	default:
		return kindCloudy
	}
}

func isNightHour(hour int) bool {
	return hour >= 20 || hour < 6
}

func drawWeatherIcon(
	dc *gg.Context,
	code int,
	cx float64,
	cy float64,
	size float64,
	night bool,
) {
	switch kindForCode(code) {
	case kindClear:
		if night {
			drawMoon(dc, cx, cy, size)
		} else {
			drawSun(dc, cx, cy, size)
		}

	case kindPartly:
		drawSun(dc, cx+size*0.16, cy-size*0.16, size*0.5)
		drawCloud(dc, cx-size*0.08, cy+size*0.10, size)

	case kindCloudy:
		drawCloud(dc, cx, cy, size)

	case kindFog:
		drawCloud(dc, cx, cy-size*0.16, size*0.8)
		drawFogLines(dc, cx, cy+size*0.22, size)

	case kindRain:
		drawCloud(dc, cx, cy-size*0.16, size*0.85)
		drawRainDrops(dc, cx, cy+size*0.24, size)

	case kindSnow:
		drawCloud(dc, cx, cy-size*0.16, size*0.85)
		drawSnowFlakes(dc, cx, cy+size*0.24, size)

	case kindThunder:
		drawCloud(dc, cx, cy-size*0.20, size*0.85)
		drawBolt(dc, cx, cy+size*0.16, size)
	}
}

func drawSun(dc *gg.Context, cx, cy, size float64) {
	dc.SetHexColor(iconSun)

	radius := size * 0.2

	for i := 0; i < 8; i++ {
		angle := float64(i) * (2 * math.Pi / 8)
		x1 := cx + 0.34*size*math.Cos(angle)
		y1 := cy + 0.34*size*math.Sin(angle)
		x2 := cx + 0.48*size*math.Cos(angle)
		y2 := cy + 0.48*size*math.Sin(angle)

		dc.SetLineWidth(size * 0.05)
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}

	dc.DrawCircle(cx, cy, radius)
	dc.Fill()
}

func drawMoon(dc *gg.Context, cx, cy, size float64) {
	dc.SetHexColor(iconMoon)
	dc.DrawCircle(cx, cy, size*0.28)
	dc.Fill()

	dc.SetHexColor(boardBackgroundHex)
	dc.DrawCircle(cx+size*0.12, cy-size*0.06, size*0.25)
	dc.Fill()
}

func drawCloud(dc *gg.Context, cx, cy, size float64) {
	dc.SetHexColor(iconCloud)

	dc.DrawCircle(cx-size*0.22, cy, size*0.18)
	dc.Fill()

	dc.DrawCircle(cx+size*0.12, cy-size*0.10, size*0.22)
	dc.Fill()

	dc.DrawCircle(cx+size*0.26, cy, size*0.15)
	dc.Fill()

	dc.DrawRoundedRectangle(
		cx-size*0.38,
		cy-size*0.02,
		size*0.76,
		size*0.20,
		size*0.10,
	)
	dc.Fill()
}

func drawRainDrops(dc *gg.Context, cx, cy, size float64) {
	dc.SetHexColor(iconPrecip)

	offsets := [3][2]float64{
		{-0.24, 0.02},
		{0.00, -0.06},
		{0.24, 0.04},
	}

	for _, offset := range offsets {
		dc.DrawCircle(
			cx+offset[0]*size,
			cy+offset[1]*size,
			size*0.06,
		)
		dc.Fill()
	}
}

func drawSnowFlakes(dc *gg.Context, cx, cy, size float64) {
	dc.SetHexColor(iconSnow)
	dc.SetLineWidth(size * 0.025)

	centres := [3][2]float64{
		{-0.24, 0.0},
		{0.0, 0.08},
		{0.24, 0.0},
	}

	for _, centre := range centres {
		fx := cx + centre[0]*size
		fy := cy + centre[1]*size
		radius := size * 0.075

		for i := 0; i < 3; i++ {
			angle := float64(i) * (math.Pi / 3)
			x1 := fx - radius*math.Cos(angle)
			y1 := fy - radius*math.Sin(angle)
			x2 := fx + radius*math.Cos(angle)
			y2 := fy + radius*math.Sin(angle)

			dc.DrawLine(x1, y1, x2, y2)
		}

		dc.Stroke()
	}
}

func drawBolt(dc *gg.Context, cx, cy, size float64) {
	dc.SetHexColor(iconBolt)

	top := cy - size*0.10
	mid := cy + size*0.04
	bottom := cy + size*0.30

	halfWidth := size * 0.16

	dc.MoveTo(cx-halfWidth, top)
	dc.LineTo(cx+halfWidth*0.4, top)
	dc.LineTo(cx+halfWidth*0.1, mid)
	dc.LineTo(cx+halfWidth, mid)
	dc.LineTo(cx-halfWidth*0.2, bottom)
	dc.LineTo(cx-halfWidth*0.05, mid+size*0.04)
	dc.LineTo(cx-halfWidth, mid+size*0.04)
	dc.ClosePath()
	dc.Fill()
}

func drawFogLines(dc *gg.Context, cx, cy, size float64) {
	dc.SetHexColor(iconCloud)
	dc.SetLineWidth(size * 0.045)

	lines := [3][2]float64{
		{-0.32, 0.0},
		{0.0, 0.14},
		{0.32, 0.0},
	}

	for _, line := range lines {
		y := cy + line[1]*size
		dc.DrawLine(cx+line[0]*size-size*0.12, y, cx+line[0]*size+size*0.12, y)
		dc.Stroke()
	}
}
