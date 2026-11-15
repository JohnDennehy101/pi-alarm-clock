package alarm

import (
	"errors"
	"time"
)

type Weekday string

const (
	Monday    Weekday = "monday"
	Tuesday   Weekday = "tuesday"
	Wednesday Weekday = "wednesday"
	Thursday  Weekday = "thursday"
	Friday    Weekday = "friday"
	Saturday  Weekday = "saturday"
	Sunday    Weekday = "sunday"
)

type Alarm struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Time    string `json:"time"`
	Enabled bool   `json:"enabled"`

	Days []Weekday  `json:"days,omitempty"`
	Date *time.Time `json:"date,omitempty"`

	Sonos SonosSettings `json:"sonos"`
	Hue   HueSettings   `json:"hue"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

var ErrNotFound = errors.New("alarm not found")
