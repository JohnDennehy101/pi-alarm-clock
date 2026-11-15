package alarm

type SonosSettings struct {
	Target        SonosTarget `json:"target"`
	Source        AudioSource `json:"source"`
	Volume        int         `json:"volume"`
	GradualVolume bool        `json:"gradualVolume"`
}
