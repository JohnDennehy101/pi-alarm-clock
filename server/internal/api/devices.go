package api

import (
	"log"
	"net/http"
)

func (a *API) listDevices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.devices.List())
}

func (a *API) revokeDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !a.devices.Revoke(id) {
		writeError(w, http.StatusNotFound, "unknown device")
		return
	}
	log.Printf("device %s revoked by %s", id, deviceIDFromContext(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"message": "device revoked"})
}

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
		SpeakerName string `json:"speakerName"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if request.SpeakerName == "" {
		writeError(w, http.StatusBadRequest, "speakerName is required")
		return
	}

	if err := a.services.Sonos.Connect(r.Context(), request.SpeakerName); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Connected to Sonos speaker",
		"status":  a.services.Sonos.Status(),
	})
}

func (a *API) getHue(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.services.Hue.Status())
}

func (a *API) connectHue(w http.ResponseWriter, r *http.Request) {
	if err := a.services.Hue.Connect(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, a.services.Hue.Status())
}

func (a *API) getHueLights(w http.ResponseWriter, r *http.Request) {
	lights, err := a.services.Hue.GetLights(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, lights)
}

func (a *API) setHueBrightness(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Brightness int `json:"brightness"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if request.Brightness < 1 || request.Brightness > 100 {
		writeError(w, http.StatusBadRequest, "brightness must be between 1 and 100")
		return
	}

	if err := a.services.Hue.SetBrightness(r.Context(), request.Brightness); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":    "Brightness updated",
		"brightness": request.Brightness,
	})
}

func (a *API) turnHueOn(w http.ResponseWriter, r *http.Request) {
	if err := a.services.Hue.TurnOn(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Hue lights switched on",
	})
}

func (a *API) turnHueOff(w http.ResponseWriter, r *http.Request) {
	if err := a.services.Hue.TurnOff(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Hue lights switched off",
	})
}
