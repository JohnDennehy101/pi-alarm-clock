package application

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vishen/go-chromecast/cast"
	pb "github.com/vishen/go-chromecast/cast/proto"
)

// fakeConn stands in for the tls cast connection. It records every payload
// sent by the application and, for each payload type in replies, pushes a
// canned receiver response (with %d substituted by the request id) back
// onto the message channel.
type fakeConn struct {
	mu      sync.Mutex
	sent    []string
	replies map[string]string

	msgChan chan *pb.CastMessage
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		replies: map[string]string{},
		msgChan: make(chan *pb.CastMessage, 16),
	}
}

func (f *fakeConn) payloads() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.sent...)
}

func (f *fakeConn) Start(addr string, port int) error { return nil }
func (f *fakeConn) Close() error                      { return nil }
func (f *fakeConn) SetDebug(debug bool)               {}
func (f *fakeConn) LocalAddr() (string, error)        { return "127.0.0.1", nil }
func (f *fakeConn) RemoteAddr() (string, error)       { return "192.0.2.10", nil }
func (f *fakeConn) RemotePort() (string, error)       { return "8009", nil }
func (f *fakeConn) MsgChan() chan *pb.CastMessage     { return f.msgChan }

func (f *fakeConn) Send(requestID int, payload cast.Payload, sourceID, destinationID, namespace string) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	f.mu.Lock()
	f.sent = append(f.sent, string(data))
	f.mu.Unlock()

	var header cast.PayloadHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return nil
	}

	template, ok := f.replies[header.Type]
	if !ok || template == "" {
		return nil
	}

	response := fmt.Sprintf(template, header.RequestId)
	msg := &pb.CastMessage{PayloadUtf8: &response}

	select {
	case f.msgChan <- msg:
	default:
	}

	return nil
}

// newTestApplication builds an Application wired to the fake connection,
// with the Default Media Receiver already running (so Load does not need
// to LAUNCH it, which the fake does not answer).
func newTestApplication(conn *fakeConn) *Application {
	app := NewApplication(
		WithConnection(conn),
		WithCacheDisabled(true),
	)

	app.application = &cast.Application{
		AppId:       defaultChromecastAppID,
		TransportId: "web-1",
		DisplayName: "Default Media Receiver",
		StatusText:  "Ready To Cast",
	}

	return app
}

const boardURL = "http://192.0.2.2:8081/boards/20261125T0846/frame-1795596960.png?t=1"

const mediaStatusOK = `{"type":"MEDIA_STATUS","requestId":%d,"status":[` +
	`{"mediaSessionId":1,"playerState":"IDLE","idleReason":"FINISHED",` +
	`"media":{"contentId":"` + boardURL + `","contentType":"image/png"}}]}`

func sentLoad(t *testing.T, conn *fakeConn) (contentType string, found bool) {
	t.Helper()

	for _, payload := range conn.payloads() {
		if !strings.Contains(payload, `"type":"LOAD"`) {
			continue
		}

		var command struct {
			Media struct {
				ContentID   string `json:"contentId"`
				ContentType string `json:"contentType"`
			} `json:"media"`
		}
		if err := json.Unmarshal([]byte(payload), &command); err != nil {
			t.Fatalf("unmarshaling LOAD payload %s: %v", payload, err)
		}

		return command.Media.ContentType, true
	}

	return "", false
}

func TestLoadWaitsForAcknowledgement(t *testing.T) {
	conn := newFakeConn()
	conn.replies["LOAD"] = mediaStatusOK

	app := newTestApplication(conn)

	if err := app.Load(boardURL, 0, "image/png", false, true, true); err != nil {
		t.Fatalf("Load: %v", err)
	}

	contentType, found := sentLoad(t, conn)
	if !found {
		t.Fatal("no LOAD was sent to the receiver")
	}

	if contentType != "image/png" {
		t.Errorf("LOAD contentType = %q, want image/png", contentType)
	}
}

func TestLoadSurfacesReceiverFailure(t *testing.T) {
	conn := newFakeConn()
	conn.replies["LOAD"] = `{"type":"LOAD_FAILED","requestId":%d}`

	app := newTestApplication(conn)

	err := app.Load(boardURL, 0, "image/png", false, true, true)
	if err == nil {
		t.Fatal("Load = nil error, want LOAD_FAILED to fail the load")
	}

	if !strings.Contains(err.Error(), "LOAD_FAILED") {
		t.Errorf("Load error = %q, want LOAD_FAILED", err)
	}
}

func TestLoadTimesOutWhenReceiverIsSilent(t *testing.T) {
	originalTimeout := loadWaitTimeout
	loadWaitTimeout = 100 * time.Millisecond
	t.Cleanup(func() { loadWaitTimeout = originalTimeout })

	conn := newFakeConn()
	app := newTestApplication(conn)

	err := app.Load(boardURL, 0, "image/png", false, true, true)
	if err == nil {
		t.Fatal("Load = nil error, want a timeout when the receiver never answers")
	}

	if !strings.Contains(err.Error(), "did not acknowledge") {
		t.Errorf("Load error = %q, want an acknowledgement timeout", err)
	}
}

func TestLoadPingKeepAliveIsWired(t *testing.T) {
	// The connection layer's keep-alive (cast/connection.go) is a real
	// tls connection concern; this test only pins the wrapper contract
	// that media loads now wait for the receiver instead of firing and
	// forgetting: a LOAD without any reply configured must block until
	// the load timeout, not return immediately.
	started := time.Now()

	originalTimeout := loadWaitTimeout
	loadWaitTimeout = 150 * time.Millisecond
	t.Cleanup(func() { loadWaitTimeout = originalTimeout })

	conn := newFakeConn()
	app := newTestApplication(conn)

	err := app.Load(boardURL, 0, "image/png", false, true, true)

	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("Load = nil error, want a timeout")
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("Load returned after %s, before the load timeout", elapsed)
	}
}
