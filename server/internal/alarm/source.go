package alarm

type AudioSource struct {
	Type string `json:"type"`
	URI  string `json:"uri,omitempty"`
	Name string `json:"name,omitempty"`
}

const (
	SourceRadio = "radio"
	SourceAudio = "audio"
)
