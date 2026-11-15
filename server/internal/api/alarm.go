package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
)

func (a *API) listAlarms(w http.ResponseWriter, r *http.Request) {
	alarms, err := a.services.Alarm.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"alarms": alarms,
	})
}

func (a *API) createAlarm(w http.ResponseWriter, r *http.Request) {
	var alarmRequest alarm.Alarm

	if err := json.NewDecoder(r.Body).Decode(&alarmRequest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid alarm payload")
		return
	}

	if err := a.services.Alarm.Create(r.Context(), &alarmRequest); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"alarm": alarmRequest,
	})
}

func (a *API) getAlarm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	alarmResponse, err := a.services.Alarm.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, alarm.ErrNotFound) {
			writeError(w, http.StatusNotFound, "alarm not found")
			return
		}

		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"alarm": alarmResponse,
	})
}

func (a *API) updateAlarm(w http.ResponseWriter, r *http.Request) {
	var alarmRequest alarm.Alarm

	if err := json.NewDecoder(r.Body).Decode(&alarmRequest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid alarm payload")
		return
	}

	alarmRequest.ID = r.PathValue("id")

	if err := a.services.Alarm.Update(r.Context(), &alarmRequest); err != nil {
		if errors.Is(err, alarm.ErrNotFound) {
			writeError(w, http.StatusNotFound, "alarm not found")
			return
		}

		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"alarm": alarmRequest,
	})
}

func (a *API) deleteAlarm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := a.services.Alarm.Delete(r.Context(), id); err != nil {
		if errors.Is(err, alarm.ErrNotFound) {
			writeError(w, http.StatusNotFound, "alarm not found")
			return
		}

		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
