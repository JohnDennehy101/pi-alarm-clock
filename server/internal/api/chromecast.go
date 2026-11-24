package api

import (
	"net/http"
	"strings"
)

// maxDisplayWindowMinutes mirrors the app-layer bound on the Chromecast
// display window (see app.chromecastMaxDisplayDurationMinutes).
const maxDisplayWindowMinutes = 240

func (a *API) getChromecast(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  a.services.Chromecast.Status(),
		"devices": a.services.Chromecast.Devices(),
	})
}

func (a *API) discoverChromecast(w http.ResponseWriter, r *http.Request) {
	devices, err := a.services.Chromecast.Discover(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"devices": devices,
		"status":  a.services.Chromecast.Status(),
	})
}

func (a *API) connectChromecast(w http.ResponseWriter, r *http.Request) {
	var request struct {
		DeviceID string `json:"deviceId"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(request.DeviceID) == "" {
		writeError(w, http.StatusBadRequest, "deviceId is required")
		return
	}

	if err := a.services.Chromecast.Connect(r.Context(), request.DeviceID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Connected to Chromecast device",
		"status":  a.services.Chromecast.Status(),
	})
}

func (a *API) displayChromecastBoard(w http.ResponseWriter, r *http.Request) {
	var request struct {
		DeviceID      string `json:"deviceId,omitempty"`
		WindowMinutes int    `json:"windowMinutes,omitempty"`
		StopNumber    string `json:"stopNumber,omitempty"`
		BoardTimeline string `json:"boardTimeline,omitempty"`
		BoardIntro    string `json:"boardIntro,omitempty"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" {
		if deviceID = a.services.Chromecast.SelectedDevice(); deviceID == "" {
			writeError(w, http.StatusBadRequest, "deviceId is required")
			return
		}
	}

	if request.WindowMinutes < 0 || request.WindowMinutes > maxDisplayWindowMinutes {
		writeError(w, http.StatusBadRequest, "windowMinutes must be between 0 and 240")
		return
	}

	if request.WindowMinutes == 0 {
		request.WindowMinutes = 1
	}

	if err := a.services.Chromecast.CastBoard(
		r.Context(),
		deviceID,
		request.WindowMinutes,
		strings.TrimSpace(request.StopNumber),
		strings.TrimSpace(request.BoardTimeline),
		strings.TrimSpace(request.BoardIntro),
	); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":       "Board displayed on Chromecast",
		"deviceId":      deviceID,
		"windowMinutes": request.WindowMinutes,
	})
}

func (a *API) stopChromecastBoard(w http.ResponseWriter, r *http.Request) {
	var request struct {
		DeviceID string `json:"deviceId,omitempty"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" {
		if deviceID = a.services.Chromecast.SelectedDevice(); deviceID == "" {
			writeError(w, http.StatusBadRequest, "deviceId is required")
			return
		}
	}

	if err := a.services.Chromecast.StopBoard(r.Context(), deviceID); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":  "Chromecast board stopped",
		"deviceId": deviceID,
	})
}
