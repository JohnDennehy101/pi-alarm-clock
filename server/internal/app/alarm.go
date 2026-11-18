package app

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
)

type AlarmService struct {
	repository alarm.Repository
}

func NewAlarmService(repository alarm.Repository) *AlarmService {
	return &AlarmService{
		repository: repository,
	}
}

func (s *AlarmService) Create(
	ctx context.Context,
	a *alarm.Alarm,
) error {
	if err := validateAlarm(a); err != nil {
		return err
	}

	return s.repository.Create(ctx, a)
}

func (s *AlarmService) Get(
	ctx context.Context,
	id string,
) (*alarm.Alarm, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("alarm ID is required")
	}

	return s.repository.Get(ctx, id)
}

func (s *AlarmService) List(
	ctx context.Context,
) ([]alarm.Alarm, error) {
	return s.repository.List(ctx)
}

func (s *AlarmService) Update(
	ctx context.Context,
	a *alarm.Alarm,
) error {
	if a == nil {
		return fmt.Errorf("alarm is required")
	}

	if strings.TrimSpace(a.ID) == "" {
		return fmt.Errorf("alarm ID is required")
	}

	if err := validateAlarm(a); err != nil {
		return err
	}

	return s.repository.Update(ctx, a)
}

func (s *AlarmService) Delete(
	ctx context.Context,
	id string,
) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("alarm ID is required")
	}

	return s.repository.Delete(ctx, id)
}

var alarmTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// chromecastMaxDisplayDurationMinutes matches the CHECK constraint on
// the chromecast_display_duration column. Zero selects the default
// duration.
const chromecastMaxDisplayDurationMinutes = 240

func validateAlarm(a *alarm.Alarm) error {
	if a == nil {
		return fmt.Errorf("alarm is required")
	}

	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("alarm name is required")
	}

	if !alarmTimePattern.MatchString(a.Time) {
		return fmt.Errorf("alarm time must use HH:MM format")
	}

	if err := validateSonos(a.Sonos); err != nil {
		return err
	}

	if err := validateSchedule(a); err != nil {
		return err
	}

	if err := validateHue(a.Hue); err != nil {
		return err
	}

	if err := validateChromecast(a.Chromecast); err != nil {
		return err
	}

	return nil
}

func validateSonos(settings alarm.SonosSettings) error {
	if settings.Volume < 0 || settings.Volume > 100 {
		return fmt.Errorf("alarm volume must be between 0 and 100")
	}

	if settings.StartVolume < 0 || settings.StartVolume > 100 {
		return fmt.Errorf("alarm start volume must be between 0 and 100")
	}

	if settings.GradualVolume && settings.StartVolume > settings.Volume {
		return fmt.Errorf("alarm start volume must not exceed the target volume")
	}

	if settings.StartOffsetMinutes < 0 || settings.StartOffsetMinutes > alarm.MaxStartOffsetMinutes {
		return fmt.Errorf(
			"Sonos start offset must be between 0 and %d minutes",
			alarm.MaxStartOffsetMinutes,
		)
	}

	if err := validateTarget(settings.Target); err != nil {
		return err
	}

	if err := validateSource(settings.Source); err != nil {
		return err
	}

	if settings.GradualVolume && settings.VolumeDuration < 1 {
		return fmt.Errorf("volume duration must be at least 1 minute")
	}

	if settings.GradualVolume && settings.VolumeDuration > 120 {
		return fmt.Errorf("volume duration cannot exceed 120 minutes")
	}

	return nil
}

func validateTarget(target alarm.SonosTarget) error {
	switch target.Type {
	case alarm.TargetSpeaker, alarm.TargetGroup:
		if strings.TrimSpace(target.ID) == "" {
			return fmt.Errorf("Sonos target ID is required")
		}
	default:
		return fmt.Errorf("invalid Sonos target type")
	}

	return nil
}

func validateSource(source alarm.AudioSource) error {
	switch source.Type {
	case alarm.SourceRadio:
		if strings.TrimSpace(source.URI) == "" {
			return fmt.Errorf("radio source URI is required")
		}
	case alarm.SourceAudio:
		if strings.TrimSpace(source.URI) == "" {
			return fmt.Errorf("audio source URI is required")
		}
	default:
		return fmt.Errorf("invalid audio source type")
	}

	return nil
}

func validateSchedule(a *alarm.Alarm) error {
	if a.Date != nil && len(a.Days) > 0 {
		return fmt.Errorf("alarm cannot have both a date and recurring days")
	}

	if a.Date == nil && len(a.Days) == 0 {
		return fmt.Errorf("alarm must have a date or at least one recurring day")
	}

	validDays := map[alarm.Weekday]bool{
		alarm.Monday:    true,
		alarm.Tuesday:   true,
		alarm.Wednesday: true,
		alarm.Thursday:  true,
		alarm.Friday:    true,
		alarm.Saturday:  true,
		alarm.Sunday:    true,
	}

	seen := make(map[alarm.Weekday]bool)

	for _, day := range a.Days {
		if !validDays[day] {
			return fmt.Errorf("invalid weekday: %s", day)
		}

		if seen[day] {
			return fmt.Errorf("duplicate weekday: %s", day)
		}

		seen[day] = true
	}

	return nil
}

func validateHue(settings alarm.HueSettings) error {
	if !settings.Enabled {
		return nil
	}

	if len(settings.LightIDs) == 0 {
		return fmt.Errorf("at least one Hue light is required")
	}

	for _, id := range settings.LightIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("Hue light ID cannot be empty")
		}
	}

	if settings.StartBrightness < 0 || settings.StartBrightness > 100 {
		return fmt.Errorf("Hue start brightness must be between 0 and 100")
	}

	if settings.StartOffsetMinutes < 0 || settings.StartOffsetMinutes > alarm.MaxStartOffsetMinutes {
		return fmt.Errorf(
			"Hue start offset must be between 0 and %d minutes",
			alarm.MaxStartOffsetMinutes,
		)
	}

	if settings.Brightness < 1 || settings.Brightness > 100 {
		return fmt.Errorf("Hue brightness must be between 1 and 100")
	}

	if settings.GradualBrightness && settings.BrightnessDuration < 1 {
		return fmt.Errorf("Hue brightness duration must be at least 1 minute")
	}

	if settings.ColourTemperature != 0 &&
		(settings.ColourTemperature < 2000 || settings.ColourTemperature > 6500) {
		return fmt.Errorf("Hue colour temperature must be between 2000 and 6500 Kelvin")
	}

	return nil
}

func validateChromecast(settings alarm.ChromecastSettings) error {
	if !settings.Enabled {
		return nil
	}

	if strings.TrimSpace(settings.DeviceID) == "" {
		return fmt.Errorf("Chromecast device is required")
	}

	if settings.DisplayOffsetMinutes < 0 ||
		settings.DisplayOffsetMinutes > alarm.MaxStartOffsetMinutes {
		return fmt.Errorf(
			"Chromecast display offset must be between 0 and %d minutes",
			alarm.MaxStartOffsetMinutes,
		)
	}

	if settings.DisplayDurationMinutes < 0 ||
		settings.DisplayDurationMinutes > chromecastMaxDisplayDurationMinutes {
		return fmt.Errorf(
			"Chromecast display duration must be between 0 and %d minutes",
			chromecastMaxDisplayDurationMinutes,
		)
	}

	return nil
}
