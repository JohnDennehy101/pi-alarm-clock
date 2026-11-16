package scheduler

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
)

func NextOccurrence(
	a alarm.Alarm,
	now time.Time,
	location *time.Location,
) (time.Time, bool) {
	if !a.Enabled || location == nil {
		return time.Time{}, false
	}

	parts := strings.Split(a.Time, ":")
	if len(parts) != 2 {
		return time.Time{}, false
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, false
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, false
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return time.Time{}, false
	}

	now = now.In(location)

	if a.Date != nil {
		date := a.Date

		occurrence := time.Date(
			date.Year(),
			date.Month(),
			date.Day(),
			hour,
			minute,
			0,
			0,
			location,
		)

		if occurrence.Before(now) {
			return time.Time{}, false
		}

		return occurrence, true
	}

	weekdays := make(map[string]bool)

	for _, day := range a.Days {
		weekdays[strings.ToLower(string(day))] = true
	}

	for offset := 0; offset <= 7; offset++ {
		date := now.AddDate(0, 0, offset)

		occurrence := time.Date(
			date.Year(),
			date.Month(),
			date.Day(),
			hour,
			minute,
			0,
			0,
			location,
		)

		weekday := strings.ToLower(occurrence.Weekday().String())

		if !weekdays[weekday] {
			continue
		}

		if occurrence.Before(now) {
			continue
		}

		return occurrence, true
	}

	return time.Time{}, false
}

const checkInterval = 15 * time.Second

type AlarmProvider interface {
	List(ctx context.Context) ([]alarm.Alarm, error)
}

type AlarmExecutor interface {
	StartAudio(ctx context.Context, a alarm.Alarm) error
	StartLighting(ctx context.Context, a alarm.Alarm) error
}

type Scheduler struct {
	alarms   AlarmProvider
	executor AlarmExecutor
	location *time.Location
}

func New(
	alarms AlarmProvider,
	executor AlarmExecutor,
	location *time.Location,
) *Scheduler {
	if location == nil {
		location = time.Local
	}

	return &Scheduler{
		alarms:   alarms,
		executor: executor,
		location: location,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	if s.alarms == nil || s.executor == nil {
		log.Println("scheduler: alarm provider or executor is not configured")
		return
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	audioFired := make(map[string]string)
	lightingFired := make(map[string]string)

	s.check(ctx, audioFired, lightingFired)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.check(ctx, audioFired, lightingFired)
		}
	}
}

func (s *Scheduler) check(
	ctx context.Context,
	audioFired map[string]string,
	lightingFired map[string]string,
) {
	alarms, err := s.alarms.List(ctx)
	if err != nil {
		log.Printf("scheduler: failed to retrieve alarms: %v", err)
		return
	}

	now := time.Now().In(s.location)
	currentMinute := now.Truncate(time.Minute)

	base := currentMinute.Add(-alarm.MaxStartOffsetMinutes * time.Minute)

	for _, a := range alarms {
		occurrence, ok := NextOccurrence(a, base, s.location)
		if !ok {
			continue
		}

		occurrenceKey := occurrence.Format("2006-01-02")

		if a.Hue.Enabled {
			lightingTime := occurrence.Add(
				-time.Duration(a.Hue.StartOffsetMinutes) * time.Minute,
			)

			lightingDue := lightingTime.Equal(currentMinute) ||
				(lightingTime.Before(currentMinute) &&
					currentMinute.Before(occurrence))

			if lightingDue && lightingFired[a.ID] != occurrenceKey {
				lightingFired[a.ID] = occurrenceKey

				if lightingTime.Before(currentMinute) {
					log.Printf(
						"scheduler: lighting start for alarm %q is overdue, starting now",
						a.ID,
					)
				}

				if err := s.executor.StartLighting(ctx, a); err != nil {
					log.Printf(
						"scheduler: failed to start lighting for alarm %q: %v",
						a.ID,
						err,
					)
				}
			}
		}

		audioTime := occurrence.Add(
			-time.Duration(a.Sonos.StartOffsetMinutes) * time.Minute,
		)

		audioOverdue := audioTime.Before(currentMinute) &&
			currentMinute.Before(occurrence)

		if (!audioTime.Equal(currentMinute) && !audioOverdue) ||
			audioFired[a.ID] == occurrenceKey {
			continue
		}

		audioFired[a.ID] = occurrenceKey

		if audioOverdue {
			log.Printf(
				"scheduler: audio start for alarm %q is overdue, starting now",
				a.ID,
			)
		}

		audioFired[a.ID] = occurrenceKey

		if err := s.executor.StartAudio(ctx, a); err != nil {
			log.Printf(
				"scheduler: failed to start audio for alarm %q: %v",
				a.ID,
				err,
			)
		}
	}
}
