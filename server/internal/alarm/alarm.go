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

	Sonos      SonosSettings      `json:"sonos"`
	Hue        HueSettings        `json:"hue"`
	Chromecast ChromecastSettings `json:"chromecast"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

var ErrNotFound = errors.New("alarm not found")

// MaxStartOffsetMinutes bounds how far before the alarm time the wake-up
// lighting or audio may start, and how far after it the Chromecast image
// may be displayed. It matches the CHECK constraints on the alarms table.
const MaxStartOffsetMinutes = 120
