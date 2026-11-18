package display

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"image/color"
	"time"

	"github.com/fogleman/gg"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/weather"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed fonts/JetBrainsMono-Regular.ttf
var fontData []byte

const (
	boardWidth  = 1280
	boardHeight = 720

	hourlySlots = 12
)

// WeatherService is anything that can supply the forecast shown on
// the board. Defined here so display does not depend on its providers.
type WeatherService interface {
	Forecast(ctx context.Context) (*weather.Forecast, error)
}

type BoardRenderer struct {
	font    *opentype.Font
	weather WeatherService
}

func NewBoardRenderer(weatherClient WeatherService) (*BoardRenderer, error) {
	parsed, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("parsing board font: %w", err)
	}

	return &BoardRenderer{
		font:    parsed,
		weather: weatherClient,
	}, nil
}

func (r *BoardRenderer) face(size float64) font.Face {
	face, err := opentype.NewFace(r.font, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil
	}

	return face
}

func (r *BoardRenderer) RenderBoard(
	ctx context.Context,
	now time.Time,
) ([]byte, error) {
	dc := gg.NewContext(boardWidth, boardHeight)

	dc.SetColor(color.RGBA{R: 13, G: 15, B: 20, A: 255})
	dc.Clear()

	r.drawClock(dc, now)

	if r.weather != nil {
		forecast, err := r.weather.Forecast(ctx)
		if err != nil {
			r.drawWeatherUnavailable(dc, err)
		} else {
			r.drawCurrent(dc, now, forecast)
			r.drawHourly(dc, now, forecast)
		}
	}

	var buf bytes.Buffer

	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("encoding board PNG: %w", err)
	}

	return buf.Bytes(), nil
}

func (r *BoardRenderer) drawClock(dc *gg.Context, now time.Time) {
	dc.SetFontFace(r.face(220))
	dc.SetColor(color.RGBA{R: 235, G: 240, B: 245, A: 255})
	dc.DrawString(now.Format("15:04"), 56, 210)

	dc.SetFontFace(r.face(52))
	dc.SetColor(color.RGBA{R: 154, G: 164, B: 178, A: 255})
	dc.DrawString(now.Format("Monday, 2 January"), 60, 268)

	dc.SetColor(color.RGBA{R: 40, G: 46, B: 58, A: 255})
	dc.SetLineWidth(2)
	dc.DrawLine(60, 320, boardWidth-60, 320)
	dc.Stroke()
}

func clockWidth(dc *gg.Context) int {
	w, _ := dc.MeasureString("00:00")
	return int(w)
}

func (r *BoardRenderer) drawCurrent(
	dc *gg.Context,
	now time.Time,
	forecast *weather.Forecast,
) {
	current := forecast.Current

	dc.SetFontFace(r.face(120))
	dc.SetColor(color.RGBA{R: 235, G: 240, B: 245, A: 255})

	temperature := fmt.Sprintf("%.0f°", current.Temperature)
	dc.DrawString(temperature, 60, 450)

	temperatureWidth, _ := dc.MeasureString(temperature)

	drawWeatherIcon(
		dc,
		current.Code,
		60+temperatureWidth+55,
		405,
		100,
		isNightHour(current.Time.Hour()),
	)

	dc.SetFontFace(r.face(40))
	dc.SetColor(color.RGBA{R: 154, G: 164, B: 178, A: 255})

	details := fmt.Sprintf(
		"%s · Feels like %.0f° · Wind %.0f km/h",
		Describe(current.Code),
		current.FeelsLike,
		current.WindSpeed,
	)
	dc.DrawString(details, 60, 496)
}

func (r *BoardRenderer) drawHourly(
	dc *gg.Context,
	now time.Time,
	forecast *weather.Forecast,
) {
	upcoming := make([]weather.Conditions, 0, hourlySlots)

	for _, hour := range forecast.Hourly {
		if hour.Time.Before(now.Truncate(time.Hour)) {
			continue
		}

		upcoming = append(upcoming, hour)

		if len(upcoming) == hourlySlots {
			break
		}
	}

	if len(upcoming) == 0 {
		return
	}

	columnWidth := float64(boardWidth-120) / float64(len(upcoming))

	for i, hour := range upcoming {
		x := 60 + columnWidth*float64(i) + columnWidth/2

		label := hour.Time.Format("3pm")
		if i == 0 {
			label = "Now"
		}

		dc.SetFontFace(r.face(34))
		dc.SetColor(color.RGBA{R: 130, G: 140, B: 155, A: 255})
		dc.DrawStringAnchored(label, x, 544, 0.5, 0.5)

		drawWeatherIcon(
			dc,
			hour.Code,
			x,
			600,
			40,
			isNightHour(hour.Time.Hour()),
		)

		dc.SetFontFace(r.face(44))
		dc.SetColor(color.RGBA{R: 235, G: 240, B: 245, A: 255})
		dc.DrawStringAnchored(
			fmt.Sprintf("%.0f°", hour.Temperature),
			x,
			652,
			0.5,
			0.5,
		)

		if hour.Precipitation > 0 {
			dc.SetFontFace(r.face(28))
			dc.SetColor(color.RGBA{R: 96, G: 165, B: 250, A: 255})
			dc.DrawStringAnchored(
				fmt.Sprintf("%d%%", hour.Precipitation),
				x,
				694,
				0.5,
				0.5,
			)
		}
	}
}

func (r *BoardRenderer) drawWeatherUnavailable(
	dc *gg.Context,
	err error,
) {
	dc.SetFontFace(r.face(40))
	dc.SetColor(color.RGBA{R: 130, G: 140, B: 155, A: 255})
	dc.DrawStringAnchored(
		"Weather unavailable",
		boardWidth/2,
		480,
		0.5,
		0.5,
	)
}

func Describe(code int) string {
	switch code {
	case 0:
		return "Clear"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Fog"
	case 51, 53, 55:
		return "Drizzle"
	case 56, 57:
		return "Freezing drizzle"
	case 61, 63, 65:
		return "Rain"
	case 66, 67:
		return "Freezing rain"
	case 71, 73, 75:
		return "Snow"
	case 77:
		return "Snow grains"
	case 80, 81, 82:
		return "Showers"
	case 85, 86:
		return "Snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with hail"
	default:
		return "Unknown"
	}
}
