package bus

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Timetable struct {
	location   *time.Location
	stopIDs    map[string]string
	stopPlaces map[string]stopPosition
	departures map[string][]stopDeparture
	calendars  map[string]serviceCalendar
	exceptions map[string]map[string]int
	tripIndex  map[string]int
	tripInfos  []tripInfo
	focusTrips map[string]*focusTrip
	cacheMmap  []byte
}

func NewTimetable(
	zipPath string,
	location *time.Location,
	focusStopNumbers ...string,
) (*Timetable, error) {
	if location == nil {
		return nil, fmt.Errorf("timezone is required")
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open GTFS dataset: %w", err)
	}
	defer reader.Close()

	timetable := &Timetable{
		location:   location,
		stopIDs:    make(map[string]string),
		stopPlaces: make(map[string]stopPosition),
		departures: make(map[string][]stopDeparture),
		calendars:  make(map[string]serviceCalendar),
		exceptions: make(map[string]map[string]int),
		tripIndex:  make(map[string]int),
	}

	if err := timetable.loadStops(&reader.Reader); err != nil {
		return nil, err
	}

	if err := timetable.loadRoutesAndTrips(&reader.Reader); err != nil {
		return nil, err
	}

	if err := timetable.loadCalendars(&reader.Reader); err != nil {
		return nil, err
	}

	if err := timetable.loadStopTimes(&reader.Reader); err != nil {
		return nil, err
	}

	if err := timetable.loadFocusTrips(&reader.Reader, focusStopNumbers); err != nil {
		return nil, err
	}

	return timetable, nil
}

func (t *Timetable) streamCSV(
	reader *zip.Reader,
	name string,
	rowFn func(record []string, headers map[string]int) error,
) error {
	for _, file := range reader.File {
		if path.Base(file.Name) != name {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("open %s: %w", name, err)
		}
		defer rc.Close()

		csvReader := csv.NewReader(rc)
		csvReader.FieldsPerRecord = -1
		csvReader.ReuseRecord = true

		record, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("read %s header: %w", name, err)
		}

		headers := make(map[string]int, len(record))
		for i, header := range record {
			headers[strings.TrimPrefix(
				strings.TrimSpace(header),
				"\ufeff",
			)] = i
		}

		for {
			record, err := csvReader.Read()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("read %s: %w", name, err)
			}

			if err := rowFn(record, headers); err != nil {
				return err
			}
		}
	}

	return nil
}

func field(record []string, headers map[string]int, name string) string {
	index, ok := headers[name]
	if !ok || index >= len(record) || index < 0 {
		return ""
	}

	return record[index]
}

func (t *Timetable) loadStops(reader *zip.Reader) error {
	return t.streamCSV(reader, "stops.txt", func(record []string, headers map[string]int) error {
		stopNumber := field(record, headers, "stop_code")
		stopID := field(record, headers, "stop_id")
		if stopNumber == "" || stopID == "" {
			return nil
		}

		t.stopIDs[stopNumber] = stopID

		lat, latErr := strconv.ParseFloat(field(record, headers, "stop_lat"), 64)
		lon, lonErr := strconv.ParseFloat(field(record, headers, "stop_lon"), 64)
		if latErr == nil && lonErr == nil {
			t.stopPlaces[stopID] = stopPosition{Lat: lat, Lon: lon}
		}

		return nil
	})
}

func (t *Timetable) loadRoutesAndTrips(reader *zip.Reader) error {
	routes := make(map[string]string)

	if err := t.streamCSV(reader, "routes.txt", func(record []string, headers map[string]int) error {
		routeID := field(record, headers, "route_id")
		if routeID == "" {
			return nil
		}

		routeName := field(record, headers, "route_short_name")
		if routeName == "" {
			routeName = routeID
		}

		routes[routeID] = routeName

		return nil
	}); err != nil {
		return err
	}

	return t.streamCSV(reader, "trips.txt", func(record []string, headers map[string]int) error {
		tripID := field(record, headers, "trip_id")
		if tripID == "" {
			return nil
		}

		t.tripIndex[tripID] = len(t.tripInfos)
		t.tripInfos = append(t.tripInfos, tripInfo{
			TripID:      tripID,
			ServiceID:   field(record, headers, "service_id"),
			Route:       routes[field(record, headers, "route_id")],
			Destination: field(record, headers, "trip_headsign"),
			ShapeID:     field(record, headers, "shape_id"),
		})

		return nil
	})
}

