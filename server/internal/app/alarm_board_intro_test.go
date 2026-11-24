package app

import (
	"strings"
	"testing"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
)

func validBoardIntroAlarm(intro string) *alarm.Alarm {
	return &alarm.Alarm{
		Name: "board intro alarm",
		Time: "07:00",
		Days: []alarm.Weekday{alarm.Monday},
		Chromecast: alarm.ChromecastSettings{
			Enabled:                true,
			DeviceID:               "cast-uuid",
			DisplayOffsetMinutes:   1,
			DisplayDurationMinutes: 45,
			BoardTimeline:          "bus-focus",
			BoardIntro:             intro,
		},
	}
}

func TestValidateChromecastBoardIntro(t *testing.T) {
	validIntros := []string{
		"",
		"weather",
		"weather,bus.map",
		"bus-focus",
	}

	for _, intro := range validIntros {
		if err := validateChromecast(validBoardIntroAlarm(intro).Chromecast); err != nil {
			t.Errorf("validateChromecast(boardIntro %q) = %v, want nil", intro, err)
		}
	}

	if err := validateChromecast(validBoardIntroAlarm(strings.Repeat("w", 201)).Chromecast); err == nil {
		t.Error("validateChromecast(201-char board intro) = nil, want error")
	}

	if err := validateChromecast(validBoardIntroAlarm("weather\nbus.map").Chromecast); err == nil {
		t.Error("validateChromecast(multi-line board intro) = nil, want error")
	}

	// Disabled chromecast settings skip validation entirely.
	disabled := validBoardIntroAlarm(strings.Repeat("w", 300))
	disabled.Chromecast.Enabled = false

	if err := validateChromecast(disabled.Chromecast); err != nil {
		t.Errorf("validateChromecast(disabled) = %v, want nil", err)
	}
}
