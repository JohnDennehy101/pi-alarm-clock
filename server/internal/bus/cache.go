package bus

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"syscall"
	"time"
	"unsafe"
)

// On-disk cache of a parsed GTFS dataset. The cache is valid only
// when its header matches the schema version, the sha256 of the GTFS
// zip it was built from, and the focus stop list; any mismatch or
// decode error is a cache miss that falls back to a streaming parse,
// so a corrupt or stale cache can never serve wrong data.
//
// The stop-times section is a page-aligned flat array of
// stopDeparture rows. It is either read into the heap (level 1) or
// mapped directly from the file (level 2), keeping the ~70 MB of rows
// in the kernel page cache instead of the Go heap.

const (
	// cacheMagic identifies the file; the final byte is NUL.
	cacheMagic   = "PIBUSC\x00"
	cacheVersion = 1

	// cacheAlignment pads the stop-times rows section so a mapped
	// []stopDeparture is naturally aligned on every platform.
	cacheAlignment = 4096

	// cacheStringMax guards against corrupt length prefixes.
	cacheStringMax = 1 << 24
)

var ErrCacheInvalid = errors.New("bus timetable cache is invalid")

func GTFSChecksum(path string) ([32]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer file.Close()

	var sum [32]byte

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return sum, err
	}

	copy(sum[:], digest.Sum(nil))

	return sum, nil
}

func OpenTimetable(
	zipPath string,
	cachePath string,
	location *time.Location,
	focusStopNumbers ...string,
) (*Timetable, error) {
	sum, sumErr := GTFSChecksum(zipPath)

	if sumErr == nil && cachePath != "" {
		timetable, err := loadTimetableCache(
			cachePath,
			sum,
			location,
			focusStopNumbers,
		)
		if err == nil {
			return timetable, nil
		}
	}

	timetable, err := NewTimetable(zipPath, location, focusStopNumbers...)
	if err != nil {
		return nil, err
	}

	if cachePath != "" && sumErr == nil {
		_ = timetable.WriteCache(cachePath, sum, focusStopNumbers)
	}

	return timetable, nil
}

func (t *Timetable) WriteCache(
	path string,
	gtfsSum [32]byte,
	focusStopNumbers []string,
) error {
	tmpPath := path + ".tmp"

	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating timetable cache: %w", err)
	}

	err = t.writeCache(bufio.NewWriterSize(file, 1<<20), gtfsSum, focusStopNumbers)

	if flushErr := file.Sync(); flushErr != nil && err == nil {
		err = flushErr
	}

	if closeErr := file.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("writing timetable cache: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("publishing timetable cache: %w", err)
	}

	return nil
}

func (t *Timetable) writeCache(
	w *bufio.Writer,
	gtfsSum [32]byte,
	focusStopNumbers []string,
) error {
	counter := &writeCounter{w: w}

	err := t.encode(counter, gtfsSum, focusStopNumbers)

	if flushErr := w.Flush(); err == nil {
		err = flushErr
	}

	return err
}

func (t *Timetable) writeCacheBuffer(
	w io.Writer,
	gtfsSum [32]byte,
	focusStopNumbers []string,
) error {
	return t.writeCache(bufio.NewWriterSize(w, 1<<16), gtfsSum, focusStopNumbers)
}

func (t *Timetable) encode(
	w *writeCounter,
	gtfsSum [32]byte,
	focusStopNumbers []string,
) error {
	if _, err := w.Write([]byte(cacheMagic)); err != nil {
		return err
	}

	if err := w.uint16(cacheVersion); err != nil {
		return err
	}

	if _, err := w.Write(gtfsSum[:]); err != nil {
		return err
	}

	if err := w.stringSlice(focusStopNumbers); err != nil {
		return err
	}

	if err := t.encodeStops(w); err != nil {
		return err
	}

	if err := t.encodeTrips(w); err != nil {
		return err
	}

	if err := t.encodeDepartures(w); err != nil {
		return err
	}

	if err := t.encodeCalendars(w); err != nil {
		return err
	}

	if err := t.encodeFocusTrips(w); err != nil {
		return err
	}

	return nil
}

