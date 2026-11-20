package app

import (
	"context"
	"sync"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/bus"
)

const realtimeSnapshotMaxAge = time.Minute

type BusRealtimeService struct {
	mu      sync.Mutex
	client  *bus.RealtimeClient
	updates map[string]map[string]time.Time
	fetched time.Time
}

func NewBusRealtimeService(client *bus.RealtimeClient) *BusRealtimeService {
	return &BusRealtimeService{client: client}
}

func (s *BusRealtimeService) Enabled() bool {
	return s != nil && s.client != nil
}

func (s *BusRealtimeService) Updates() map[string]map[string]time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.Enabled() || len(s.updates) == 0 {
		return nil
	}

	updates := make(map[string]map[string]time.Time, len(s.updates))
	for stopID, byTrip := range s.updates {
		stop := make(map[string]time.Time, len(byTrip))
		for tripID, expected := range byTrip {
			stop[tripID] = expected
		}

		updates[stopID] = stop
	}

	return updates
}

func (s *BusRealtimeService) Refresh(ctx context.Context, stopNumber string) error {
	if !s.Enabled() || stopNumber == "" {
		return nil
	}

	s.mu.Lock()
	if !s.fetched.IsZero() && time.Since(s.fetched) < realtimeSnapshotMaxAge {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	updates, err := s.client.Fetch(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.updates = indexRealtimeUpdates(updates)
	s.fetched = time.Now()

	return nil
}

func (s *BusRealtimeService) Enrich(scheduled []bus.Departure) []bus.Departure {
	enriched := append([]bus.Departure(nil), scheduled...)

	if !s.Enabled() {
		return enriched
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range enriched {
		stop, ok := s.updates[enriched[i].StopID]
		if !ok {
			continue
		}

		expected, ok := stop[enriched[i].TripID]
		if !ok || expected.IsZero() {
			continue
		}

		enriched[i].ExpectedTime = expected
		enriched[i].IsRealtime = true
	}

	return enriched
}

func indexRealtimeUpdates(
	updates []bus.RealtimeUpdate,
) map[string]map[string]time.Time {
	byStop := make(map[string]map[string]time.Time)

	for _, update := range updates {
		stop, ok := byStop[update.StopID]
		if !ok {
			stop = make(map[string]time.Time)
			byStop[update.StopID] = stop
		}

		stop[update.TripID] = update.Time
	}

	return byStop
}
