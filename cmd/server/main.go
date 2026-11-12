package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/internal/api"
	"github.com/johndennehy101/pi-alarm-clock/internal/app"
	"github.com/johndennehy101/pi-alarm-clock/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
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

	services, err := app.NewServices(
		ctx,
		cfg.HTTPPort,
		cfg.HueAPIKey,
		cfg.SonosEnabled,
	)
	if err != nil {
		return fmt.Errorf("initialising services: %w", err)
	}

	router := api.NewRouter(services, cfg.APIToken)

	return serve(ctx, cfg.HTTPPort, router)
}

func serve(ctx context.Context, port int, handler http.Handler) error {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("HTTP server listening on port %d", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	log.Println("Application started successfully")

	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received")

	case err := <-serverErrors:
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful HTTP shutdown failed: %v", err)

		if closeErr := server.Close(); closeErr != nil {
			return fmt.Errorf("closing HTTP server: %w", closeErr)
		}
	}

	log.Println("Application stopped")
	return nil
}
