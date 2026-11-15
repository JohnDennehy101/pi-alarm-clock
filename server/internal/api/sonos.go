package api

import (
	"net/http"
	"strings"
)

func (a *API) getSonos(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   a.services.Sonos.Status(),
		"speakers": a.services.Sonos.Speakers(),
	})
}

func (a *API) discoverSonos(w http.ResponseWriter, r *http.Request) {
	speakers, err := a.services.Sonos.Discover(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"speakers": speakers,
		"status":   a.services.Sonos.Status(),
	})
}

func (a *API) connectSonos(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SpeakerID string `json:"speakerId"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(request.SpeakerID) == "" {
		writeError(w, http.StatusBadRequest, "speakerId is required")
		return
	}

	if err := a.services.Sonos.Connect(r.Context(), request.SpeakerID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Connected to Sonos speaker",
		"status":  a.services.Sonos.Status(),
	})
}
