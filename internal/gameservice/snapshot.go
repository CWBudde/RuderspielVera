package gameservice

import (
	"fmt"
	"maps"
	"time"

	"ruderspiel/internal/engine"
)

// SnapshotVersion is bumped when the snapshot format changes incompatibly or when a rule change
// makes old decision logs replay differently (2: per-mille incidents, no Motivation training,
// Hildesheim loan boats, Motivation weight 1 everywhere; 3: season-aware Ereignisse, the larger
// event and Ereignis decks, trainer mood).
const SnapshotVersion = 3

// Snapshot is everything needed to rebuild a game with engine.Replay: config (incl. seed),
// tokens and the decision log. It is the on-disk file of the HTTP server and the localStorage
// entry of the WebAssembly build.
type Snapshot struct {
	Version   int                        `json:"version"`
	Config    engine.Config              `json:"config"`
	Tokens    map[string]string          `json:"tokens"` // bearer token -> player ID
	Decisions []engine.SubmittedDecision `json:"decisions"`
	CreatedAt time.Time                  `json:"createdAt"`
	UpdatedAt time.Time                  `json:"updatedAt"`
}

// snapshot builds the snapshot of a live game. Caller holds e.mu.
func (s *Service) snapshot(e *gameEntry) Snapshot {
	return Snapshot{
		Version:   SnapshotVersion,
		Config:    e.game.Config(),
		Tokens:    maps.Clone(e.tokens),
		Decisions: e.game.Decisions(),
		CreatedAt: e.createdAt,
		UpdatedAt: s.now(),
	}
}

// save hands the game to the store, if any. Caller holds e.mu.
func (s *Service) save(e *gameEntry) error {
	if s.store == nil {
		return nil
	}

	err := s.store.Save(s.snapshot(e))
	if err != nil {
		s.log.Error("save failed", "game", e.id, "err", err)
		return newError(StatusInternal, "Das Spiel konnte nicht gespeichert werden.")
	}

	return nil
}

// Export returns the snapshot of a game. Callers outside the service are responsible for
// keeping it private: it contains the bearer tokens.
func (s *Service) Export(gameID string) (Snapshot, error) {
	e := s.lookup(gameID)
	if e == nil {
		return Snapshot{}, newError(StatusNotFound, "Dieses Spiel gibt es nicht.")
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
		return "", newError(StatusBadRequest,
			"Dieser Spielstand stammt aus einer älteren Spielversion mit anderen Regeln und kann nicht "+
				"mehr geladen werden. Bitte starte ein neues Spiel.")
	}

	if snap.Version != SnapshotVersion {
		return "", newError(StatusBadRequest, fmt.Sprintf("Unbekanntes Spielstand-Format (Version %d).", snap.Version))
	}

	id := snap.Config.GameID
	if !ValidGameID(id) {
		return "", newError(StatusBadRequest, "Ungültige Spiel-ID im Spielstand.")
	}

	g, err := engine.Replay(snap.Config, snap.Decisions)
	if err != nil {
		s.log.Error("replay failed", "game", id, "err", err)
		return "", newError(StatusBadRequest, "Der Spielstand konnte nicht wiederhergestellt werden.")
	}

	tokens := snap.Tokens
	if tokens == nil {
		tokens = map[string]string{}
	}

	e := newGameEntry(id, g, tokens, snap.CreatedAt)

	s.mu.Lock()
	s.games[id] = e
	s.mu.Unlock()

	return id, nil
}

// Count returns the number of registered games.
func (s *Service) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.games)
}
