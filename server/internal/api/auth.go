package api

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/auth"
)

const (
	maxPairFailures = 5
	pairLockout     = 15 * time.Minute
)

type PairingHandler struct {
	pin         string
	store       *auth.Store
	mu          sync.Mutex
	failures    map[string]int
	lockedUntil map[string]time.Time
}

func NewPairingHandler(pin string, store *auth.Store) *PairingHandler {
	return &PairingHandler{
		pin:         pin,
		store:       store,
		failures:    make(map[string]int),
		lockedUntil: make(map[string]time.Time),
	}
}

type pairRequest struct {
	PIN        string `json:"pin"`
	DeviceName string `json:"deviceName"`
}

func (h *PairingHandler) Pair(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	if ip == nil || !isLocalIP(ip) {
		writeError(w, http.StatusForbidden, "pairing rejected for non-local address")
		return
	}

	var request pairRequest
	if !decodePairRequest(w, r, &request) {
		return
	}

	if !h.verifyWithLockout(w, ip, request) {
		return
	}

	device, token, err := h.store.Pair(request.DeviceName)
	if err != nil {
		log.Printf("pairing for %s failed: %v", ip, err)
		writeError(w, http.StatusInternalServerError, "pairing failed")
		return
	}

	log.Printf("device %q paired as %s from %s", request.DeviceName, device.ID, ip)
	writeJSON(w, http.StatusCreated, map[string]string{
		"deviceId": device.ID,
		"token":    token,
	})
}

func isLocalIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback()
}

func decodePairRequest(w http.ResponseWriter, r *http.Request, request *pairRequest) bool {
	if err := decodeJSON(w, r, request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return false
	}

	if len(request.DeviceName) < 1 || len(request.DeviceName) > 64 {
		writeError(w, http.StatusBadRequest, "deviceName must be between 1 and 64 characters")
		return false
	}

	return true
}

func (h *PairingHandler) verifyWithLockout(w http.ResponseWriter, ip net.IP, request pairRequest) bool {
	if locked, retryAfter := h.isLocked(ip); locked {
		writeLocked(w, retryAfter)
		return false
	}

	if auth.VerifyPIN(request.PIN, h.pin) {
		h.clearFailures(ip)
		return true
	}

	log.Printf("pairing denied for %s (%s)", ip, request.DeviceName)

	if h.recordFailure(ip) {
		writeLocked(w, pairLockout)
		return false
	}

	writeError(w, http.StatusUnauthorized, "invalid PIN")
	return false
}

func writeLocked(w http.ResponseWriter, retryAfter time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
	writeError(w, http.StatusTooManyRequests, "too many failed pairing attempts")
}

func (h *PairingHandler) isLocked(ip net.IP) (bool, time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	until, ok := h.lockedUntil[ip.String()]
	if !ok {
		return false, 0
	}

	if time.Now().Before(until) {
		return true, time.Until(until)
	}

	delete(h.lockedUntil, ip.String())
	return false, 0
}

func (h *PairingHandler) recordFailure(ip net.IP) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := ip.String()
	h.failures[key]++

	if h.failures[key] >= maxPairFailures {
		h.lockedUntil[key] = time.Now().Add(pairLockout)
		delete(h.failures, key)
		return true
	}

	return false
}

func (h *PairingHandler) clearFailures(ip net.IP) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.failures, ip.String())
	delete(h.lockedUntil, ip.String())
}

func clientIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	return net.ParseIP(host)
}
