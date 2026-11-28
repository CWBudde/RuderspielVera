package gameservice

import (
	"maps"
	"slices"
	"time"

	"ruderspiel/internal/engine"
)

// SnapshotVersion is bumped when the snapshot format changes incompatibly or when a rule change
// makes old decision logs replay differently (2: per-mille incidents, no Motivation training,
// Hildesheim loan boats, Motivation weight 1 everywhere; 3: season-aware Ereignisse, the larger
// event and Ereignis decks, trainer mood; 4: temporary Siegesmotivation instead of permanent Gold
// motivation, weaker guest crews, shorter regular regattas; 5: Krebs chance scaled by boat size
// and crew Technik, Seelenzwilling links a teammate, Wanderfahrt for Junioren only; 6: named city
// boats dealt at random with values 1/2/4, loan boats worth 0, own eights Kraftwerk / Paintrain,
// boat values count once instead of twice per seat; 7: Hannover content pack, eight more
// Ereignisse and six more event cards change the seeded deck draws; 8: premium boats Vierer mit
// Freunden and Blitz enlarge the draft pool; 9: trainer age bonuses change entry scores; 10: trainer
// id frau-linke renamed to frau-lincke).
const SnapshotVersion = 10

// Snapshot is everything needed to rebuild a game with engine.Replay: config (incl. seed),
// tokens and the decision log. It is the on-disk file of the HTTP server and the localStorage
// entry of the WebAssembly build.
type Snapshot struct {
	Version   int                        `json:"version"`
	Config    engine.Config              `json:"config"`
	Tokens    map[string]string          `json:"tokens"` // bearer token -> player ID
	Decisions []engine.SubmittedDecision `json:"decisions"`
	Lobby     *Lobby                     `json:"lobby,omitempty"` // multiplayer games only
	CreatedAt time.Time                  `json:"createdAt"`
	UpdatedAt time.Time                  `json:"updatedAt"`
}

// snapshot builds the snapshot of a live game. Caller holds e.mu.
func (s *Service) snapshot(e *gameEntry) Snapshot {
	e.tokMu.RLock()
	tokens := maps.Clone(e.tokens)
	e.tokMu.RUnlock()

	snap := Snapshot{
		Version:   SnapshotVersion,
		Tokens:    tokens,
		CreatedAt: e.createdAt,
		UpdatedAt: s.now(),
	}

	if e.lobby != nil {
		l := *e.lobby
		snap.Lobby = &l
	}

	if e.game == nil {
		snap.Config = e.cfg
		snap.Config.Players = slices.Clone(e.cfg.Players)
		snap.Decisions = []engine.SubmittedDecision{}
	} else {
		snap.Config = e.game.Config()
		snap.Decisions = e.game.Decisions()
	}

	return snap
}

// save hands the game to the store, if any. Caller holds e.mu.
func (s *Service) save(e *gameEntry) error {
	if s.store != nil {
		err := s.store.Save(s.snapshot(e))
		if err != nil {
			s.log.Error("save failed", "game", e.id, "err", err)
			return newError(StatusInternal, "err.saveFailed")
		}
	}

	// Every mutation ends in save, so this is the one place that tells subscribers.
	s.events.notify(e.id)

	return nil
}

// Export returns the snapshot of a game. Callers outside the service are responsible for
// keeping it private: it contains the bearer tokens.
func (s *Service) Export(gameID string) (Snapshot, error) {
	e := s.lookup(gameID)
	if e == nil {
		return Snapshot{}, newError(StatusNotFound, "err.gameNotFound")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	return s.snapshot(e), nil
}

// Import replays a snapshot and registers the game, replacing a game with the same ID.
// It does not call the store. Bots are not run: a snapshot is always saved after the bots
// handed control back, so replaying it yields a game that waits on a human (or is over).
func (s *Service) Import(snap Snapshot) (string, error) {
	if snap.Version > 0 && snap.Version < SnapshotVersion {
		return "", newError(StatusBadRequest, "err.snapshotOld")
	}

	if snap.Version != SnapshotVersion {
		return "", newError(StatusBadRequest, "err.snapshotVersion", "version", snap.Version)
	}

	id := snap.Config.GameID
	if !ValidGameID(id) {
		return "", newError(StatusBadRequest, "err.snapshotGameID")
	}

	tokens := snap.Tokens
	if tokens == nil {
		tokens = map[string]string{}
	}

	var e *gameEntry

	if snap.Lobby != nil && !snap.Lobby.Started {
		if !validLobby(snap.Lobby, snap.Config) {
			return "", newError(StatusBadRequest, "err.snapshotReplay")
		}

		e = newGameEntry(id, nil, tokens, snap.CreatedAt)
		e.cfg = snap.Config
	} else {
		g, err := engine.Replay(snap.Config, snap.Decisions)
		if err != nil {
			s.log.Error("replay failed", "game", id, "err", err)
			return "", newError(StatusBadRequest, "err.snapshotReplay")
		}

		e = newGameEntry(id, g, tokens, snap.CreatedAt)
		s.markWaiting(e) // the turn clock restarts on load
	}

	if snap.Lobby != nil {
		l := *snap.Lobby
		e.lobby = &l
	}

	s.mu.Lock()
	s.games[id] = e

	if e.lobby != nil {
		s.codes[e.lobby.Code] = id
	}
	s.mu.Unlock()

	return id, nil
}

// Count returns the number of registered games.
func (s *Service) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.games)
}