func (t *Timetable) loadCalendars(reader *zip.Reader) error {
	if err := t.streamCSV(reader, "calendar.txt", func(record []string, headers map[string]int) error {
		startDate, err := parseGTFSDate(field(record, headers, "start_date"), t.location)
		if err != nil {
			return fmt.Errorf("parse calendar start date: %w", err)
		}

		endDate, err := parseGTFSDate(field(record, headers, "end_date"), t.location)
		if err != nil {
			return fmt.Errorf("parse calendar end date: %w", err)
		}

		t.calendars[field(record, headers, "service_id")] = serviceCalendar{
			StartDate: startDate,
			EndDate:   endDate,
			Weekdays: [7]bool{
				field(record, headers, "monday") == "1",
				field(record, headers, "tuesday") == "1",
				field(record, headers, "wednesday") == "1",
				field(record, headers, "thursday") == "1",
				field(record, headers, "friday") == "1",
				field(record, headers, "saturday") == "1",
				field(record, headers, "sunday") == "1",
			},
		}

		return nil
	}); err != nil {
		return err
	}

	return t.streamCSV(reader, "calendar_dates.txt", func(record []string, headers map[string]int) error {
		serviceID := field(record, headers, "service_id")

		date, err := parseGTFSDate(field(record, headers, "date"), t.location)
		if err != nil {
			return fmt.Errorf("parse calendar exception date: %w", err)
		}

		if t.exceptions[serviceID] == nil {
			t.exceptions[serviceID] = make(map[string]int)
		}

		exceptionType, err := strconv.Atoi(field(record, headers, "exception_type"))
		if err != nil {
			return fmt.Errorf("parse calendar exception type: %w", err)
		}

		t.exceptions[serviceID][date.Format("20060102")] = exceptionType

		return nil
	})
}

func (t *Timetable) loadStopTimes(reader *zip.Reader) error {
	return t.streamCSV(reader, "stop_times.txt", func(record []string, headers map[string]int) error {
		tripKey, ok := headerField(headers, "trip_id")
		if !ok || tripKey >= len(record) {
			return nil
		}

		tripIndex, ok := t.tripIndex[record[tripKey]]
		if !ok {
			return nil
		}

		stopKey, ok := headerField(headers, "stop_id")
		if !ok || stopKey >= len(record) {
			return nil
		}

		departureValue := field(record, headers, "departure_time")
		if departureValue == "" {
			departureValue = field(record, headers, "arrival_time")
		}

		departure, err := parseGTFSTime(departureValue)
		if err != nil {
			return nil
		}

		stopID := record[stopKey]

		t.departures[stopID] = append(
			t.departures[stopID],
			stopDeparture{
				TripIndex: int32(tripIndex),
				Seconds:   int32(departure / time.Second),
			},
		)

		return nil
	})
}

func headerField(headers map[string]int, name string) (int, bool) {
	index, ok := headers[name]

	return index, ok
}

func parseGTFSDate(value string, location *time.Location) (time.Time, error) {
	return time.ParseInLocation("20060102", value, location)
}

func parseGTFSTime(value string) (time.Duration, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid GTFS time %q", value)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, err
	}

	if hours < 0 || minutes < 0 || minutes > 59 || seconds < 0 || seconds > 59 {
		return 0, fmt.Errorf("invalid GTFS time %q", value)
	}

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second, nil
}

func (t *Timetable) GetDepartures(
	stopNumber string,
	now time.Time,
	limit int,
) ([]Departure, error) {
	stopID, exists := t.stopIDs[stopNumber]
	if !exists {
		return nil, fmt.Errorf("unknown bus stop number %q", stopNumber)
	}

	if limit <= 0 {
		limit = 5
	}

	now = now.In(t.location)

	today := time.Date(
		now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, t.location,
	)

	var results []Departure

	for offset := -1; offset <= 1; offset++ {
		serviceDate := today.AddDate(0, 0, offset)

		for _, scheduled := range t.departures[stopID] {
			info := t.tripInfos[scheduled.TripIndex]

			if !t.isServiceActive(info.ServiceID, serviceDate) {
				continue
			}

			departureTime := serviceDate.Add(
				time.Duration(scheduled.Seconds) * time.Second,
			)

			if departureTime.Before(now) {
				continue
			}

			minutesAway := int(math.Ceil(
				departureTime.Sub(now).Minutes(),
			))

			results = append(results, Departure{
				StopNumber:    stopNumber,
				StopID:        stopID,
				TripID:        info.TripID,
				Route:         info.Route,
				Destination:   info.Destination,
				ScheduledTime: departureTime,
				ExpectedTime:  departureTime,
				MinutesAway:   minutesAway,
				IsRealtime:    false,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ScheduledTime.Before(results[j].ScheduledTime)
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (t *Timetable) isServiceActive(serviceID string, date time.Time) bool {
	dateKey := date.Format("20060102")

	// Calendar exceptions override the normal weekly calendar.
	if exceptions, exists := t.exceptions[serviceID]; exists {
		if exceptionType, exists := exceptions[dateKey]; exists {
			return exceptionType == 1
		}
	}

	calendar, exists := t.calendars[serviceID]
	if !exists {
		return false
	}

	if date.Before(calendar.StartDate) || date.After(calendar.EndDate) {
		return false
	}

	weekday := int(date.Weekday() - time.Monday)
	if weekday < 0 {
		weekday = 6
	}

	return calendar.Weekdays[weekday]
}
