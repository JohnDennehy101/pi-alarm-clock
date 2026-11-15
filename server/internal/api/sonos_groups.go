package api

import (
	"context"
	"net/http"
	"strings"
)

func (a *API) getSonosGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := a.services.Sonos.Groups(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"groups": groups,
	})
}

func (a *API) playSonosGroup(w http.ResponseWriter, r *http.Request) {
	var request struct {
		URI      string `json:"uri"`
		Metadata string `json:"metadata"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(request.URI) == "" {
		writeError(w, http.StatusBadRequest, "uri is required")
		return
	}

	a.sonosGroupAction(
		w,
		r,
		sonosGroupID(r),
		"playback started",
		func(ctx context.Context, groupID string) error {
			return a.services.Sonos.PlayGroupURI(
				ctx,
				groupID,
				request.URI,
				request.Metadata,
			)
		},
	)
}

func (a *API) playSonosGroupRadio(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(request.Name) == "" {
		writeError(w, http.StatusBadRequest, "station name is required")
		return
	}

	a.sonosGroupAction(
		w,
		r,
		sonosGroupID(r),
		"radio playback started",
		func(ctx context.Context, groupID string) error {
			return a.services.Sonos.PlayGroupRadioStation(
				ctx,
				groupID,
				request.Name,
			)
		},
	)
}

func (a *API) getSonosGroupRadioStations(w http.ResponseWriter, r *http.Request) {
	stations, err := a.services.Sonos.GetGroupRadioStations(
		r.Context(),
		sonosGroupID(r),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"stations": stations,
	})
}

func (a *API) stopSonosGroup(w http.ResponseWriter, r *http.Request) {
	a.sonosGroupAction(
		w,
		r,
		sonosGroupID(r),
		"playback stopped",
		a.services.Sonos.StopGroup,
	)
}

func (a *API) setSonosGroupVolume(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Volume int `json:"volume"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateRange(request.Volume, 0, 100, "volume"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := a.services.Sonos.SetGroupVolume(
		r.Context(),
		sonosGroupID(r),
		request.Volume,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "group volume updated",
		"volume":  request.Volume,
	})
}

func sonosGroupID(r *http.Request) string {
	return r.PathValue("groupId")
}

func (a *API) sonosGroupAction(
	w http.ResponseWriter,
	r *http.Request,
	groupID string,
	message string,
	action func(context.Context, string) error,
) {
	if err := action(r.Context(), groupID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": message,
	})
}
