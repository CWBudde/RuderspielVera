package gameservice

import (
	"slices"
	"time"

	"ruderspiel/internal/engine"
)

// Background maintenance ("janitor") for the HTTP server:
//
//   - Turn timeout: in a started multiplayer game that waits only on humans, a player who has not
//     decided within TurnTimeout of the phase starting gets a bot decision (same Decider as the
//     bot seats). The clock starts when a phase begins waiting on humans; it lives in memory only,
//     so a restart restarts it.
//   - Stale lobbies: lobbies that were never started are dropped after LobbyMaxAge, together with
//     their join code and (if the store supports it) their snapshot.
//
// The WebAssembly build never starts the janitor.

// Deleter is an optional Store extension: Delete removes a persisted game. Called with the game
// lock held.
type Deleter interface {
	Delete(gameID string) error
}

// DefaultJanitorInterval is how often StartJanitor checks timeouts and stale lobbies.
const DefaultJanitorInterval = 30 * time.Second

// markWaiting restarts the turn clock when the game entered a new phase. Caller holds e.mu.
func (s *Service) markWaiting(e *gameEntry) {
	if e.game == nil {
		return
	}

	if id := e.game.PhaseID(); id != e.waitPhase {
		e.waitPhase = id
		e.waitSince = s.now()
	}
}

// StartJanitor runs Sweep every interval (DefaultJanitorInterval if <= 0) in a background
// goroutine until Close. Calling it twice is a no-op.
func (s *Service) StartJanitor(interval time.Duration) {
	if interval <= 0 {
		interval = DefaultJanitorInterval
	}

	s.janitorMu.Lock()
	defer s.janitorMu.Unlock()

	if s.stop != nil {
		return
	}

	s.stop = make(chan struct{})
	s.done = make(chan struct{})

	go func(stop <-chan struct{}, done chan<- struct{}) {
		defer close(done)

		t := time.NewTicker(interval)
		defer t.Stop()

		for {
			select {
			case <-stop:
				return
			case <-t.C:
				s.Sweep()
			}
		}
	}(s.stop, s.done)
}

// Close stops the janitor (if running) and waits for a running sweep to finish.
func (s *Service) Close() {
	s.janitorMu.Lock()
	defer s.janitorMu.Unlock()

	if s.stop == nil {
		return
	}

	close(s.stop)
	<-s.done
	s.stop, s.done = nil, nil
}

// SweepResult counts what one Sweep did.
type SweepResult struct {
	Takeovers      int // bot decisions made for timed-out humans
	LobbiesRemoved int
}

// Sweep makes bot decisions for humans past the turn timeout and removes stale lobbies.
func (s *Service) Sweep() SweepResult {
	s.mu.RLock()
	entries := make([]*gameEntry, 0, len(s.games))

	for _, e := range s.games {
		if e.lobby != nil { // solo games are never touched
			entries = append(entries, e)
		}
	}

	s.mu.RUnlock()

	var res SweepResult

	for _, e := range entries {
		n, removed := s.sweepEntry(e)
		res.Takeovers += n

		if removed {
			res.LobbiesRemoved++
		}
	}

	return res
}

func (s *Service) sweepEntry(e *gameEntry) (int, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch {
	case e.removed:
		return 0, false
	case !e.lobby.Started:
		return 0, s.removeStaleLobby(e)
	default:
		return s.takeOver(e), false
	}
}

// removeStaleLobby drops an open lobby older than lobbyMaxAge. Caller holds e.mu.
func (s *Service) removeStaleLobby(e *gameEntry) bool {
	if s.lobbyMaxAge <= 0 || s.now().Sub(e.createdAt) < s.lobbyMaxAge {
		return false
	}

	if d, ok := s.store.(Deleter); ok {
		err := d.Delete(e.id)
		if err != nil {
			s.log.Error("delete stale lobby failed", "game", e.id, "err", err)
			return false
		}
	}

	e.removed = true

	s.mu.Lock()
	delete(s.games, e.id)

	if s.codes[e.lobby.Code] == e.id {
		delete(s.codes, e.lobby.Code)
	}
	s.mu.Unlock()

	s.log.Info("stale lobby removed", "game", e.id, "code", e.lobby.Code,
		"age", s.now().Sub(e.createdAt).Round(time.Second))

	return true
}

// takeOver lets the bot decide for every human the current phase has waited on for longer than
// the turn timeout, then runs the bots and saves. Caller holds e.mu.
func (s *Service) takeOver(e *gameEntry) int {
	g := e.game
	if s.turnTimeout <= 0 || g == nil || g.Over() || g.PhaseID() != e.waitPhase ||
		s.now().Sub(e.waitSince) < s.turnTimeout {
		return 0
	}

	players := g.Config().Players
	phase := g.PhaseID()
	taken := 0

	for _, pid := range g.Pending() {
		if g.PhaseID() != phase || !slices.Contains(g.Pending(), pid) {
			break // the phase resolved with an earlier takeover
		}

		seat := slices.IndexFunc(players, func(p engine.PlayerConfig) bool { return p.ID == pid })
		if seat < 0 || players[seat].IsBot {
			continue
		}

		d, err := s.decide(g, pid, seat, e.botRNG(seat))
		if err == nil {
			err = g.Submit(pid, d)
		}

		if err != nil {
			s.log.Error("turn timeout: bot decision failed", "game", e.id, "player", pid, "phase", phase, "err", err)
			continue
		}

		taken++

		s.log.Info("turn timeout: bot decided for player", "game", e.id, "player", pid, "phase", phase,
			"waited", s.now().Sub(e.waitSince).Round(time.Second))
	}

	if taken == 0 {
		return 0
	}

	err := s.runBots(e)
	if err != nil {
		s.log.Error("turn timeout: bots failed", "game", e.id, "err", err)
	}

	// save logs its own errors; there is nobody to report them to.
	_ = s.save(e)

	return taken
}
