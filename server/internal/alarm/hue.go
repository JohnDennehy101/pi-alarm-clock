package alarm

type HueSettings struct {
	Enabled bool `json:"enabled"`

	LightIDs []string `json:"lightIds,omitempty"`

	StartBrightness int `json:"startBrightness"`
	Brightness      int `json:"brightness"`

	GradualBrightness  bool `json:"gradualBrightness"`
	BrightnessDuration int  `json:"brightnessDuration"`

	ColourTemperature int `json:"colourTemperature,omitempty"`
}
