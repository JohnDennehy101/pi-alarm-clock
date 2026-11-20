package display

import (
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

func Handler(renderer *BoardRenderer, boardsRoot string) http.Handler {
	mux := http.NewServeMux()

	if renderer != nil {
		mux.HandleFunc("/display/board.png", liveBoard(renderer))
	}

	if boardsRoot != "" {
		fileServer := http.FileServer(http.Dir(filepath.Join(boardsRoot)))
		mux.Handle(
			"/display/boards/",
			noStoreHeaders(http.StripPrefix("/display/boards/", fileServer)),
		)
	}

	return mux
}

func liveBoard(renderer *BoardRenderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		panel := PanelWeather
		if value := r.URL.Query().Get("panel"); value != "" {
			parsed, err := ParsePanel(value)
			if err != nil {
				http.Error(w, "unknown board panel", http.StatusBadRequest)
				return
			}

			panel = parsed
		}

		png, err := renderer.RenderBoard(r.Context(), time.Now(), BoardView{
			Panel: panel,
		})
		if err != nil {
			http.Error(w, "rendering board failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Length", strconv.Itoa(len(png)))
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodHead {
			return
		}

		_, _ = w.Write(png)
	}
}

func noStoreHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
