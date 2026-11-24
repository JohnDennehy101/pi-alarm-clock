package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/app"
)

func (a *API) getBoardTimeline(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.services.Board.TimelineSettings())
}

func (a *API) updateBoardTimelineView(w http.ResponseWriter, r *http.Request) {
	var request struct {
		DurationSeconds *int64 `json:"durationSeconds"`
		RefreshSeconds  *int64 `json:"refreshSeconds"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if request.DurationSeconds == nil && request.RefreshSeconds == nil {
		writeError(w, http.StatusBadRequest, "durationSeconds or refreshSeconds is required")
		return
	}

	if err := a.services.Board.UpdateViewSettings(
		r.PathValue("view"),
		request.DurationSeconds,
		request.RefreshSeconds,
	); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, a.services.Board.TimelineSettings())
}

func (a *API) setBoardTimelineOrder(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Order json.RawMessage `json:"order"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(request.Order) == 0 {
		writeError(w, http.StatusBadRequest, "order is required")
		return
	}

	var viewList []string
	if err := json.Unmarshal(request.Order, &viewList); err == nil {
		a.applyTimelineOrderViews(w, viewList)
		return
	}

	a.applyTimelineOrderPreset(w, request.Order)
}

func (a *API) applyTimelineOrderViews(w http.ResponseWriter, views []string) {
	if len(views) == 0 {
		writeError(w, http.StatusBadRequest, "order must contain at least one view")
		return
	}

	if err := a.services.Board.SetTimelineOrderViews(views); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, a.services.Board.TimelineSettings())
}

func (a *API) applyTimelineOrderPreset(w http.ResponseWriter, rawOrder json.RawMessage) {
	var expr string
	if err := json.Unmarshal(rawOrder, &expr); err != nil {
		writeError(w, http.StatusBadRequest, "order must be a preset name or a list of view names")
		return
	}

	if strings.TrimSpace(expr) == "" {
		writeError(w, http.StatusBadRequest, "order is required")
		return
	}

	if err := a.services.Board.SetTimelineOrder(expr); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, a.services.Board.TimelineSettings())
}

func (a *API) setBoardTimelineIntro(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Intro *[]string `json:"intro"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if request.Intro == nil {
		writeError(w, http.StatusBadRequest, "intro is required (use an empty list to clear)")
		return
	}

	if err := a.services.Board.SetTimelineIntro(*request.Intro); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, a.services.Board.TimelineSettings())
}

func (a *API) listBoardTimelinePresets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"presets": a.services.Board.ListPresets(),
	})
}

func (a *API) createBoardTimelinePreset(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name  string   `json:"name"`
		Order []string `json:"order"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(request.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	a.saveBoardTimelinePreset(w, presetWriteRequest{
		name:       request.Name,
		order:      request.Order,
		statusCode: http.StatusCreated,
	}, a.services.Board.CreatePreset)
}

func (a *API) getBoardTimelinePreset(w http.ResponseWriter, r *http.Request) {
	preset, err := a.services.Board.GetPreset(r.PathValue("name"))
	if err != nil {
		writeBoardError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, preset)
}

func (a *API) updateBoardTimelinePreset(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Order []string `json:"order"`
	}

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	a.saveBoardTimelinePreset(w, presetWriteRequest{
		name:       r.PathValue("name"),
		order:      request.Order,
		statusCode: http.StatusOK,
	}, a.services.Board.UpdatePreset)
}

type presetWriteRequest struct {
	name       string
	order      []string
	statusCode int
}

func (a *API) saveBoardTimelinePreset(
	w http.ResponseWriter,
	req presetWriteRequest,
	save func(name string, order []string) error,
) {
	if len(req.order) == 0 {
		writeError(w, http.StatusBadRequest, "order is required")
		return
	}

	if err := save(req.name, req.order); err != nil {
		writeBoardError(w, err)
		return
	}

	preset, err := a.services.Board.GetPreset(req.name)
	if err != nil {
		writeBoardError(w, err)
		return
	}

	writeJSON(w, req.statusCode, preset)
}

func (a *API) deleteBoardTimelinePreset(w http.ResponseWriter, r *http.Request) {
	if err := a.services.Board.DeletePreset(r.PathValue("name")); err != nil {
		writeBoardError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeBoardError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrBoardTimelinePresetExists):
		writeError(w, http.StatusConflict, err.Error())

	case errors.Is(err, app.ErrBoardTimelinePresetNotFound):
		writeError(w, http.StatusNotFound, err.Error())

	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}
