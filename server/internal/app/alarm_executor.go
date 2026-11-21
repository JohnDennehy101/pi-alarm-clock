package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
)

type AlarmExecutor struct {
	sonos      *SonosService
	hue        *HueService
	chromecast *ChromecastService
}

func NewAlarmExecutor(
	sonos *SonosService,
	hue *HueService,
	chromecast *ChromecastService,
) *AlarmExecutor {
	return &AlarmExecutor{
		sonos:      sonos,
		hue:        hue,
		chromecast: chromecast,
	}
}

func (e *AlarmExecutor) StartLighting(
	ctx context.Context,
	a alarm.Alarm,
) error {
	if !a.Hue.Enabled {
		return nil
	}

	if e.hue == nil {
		return fmt.Errorf("Hue is enabled but Hue service is not configured")
	}

	go func() {
		if err := e.hue.RunWakeUpLighting(ctx, a.Hue); err != nil {
			log.Printf(
				"alarm %s: Hue wake-up lighting failed: %v",
				a.ID,
				err,
			)
		}
	}()

	return nil
}

func (e *AlarmExecutor) StartBoardDisplay(
	ctx context.Context,
	a alarm.Alarm,
) error {
	if !a.Chromecast.Enabled {
		return nil
	}

	if e.chromecast == nil {
		return fmt.Errorf(
			"Chromecast is enabled but Chromecast service is not configured",
		)
	}

	go func() {
		if err := e.chromecast.RunBoardCast(
			ctx,
			a.Chromecast.DeviceID,
			a.Chromecast.DisplayDurationMinutes,
			"",
			a.Chromecast.BoardTimeline,
		); err != nil {
			log.Printf(
				"alarm %s: displaying Chromecast board failed: %v",
				a.ID,
				err,
			)
		}
	}()

	return nil
}

func (e *AlarmExecutor) StartAudio(
	ctx context.Context,
	a alarm.Alarm,
) error {
	if e.sonos == nil {
		return fmt.Errorf("Sonos service is not configured")
	}

	if a.Sonos.GradualVolume {
		if err := e.sonos.SetTargetVolume(
			ctx,
			a.Sonos.Target,
			a.Sonos.StartVolume,
		); err != nil {
			return fmt.Errorf("setting initial Sonos volume: %w", err)
		}
	} else {
		if err := e.sonos.SetTargetVolume(
			ctx,
			a.Sonos.Target,
			a.Sonos.Volume,
		); err != nil {
			return fmt.Errorf("setting Sonos volume: %w", err)
		}
	}

	if err := e.playAlarmSource(ctx, a); err != nil {
		return fmt.Errorf("starting Sonos playback: %w", err)
	}

	if a.Sonos.GradualVolume {
		go func() {
			duration := time.Duration(a.Sonos.VolumeDuration) * time.Minute

			if err := e.sonos.RunGradualVolume(
				ctx,
				a.Sonos.Target,
				a.Sonos.StartVolume,
				a.Sonos.Volume,
				duration,
			); err != nil {
				log.Printf(
					"alarm %s: gradual Sonos volume failed: %v",
					a.ID,
					err,
				)
			}
		}()
	}

	return nil
}

func (e *AlarmExecutor) playAlarmSource(
	ctx context.Context,
	a alarm.Alarm,
) error {
	switch a.Sonos.Target.Type {
	case alarm.TargetSpeaker:
		switch a.Sonos.Source.Type {
		case alarm.SourceRadio:
			return e.sonos.PlaySpeakerRadioStation(
				ctx,
				a.Sonos.Target.ID,
				a.Sonos.Source.Name,
			)

		case alarm.SourceAudio:
			return e.sonos.PlaySpeakerURI(
				ctx,
				a.Sonos.Target.ID,
				a.Sonos.Source.URI,
				"",
			)

		default:
			return fmt.Errorf(
				"unsupported audio source type %q",
				a.Sonos.Source.Type,
			)
		}

	case alarm.TargetGroup:
		switch a.Sonos.Source.Type {
		case alarm.SourceRadio:
			return e.sonos.PlayGroupRadioStation(
				ctx,
				a.Sonos.Target.ID,
				a.Sonos.Source.Name,
			)

		case alarm.SourceAudio:
			return e.sonos.PlayGroupURI(
				ctx,
				a.Sonos.Target.ID,
				a.Sonos.Source.URI,
				"",
			)

		default:
			return fmt.Errorf(
				"unsupported audio source type %q",
				a.Sonos.Source.Type,
			)
		}

	default:
		return fmt.Errorf(
			"unsupported Sonos target type %q",
			a.Sonos.Target.Type,
		)
	}
}
