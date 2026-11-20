package bus

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultRealtimeURL = "https://api.nationaltransport.ie/gtfsr/v2/gtfsr"

	realtimeMaxResponseBytes = 20 << 20
)

type RealtimeClient struct {
	url    string
	apiKey string
	client *http.Client
}

func NewRealtimeClient(url string, apiKey string) *RealtimeClient {
	if url == "" {
		url = DefaultRealtimeURL
	}

	return &RealtimeClient{
		url:    url,
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

type RealtimeUpdate struct {
	TripID string
	StopID string
	Time   time.Time
}

func (c *RealtimeClient) Fetch(ctx context.Context) ([]RealtimeUpdate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("create realtime request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download realtime feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("realtime feed returned status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, realtimeMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read realtime feed: %w", err)
	}

	updates, err := parseRealtimeFeed(body)
	if err != nil {
		return nil, fmt.Errorf("parse realtime feed: %w", err)
	}

	return updates, nil
}

//	FeedMessage{
//	  entity = 2 => FeedEntity{
//	    trip_update = 3 => TripUpdate{
//	      trip = 1 => TripDescriptor{ trip_id = 1 },
//	      stop_time_update = 2 => StopTimeUpdate{
//	        stop_id = 4, arrival = 2, departure = 3,
//	        schedule_relationship = 5 } } } }
//
// StopTimeEvent{ time = 2 }
func parseRealtimeFeed(data []byte) ([]RealtimeUpdate, error) {
	var updates []RealtimeUpdate

	err := walkProto(data, func(field int, value []byte, _ uint64) error {
		if field == 2 {
			return parseRealtimeEntity(value, &updates)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updates, nil
}

func parseRealtimeEntity(data []byte, updates *[]RealtimeUpdate) error {
	return walkProto(data, func(field int, value []byte, _ uint64) error {
		if field == 3 {
			return parseRealtimeTripUpdate(value, updates)
		}

		return nil
	})
}

func parseRealtimeTripUpdate(data []byte, updates *[]RealtimeUpdate) error {
	tripID := ""

	return walkProto(data, func(field int, value []byte, _ uint64) error {
		switch field {
		case 1:
			return walkProto(value, func(field int, value []byte, _ uint64) error {
				if field == 1 {
					tripID = string(value)
				}

				return nil
			})

		case 2:
			stopID, at, skipped, err := parseRealtimeStopTimeUpdate(value)
			if err != nil {
				return err
			}

			if skipped || tripID == "" || stopID == "" || at.IsZero() {
				return nil
			}

			*updates = append(*updates, RealtimeUpdate{
				TripID: tripID,
				StopID: stopID,
				Time:   at,
			})

			return nil

		default:
			return nil
		}
	})
}

func parseRealtimeStopTimeUpdate(data []byte) (string, time.Time, bool, error) {
	var (
		stopID    string
		arrival   int64
		departure int64
		skipped   bool
	)

	err := walkProto(data, func(field int, value []byte, varint uint64) error {
		switch field {
		case 2:
			at, err := parseRealtimeStopTimeEvent(value)
			if err != nil {
				return err
			}

			arrival = at

		case 3:
			at, err := parseRealtimeStopTimeEvent(value)
			if err != nil {
				return err
			}

			departure = at

		case 4:
			stopID = string(value)

		case 5:
			// ScheduleRelationship SKIPPED (1).
			if varint == 1 {
				skipped = true
			}
		}

		return nil
	})
	if err != nil {
		return "", time.Time{}, false, err
	}

	if skipped {
		return "", time.Time{}, true, nil
	}

	at := departure
	if at == 0 {
		at = arrival
	}
	if at == 0 {
		return "", time.Time{}, false, nil
	}

	return stopID, time.Unix(at, 0), false, nil
}

func parseRealtimeStopTimeEvent(data []byte) (int64, error) {
	at := int64(0)

	err := walkProto(data, func(field int, _ []byte, varint uint64) error {
		if field == 2 {
			at = int64(varint)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return at, nil
}

func walkProto(
	data []byte,
	visit func(field int, value []byte, varint uint64) error,
) error {
	for len(data) > 0 {
		tag, n := binary.Uvarint(data)
		if n <= 0 {
			return fmt.Errorf("invalid protobuf tag")
		}
		data = data[n:]

		switch wire := tag & 7; wire {
		case 0:
			varint, n := binary.Uvarint(data)
			if n <= 0 {
				return fmt.Errorf("invalid protobuf varint")
			}
			data = data[n:]

			if err := visit(int(tag>>3), nil, varint); err != nil {
				return err
			}

		case 1:
			if len(data) < 8 {
				return fmt.Errorf("truncated protobuf 64-bit field")
			}

			if err := visit(int(tag>>3), data[:8], 0); err != nil {
				return err
			}

			data = data[8:]

		case 2:
			length, n := binary.Uvarint(data)
			if n <= 0 {
				return fmt.Errorf("invalid protobuf length")
			}
			data = data[n:]

			if uint64(len(data)) < length {
				return fmt.Errorf("truncated protobuf field")
			}

			if err := visit(int(tag>>3), data[:length], 0); err != nil {
				return err
			}

			data = data[length:]

		case 5:
			if len(data) < 4 {
				return fmt.Errorf("truncated protobuf 32-bit field")
			}

			if err := visit(int(tag>>3), data[:4], 0); err != nil {
				return err
			}

			data = data[4:]

		default:
			return fmt.Errorf("unsupported protobuf wire type %d", wire)
		}
	}

	return nil
}
