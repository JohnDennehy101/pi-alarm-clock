package api

import (
	"net/http"
)

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

	if err := validateRange(request.Brightness, 1, 100, "brightness"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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
