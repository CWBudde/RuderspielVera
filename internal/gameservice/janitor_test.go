package gameservice

import (
	"slices"
	"sync"
	"testing"
	"time"

	"ruderspiel/internal/engine"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// delStore is a memStore that also supports Delete.
type delStore struct {
	memStore

	deleted []string
}

func (d *delStore) Delete(id string) error {
	delete(d.saved, id)
	d.deleted = append(d.deleted, id)

	return nil
}

func newJanitorService(t *testing.T, store Store, clk *fakeClock) *Service {
	t.Helper()

	s, err := New(Options{
		Decider: SimpleDecider, Store: store, Now: clk.now,
		TurnTimeout: 10 * time.Minute, LobbyMaxAge: 24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	return s
}

// phaseState returns the current phase ID and pending players.
func phaseState(t *testing.T, s *Service, id string) (string, []string) {
	t.Helper()

	var (
		phase   string
		pending []string
	)

	err := s.WithGame(id, func(g *engine.Game) error {
		phase, pending = g.PhaseID(), g.Pending()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return phase, pending
}

// TestTurnTimeoutTakeover: two humans, nobody decides; before the timeout nothing happens, after
// it the bot decides for them and the game moves on and is saved. A solo game is never touched.
func TestTurnTimeoutTakeover(t *testing.T) {
	t.Parallel()

	clk := &fakeClock{t: time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)}
	store := &delStore{saved: map[string]Snapshot{}}
	s := newJanitorService(t, store, clk)

	solo := create(t, s, 3)

	seed := uint64(11)

	host, err := s.CreateLobby(CreateLobbyRequest{PlayerName: "Vera", Seats: 2, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.JoinLobby(host.Lobby.Code, JoinLobbyRequest{PlayerName: "Max"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.StartLobby(host.GameID, host.Token)
	if err != nil {
		t.Fatal(err)
	}

	phase0, pending0 := phaseState(t, s, host.GameID)
	if len(pending0) == 0 {
		t.Fatal("started game waits on nobody")
	}

	soloPhase, soloPending := phaseState(t, s, solo.GameID)
	saves := len(store.saved[host.GameID].Decisions)

	clk.advance(9 * time.Minute)

	if res := s.Sweep(); res.Takeovers != 0 {
		t.Fatalf("takeover before the timeout: %+v", res)
	}

	clk.advance(2 * time.Minute)

	res := s.Sweep()
	if res.Takeovers == 0 || res.Takeovers > len(pending0) {
		t.Fatalf("takeovers = %d, pending were %v", res.Takeovers, pending0)
	}

	phase1, pending1 := phaseState(t, s, host.GameID)
	if phase1 == phase0 && len(pending1) >= len(pending0) {
		t.Fatalf("game did not move: phase %s pending %v", phase1, pending1)
	}

	if got := len(store.saved[host.GameID].Decisions); got <= saves {
		t.Fatalf("takeover not saved: %d decisions, had %d", got, saves)
	}

	// The new phase starts a fresh clock: an immediate second sweep does nothing.
	if phase1 != phase0 {
		if res := s.Sweep(); res.Takeovers != 0 {
			t.Fatalf("second sweep took over again: %+v", res)
		}
	}

	if p, pend := phaseState(t, s, solo.GameID); p != soloPhase || !slices.Equal(pend, soloPending) {
		t.Fatalf("solo game changed: %s %v -> %s %v", soloPhase, soloPending, p, pend)
	}
}

// TestTurnTimeoutDisabled: TurnTimeout 0 never takes over.
func TestTurnTimeoutDisabled(t *testing.T) {
	t.Parallel()

	clk := &fakeClock{t: time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)}

	s, err := New(Options{Decider: SimpleDecider, Now: clk.now})
	if err != nil {
		t.Fatal(err)
	}

	host, err := s.CreateLobby(CreateLobbyRequest{PlayerName: "Vera", Seats: 2})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.StartLobby(host.GameID, host.Token)
	if err != nil {
		t.Fatal(err)
	}

	clk.advance(1000 * time.Hour)

	if res := s.Sweep(); res != (SweepResult{}) {
		t.Fatalf("sweep with janitor options off: %+v", res)
	}
}

// TestStaleLobbyCleanup: an open lobby older than LobbyMaxAge is removed from memory, the code
// map and the store; a started lobby of the same age stays.
func TestStaleLobbyCleanup(t *testing.T) {
	t.Parallel()

	clk := &fakeClock{t: time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)}
	store := &delStore{saved: map[string]Snapshot{}}
	s := newJanitorService(t, store, clk)

	open, err := s.CreateLobby(CreateLobbyRequest{PlayerName: "Vera", Seats: 3})
	if err != nil {
		t.Fatal(err)
	}

	started, err := s.CreateLobby(CreateLobbyRequest{PlayerName: "Max", Seats: 2})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.StartLobby(started.GameID, started.Token)
	if err != nil {
		t.Fatal(err)
	}

	clk.advance(23 * time.Hour)

	if res := s.Sweep(); res.LobbiesRemoved != 0 {
		t.Fatalf("removed too early: %+v", res)
	}

	clk.advance(2 * time.Hour)

	if res := s.Sweep(); res.LobbiesRemoved != 1 {
		t.Fatalf("LobbiesRemoved = %d, want 1", res.LobbiesRemoved)
	}

	if !slices.Equal(store.deleted, []string{open.GameID}) {
		t.Fatalf("deleted = %v", store.deleted)
	}

	_, err = s.Lobby(open.GameID, open.Token)
	if AsError(err).Status != StatusNotFound {
		t.Fatalf("lobby still reachable: %v", err)
	}

	_, err = s.JoinLobby(open.Lobby.Code, JoinLobbyRequest{PlayerName: "Late"})
	if AsError(err).Msg.Key != "err.lobbyNotFound" {
		t.Fatalf("join after cleanup: %v", err)
	}

	s.mu.RLock()
	_, codeLeft := s.codes[open.Lobby.Code]
	s.mu.RUnlock()

	if codeLeft {
		t.Fatal("join code left behind")
	}

	_, err = s.Lobby(started.GameID, started.Token)
	if err != nil {
		t.Fatalf("started game removed: %v", err)
	}
}

// TestJanitorStartClose: the background janitor starts and stops cleanly (also twice).
func TestJanitorStartClose(t *testing.T) {
	t.Parallel()

	s := newService(t, nil)
	s.StartJanitor(time.Millisecond)
	s.StartJanitor(time.Millisecond)
	time.Sleep(5 * time.Millisecond) // let a few ticks run
	s.Close()
	s.Close()
}
