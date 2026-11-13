package audio

import (
	"fmt"
	"net/url"
	"path"
)

type AudioManager struct {
	baseURL string
}

func NewAudioManager(port int) (*AudioManager, error) {
	ip, err := GetLocalIP()
	if err != nil {
		return nil, fmt.Errorf("getting local IP: %w", err)
	}

	return &AudioManager{
		baseURL: fmt.Sprintf("http://%s:%d/audio", ip, port),
	}, nil
}

func (a *AudioManager) GetAudioURL(filename string) (string, error) {
	filename = path.Base(filename)

	if filename == "." || filename == "/" {
		return "", fmt.Errorf("invalid audio filename")
	}

	return a.baseURL + "/" + url.PathEscape(filename), nil
}
