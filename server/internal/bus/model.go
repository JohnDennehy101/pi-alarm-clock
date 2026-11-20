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

type scheduledDeparture struct {
	ServiceID   string
	TripID      string
	Route       string
	Destination string
	Departure   time.Duration
}

type serviceCalendar struct {
	StartDate time.Time
	EndDate   time.Time
	Weekdays  [7]bool
}
