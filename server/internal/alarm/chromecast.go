package alarm

type ChromecastSettings struct {
	Enabled bool `json:"enabled"`

	DeviceID string `json:"deviceId,omitempty"`

	DisplayOffsetMinutes int `json:"displayOffsetMinutes"`

	DisplayDurationMinutes int `json:"displayDurationMinutes"`

	// BoardTimeline selects the board view rotation for this alarm:
	// empty plays the server's default timeline, a preset name selects
	// a board-timeline.yml preset, and a comma-separated view list is
	// a one-off order.
	BoardTimeline string `json:"boardTimeline,omitempty"`
}