func (t *Timetable) encodeStops(w *writeCounter) error {
	numbers := make([]string, 0, len(t.stopIDs))
	for stopNumber := range t.stopIDs {
		numbers = append(numbers, stopNumber)
	}
	sortStrings(numbers)

	if err := w.uint32(uint32(len(numbers))); err != nil {
		return err
	}

	for _, stopNumber := range numbers {
		stopID := t.stopIDs[stopNumber]
		place := t.stopPlaces[stopID]

		if err := w.string(stopNumber); err != nil {
			return err
		}

		if err := w.string(stopID); err != nil {
			return err
		}

		if err := w.float64(place.Lat); err != nil {
			return err
		}

		if err := w.float64(place.Lon); err != nil {
			return err
		}
	}

	return nil
}

func (t *Timetable) encodeTrips(w *writeCounter) error {
	if err := w.uint32(uint32(len(t.tripInfos))); err != nil {
		return err
	}

	for _, info := range t.tripInfos {
		for _, value := range []string{
			info.TripID,
			info.ServiceID,
			info.Route,
			info.Destination,
			info.ShapeID,
		} {
			if err := w.string(value); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *Timetable) encodeDepartures(w *writeCounter) error {
	stopIDs := make([]string, 0, len(t.departures))
	for stopID := range t.departures {
		stopIDs = append(stopIDs, stopID)
	}
	sortStrings(stopIDs)

	if err := w.uint32(uint32(len(stopIDs))); err != nil {
		return err
	}

	totalRows := 0
	for _, stopID := range stopIDs {
		totalRows += len(t.departures[stopID])
	}

	offset := 0

	for _, stopID := range stopIDs {
		entries := len(t.departures[stopID])

		if err := w.string(stopID); err != nil {
			return err
		}

		if err := w.uint32(uint32(entries)); err != nil {
			return err
		}

		// Byte offset into the rows array, matching the decoder.
		if err := w.uint64(uint64(offset) * 8); err != nil {
			return err
		}

		offset += entries
	}

	// Pad to a page boundary so a mapped rows array is aligned.
	pad := (cacheAlignment - int(w.n%cacheAlignment)) % cacheAlignment
	if err := w.zeros(pad); err != nil {
		return err
	}

	if err := w.uint64(uint64(totalRows)); err != nil {
		return err
	}

	for _, stopID := range stopIDs {
		for _, departure := range t.departures[stopID] {
			if err := w.int32(departure.TripIndex); err != nil {
				return err
			}

			if err := w.int32(departure.Seconds); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *Timetable) encodeCalendars(w *writeCounter) error {
	serviceIDs := make([]string, 0, len(t.calendars))
	for serviceID := range t.calendars {
		serviceIDs = append(serviceIDs, serviceID)
	}
	sortStrings(serviceIDs)

	if err := w.uint32(uint32(len(serviceIDs))); err != nil {
		return err
	}

	for _, serviceID := range serviceIDs {
		calendar := t.calendars[serviceID]

		if err := w.string(serviceID); err != nil {
			return err
		}

		if err := w.int64(calendar.StartDate.Unix()); err != nil {
			return err
		}

		if err := w.int64(calendar.EndDate.Unix()); err != nil {
			return err
		}

		var weekdays [7]byte
		for i, day := range calendar.Weekdays {
			if day {
				weekdays[i] = 1
			}
		}

		if _, err := w.Write(weekdays[:]); err != nil {
			return err
		}
	}

	exceptionServiceIDs := make([]string, 0, len(t.exceptions))
	for serviceID := range t.exceptions {
		exceptionServiceIDs = append(exceptionServiceIDs, serviceID)
	}
	sortStrings(exceptionServiceIDs)

	if err := w.uint32(uint32(len(exceptionServiceIDs))); err != nil {
		return err
	}

	for _, serviceID := range exceptionServiceIDs {
		if err := w.string(serviceID); err != nil {
			return err
		}

		dateKeys := make([]string, 0, len(t.exceptions[serviceID]))
		for dateKey := range t.exceptions[serviceID] {
			dateKeys = append(dateKeys, dateKey)
		}
		sortStrings(dateKeys)

		if err := w.uint32(uint32(len(dateKeys))); err != nil {
			return err
		}

		for _, dateKey := range dateKeys {
			if err := w.string(dateKey); err != nil {
				return err
			}

			if err := w.uint8(byte(t.exceptions[serviceID][dateKey])); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *Timetable) encodeFocusTrips(w *writeCounter) error {
	tripIDs := make([]string, 0, len(t.focusTrips))
	for tripID := range t.focusTrips {
		tripIDs = append(tripIDs, tripID)
	}
	sortStrings(tripIDs)

	if err := w.uint32(uint32(len(tripIDs))); err != nil {
		return err
	}

	for _, tripID := range tripIDs {
		trip := t.focusTrips[tripID]

		for _, value := range []string{
			trip.TripID,
			trip.Route,
			trip.Destination,
			trip.ServiceID,
			trip.ShapeID,
		} {
			if err := w.string(value); err != nil {
				return err
			}
		}

		if err := w.int32(int32(trip.focusIndex)); err != nil {
			return err
		}

		if err := w.uint32(uint32(len(trip.Stops))); err != nil {
			return err
		}

		for _, stop := range trip.Stops {
			if err := w.string(stop.StopID); err != nil {
				return err
			}

			if err := w.float64(stop.Lat); err != nil {
				return err
			}

			if err := w.float64(stop.Lon); err != nil {
				return err
			}

			if err := w.int32(int32(stop.Sequence)); err != nil {
				return err
			}

			if err := w.int64(int64(stop.Duration / time.Second)); err != nil {
				return err
			}
		}

		if err := w.uint32(uint32(len(trip.Shape))); err != nil {
			return err
		}

		for _, point := range trip.Shape {
			if err := w.float64(point.Lat); err != nil {
				return err
			}

			if err := w.float64(point.Lon); err != nil {
				return err
			}

			if err := w.float64(point.Dist); err != nil {
				return err
			}
		}

		if err := w.uint32(uint32(len(trip.shapeAtStop))); err != nil {
			return err
		}

		for _, distance := range trip.shapeAtStop {
			if err := w.float64(distance); err != nil {
				return err
			}
		}
	}

	return nil
}

func loadTimetableCache(
	path string,
	gtfsSum [32]byte,
	location *time.Location,
	focusStopNumbers []string,
) (*Timetable, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() > 0 {
		data, mapErr := syscall.Mmap(
			int(file.Fd()),
			0,
			int(info.Size()),
			syscall.PROT_READ,
			syscall.MAP_PRIVATE,
		)
		if mapErr == nil {
			timetable, decodeErr := decodeCacheBytes(data, gtfsSum, location, focusStopNumbers)
			if decodeErr == nil {
				timetable.cacheMmap = data

				return timetable, nil
			}

			_ = syscall.Munmap(data)

			if errors.Is(decodeErr, ErrCacheInvalid) {
				return nil, decodeErr
			}

			return nil, decodeErr
		}
	}

	counter := &readCounter{r: bufio.NewReaderSize(file, 1<<20)}

	timetable, err := decodeCache(counter, gtfsSum, location, focusStopNumbers, func(rows int) ([]stopDeparture, error) {
		entries := make([]stopDeparture, rows)

		if err := binary.Read(counter.r, binary.LittleEndian, &entries); err != nil {
			return nil, err
		}

		counter.n += int64(rows) * 8

		return entries, nil
	})
	if err != nil {
		return nil, err
	}

	if err := counter.ensureDrained(); err != nil {
		return nil, err
	}

	return timetable, nil
}

type cacheCursor interface {
	next(n int) ([]byte, error)
	pos() int64
}

type bytesCursor struct {
	data []byte
	off  int
}

func (c *bytesCursor) next(n int) ([]byte, error) {
	if n < 0 || c.off+n > len(c.data) {
		return nil, ErrCacheInvalid
	}

	b := c.data[c.off : c.off+n]
	c.off += n

	return b, nil
}

func (c *bytesCursor) pos() int64 {
	return int64(c.off)
}

func decodeCacheBytes(
	data []byte,
	gtfsSum [32]byte,
	location *time.Location,
	focusStopNumbers []string,
) (*Timetable, error) {
	cursor := &bytesCursor{data: data}

	return decodeCache(cursor, gtfsSum, location, focusStopNumbers,
		func(rows int) ([]stopDeparture, error) {
			base := cursor.pos()

			b := data[base : base+int64(rows)*8]
			cursor.off += rows * 8

			return unsafe.Slice(
				(*stopDeparture)(unsafe.Pointer(&b[0])),
				rows,
			), nil
		})
}

func decodeCache(
	cursor cacheCursor,
	gtfsSum [32]byte,
	location *time.Location,
	focusStopNumbers []string,
	loadRows func(rows int) ([]stopDeparture, error),
) (*Timetable, error) {
	magic, err := cursor.next(len(cacheMagic))
	if err != nil || string(magic) != cacheMagic {
		return nil, ErrCacheInvalid
	}

	version, err := cursorUint16(cursor)
	if err != nil || version != cacheVersion {
		return nil, ErrCacheInvalid
	}

	sum, err := cursor.next(32)
	if err != nil || string(sum) != string(gtfsSum[:]) {
		return nil, ErrCacheInvalid
	}

	focus, err := readStringSlice(cursor)
	if err != nil {
		return nil, err
	}

	if !stringSlicesEqual(focus, focusStopNumbers) {
		return nil, ErrCacheInvalid
	}

	timetable, err := readStops(cursor, location)
	if err != nil {
		return nil, err
	}

	if err := readTrips(cursor, timetable); err != nil {
		return nil, err
	}

	if err := readDepartures(cursor, timetable, loadRows); err != nil {
		return nil, err
	}

	if err := readCalendars(cursor, timetable); err != nil {
		return nil, err
	}

	if err := readFocusTrips(cursor, timetable); err != nil {
		return nil, err
	}

	return timetable, nil
}

func readStops(cursor cacheCursor, location *time.Location) (*Timetable, error) {
	timetable := &Timetable{
		location:   location,
		stopIDs:    make(map[string]string),
		stopPlaces: make(map[string]stopPosition),
		departures: make(map[string][]stopDeparture),
		calendars:  make(map[string]serviceCalendar),
		exceptions: make(map[string]map[string]int),
		tripIndex:  make(map[string]int),
	}

	count, err := cursorUint32(cursor)
	if err != nil {
		return nil, err
	}

	for i := uint32(0); i < count; i++ {
		stopNumber, err := readString(cursor)
		if err != nil {
			return nil, err
		}

		stopID, err := readString(cursor)
		if err != nil {
			return nil, err
		}

		lat, err := cursorFloat64(cursor)
		if err != nil {
			return nil, err
		}

		lon, err := cursorFloat64(cursor)
		if err != nil {
			return nil, err
		}

		timetable.stopIDs[stopNumber] = stopID
		timetable.stopPlaces[stopID] = stopPosition{Lat: lat, Lon: lon}
	}

	return timetable, nil
}

func readTrips(cursor cacheCursor, timetable *Timetable) error {
	count, err := cursorUint32(cursor)
	if err != nil {
		return err
	}

	timetable.tripInfos = make([]tripInfo, 0, count)

	for i := uint32(0); i < count; i++ {
		var info tripInfo

		for idx := 0; idx < 5; idx++ {
			field, err := readString(cursor)
			if err != nil {
				return err
			}

			switch idx {
			case 0:
				info.TripID = field
			case 1:
				info.ServiceID = field
			case 2:
				info.Route = field
			case 3:
				info.Destination = field
			case 4:
				info.ShapeID = field
			}
		}

		timetable.tripIndex[info.TripID] = len(timetable.tripInfos)
		timetable.tripInfos = append(timetable.tripInfos, info)
	}

	return nil
}

func readDepartures(
	cursor cacheCursor,
	timetable *Timetable,
	loadRows func(rows int) ([]stopDeparture, error),
) error {
	count, err := cursorUint32(cursor)
	if err != nil {
		return err
	}

	type stopSpan struct {
		entries uint32
		offset  uint64
	}

	spans := make(map[string]stopSpan, count)

	for i := uint32(0); i < count; i++ {
		stopID, err := readString(cursor)
		if err != nil {
			return err
		}

		entries, err := cursorUint32(cursor)
		if err != nil {
			return err
		}

		offset, err := cursorUint64(cursor)
		if err != nil {
			return err
		}

		spans[stopID] = stopSpan{entries: entries, offset: offset}
	}

	remainder := cursor.pos() % int64(cacheAlignment)
	if remainder != 0 {
		if _, err := cursor.next(
			int(int64(cacheAlignment) - remainder),
		); err != nil {
			return err
		}
	}

	rowsCount, err := cursorUint64(cursor)
	if err != nil {
		return err
	}

	if rowsCount > 1<<32 {
		return ErrCacheInvalid
	}

	rows, err := loadRows(int(rowsCount))
	if err != nil {
		return err
	}

	for stopID, span := range spans {
		if span.offset%8 != 0 {
			return ErrCacheInvalid
		}

		start := span.offset / 8
		end := start + uint64(span.entries)

		if end > uint64(len(rows)) {
			return ErrCacheInvalid
		}

		timetable.departures[stopID] = rows[start:end]
	}

	return nil
}

func readCalendars(cursor cacheCursor, timetable *Timetable) error {
	count, err := cursorUint32(cursor)
	if err != nil {
		return err
	}

	for i := uint32(0); i < count; i++ {
		serviceID, err := readString(cursor)
		if err != nil {
			return err
		}

		startUnix, err := cursorInt64(cursor)
		if err != nil {
			return err
		}

		endUnix, err := cursorInt64(cursor)
		if err != nil {
			return err
		}

		weekdaysBytes, err := cursor.next(7)
		if err != nil {
			return err
		}

		var weekdays [7]bool
		for day, value := range weekdaysBytes {
			weekdays[day] = value == 1
		}

		timetable.calendars[serviceID] = serviceCalendar{
			StartDate: time.Unix(startUnix, 0),
			EndDate:   time.Unix(endUnix, 0),
			Weekdays:  weekdays,
		}
	}

	exceptionCount, err := cursorUint32(cursor)
	if err != nil {
		return err
	}

	for i := uint32(0); i < exceptionCount; i++ {
		serviceID, err := readString(cursor)
		if err != nil {
			return err
		}

		entryCount, err := cursorUint32(cursor)
		if err != nil {
			return err
		}

		if timetable.exceptions[serviceID] == nil {
			timetable.exceptions[serviceID] = make(map[string]int)
		}

		for j := uint32(0); j < entryCount; j++ {
			dateKey, err := readString(cursor)
			if err != nil {
				return err
			}

			exceptionType, err := cursorUint8(cursor)
			if err != nil {
				return err
			}

			timetable.exceptions[serviceID][dateKey] = int(exceptionType)
		}
	}

	return nil
}

func readFocusTrips(cursor cacheCursor, timetable *Timetable) error {
	count, err := cursorUint32(cursor)
	if err != nil {
		return err
	}

	timetable.focusTrips = make(map[string]*focusTrip, count)

	for i := uint32(0); i < count; i++ {
		trip := &focusTrip{}

		for idx := 0; idx < 5; idx++ {
			value, err := readString(cursor)
			if err != nil {
				return err
			}

			switch idx {
			case 0:
				trip.TripID = value
			case 1:
				trip.Route = value
			case 2:
				trip.Destination = value
			case 3:
				trip.ServiceID = value
			case 4:
				trip.ShapeID = value
			}
		}

		focusIndex, err := cursorInt32(cursor)
		if err != nil {
			return err
		}

		trip.focusIndex = int(focusIndex)

		stopCount, err := cursorUint32(cursor)
		if err != nil {
			return err
		}

		trip.Stops = make([]focusTripStop, 0, stopCount)

		for j := uint32(0); j < stopCount; j++ {
			stopID, err := readString(cursor)
			if err != nil {
				return err
			}

			lat, err := cursorFloat64(cursor)
			if err != nil {
				return err
			}

			lon, err := cursorFloat64(cursor)
			if err != nil {
				return err
			}

			sequence, err := cursorInt32(cursor)
			if err != nil {
				return err
			}

			seconds, err := cursorInt64(cursor)
			if err != nil {
				return err
			}

			trip.Stops = append(trip.Stops, focusTripStop{
				StopID:   stopID,
				Lat:      lat,
				Lon:      lon,
				Sequence: int(sequence),
				Duration: time.Duration(seconds) * time.Second,
			})
		}

		shapeCount, err := cursorUint32(cursor)
		if err != nil {
			return err
		}

		trip.Shape = make([]shapePoint, 0, shapeCount)

		for j := uint32(0); j < shapeCount; j++ {
			lat, err := cursorFloat64(cursor)
			if err != nil {
				return err
			}

			lon, err := cursorFloat64(cursor)
			if err != nil {
				return err
			}

			dist, err := cursorFloat64(cursor)
			if err != nil {
				return err
			}

			trip.Shape = append(trip.Shape, shapePoint{
				Lat:  lat,
				Lon:  lon,
				Dist: dist,
			})
		}

		distCount, err := cursorUint32(cursor)
		if err != nil {
			return err
		}

		trip.shapeAtStop = make([]float64, 0, distCount)

		for j := uint32(0); j < distCount; j++ {
			distance, err := cursorFloat64(cursor)
			if err != nil {
				return err
			}

			trip.shapeAtStop = append(trip.shapeAtStop, distance)
		}

		if trip.focusIndex < 0 || trip.focusIndex >= len(trip.Stops) {
			return ErrCacheInvalid
		}

		timetable.focusTrips[trip.TripID] = trip
	}

	return nil
}

func cursorUint8(cursor cacheCursor) (uint8, error) {
	b, err := cursor.next(1)
	if err != nil {
		return 0, err
	}

	return b[0], nil
}

func cursorUint16(cursor cacheCursor) (uint16, error) {
	b, err := cursor.next(2)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint16(b), nil
}

func cursorUint32(cursor cacheCursor) (uint32, error) {
	b, err := cursor.next(4)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(b), nil
}

func cursorUint64(cursor cacheCursor) (uint64, error) {
	b, err := cursor.next(8)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(b), nil
}

func cursorInt32(cursor cacheCursor) (int32, error) {
	value, err := cursorUint32(cursor)

	return int32(value), err
}

func cursorInt64(cursor cacheCursor) (int64, error) {
	value, err := cursorUint64(cursor)

	return int64(value), err
}

func cursorFloat64(cursor cacheCursor) (float64, error) {
	bits, err := cursorUint64(cursor)

	return math.Float64frombits(bits), err
}

func readString(cursor cacheCursor) (string, error) {
	length, err := cursorUint32(cursor)
	if err != nil {
		return "", err
	}

	if length > cacheStringMax {
		return "", ErrCacheInvalid
	}

	b, err := cursor.next(int(length))
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func readStringSlice(cursor cacheCursor) ([]string, error) {
	count, err := cursorUint32(cursor)
	if err != nil {
		return nil, err
	}

	if count > 1024 {
		return nil, ErrCacheInvalid
	}

	values := make([]string, 0, count)
	for i := uint32(0); i < count; i++ {
		value, err := readString(cursor)
		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	return values, nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func sortStrings(values []string) {
	sort.Strings(values)
}

type writeCounter struct {
	w io.Writer
	n int64
}

func (c *writeCounter) Write(p []byte) (int, error) {
	written, err := c.w.Write(p)
	c.n += int64(written)

	return written, err
}

func (c *writeCounter) uint8(value uint8) error {
	_, err := c.Write([]byte{value})

	return err
}

func (c *writeCounter) uint16(value uint16) error {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], value)

	_, err := c.Write(buf[:])

	return err
}

func (c *writeCounter) uint32(value uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], value)

	_, err := c.Write(buf[:])

	return err
}

func (c *writeCounter) uint64(value uint64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], value)

	_, err := c.Write(buf[:])

	return err
}

func (c *writeCounter) int32(value int32) error {
	return c.uint32(uint32(value))
}

func (c *writeCounter) int64(value int64) error {
	return c.uint64(uint64(value))
}

func (c *writeCounter) float64(value float64) error {
	return c.uint64(math.Float64bits(value))
}

func (c *writeCounter) string(value string) error {
	if len(value) > cacheStringMax {
		return ErrCacheInvalid
	}

	if err := c.uint32(uint32(len(value))); err != nil {
		return err
	}

	_, err := c.Write([]byte(value))

	return err
}

func (c *writeCounter) stringSlice(values []string) error {
	if err := c.uint32(uint32(len(values))); err != nil {
		return err
	}

	for _, value := range values {
		if err := c.string(value); err != nil {
			return err
		}
	}

	return nil
}

func (c *writeCounter) zeros(n int) error {
	if n <= 0 {
		return nil
	}

	buf := make([]byte, 4096)
	for n > 0 {
		chunk := n
		if chunk > len(buf) {
			chunk = len(buf)
		}

		if _, err := c.Write(buf[:chunk]); err != nil {
			return err
		}

		n -= chunk
	}

	return nil
}

type readCounter struct {
	r *bufio.Reader
	n int64
}

func (c *readCounter) next(n int) ([]byte, error) {
	b := make([]byte, n)

	if _, err := io.ReadFull(c.r, b); err != nil {
		return nil, ErrCacheInvalid
	}

	c.n += int64(n)

	return b, nil
}

func (c *readCounter) pos() int64 {
	return c.n
}

func (c *readCounter) ensureDrained() error {
	if _, err := c.r.ReadByte(); err == nil {
		return ErrCacheInvalid
	}

	return nil
}
