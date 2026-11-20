package bus

import (
	"math"
	"testing"
	"time"
)

func TestShapePointAtDist(t *testing.T) {
	shape := []shapePoint{
		{Lat: 53.3100, Lon: -6.2100, Dist: 0},
		{Lat: 53.3120, Lon: -6.2100, Dist: 222.64},
		{Lat: 53.3140, Lon: -6.2100, Dist: 445.28},
	}

	lat, lon := shapePointAtDist(shape, -10)
	if lat != shape[0].Lat || lon != shape[0].Lon {
		t.Errorf("position before start = %v,%v, want first point", lat, lon)
	}

	lat, lon = shapePointAtDist(shape, math.MaxFloat64)
	last := shape[len(shape)-1]
	if lat != last.Lat || lon != last.Lon {
		t.Errorf("position past end = %v,%v, want last point", lat, lon)
	}

	// Halfway between the first two points.
	lat, lon = shapePointAtDist(shape, 111.32)
	if math.Abs(lat-53.3110) > 0.0001 {
		t.Errorf("midpoint lat = %v, want ~53.311", lat)
	}
}

func TestPositionOnShapeAtInterpolates(t *testing.T) {
	trip := &focusTrip{
		Stops: []focusTripStop{
			{StopID: "s1", Lat: 53.3100, Lon: -6.2100, Duration: 10 * time.Minute},
			{StopID: "s2", Lat: 53.3120, Lon: -6.2100, Duration: 20 * time.Minute},
			{StopID: "s3", Lat: 53.3140, Lon: -6.2100, Duration: 30 * time.Minute},
		},
		Shape: []shapePoint{
			{Lat: 53.3100, Lon: -6.2100, Dist: 0},
			{Lat: 53.3120, Lon: -6.2100, Dist: 222.64},
			{Lat: 53.3140, Lon: -6.2100, Dist: 445.28},
		},
		shapeAtStop: []float64{0, 222.64, 445.28},
	}

	lat, lon := positionOnShapeAt(trip, 5*time.Minute)
	if lat != trip.Stops[0].Lat || lon != trip.Stops[0].Lon {
		t.Errorf("position before start = %v,%v, want first stop", lat, lon)
	}

	// Halfway between s1 and s2 by time.
	lat, lon = positionOnShapeAt(trip, 15*time.Minute)
	if math.Abs(lat-53.3110) > 0.0001 {
		t.Errorf("halfway lat = %v, want ~53.311", lat)
	}
	if math.Abs(lon-(-6.2100)) > 0.0001 {
		t.Errorf("halfway lon = %v, want -6.21", lon)
	}

	lat, lon = positionOnShapeAt(trip, 35*time.Minute)
	last := trip.Stops[len(trip.Stops)-1]
	if lat != last.Lat || lon != last.Lon {
		t.Errorf("position after end = %v,%v, want last stop", lat, lon)
	}
}

func TestTripDelayAtFocusStop(t *testing.T) {
	serviceDate := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)

	trip := &focusTrip{
		Stops: []focusTripStop{
			{StopID: "s1", Duration: 10 * time.Minute},
			{StopID: "s2", Duration: 20 * time.Minute},
		},
		focusIndex: 1,
	}

	updates := map[string]map[string]time.Time{
		"s2": {
			trip.TripID: serviceDate.Add(24 * time.Minute),
		},
	}

	delay, isRealtime := tripDelay(trip, serviceDate, updates)
	if !isRealtime || delay != 4*time.Minute {
		t.Errorf("tripDelay = %v (realtime %v), want 4m realtime", delay, isRealtime)
	}

	delay, isRealtime = tripDelay(trip, serviceDate, nil)
	if isRealtime || delay != 0 {
		t.Errorf(
			"tripDelay without updates = %v (realtime %v), want 0m",
			delay,
			isRealtime,
		)
	}
}

func TestTripDelayIgnoresNoise(t *testing.T) {
	serviceDate := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)

	trip := &focusTrip{
		TripID: "t1",
		Stops: []focusTripStop{
			{StopID: "s1", Duration: 10 * time.Minute},
		},
		focusIndex: 0,
	}

	// A wildly distant expected time reads as feed noise, not a delay.
	updates := map[string]map[string]time.Time{
		"s1": {
			"t1": serviceDate.Add(10 * time.Hour),
		},
	}

	delay, isRealtime := tripDelay(trip, serviceDate, updates)
	if !isRealtime || delay != 0 {
		t.Errorf(
			"tripDelay = %v (realtime %v), want 0m realtime",
			delay,
			isRealtime,
		)
	}
}
