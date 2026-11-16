package alarm

type SonosSettings struct {
	Target SonosTarget `json:"target"`
	Source AudioSource `json:"source"`
	Volume int         `json:"volume"`

	// StartVolume is the volume gradual playback begins at before ramping
	// up to Volume. Zero ramps from silence.
	StartVolume int `json:"startVolume"`

	GradualVolume  bool `json:"gradualVolume"`
	VolumeDuration int  `json:"volumeDuration"`

	// StartOffsetMinutes is how many minutes before the alarm time playback
	// (and the gradual volume ramp, if enabled) begins. Zero starts audio at
	// the alarm time.
	StartOffsetMinutes int `json:"startOffsetMinutes"`
}
