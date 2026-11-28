package api

import (
	"bufio"
	"context"
	mrand "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"ruderspiel/internal/engine"
	"ruderspiel/internal/gameservice"
)

// newShortTimeoutServer is newTestServer with read/write timeouts far below the test's run time,
// so the event stream must clear them to survive.
func newShortTimeoutServer(t *testing.T) (*Server, *client) {
	t.Helper()

	s, err := New(Options{Decider: SimpleDecider})
	if err != nil {
		t.Fatal(err)
	}

	hs := httptest.NewUnstartedServer(s.Handler(nil))
	hs.Config.ReadTimeout = 150 * time.Millisecond
	hs.Config.WriteTimeout = 150 * time.Millisecond
	hs.Start()
	t.Cleanup(hs.Close)

	return s, &client{t: t, srv: hs}
}

// openEvents connects to the event stream and returns the versions of the "change" events.
func (c *client) openEvents(gameID, token string) <-chan uint64 {
	c.t.Helper()

	ctx, cancel := context.WithCancel(c.t.Context())
	c.t.Cleanup(cancel)

	u := c.srv.URL + "/api/games/" + gameID + "/events?token=" + url.QueryEscape(token)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		c.t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req) //nolint:bodyclose // closed by the reader goroutine below
	if err != nil {
		c.t.Fatal(err)
	}

	if ct := res.Header.Get("Content-Type"); res.StatusCode != http.StatusOK || ct != "text/event-stream" {
		_ = res.Body.Close()
		c.t.Fatalf("events: status %d, content type %q", res.StatusCode, ct)
	}

	versions := make(chan uint64, 16)
	connected := make(chan struct{})

	go func() {
		defer func() { _ = res.Body.Close() }()

		sc := bufio.NewScanner(res.Body)
		event := ""

		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == ": connected":
				close(connected)
			case strings.HasPrefix(line, "event: "):
				event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: ") && event == "change":
				v, err := strconv.ParseUint(strings.TrimPrefix(line, "data: "), 10, 64)
				if err == nil {
					versions <- v
				}
			case line == "":
				event = ""
			}
		}
	}()

	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		c.t.Fatal("events: no connected comment")
	}

	return versions
}

func expectChange(t *testing.T, versions <-chan uint64, after uint64, what string) uint64 {
	t.Helper()

	select {
	case v := <-versions:
		if v <= after {
			t.Fatalf("%s: version %d not after %d", what, v, after)
		}

		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: no change event", what)
		return 0
	}
}

// TestEventsStream: a host listening on the event stream sees the join, the start and a
// decision of the other player, also after the server's read/write timeouts have passed.
func TestEventsStream(t *testing.T) {
	t.Parallel()

	srv, cl := newShortTimeoutServer(t)

	seed := uint64(11)

	var host gameservice.LobbyResponse
	if st := cl.do("POST", "/api/lobbies", "", gameservice.CreateLobbyRequest{
		PlayerName: "Vera", Seats: 2, Seed: &seed,
	}, &host); st != http.StatusCreated {
		t.Fatalf("create lobby: %d", st)
	}

	events := cl.openEvents(host.GameID, host.Token)

	time.Sleep(400 * time.Millisecond) // well past ReadTimeout and WriteTimeout

	var friend gameservice.LobbyResponse
	if st := cl.do("POST", "/api/lobbies/"+host.Lobby.Code+"/join", "", gameservice.JoinLobbyRequest{
		PlayerName: "Max",
	}, &friend); st != http.StatusCreated {
		t.Fatalf("join: %d", st)
	}

	v := expectChange(t, events, 0, "join")

	var view engine.GameView
	if st := cl.do("POST", "/api/games/"+host.GameID+"/start", host.Token, nil, &view); st != http.StatusOK {
		t.Fatalf("start: %d", st)
	}

	v = expectChange(t, events, v, "start")

	// The friend decides; the host hears about it.
	d := humanDecision(t, srv, host.GameID, friend.PlayerID, mrand.New(mrand.NewPCG(1, 2)))
	if st := cl.do("POST", "/api/games/"+host.GameID+"/decisions", friend.Token, d, nil); st != http.StatusOK {
		t.Fatalf("decision: %d", st)
	}

	expectChange(t, events, v, "decision")
}

// TestEventsAuth: the stream needs a valid ?token= of the game.
func TestEventsAuth(t *testing.T) {
	t.Parallel()

	_, cl := newTestServer(t, "")
	cr := cl.create(5)

	for _, tc := range []struct {
		path, token string
		status      int
		key         string
	}{
		{"/api/games/" + cr.GameID + "/events", "", http.StatusUnauthorized, "err.noToken"},
		{"/api/games/" + cr.GameID + "/events?token=nope", "", http.StatusUnauthorized, "err.badToken"},
		// A bearer header is not accepted here (EventSource cannot send one).
		{"/api/games/" + cr.GameID + "/events", cr.Token, http.StatusUnauthorized, "err.noToken"},
		{"/api/games/abcd/events?token=" + cr.Token, "", http.StatusNotFound, "err.gameNotFound"},
	} {
		var eb ErrorBody
		if st := cl.do("GET", tc.path, tc.token, nil, &eb); st != tc.status || eb.Error != tc.key {
			t.Fatalf("%s: got %d %q, want %d %q", tc.path, st, eb.Error, tc.status, tc.key)
		}
	}
}
