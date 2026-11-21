package database

import (
	"context"
	"testing"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
)

func TestChromecastRepoRoundTrip(t *testing.T) {
	ctx := context.Background()

	db, err := New(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close()

	a := &alarm.Alarm{
		Name:    "test",
		Time:    "07:30",
		Enabled: true,
		Days:    []alarm.Weekday{alarm.Monday},
		Sonos: alarm.SonosSettings{
			Target:             alarm.SonosTarget{Type: alarm.TargetSpeaker, ID: "spk"},
			Source:             alarm.AudioSource{Type: alarm.SourceRadio, URI: "http://x"},
			Volume:             40,
			StartOffsetMinutes: 5,
			GradualVolume:      true,
			StartVolume:        10,
			VolumeDuration:     10,
		},
		Hue: alarm.HueSettings{
			Enabled:            true,
			LightIDs:           []string{"l1"},
			StartOffsetMinutes: 10,
		},
		Chromecast: alarm.ChromecastSettings{
			Enabled:                true,
			DeviceID:               "cast-uuid",
			DisplayOffsetMinutes:   7,
			DisplayDurationMinutes: 90,
		},
	}

	if err := db.AlarmRepository.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := db.AlarmRepository.Get(ctx, a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Chromecast != a.Chromecast {
		t.Fatalf("get mismatch: %+v vs %+v", got.Chromecast, a.Chromecast)
	}

	got.Chromecast.DisplayDurationMinutes = 30
	got.Chromecast.DisplayOffsetMinutes = 15
	got.Hue.StartOffsetMinutes = 3

	if err := db.AlarmRepository.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}

	updated, err := db.AlarmRepository.Get(ctx, got.ID)
	if err != nil {
		t.Fatalf("get2: %v", err)
	}

	if updated.Chromecast.DisplayDurationMinutes != 30 ||
		updated.Chromecast.DisplayOffsetMinutes != 15 ||
		updated.Chromecast.DeviceID != "cast-uuid" ||
		!updated.Chromecast.Enabled {
		t.Fatalf("update chromecast mismatch: %+v", updated.Chromecast)
	}

	if updated.Hue.StartOffsetMinutes != 3 {
		t.Fatalf("hue start offset was not persisted: %d", updated.Hue.StartOffsetMinutes)
	}

	list, err := db.AlarmRepository.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
}
