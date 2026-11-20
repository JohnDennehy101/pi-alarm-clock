package bus

import "time"

type Departure struct {
	StopNumber    string    `json:"stopNumber"`
	StopID        string    `json:"stopId,omitempty"`
	TripID        string    `json:"tripId,omitempty"`
	Route         string    `json:"route"`
	Destination   string    `json:"destination"`
	ScheduledTime time.Time `json:"scheduledTime"`
	ExpectedTime  time.Time `json:"expectedTime"`
	MinutesAway   int       `json:"minutesAway"`
	IsRealtime    bool      `json:"isRealtime"`
}

type tripInfo struct {
	TripID      string
	ServiceID   string
	Route       string
	Destination string
	ShapeID     string
}

type stopDeparture struct {
	TripIndex int32
	Seconds   int32
}

type serviceCalendar struct {
	StartDate time.Time
	EndDate   time.Time
	Weekdays  [7]bool
}

type stopPosition struct {
	Lat float64
	Lon float64
}

type shapePoint struct {
	Lat  float64
	Lon  float64
	Dist float64
}

type focusTripStop struct {
	StopID   string
	Lat      float64
	Lon      float64
	Sequence int
	Duration time.Duration
}

type focusTrip struct {
	TripID      string
	Route       string
	Destination string
	ServiceID   string
	ShapeID     string
	Stops       []focusTripStop
	Shape       []shapePoint
	shapeAtStop []float64
	focusIndex  int
}

type MapTrip struct {
	TripID           string
	Route            string
	Destination      string
	ShapeID          string
	Lat              float64
	Lon              float64
	PrevLat          float64
	PrevLon          float64
	Arrival          time.Time
	MinutesToStop    int
	IsRealtime       bool
	ProgressFraction float64
	PreviousFraction float64
	StopFractions    []float64
	StopsRemaining   int
	ScheduledArrival time.Time
	Polyline         []MapCoordinate
	Stops            []MapCoordinate
}

type MapCoordinate struct {
	Lat float64
	Lon float64
}
