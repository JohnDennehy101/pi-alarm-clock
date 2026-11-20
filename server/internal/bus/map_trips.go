package bus

import (
	"archive/zip"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

const (
	mapTripWindowMinutes = 25
)

func (t *Timetable) loadFocusTrips(
	reader *zip.Reader,
	stopNumbers []string,
) error {
	if len(stopNumbers) == 0 || len(t.departures) == 0 {
		return nil
	}

	focusStopIDs := make(map[string]bool, len(stopNumbers))
	for _, stopNumber := range stopNumbers {
		if stopID, ok := t.stopIDs[stopNumber]; ok {
			focusStopIDs[stopID] = true
		}
	}

	if len(focusStopIDs) == 0 {
		return nil
	}

	tripIndexes := t.focusTripIndexes(focusStopIDs)

	if len(tripIndexes) == 0 {
		return nil
	}

	stopsByTrip, err := t.loadFocusStopTimes(reader, tripIndexes, focusStopIDs)
	if err != nil {
		return err
	}

	shapes, err := t.loadFocusShapes(reader, tripIndexes)
	if err != nil {
		return err
	}

	t.focusTrips = make(map[string]*focusTrip, len(stopsByTrip))

	for tripID, stops := range stopsByTrip {
		info := t.tripInfos[t.tripIndex[tripID]]

		trip := &focusTrip{
			TripID:      tripID,
			Route:       info.Route,
			Destination: info.Destination,
			ServiceID:   info.ServiceID,
			ShapeID:     info.ShapeID,
			Stops:       stops,
			focusIndex:  -1,
		}

		for i, stop := range stops {
			if focusStopIDs[stop.StopID] {
				trip.focusIndex = i
				break
			}
		}

		if trip.focusIndex < 0 {
			continue
		}

		trip.Shape = shapes[trip.ShapeID]
		if len(trip.Shape) == 0 {
			trip.Shape = shapeFromStops(stops)
		}

		trip.shapeAtStop = projectStopsOnShape(stops, trip.Shape)

		t.focusTrips[tripID] = trip
	}

	return nil
}

func (t *Timetable) focusTripIndexes(
	focusStopIDs map[string]bool,
) map[string]int {
	tripIndexes := make(map[string]int)

	for stopID := range focusStopIDs {
		for _, scheduled := range t.departures[stopID] {
			index := int(scheduled.TripIndex)
			tripIndexes[t.tripInfos[index].TripID] = index
		}
	}

	return tripIndexes
}

func (t *Timetable) loadFocusStopTimes(
	reader *zip.Reader,
	tripIndexes map[string]int,
	focusStopIDs map[string]bool,
) (map[string][]focusTripStop, error) {
	stopsByTrip := make(map[string][]focusTripStop, len(tripIndexes))

	err := t.streamCSV(reader, "stop_times.txt", func(record []string, headers map[string]int) error {
		tripID := field(record, headers, "trip_id")
		if _, wanted := tripIndexes[tripID]; !wanted {
			return nil
		}

		stopID := field(record, headers, "stop_id")

		timeValue := field(record, headers, "departure_time")
		if timeValue == "" {
			timeValue = field(record, headers, "arrival_time")
		}

		offset, err := parseGTFSTime(timeValue)
		if err != nil {
			return nil
		}

		sequence, err := strconv.Atoi(field(record, headers, "stop_sequence"))
		if err != nil {
			return nil
		}

		place, ok := t.stopPlaces[stopID]
		if !ok {
			return nil
		}

		stopsByTrip[tripID] = append(stopsByTrip[tripID], focusTripStop{
			StopID:   stopID,
			Lat:      place.Lat,
			Lon:      place.Lon,
			Sequence: sequence,
			Duration: offset,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, stops := range stopsByTrip {
		sort.SliceStable(stops, func(i, j int) bool {
			return stops[i].Sequence < stops[j].Sequence
		})
	}

	return stopsByTrip, nil
}

func (t *Timetable) loadFocusShapes(
	reader *zip.Reader,
	tripIndexes map[string]int,
) (map[string][]shapePoint, error) {
	wanted := make(map[string]bool)
	for _, index := range tripIndexes {
		if shapeID := t.tripInfos[index].ShapeID; shapeID != "" {
			wanted[shapeID] = true
		}
	}

	if len(wanted) == 0 {
		return nil, nil
	}

	indexByShape := make(map[string][]int, len(wanted))
	for _, index := range tripIndexes {
		info := t.tripInfos[index]
		if wanted[info.ShapeID] {
			indexByShape[info.ShapeID] = append(
				indexByShape[info.ShapeID],
				index,
			)
		}
	}

	rawByShape := make(map[string][]shapePoint, len(wanted))

	err := t.streamCSV(reader, "shapes.txt", func(record []string, headers map[string]int) error {
		shapeID := field(record, headers, "shape_id")
		if !wanted[shapeID] {
			return nil
		}

		lat, latErr := strconv.ParseFloat(field(record, headers, "shape_pt_lat"), 64)
		lon, lonErr := strconv.ParseFloat(field(record, headers, "shape_pt_lon"), 64)
		if latErr != nil || lonErr != nil {
			return nil
		}

		sequence, err := strconv.Atoi(field(record, headers, "shape_pt_sequence"))
		if err != nil {
			return nil
		}

		rawByShape[shapeID] = append(rawByShape[shapeID], shapePoint{
			Lat:  lat,
			Lon:  lon,
			Dist: float64(sequence),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	shapes := make(map[string][]shapePoint, len(rawByShape))
	for shapeID, points := range rawByShape {
		sort.Slice(points, func(i, j int) bool {
			return points[i].Dist < points[j].Dist
		})

		for i := range points {
			points[i].Dist = 0
			if i > 0 {
				points[i].Dist =
					points[i-1].Dist + haversineMeters(
						points[i-1].Lat,
						points[i-1].Lon,
						points[i].Lat,
						points[i].Lon,
					)
			}
		}

		shapes[shapeID] = points
	}

	return shapes, nil
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	return 6371000 * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func shapeFromStops(stops []focusTripStop) []shapePoint {
	shape := make([]shapePoint, len(stops))

	for i, stop := range stops {
		point := shapePoint{Lat: stop.Lat, Lon: stop.Lon}
		if i > 0 {
			point.Dist =
				shape[i-1].Dist + haversineMeters(
					shape[i-1].Lat,
					shape[i-1].Lon,
					point.Lat,
					point.Lon,
				)
		}

		shape[i] = point
	}

	return shape
}

func projectStopsOnShape(
	stops []focusTripStop,
	shape []shapePoint,
) []float64 {
	distances := make([]float64, len(stops))

	for i, stop := range stops {
		bestIndex := 0
		bestDist := math.MaxFloat64

		for j, point := range shape {
			d := (point.Lat-stop.Lat)*(point.Lat-stop.Lat) +
				(point.Lon-stop.Lon)*(point.Lon-stop.Lon)

			if d < bestDist {
				bestDist = d
				bestIndex = j
			}
		}

		distances[i] = shape[bestIndex].Dist
	}

	return distances
}

func shapePointAtDist(
	shape []shapePoint,
	dist float64,
) (float64, float64) {
	if len(shape) == 0 {
		return 0, 0
	}

	if dist <= shape[0].Dist {
		return shape[0].Lat, shape[0].Lon
	}

	last := shape[len(shape)-1]
	if dist >= last.Dist {
		return last.Lat, last.Lon
	}

	for i := 1; i < len(shape); i++ {
		if shape[i].Dist >= dist {
			prev := shape[i-1]
			next := shape[i]

			span := next.Dist - prev.Dist
			if span <= 0 {
				return next.Lat, next.Lon
			}

			f := (dist - prev.Dist) / span

			return prev.Lat + (next.Lat-prev.Lat)*f,
				prev.Lon + (next.Lon-prev.Lon)*f
		}
	}

	return last.Lat, last.Lon
}

func positionOnShapeAt(
	trip *focusTrip,
	offset time.Duration,
) (float64, float64) {
	dist, _ := distanceOnShapeAt(trip, offset)

	return shapePointAtDist(trip.Shape, dist)
}

func distanceOnShapeAt(
	trip *focusTrip,
	offset time.Duration,
) (float64, int) {
	stops := trip.Stops

	if offset <= stops[0].Duration {
		return trip.shapeAtStop[0], 0
	}

	for i := 1; i < len(stops); i++ {
		if offset < stops[i].Duration {
			span := stops[i].Duration - stops[i-1].Duration
			if span <= 0 {
				return trip.shapeAtStop[i], i - 1
			}

			f := float64(offset-stops[i-1].Duration) / float64(span)

			front := trip.shapeAtStop[i-1]
			back := trip.shapeAtStop[i]

			return front + (back-front)*f, i - 1
		}
	}

	return trip.shapeAtStop[len(stops)-1], len(stops) - 1
}

func (t *Timetable) MapTrips(
	stopNumber string,
	now time.Time,
	windowMinutes int,
	updates map[string]map[string]time.Time,
) ([]MapTrip, error) {
	stopID, exists := t.stopIDs[stopNumber]
	if !exists {
		return nil, fmt.Errorf("unknown bus stop number %q", stopNumber)
	}

	if t.focusTrips == nil {
		return nil, fmt.Errorf("map trips unavailable for stop %q", stopNumber)
	}

	if windowMinutes <= 0 {
		windowMinutes = mapTripWindowMinutes
	}

	now = now.In(t.location)

	today := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		t.location,
	)

	var trips []MapTrip

	for offset := -1; offset <= 1; offset++ {
		serviceDate := today.AddDate(0, 0, offset)

		for _, trip := range t.focusTrips {
			if trip.Stops[trip.focusIndex].StopID != stopID {
				continue
			}

			if !t.isServiceActive(trip.ServiceID, serviceDate) {
				continue
			}

			arrival := serviceDate.Add(
				trip.Stops[trip.focusIndex].Duration,
			)

			delay, isRealtime := tripDelay(trip, serviceDate, updates)
			expected := arrival.Add(delay)

			if expected.Before(now) {
				continue
			}

			if expected.After(now.Add(
				time.Duration(windowMinutes) * time.Minute,
			)) {
				continue
			}

			tripOffset := now.Sub(serviceDate) - delay
			lat, lon := positionOnShapeAt(trip, tripOffset)
			prevLat, prevLon := positionOnShapeAt(
				trip,
				tripOffset-time.Minute,
			)

			distNow, segmentIndex := distanceOnShapeAt(trip, tripOffset)
			distPrev, _ := distanceOnShapeAt(
				trip,
				tripOffset-time.Minute,
			)

			focusDist := trip.shapeAtStop[trip.focusIndex]
			if focusDist <= 0 {
				focusDist = 1
			}

			stopFractions := make([]float64, 0, trip.focusIndex+1)
			for i := 0; i <= trip.focusIndex; i++ {
				stopFractions = append(
					stopFractions,
					trip.shapeAtStop[i]/focusDist,
				)
			}

			stopsRemaining := trip.focusIndex - segmentIndex
			if stopsRemaining < 0 {
				stopsRemaining = 0
			}

			trips = append(trips, MapTrip{
				TripID:      trip.TripID,
				Route:       trip.Route,
				Destination: trip.Destination,
				ShapeID:     trip.ShapeID,

				Lat:     lat,
				Lon:     lon,
				PrevLat: prevLat,
				PrevLon: prevLon,

				Arrival: expected,
				MinutesToStop: int(math.Ceil(
					expected.Sub(now).Minutes(),
				)),
				IsRealtime: isRealtime,

				ProgressFraction: distNow / focusDist,
				PreviousFraction: distPrev / focusDist,
				StopFractions:    stopFractions,
				StopsRemaining:   stopsRemaining,
				ScheduledArrival: arrival,

				Polyline: shapeCoordinates(trip.Shape),
				Stops:    stopCoordinates(trip.Stops),
			})
		}
	}

	sort.Slice(trips, func(i, j int) bool {
		return trips[i].Arrival.Before(trips[j].Arrival)
	})

	return trips, nil
}

func (t *Timetable) StopPosition(stopNumber string) (MapCoordinate, error) {
	stopID, ok := t.stopIDs[stopNumber]
	if !ok {
		return MapCoordinate{}, fmt.Errorf("unknown bus stop number %q", stopNumber)
	}

	place, ok := t.stopPlaces[stopID]
	if !ok {
		return MapCoordinate{}, fmt.Errorf(
			"no coordinates for bus stop number %q",
			stopNumber,
		)
	}

	return MapCoordinate{Lat: place.Lat, Lon: place.Lon}, nil
}

func tripDelay(
	trip *focusTrip,
	serviceDate time.Time,
	updates map[string]map[string]time.Time,
) (time.Duration, bool) {
	if len(updates) == 0 {
		return 0, false
	}

	focusStop := trip.Stops[trip.focusIndex]

	if expected, ok := updates[focusStop.StopID][trip.TripID]; ok {
		scheduled := serviceDate.Add(focusStop.Duration)
		if expected.After(scheduled) &&
			expected.Sub(scheduled) < 2*time.Hour {
			return expected.Sub(scheduled), true
		}

		return 0, true
	}

	stopDurations := make(map[string]time.Duration, len(trip.Stops))
	for _, stop := range trip.Stops {
		stopDurations[stop.StopID] = stop.Duration
	}

	for stopID, byTrip := range updates {
		expected, ok := byTrip[trip.TripID]
		if !ok {
			continue
		}

		duration, ok := stopDurations[stopID]
		if !ok {
			continue
		}

		scheduled := serviceDate.Add(duration)
		if expected.After(scheduled) &&
			expected.Sub(scheduled) < 2*time.Hour {
			return expected.Sub(scheduled), true
		}

		return 0, true
	}

	return 0, false
}

func shapeCoordinates(shape []shapePoint) []MapCoordinate {
	coords := make([]MapCoordinate, len(shape))
	for i, point := range shape {
		coords[i] = MapCoordinate{Lat: point.Lat, Lon: point.Lon}
	}

	return coords
}

func stopCoordinates(stops []focusTripStop) []MapCoordinate {
	coords := make([]MapCoordinate, len(stops))
	for i, stop := range stops {
		coords[i] = MapCoordinate{Lat: stop.Lat, Lon: stop.Lon}
	}

	return coords
}
