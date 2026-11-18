package alarm

type ChromecastSettings struct {
	Enabled bool `json:"enabled"`

	DeviceID string `json:"deviceId,omitempty"`

	DisplayOffsetMinutes int `json:"displayOffsetMinutes"`

	DisplayDurationMinutes int `json:"displayDurationMinutes"`
}
