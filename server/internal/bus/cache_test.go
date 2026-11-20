package bus

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testLocation(t *testing.T) *time.Location {
	t.Helper()

	location, err := time.LoadLocation("Europe/Dublin")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	return location
}

// testTimetable builds a small timetable through NewTimetable's zip
// path, so round-trips exercise the real encoder input.
func testTimetable(t *testing.T, focus []string) *Timetable {
	t.Helper()

	timetable, err := NewTimetable(
		filepath.Join("testdata", "gtfs.zip"),
		testLocation(t),
		focus...,
	)
	if err != nil {
		t.Fatalf("NewTimetable: %v", err)
	}

	return timetable
}

// decodeCacheFromBytesForTest decodes an encoded cache from memory
// with a heap rows loader, mirroring the level 1 path.
func decodeCacheFromBytesForTest(
	data []byte,
	gtfsSum [32]byte,
	location *time.Location,
	focus []string,
) (*Timetable, error) {
	cursor := &bytesCursor{data: data}

	return decodeCache(cursor, gtfsSum, location, focus, func(rows int) ([]stopDeparture, error) {
		base := cursor.pos()

		entries := make([]stopDeparture, rows)
		for i := range entries {
			b := data[base+int64(i)*8 : base+int64(i)*8+8]
			entries[i] = stopDeparture{
				TripIndex: int32(binary.LittleEndian.Uint32(b)),
				Seconds:   int32(binary.LittleEndian.Uint32(b[4:])),
			}
		}

		cursor.off += rows * 8

		return entries, nil
	})
}

func TestCacheRoundTrip(t *testing.T) {
	focus := []string{"000001"}
	original := testTimetable(t, focus)

	sum := [32]byte{1, 2, 3}

	var buf bytes.Buffer

	if err := original.writeCacheBuffer(&buf, sum, focus); err != nil {
		t.Fatalf("encode: %v", err)
	}

	data := buf.Bytes()

	tests := []struct {
		name string
		load func() (*Timetable, error)
	}{
		{
			name: "level1-heap",
			load: func() (*Timetable, error) {
				return decodeCacheFromBytesForTest(
					data,
					sum,
					testLocation(t),
					focus,
				)
			},
		},
		{
			name: "level2-mmap",
			load: func() (*Timetable, error) {
				path := filepath.Join(t.TempDir(), "timetable.cache")
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatalf("write cache: %v", err)
				}

				return loadTimetableCache(path, sum, testLocation(t), focus)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restored, err := test.load()
			if err != nil {
				t.Fatalf("load: %v", err)
			}

			if len(restored.stopIDs) != len(original.stopIDs) {
				t.Errorf("stops = %d, want %d", len(restored.stopIDs), len(original.stopIDs))
			}

			if len(restored.tripInfos) != len(original.tripInfos) {
				t.Errorf("trips = %d, want %d", len(restored.tripInfos), len(original.tripInfos))
			}

			if len(restored.focusTrips) != len(original.focusTrips) {
				t.Errorf("focus trips = %d, want %d", len(restored.focusTrips), len(original.focusTrips))
			}

			for stopNumber, stopID := range original.stopIDs {
				if restored.stopIDs[stopNumber] != stopID {
					t.Errorf("stop %s = %q, want %q", stopNumber, restored.stopIDs[stopNumber], stopID)
				}
			}

			for stopID, departures := range original.departures {
				restoredDepartures := restored.departures[stopID]
				if len(restoredDepartures) != len(departures) {
					t.Errorf(
						"stop %s departures = %d, want %d",
						stopID,
						len(restoredDepartures),
						len(departures),
					)

					continue
				}

				for i, departure := range departures {
					if restoredDepartures[i] != departure {
						t.Errorf(
							"stop %s departure %d = %+v, want %+v",
							stopID,
							i,
							restoredDepartures[i],
							departure,
						)
					}
				}
			}

			assertSameDepartures(t, original, restored)
			assertSameMapTrips(t, original, restored)
		})
	}
}

func assertSameDepartures(t *testing.T, original, restored *Timetable) {
	t.Helper()

	now := time.Date(2026, 3, 2, 7, 45, 0, 0, testLocation(t))

	for _, stopNumber := range []string{"000001", "000002"} {
		originalDepartures, err := original.GetDepartures(stopNumber, now, 5)
		if err != nil {
			// Stop may legitimately have no departures in this tiny
			// dataset at this time; the interesting case is when the
			// original succeeds.
			continue
		}

		restoredDepartures, err := restored.GetDepartures(stopNumber, now, 5)
		if err != nil {
			t.Fatalf("restored GetDepartures(%s): %v", stopNumber, err)
		}

		if len(originalDepartures) == 0 && len(restoredDepartures) == 0 {
			continue
		}

		if len(restoredDepartures) != len(originalDepartures) {
			t.Fatalf(
				"restored departures for %s = %d, want %d",
				stopNumber,
				len(restoredDepartures),
				len(originalDepartures),
			)
		}

		for i := range originalDepartures {
			if !sameDepartures(originalDepartures[i], restoredDepartures[i]) {
				t.Errorf(
					"departure %d for %s = %+v, want %+v",
					i,
					stopNumber,
					restoredDepartures[i],
					originalDepartures[i],
				)
			}
		}
	}
}

