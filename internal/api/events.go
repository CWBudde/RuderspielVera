package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// sseHeartbeat is how often an idle event stream sends a comment, so proxies keep it open.
const sseHeartbeat = 25 * time.Second

// sseRetryMillis tells EventSource how long to wait before reconnecting after a drop.
const sseRetryMillis = 3000

// handleEvents streams "game changed" notifications as Server-Sent Events:
//
//	event: change
//	data: <version>
//
// EventSource cannot set headers, so this endpoint (only this one) takes the token as ?token=.
// The client refetches the game or lobby on each event; the payload is just a counter.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	changes, unsubscribe, err := s.svc.Subscribe(r.PathValue("id"), r.URL.Query().Get("token"))
	if err != nil {
		reply(w, 0, nil, err)
		return
	}
	defer unsubscribe()

	// The server's read and write timeouts are meant for short requests; a stream lives as long as
	// the client stays. (A stale read deadline would also cancel the request context.)
	rc := http.NewResponseController(w)

	err = rc.SetWriteDeadline(time.Time{})
	if err == nil {
		err = rc.SetReadDeadline(time.Time{})
	}

	if err != nil && !errors.Is(err, http.ErrNotSupported) {
		s.log.Warn("events: clear deadlines", "err", err)
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no") // nginx: do not buffer the stream
	w.WriteHeader(http.StatusOK)

	send := func(format string, args ...any) bool {
		_, err := fmt.Fprintf(w, format, args...)
		if err == nil {
			err = rc.Flush()
		}

		return err == nil
	}

	if !send("retry: %d\n: connected\n\n", sseRetryMillis) {
		return
	}

	heartbeat := time.NewTicker(sseHeartbeat)
	defer heartbeat.Stop()

	for {
		var ok bool

		select {
		case <-r.Context().Done():
			return
		case v := <-changes:
			ok = send("event: change\ndata: %d\n\n", v)
		case <-heartbeat.C:
			ok = send(": ping\n\n")
		}

		if !ok {
			return
		}
	}
}
