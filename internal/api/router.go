package api

import (
	"net/http"

	"github.com/johndennehy101/pi-alarm-clock/internal/app"
	"github.com/johndennehy101/pi-alarm-clock/internal/audio"
)

type API struct {
	services *app.Services
}

func NewRouter(services *app.Services, apiToken string) http.Handler {
	a := &API{
		services: services,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", a.health)

	mux.HandleFunc("GET /api/v1/devices/sonos", a.getSonos)
	mux.HandleFunc("POST /api/v1/devices/sonos/discover", a.discoverSonos)
	mux.HandleFunc("POST /api/v1/devices/sonos/connect", a.connectSonos)

	mux.HandleFunc("GET /api/v1/devices/hue", a.getHue)
	mux.HandleFunc("POST /api/v1/devices/hue/connect", a.connectHue)
	mux.HandleFunc("GET /api/v1/devices/hue/lights", a.getHueLights)
	mux.HandleFunc("PUT /api/v1/devices/hue/lights/brightness", a.setHueBrightness)
	mux.HandleFunc("POST /api/v1/devices/hue/lights/on", a.turnHueOn)
	mux.HandleFunc("POST /api/v1/devices/hue/lights/off", a.turnHueOff)

	mux.HandleFunc("POST /api/v1/audio/play", a.playAudio)
	mux.HandleFunc("POST /api/v1/audio/stop", a.stopAudio)
	mux.HandleFunc("PUT /api/v1/audio/volume", a.setVolume)
	mux.HandleFunc("GET /api/v1/audio/radio-stations", a.getRadioStations)

	root := http.NewServeMux()
	root.Handle("/api/", authMiddleware(mux, apiToken))

	root.Handle(
		"/audio/",
		http.StripPrefix("/audio/", audio.Handler()),
	)

	return root
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
