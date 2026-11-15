package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
)

type AlarmRepository struct {
	db *sql.DB
}

func NewAlarmRepository(db *sql.DB) *AlarmRepository {
	return &AlarmRepository{db: db}
}

func (r *AlarmRepository) Create(
	ctx context.Context,
	a *alarm.Alarm,
) error {
	now := time.Now().UTC()

	id, err := newID()
	if err != nil {
		return fmt.Errorf("generating alarm ID: %w", err)
	}

	days, err := json.Marshal(a.Days)
	if err != nil {
		return fmt.Errorf("encoding alarm days: %w", err)
	}

	lightIDs, err := json.Marshal(a.Hue.LightIDs)
	if err != nil {
		return fmt.Errorf("encoding Hue light IDs: %w", err)
	}

	var date any
	if a.Date != nil {
		date = a.Date.Format("2006-01-02")
	}

	var colourTemperature any
	if a.Hue.ColourTemperature != 0 {
		colourTemperature = a.Hue.ColourTemperature
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO alarms (
			id, name, time, enabled, days, date,
			target_type, target_id,
			source_type, source_uri, source_name,
			volume, gradual_volume,
			hue_enabled, hue_light_ids,
			hue_start_brightness, hue_brightness,
			hue_gradual_brightness, hue_brightness_duration,
			hue_colour_temperature,
			created_at, updated_at
		)
		VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?,
			?, ?,
			?, ?,
			?, ?,
			?,
			?, ?
		)
	`,
		id,
		a.Name,
		a.Time,
		a.Enabled,
		string(days),
		date,
		a.Sonos.Target.Type,
		a.Sonos.Target.ID,
		a.Sonos.Source.Type,
		a.Sonos.Source.URI,
		a.Sonos.Source.Name,
		a.Sonos.Volume,
		a.Sonos.GradualVolume,
		a.Hue.Enabled,
		string(lightIDs),
		a.Hue.StartBrightness,
		a.Hue.Brightness,
		a.Hue.GradualBrightness,
		a.Hue.BrightnessDuration,
		colourTemperature,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("creating alarm: %w", err)
	}

	a.ID = id
	a.CreatedAt = now
	a.UpdatedAt = now

	return nil
}

func (r *AlarmRepository) Get(
	ctx context.Context,
	id string,
) (*alarm.Alarm, error) {
	a, err := scanAlarm(r.db.QueryRowContext(ctx, `
		SELECT
			id, name, time, enabled, days, date,
			target_type, target_id,
			source_type, source_uri, source_name,
			volume, gradual_volume,
			hue_enabled, hue_light_ids,
			hue_start_brightness, hue_brightness,
			hue_gradual_brightness, hue_brightness_duration,
			hue_colour_temperature,
			created_at, updated_at
		FROM alarms
		WHERE id = ?
	`, id))

	if errors.Is(err, sql.ErrNoRows) {
		return nil, alarm.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("getting alarm: %w", err)
	}

	return a, nil
}

func (r *AlarmRepository) List(
	ctx context.Context,
) ([]alarm.Alarm, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, name, time, enabled, days, date,
			target_type, target_id,
			source_type, source_uri, source_name,
			volume, gradual_volume,
			hue_enabled, hue_light_ids,
			hue_start_brightness, hue_brightness,
			hue_gradual_brightness, hue_brightness_duration,
			hue_colour_temperature,
			created_at, updated_at
		FROM alarms
		ORDER BY time ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing alarms: %w", err)
	}
	defer rows.Close()

	alarms := make([]alarm.Alarm, 0)

	for rows.Next() {
		a, err := scanAlarm(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning alarm: %w", err)
		}

		alarms = append(alarms, *a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating alarms: %w", err)
	}

	return alarms, nil
}

func (r *AlarmRepository) Update(
	ctx context.Context,
	a *alarm.Alarm,
) error {
	days, err := json.Marshal(a.Days)
	if err != nil {
		return fmt.Errorf("encoding alarm days: %w", err)
	}

	lightIDs, err := json.Marshal(a.Hue.LightIDs)
	if err != nil {
		return fmt.Errorf("encoding Hue light IDs: %w", err)
	}

	var date any
	if a.Date != nil {
		date = a.Date.Format("2006-01-02")
	}

	var colourTemperature any
	if a.Hue.ColourTemperature != 0 {
		colourTemperature = a.Hue.ColourTemperature
	}

	now := time.Now().UTC()

	result, err := r.db.ExecContext(ctx, `
		UPDATE alarms
		SET
			name = ?,
			time = ?,
			enabled = ?,
			days = ?,
			date = ?,
			target_type = ?,
			target_id = ?,
			source_type = ?,
			source_uri = ?,
			source_name = ?,
			volume = ?,
			gradual_volume = ?,
			hue_enabled = ?,
			hue_light_ids = ?,
			hue_start_brightness = ?,
			hue_brightness = ?,
			hue_gradual_brightness = ?,
			hue_brightness_duration = ?,
			hue_colour_temperature = ?,
			updated_at = ?
		WHERE id = ?
	`,
		a.Name,
		a.Time,
		a.Enabled,
		string(days),
		date,
		a.Sonos.Target.Type,
		a.Sonos.Target.ID,
		a.Sonos.Source.Type,
		a.Sonos.Source.URI,
		a.Sonos.Source.Name,
		a.Sonos.Volume,
		a.Sonos.GradualVolume,
		a.Hue.Enabled,
		string(lightIDs),
		a.Hue.StartBrightness,
		a.Hue.Brightness,
		a.Hue.GradualBrightness,
		a.Hue.BrightnessDuration,
		colourTemperature,
		now.Format(time.RFC3339Nano),
		a.ID,
	)
	if err != nil {
		return fmt.Errorf("updating alarm: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking updated alarm: %w", err)
	}

	if affected == 0 {
		return alarm.ErrNotFound
	}

	a.UpdatedAt = now

	return nil
}

func (r *AlarmRepository) Delete(
	ctx context.Context,
	id string,
) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM alarms
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("deleting alarm: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking deleted alarm: %w", err)
	}

	if affected == 0 {
		return alarm.ErrNotFound
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAlarm(row rowScanner) (*alarm.Alarm, error) {
	var a alarm.Alarm

	var (
		enabled           bool
		gradualVolume     bool
		hueEnabled        bool
		gradualBrightness bool
		days              string
		lightIDs          string
		date              sql.NullString
		colourTemperature sql.NullInt64
		createdAt         string
		updatedAt         string
	)

	err := row.Scan(
		&a.ID,
		&a.Name,
		&a.Time,
		&enabled,
		&days,
		&date,
		&a.Sonos.Target.Type,
		&a.Sonos.Target.ID,
		&a.Sonos.Source.Type,
		&a.Sonos.Source.URI,
		&a.Sonos.Source.Name,
		&a.Sonos.Volume,
		&gradualVolume,
		&hueEnabled,
		&lightIDs,
		&a.Hue.StartBrightness,
		&a.Hue.Brightness,
		&gradualBrightness,
		&a.Hue.BrightnessDuration,
		&colourTemperature,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(days), &a.Days); err != nil {
		return nil, fmt.Errorf("decoding alarm days: %w", err)
	}

	if err := json.Unmarshal([]byte(lightIDs), &a.Hue.LightIDs); err != nil {
		return nil, fmt.Errorf("decoding Hue light IDs: %w", err)
	}

	if date.Valid {
		parsed, err := time.Parse("2006-01-02", date.String)
		if err != nil {
			return nil, fmt.Errorf("parsing alarm date: %w", err)
		}
		a.Date = &parsed
	}

	if colourTemperature.Valid {
		a.Hue.ColourTemperature = int(colourTemperature.Int64)
	}

	a.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing creation time: %w", err)
	}

	a.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing update time: %w", err)
	}

	a.Enabled = enabled
	a.Sonos.GradualVolume = gradualVolume
	a.Hue.Enabled = hueEnabled
	a.Hue.GradualBrightness = gradualBrightness

	return &a, nil
}

func newID() (string, error) {
	var id [16]byte

	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}

	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80

	encoded := make([]byte, 32)
	hex.Encode(encoded, id[:])

	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		encoded[0:8],
		encoded[8:12],
		encoded[12:16],
		encoded[16:20],
		encoded[20:32],
	), nil
}

var _ alarm.Repository = (*AlarmRepository)(nil)
