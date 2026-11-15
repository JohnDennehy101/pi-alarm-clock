package alarm

type SonosTarget struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

const (
	TargetSpeaker = "speaker"
	TargetGroup   = "group"
)