// sameDepartures compares Departures field by field, using Equal for
// times: Time's == compares location pointers, not instants.
func sameDepartures(a, b Departure) bool {
	return a.StopNumber == b.StopNumber &&
		a.StopID == b.StopID &&
		a.TripID == b.TripID &&
		a.Route == b.Route &&
		a.Destination == b.Destination &&
		a.ScheduledTime.Equal(b.ScheduledTime) &&
		a.ExpectedTime.Equal(b.ExpectedTime) &&
		a.MinutesAway == b.MinutesAway &&
		a.IsRealtime == b.IsRealtime
}

func assertSameMapTrips(t *testing.T, original, restored *Timetable) {
	t.Helper()

	now := time.Date(2026, 3, 2, 7, 50, 0, 0, testLocation(t))

	originalTrips, err := original.MapTrips("000001", now, 60, nil)
	if err != nil {
		t.Fatalf("original MapTrips: %v", err)
	}

	restoredTrips, err := restored.MapTrips("000001", now, 60, nil)
	if err != nil {
		t.Fatalf("restored MapTrips: %v", err)
	}

	if len(originalTrips) == 0 {
		t.Fatal("test dataset has no focus trips; extend testdata")
	}

	if len(restoredTrips) != len(originalTrips) {
		t.Fatalf("map trips = %d, want %d", len(restoredTrips), len(originalTrips))
	}

	for i := range originalTrips {
		original, restored := originalTrips[i], restoredTrips[i]

		if original.Route != restored.Route ||
			original.TripID != restored.TripID ||
			original.Lat != restored.Lat ||
			original.Lon != restored.Lon ||
			original.PrevLat != restored.PrevLat ||
			original.PrevLon != restored.PrevLon ||
			original.ProgressFraction != restored.ProgressFraction ||
			original.StopsRemaining != restored.StopsRemaining {
			t.Errorf("map trip %d = %+v, want %+v", i, restored, original)
		}
	}
}

func TestCacheInvalidation(t *testing.T) {
	focus := []string{"000001"}
	original := testTimetable(t, focus)

	path := filepath.Join(t.TempDir(), "timetable.cache")

	if err := original.WriteCache(path, [32]byte{9}, focus); err != nil {
		t.Fatalf("WriteCache: %v", err)
	}

	tests := []struct {
		name    string
		sum     [32]byte
		focus   []string
		wantErr bool
	}{
		{name: "matching", sum: [32]byte{9}, focus: focus, wantErr: false},
		{name: "dataset changed", sum: [32]byte{8}, focus: focus, wantErr: true},
		{name: "focus changed", sum: [32]byte{9}, focus: []string{"000002"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := loadTimetableCache(path, test.sum, testLocation(t), test.focus)
			if (err != nil) != test.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestCacheTruncated(t *testing.T) {
	focus := []string{"000001"}
	original := testTimetable(t, focus)

	var buf bytes.Buffer
	if err := original.writeCacheBuffer(&buf, [32]byte{9}, focus); err != nil {
		t.Fatalf("encode: %v", err)
	}

	path := filepath.Join(t.TempDir(), "timetable.cache")

	data := buf.Bytes()

	for _, cut := range []int{5, 50, len(data) / 2, len(data) - 3} {
		if err := os.WriteFile(path, data[:cut], 0o644); err != nil {
			t.Fatalf("write truncated cache: %v", err)
		}

		if _, err := loadTimetableCache(path, [32]byte{9}, testLocation(t), focus); err == nil {
			t.Errorf("truncated cache at %d bytes decoded without error", cut)
		}
	}
}

func TestOpenTimetableMissAndHit(t *testing.T) {
	zipPath := filepath.Join("testdata", "gtfs.zip")
	cachePath := filepath.Join(t.TempDir(), "timetable.cache")
	focus := []string{"000001"}

	now := time.Date(2026, 3, 2, 7, 45, 0, 0, testLocation(t))

	first, err := OpenTimetable(zipPath, cachePath, testLocation(t), focus...)
	if err != nil {
		t.Fatalf("OpenTimetable (miss): %v", err)
	}

	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache not written after miss: %v", err)
	}

	second, err := OpenTimetable(zipPath, cachePath, testLocation(t), focus...)
	if err != nil {
		t.Fatalf("OpenTimetable (hit): %v", err)
	}

	// The warm load should map the cache (level 2) and still carry
	// the same departure data as the cold, stream-parsed load.
	if second.cacheMmap == nil {
		t.Fatal("warm load did not mmap the cache")
	}

	if len(first.departures) != len(second.departures) {
		t.Errorf(
			"hit departures = %d stops, miss had %d",
			len(second.departures),
			len(first.departures),
		)
	}

	restoredDepartures, err := second.GetDepartures("000001", now, 5)
	if err != nil {
		t.Fatalf("warm GetDepartures: %v", err)
	}

	if len(restoredDepartures) == 0 {
		t.Fatal("test dataset has no departures; extend testdata")
	}
}
