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

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "device revoked",
	})
}
