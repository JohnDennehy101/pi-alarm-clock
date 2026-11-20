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
	location *time.Location

	stopIDs      map[string]string
	departures   map[string][]scheduledDeparture
	calendars    map[string]serviceCalendar
	exceptions   map[string]map[string]int
	tripMetadata map[string]scheduledDeparture
}

func NewTimetable(zipPath string, location *time.Location) (*Timetable, error) {
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
		departures: make(map[string][]scheduledDeparture),
		calendars:  make(map[string]serviceCalendar),
		exceptions: make(map[string]map[string]int),
	}

	files, err := readGTFSFiles(&reader.Reader)
	if err != nil {
		return nil, err
	}

	if err := timetable.loadStops(files); err != nil {
		return nil, err
	}

	if err := timetable.loadRoutesAndTrips(files); err != nil {
		return nil, err
	}

	if err := timetable.loadCalendars(files); err != nil {
		return nil, err
	}

	if err := timetable.loadStopTimes(files); err != nil {
		return nil, err
	}

	return timetable, nil
}

func readGTFSFiles(reader *zip.Reader) (map[string][]map[string]string, error) {
	files := make(map[string][]map[string]string)

	for _, file := range reader.File {
		name := path.Base(file.Name)

		if !strings.HasSuffix(name, ".txt") {
			continue
		}

		content, err := readCSVFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}

		files[name] = content
	}

	return files, nil
}

func readCSVFile(file *zip.File) ([]map[string]string, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1

	headers, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	for i := range headers {
		headers[i] = strings.TrimPrefix(headers[i], "\ufeff")
		headers[i] = strings.TrimSpace(headers[i])
	}

	var rows []map[string]string

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		row := make(map[string]string, len(headers))

		for i, header := range headers {
			if i < len(record) {
				row[header] = strings.TrimSpace(record[i])
			}
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func (t *Timetable) loadStops(files map[string][]map[string]string) error {
	for _, row := range files["stops.txt"] {
		stopNumber := row["stop_code"]
		stopID := row["stop_id"]

		if stopNumber == "" || stopID == "" {
			continue
		}

		t.stopIDs[stopNumber] = stopID
	}

	return nil
}

func (t *Timetable) loadRoutesAndTrips(files map[string][]map[string]string) error {
	routes := make(map[string]string)

	for _, row := range files["routes.txt"] {
		routeID := row["route_id"]
		routeName := row["route_short_name"]

		if routeID == "" {
			continue
		}

		if routeName == "" {
			routeName = routeID
		}

		routes[routeID] = routeName
	}

	trips := make(map[string]scheduledDeparture)

	for _, row := range files["trips.txt"] {
		tripID := row["trip_id"]
		if tripID == "" {
			continue
		}

		routeID := row["route_id"]

		trips[tripID] = scheduledDeparture{
			ServiceID:   row["service_id"],
			TripID:      tripID,
			Route:       routes[routeID],
			Destination: row["trip_headsign"],
		}
	}

	// Temporarily store the trip metadata so loadStopTimes can use it.
	t.tripMetadata = trips

	return nil
}

func (t *Timetable) loadCalendars(files map[string][]map[string]string) error {
	for _, row := range files["calendar.txt"] {
		startDate, err := parseGTFSDate(row["start_date"], t.location)
		if err != nil {
			return fmt.Errorf("parse calendar start date: %w", err)
		}

		endDate, err := parseGTFSDate(row["end_date"], t.location)
		if err != nil {
			return fmt.Errorf("parse calendar end date: %w", err)
		}

		calendar := serviceCalendar{
			StartDate: startDate,
			EndDate:   endDate,
			Weekdays: [7]bool{
				row["monday"] == "1",
				row["tuesday"] == "1",
				row["wednesday"] == "1",
				row["thursday"] == "1",
				row["friday"] == "1",
				row["saturday"] == "1",
				row["sunday"] == "1",
			},
		}

		t.calendars[row["service_id"]] = calendar
	}

	for _, row := range files["calendar_dates.txt"] {
		serviceID := row["service_id"]

		date, err := parseGTFSDate(row["date"], t.location)
		if err != nil {
			return fmt.Errorf("parse calendar exception date: %w", err)
		}

		if t.exceptions[serviceID] == nil {
			t.exceptions[serviceID] = make(map[string]int)
		}

		exceptionType, err := strconv.Atoi(row["exception_type"])
		if err != nil {
			return fmt.Errorf("parse calendar exception type: %w", err)
		}

		t.exceptions[serviceID][date.Format("20060102")] = exceptionType
	}

	return nil
}

func (t *Timetable) loadStopTimes(files map[string][]map[string]string) error {
	for _, row := range files["stop_times.txt"] {
		stopID := row["stop_id"]
		tripID := row["trip_id"]

		trip, exists := t.tripMetadata[tripID]
		if !exists {
			continue
		}

		departureValue := row["departure_time"]
		if departureValue == "" {
			departureValue = row["arrival_time"]
		}

		departureTime, err := parseGTFSTime(departureValue)
		if err != nil {
			return fmt.Errorf("parse departure time for trip %s: %w", tripID, err)
		}

		departure := trip
		departure.Departure = departureTime

		t.departures[stopID] = append(
			t.departures[stopID],
			departure,
		)
	}

	return nil
}

func parseGTFSDate(value string, location *time.Location) (time.Time, error) {
	return time.ParseInLocation("20060102", value, location)
}

// GTFS allows hours greater than 23, e.g. 25:15:00.
// These represent times after midnight on the following day.
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
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		t.location,
	)

	var results []Departure

	for offset := -1; offset <= 1; offset++ {
		serviceDate := today.AddDate(0, 0, offset)

		for _, scheduled := range t.departures[stopID] {
			if !t.isServiceActive(scheduled.ServiceID, serviceDate) {
				continue
			}

			departureTime := serviceDate.Add(scheduled.Departure)

			if departureTime.Before(now) {
				continue
			}

			minutesAway := int(math.Ceil(
				departureTime.Sub(now).Minutes(),
			))

			results = append(results, Departure{
				StopNumber:    stopNumber,
				StopID:        stopID,
				TripID:        scheduled.TripID,
				Route:         scheduled.Route,
				Destination:   scheduled.Destination,
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
