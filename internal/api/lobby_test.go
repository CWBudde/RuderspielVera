package api

import (
	mrand "math/rand/v2"
	"net/http"
	"strings"
	"testing"

	"ruderspiel/internal/engine"
	"ruderspiel/internal/gameservice"
)

// TestLobbyFlow: host opens 3 seats, a friend joins with a lower-case code, the host starts, the
// third seat becomes a bot, and both humans play decisions until a phase resolves. The open
// lobby and the running game both survive a server restart.
func TestLobbyFlow(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, cl := newTestServer(t, dir)

	seed := uint64(7)

	var host gameservice.LobbyResponse
	if st := cl.do("POST", "/api/lobbies", "", gameservice.CreateLobbyRequest{
		PlayerName: "Vera", Seats: 3, Seed: &seed,
	}, &host); st != http.StatusCreated {
		t.Fatalf("create lobby: %d", st)
	}

	if !host.Lobby.Host || host.PlayerID != "p1" || len(host.Lobby.Code) != 6 {
		t.Fatalf("unexpected host response: %+v", host)
	}

	// Not started yet: the game view is a 409, the lobby view works.
	var eb ErrorBody
	if st := cl.do("GET", "/api/games/"+host.GameID, host.Token, nil, &eb); st != http.StatusConflict ||
		eb.Error != "err.gameNotStarted" {
		t.Fatalf("view before start: %d %q", st, eb.Error)
	}

	// Restart with the lobby open.
	_, cl = newTestServer(t, dir)

	var friend gameservice.LobbyResponse

	code := strings.ToLower(host.Lobby.Code[:3]) + "-" + host.Lobby.Code[3:]
	if st := cl.do("POST", "/api/lobbies/"+code+"/join", "", gameservice.JoinLobbyRequest{
		PlayerName: "Max",
	}, &friend); st != http.StatusCreated {
		t.Fatalf("join: %d", st)
	}

	if friend.PlayerID != "p2" || friend.Lobby.Host || len(friend.Lobby.Players) != 2 {
		t.Fatalf("unexpected join response: %+v", friend)
	}

	if st := cl.do("POST", "/api/games/"+host.GameID+"/start", friend.Token, nil, &eb); st != http.StatusForbidden {
		t.Fatalf("start by non-host: %d", st)
	}

	var view engine.GameView
	if st := cl.do("POST", "/api/games/"+host.GameID+"/start", host.Token, nil, &view); st != http.StatusOK {
		t.Fatalf("start: %d", st)
	}

	if len(view.Players) != 3 || !view.Players[2].IsBot || view.Players[1].Name != "Max" {
		t.Fatalf("unexpected players: %+v", view.Players)
	}

	if st := cl.do("POST", "/api/games/"+host.GameID+"/start", host.Token, nil, &eb); st != http.StatusConflict {
		t.Fatalf("second start: %d", st)
	}

	if st := cl.do("POST", "/api/lobbies/"+host.Lobby.Code+"/join", "", gameservice.JoinLobbyRequest{
		PlayerName: "Late",
	}, &eb); st != http.StatusConflict || eb.Error != "err.lobbyStarted" {
		t.Fatalf("join after start: %d %q", st, eb.Error)
	}

	// Restart with the game running, then the host decides first and waits for Max.
	s, cl := newTestServer(t, dir)
	firstPhase := view.Phase.ID

	submit := func(token, pid string) engine.GameView {
		t.Helper()

		var d engine.Decision

		err := s.Service().WithGame(host.GameID, func(g *engine.Game) error {
			var err error
			d, err = SimpleDecider(g, pid, 0, mrand.New(mrand.NewPCG(1, 2)))

			return err
		})
		if err != nil {
			t.Fatal(err)
		}

		var v engine.GameView
		if st := cl.do("POST", "/api/games/"+host.GameID+"/decisions", token, d, &v); st != http.StatusOK {
			t.Fatalf("decision by %s: %d", pid, st)
		}

		return v
	}

	if v := submit(host.Token, "p1"); !v.Phase.Waiting || v.Phase.ID != firstPhase {
		t.Fatalf("host should wait for Max: waiting=%v phase=%s", v.Phase.Waiting, v.Phase.ID)
	}

	if v := submit(friend.Token, "p2"); v.Phase.ID == firstPhase {
		t.Fatal("phase did not resolve after both humans decided")
	}

	var lv gameservice.LobbyView
	if st := cl.do("GET", "/api/games/"+host.GameID+"/lobby", friend.Token, nil, &lv); st != http.StatusOK ||
		!lv.Started || len(lv.Players) != 3 {
		t.Fatalf("lobby after start: %d %+v", st, lv)
	}
}

func TestLobbyErrors(t *testing.T) {
	t.Parallel()

	_, cl := newTestServer(t, "")

	var eb ErrorBody

	st := cl.do("POST", "/api/lobbies", "", gameservice.CreateLobbyRequest{Seats: 5}, &eb)
	if st != http.StatusBadRequest || eb.Error != "err.seatCount" {
		t.Fatalf("5 seats: %d %q", st, eb.Error)
	}

	st = cl.do("POST", "/api/lobbies/ZZZZZZ/join", "", gameservice.JoinLobbyRequest{}, &eb)
	if st != http.StatusNotFound {
		t.Fatalf("unknown code: %d", st)
	}

	var host gameservice.LobbyResponse
	cl.do("POST", "/api/lobbies", "", gameservice.CreateLobbyRequest{PlayerName: "A", Seats: 2}, &host)

	join := "/api/lobbies/" + host.Lobby.Code + "/join"
	if st := cl.do("POST", join, "", gameservice.JoinLobbyRequest{PlayerName: "B"}, nil); st != http.StatusCreated {
		t.Fatalf("join: %d", st)
	}

	st = cl.do("POST", join, "", gameservice.JoinLobbyRequest{PlayerName: "C"}, &eb)
	if st != http.StatusConflict || eb.Error != "err.lobbyFull" {
		t.Fatalf("full: %d %q", st, eb.Error)
	}
}
