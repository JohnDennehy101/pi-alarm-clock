package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/app"
)

func newBoardTimelineTestAPI(t *testing.T) http.Handler {
	t.Helper()

	board := app.NewBoardService(
		"",
		nil,
		nil,
		app.BoardTimelineConfig{
			Timeline:     nil,
			Catalog:      nil,
			TimelinePath: filepath.Join(t.TempDir(), "board-timeline.yml"),
		},
	)

	a := &API{services: &app.Services{Board: board}}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/board/timeline", a.getBoardTimeline)
	mux.HandleFunc("PUT /api/v1/board/timeline/views/{view}", a.updateBoardTimelineView)
	mux.HandleFunc("PUT /api/v1/board/timeline/order", a.setBoardTimelineOrder)
	mux.HandleFunc("PUT /api/v1/board/timeline/intro", a.setBoardTimelineIntro)
	mux.HandleFunc("GET /api/v1/board/timeline/presets", a.listBoardTimelinePresets)
	mux.HandleFunc("POST /api/v1/board/timeline/presets", a.createBoardTimelinePreset)
	mux.HandleFunc("GET /api/v1/board/timeline/presets/{name}", a.getBoardTimelinePreset)
	mux.HandleFunc("PUT /api/v1/board/timeline/presets/{name}", a.updateBoardTimelinePreset)
	mux.HandleFunc("DELETE /api/v1/board/timeline/presets/{name}", a.deleteBoardTimelinePreset)

	return mux
}

func performBoardRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

func TestBoardTimelineAPIGetAndUpdateView(t *testing.T) {
	handler := newBoardTimelineTestAPI(t)

	rec := performBoardRequest(t, handler, http.MethodGet, "/api/v1/board/timeline", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET timeline status = %d, body %s", rec.Code, rec.Body)
	}

	var settings app.BoardTimelineSettings
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}

	if len(settings.Views) == 0 || len(settings.AvailableViews) == 0 {
		t.Fatalf("settings missing views or availableViews: %+v", settings)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/views/weather",
		`{"durationSeconds":75,"refreshSeconds":20}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT view status = %d, body %s", rec.Code, rec.Body)
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode updated settings: %v", err)
	}

	if settings.Views["weather"].DurationSeconds != 75 {
		t.Fatalf("weather duration = %d, want 75", settings.Views["weather"].DurationSeconds)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/views/weather",
		`{"durationSeconds":0}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT invalid view status = %d, body %s", rec.Code, rec.Body)
	}
}

func TestBoardTimelineAPIOrder(t *testing.T) {
	handler := newBoardTimelineTestAPI(t)

	rec := performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/order",
		`{"order":["bus","progress"]}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT order array status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/order",
		`{"order":"bus-focus"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT order preset status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/order",
		`{"order":"no-such-preset"}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT unknown order status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/order",
		`{"order":[]}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT empty order status = %d, body %s", rec.Code, rec.Body)
	}
}

func TestBoardTimelineAPIPresets(t *testing.T) {
	handler := newBoardTimelineTestAPI(t)

	rec := performBoardRequest(
		t,
		handler,
		http.MethodPost,
		"/api/v1/board/timeline/presets",
		`{"name":"morning","order":["weather","bus.map"]}`,
	)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST preset status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPost,
		"/api/v1/board/timeline/presets",
		`{"name":"morning","order":["weather"]}`,
	)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate POST preset status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(t, handler, http.MethodGet, "/api/v1/board/timeline/presets/morning", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET preset status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(t, handler, http.MethodGet, "/api/v1/board/timeline/presets/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET missing preset status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/presets/morning",
		`{"order":["bus.progress"]}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT preset status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(t, handler, http.MethodDelete, "/api/v1/board/timeline/presets/morning", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE preset status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(t, handler, http.MethodDelete, "/api/v1/board/timeline/presets/default", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("DELETE builtin preset status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(t, handler, http.MethodDelete, "/api/v1/board/timeline/presets/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing preset status = %d, body %s", rec.Code, rec.Body)
	}
}

func TestBoardTimelineAPIIntro(t *testing.T) {
	handler := newBoardTimelineTestAPI(t)

	var settings app.BoardTimelineSettings

	rec := performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/intro",
		`{"intro":["weather"]}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT intro status = %d, body %s", rec.Code, rec.Body)
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}

	if len(settings.Intro) != 1 || settings.Intro[0] != "weather" {
		t.Fatalf("intro = %v, want [weather]", settings.Intro)
	}

	if len(settings.Order) == 0 {
		t.Fatalf("intro update should not clear order: %+v", settings)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/intro",
		`{"intro":["weather","mup"]}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT intro unknown view status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/intro",
		`{}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT intro missing field status = %d, body %s", rec.Code, rec.Body)
	}

	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/intro",
		`{"intro":null}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT intro null status = %d, body %s", rec.Code, rec.Body)
	}

	// An empty list is the explicit "no intro" form.
	rec = performBoardRequest(
		t,
		handler,
		http.MethodPut,
		"/api/v1/board/timeline/intro",
		`{"intro":[]}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT intro clear status = %d, body %s", rec.Code, rec.Body)
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode cleared settings: %v", err)
	}

	if len(settings.Intro) != 0 {
		t.Fatalf("intro after clear = %v, want empty", settings.Intro)
	}

	rec = performBoardRequest(t, handler, http.MethodGet, "/api/v1/board/timeline", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET timeline status = %d, body %s", rec.Code, rec.Body)
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode reloaded settings: %v", err)
	}

	if len(settings.Intro) != 0 {
		t.Fatalf("intro after reload = %v, want empty", settings.Intro)
	}
}
