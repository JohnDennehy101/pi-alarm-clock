package api

import (
	"net/http"
	"strings"
)

type playAudioRequest struct {
	Type     string `json:"type"`
	Station  string `json:"station,omitempty"`
	Filename string `json:"filename,omitempty"`
}

func (a *API) playAudio(w http.ResponseWriter, r *http.Request) {
	var req playAudioRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	switch strings.ToLower(req.Type) {
	case "radio":
		a.playRadio(w, r, req.Station)
	case "file":
		a.playFile(w, r, req.Filename)
	default:
		writeError(w, http.StatusBadRequest, "type must be radio or file")
	}
}

func (a *API) playRadio(w http.ResponseWriter, r *http.Request, station string) {
	if station == "" {
		writeError(w, http.StatusBadRequest, "station is required")
		return
	}

	if err := a.services.Sonos.PlayRadioStation(r.Context(), station); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Playback started",
	})
}

func (a *API) playFile(w http.ResponseWriter, r *http.Request, filename string) {
	if filename == "" {
		writeError(w, http.StatusBadRequest, "filename is required")
		return
	}

	if a.services.Audio == nil {
		writeError(w, http.StatusServiceUnavailable, "Audio service unavailable")
		return
	}

	audioURL, err := a.services.Audio.GetAudioURL(filename)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := a.services.Sonos.PlayURI(r.Context(), audioURL, ""); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Playback started",
	})
}

func (a *API) stopAudio(w http.ResponseWriter, r *http.Request) {
	if err := a.services.Sonos.Stop(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Playback stopped",
	})
}

func (a *API) setVolume(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Volume int `json:"volume"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if request.Volume < 0 || request.Volume > 100 {
		writeError(w, http.StatusBadRequest, "volume must be between 0 and 100")
		return
	}

	if err := a.services.Sonos.SetVolume(r.Context(), request.Volume); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{
		"volume": request.Volume,
	})
}

func (a *API) getRadioStations(w http.ResponseWriter, r *http.Request) {
	stations, err := a.services.Sonos.GetRadioStations(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stations)
}
