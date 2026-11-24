package api

import (
	"net/http"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/app"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/auth"
)

type API struct {
	services *app.Services
	devices  *auth.Store
}

func NewRouter(services *app.Services, devices *auth.Store, pairing *PairingHandler) http.Handler {
	a := &API{
		services: services,
		devices:  devices,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", a.health)

	mux.HandleFunc("GET /api/v1/devices", a.listDevices)
	mux.HandleFunc("DELETE /api/v1/devices/{id}", a.revokeDevice)

	mux.HandleFunc("GET /api/v1/devices/sonos", a.getSonos)
	mux.HandleFunc("POST /api/v1/devices/sonos/discover", a.discoverSonos)
	mux.HandleFunc("POST /api/v1/devices/sonos/connect", a.connectSonos)
	mux.HandleFunc("GET /api/v1/devices/sonos/groups", a.getSonosGroups)
	mux.HandleFunc("POST /api/v1/devices/sonos/groups/{groupId}/play", a.playSonosGroup)
	mux.HandleFunc("POST /api/v1/devices/sonos/groups/{groupId}/radio", a.playSonosGroupRadio)
	mux.HandleFunc("GET /api/v1/devices/sonos/groups/{groupId}/radio-stations", a.getSonosGroupRadioStations)
	mux.HandleFunc("POST /api/v1/devices/sonos/groups/{groupId}/stop", a.stopSonosGroup)
	mux.HandleFunc("PUT /api/v1/devices/sonos/groups/{groupId}/volume", a.setSonosGroupVolume)

	mux.HandleFunc("GET /api/v1/devices/hue", a.getHue)
	mux.HandleFunc("POST /api/v1/devices/hue/connect", a.connectHue)
	mux.HandleFunc("GET /api/v1/devices/hue/lights", a.getHueLights)
	mux.HandleFunc("PUT /api/v1/devices/hue/lights/brightness", a.setHueBrightness)
	mux.HandleFunc("POST /api/v1/devices/hue/lights/on", a.turnHueOn)
	mux.HandleFunc("POST /api/v1/devices/hue/lights/off", a.turnHueOff)

	mux.HandleFunc("GET /api/v1/devices/chromecast", a.getChromecast)
	mux.HandleFunc("POST /api/v1/devices/chromecast/discover", a.discoverChromecast)
	mux.HandleFunc("POST /api/v1/devices/chromecast/connect", a.connectChromecast)
	mux.HandleFunc("POST /api/v1/devices/chromecast/display", a.displayChromecastBoard)
	mux.HandleFunc("POST /api/v1/devices/chromecast/stop", a.stopChromecastBoard)

	mux.HandleFunc("POST /api/v1/audio/play", a.playAudio)
	mux.HandleFunc("POST /api/v1/audio/stop", a.stopAudio)
	mux.HandleFunc("PUT /api/v1/audio/volume", a.setVolume)
	mux.HandleFunc("GET /api/v1/audio/radio-stations", a.getRadioStations)

	mux.HandleFunc("GET /api/v1/board/timeline", a.getBoardTimeline)
	mux.HandleFunc("PUT /api/v1/board/timeline/views/{view}", a.updateBoardTimelineView)
	mux.HandleFunc("PUT /api/v1/board/timeline/order", a.setBoardTimelineOrder)
	mux.HandleFunc("PUT /api/v1/board/timeline/intro", a.setBoardTimelineIntro)
	mux.HandleFunc("GET /api/v1/board/timeline/presets", a.listBoardTimelinePresets)
	mux.HandleFunc("POST /api/v1/board/timeline/presets", a.createBoardTimelinePreset)
	mux.HandleFunc("GET /api/v1/board/timeline/presets/{name}", a.getBoardTimelinePreset)
	mux.HandleFunc("PUT /api/v1/board/timeline/presets/{name}", a.updateBoardTimelinePreset)
	mux.HandleFunc("DELETE /api/v1/board/timeline/presets/{name}", a.deleteBoardTimelinePreset)

	mux.HandleFunc("GET /api/v1/alarms", a.listAlarms)
	mux.HandleFunc("POST /api/v1/alarms", a.createAlarm)
	mux.HandleFunc("GET /api/v1/alarms/{id}", a.getAlarm)
	mux.HandleFunc("PUT /api/v1/alarms/{id}", a.updateAlarm)
	mux.HandleFunc("DELETE /api/v1/alarms/{id}", a.deleteAlarm)

	root := http.NewServeMux()
	root.HandleFunc("POST /api/v1/auth/pair", pairing.Pair)
	root.Handle("/api/", authMiddleware(mux, devices))

	return root
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
