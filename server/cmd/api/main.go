package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/api"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/app"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/audio"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/auth"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

type application struct {
	cfg          *config.Config
	router       http.Handler
	audioHandler http.Handler
	certFile     string
	keyFile      string
	pin          string
	fingerprint  string
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	app, err := newApplication(ctx, cfg)
	if err != nil {
		return err
	}

	app.logStartup()

	return serve(ctx, app)
}

func newApplication(ctx context.Context, cfg *config.Config) (*application, error) {
	services, err := app.NewServices(
		ctx,
		cfg.AudioPort,
		cfg.HueAPIKey,
		cfg.SonosEnabled,
	)
	if err != nil {
		return nil, fmt.Errorf("initialising services: %w", err)
	}

	deviceStore, err := auth.OpenStore(filepath.Join(cfg.DataDir, "devices.json"))
	if err != nil {
		return nil, fmt.Errorf("opening device store: %w", err)
	}

	certFile, keyFile, fingerprint, err := auth.EnsureServerCert(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("preparing TLS certificate: %w", err)
	}

	pin := auth.NewPIN()
	pairing := api.NewPairingHandler(pin, deviceStore)

	return &application{
		cfg:          cfg,
		router:       api.NewRouter(services, deviceStore, pairing),
		audioHandler: newAudioHandler(),
		certFile:     certFile,
		keyFile:      keyFile,
		pin:          pin,
		fingerprint:  fingerprint,
	}, nil
}

func newAudioHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/audio/", http.StripPrefix("/audio/", audio.Handler()))
	return mux
}

func (a *application) logStartup() {
	log.Println("================================================")
	log.Printf("  Pairing PIN:      %s", a.pin)
	log.Printf("  Cert SHA-256:     %s", a.fingerprint)
	log.Println("================================================")
}

func serve(ctx context.Context, app *application) error {
	apiServer := newServer(app.cfg.HTTPPort, app.router)
	audioServer := newServer(app.cfg.AudioPort, app.audioHandler)

	serverErrors := make(chan error, 2)

	go func() {
		log.Printf("HTTPS API listening on port %d", app.cfg.HTTPPort)
		if err := apiServer.ListenAndServeTLS(app.certFile, app.keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("API server failed: %w", err)
		}
	}()

	go func() {
		log.Printf("HTTP audio listening on port %d", app.cfg.AudioPort)
		if err := audioServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("audio server failed: %w", err)
		}
	}()

	log.Println("Application started successfully")

	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received")

	case err := <-serverErrors:
		return err
	}

	return shutdownServers(apiServer, audioServer)
}

func shutdownServers(servers ...*http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, server := range servers {
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)

			if closeErr := server.Close(); closeErr != nil {
				return fmt.Errorf("closing server: %w", closeErr)
			}
		}
	}

	log.Println("Application stopped")
	return nil
}

func newServer(port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
