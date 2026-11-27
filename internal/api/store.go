package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	mrand "math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ruderspiel/internal/engine"
)

// fileVersion is bumped when the persisted game file format changes incompatibly.
const fileVersion = 1

// gameFile is the on-disk form of one game: everything needed to rebuild it with engine.Replay.
type gameFile struct {
	Version   int                        `json:"version"`
	Config    engine.Config              `json:"config"`
	Tokens    map[string]string          `json:"tokens"` // bearer token -> player ID
	Decisions []engine.SubmittedDecision `json:"decisions"`
	CreatedAt time.Time                  `json:"createdAt"`
	UpdatedAt time.Time                  `json:"updatedAt"`
}

// gameEntry is one live game. mu serialises all access to game and rngs.
type gameEntry struct {
	mu        sync.Mutex
	id        string
	game      *engine.Game
	tokens    map[string]string // token -> player ID (immutable after creation)
	createdAt time.Time
	rngs      map[int]*mrand.Rand // bot RNG per seat
}

// botRNG returns the bot's private RNG for a seat, seeded from (game seed, seat). After a server
// restart the stream starts over; that is fine because past bot decisions are in the log.
func (e *gameEntry) botRNG(seat int) *mrand.Rand {
	if r, ok := e.rngs[seat]; ok {
		return r
	}
	r := mrand.New(mrand.NewPCG(e.game.Config().Seed, 0xB0700+uint64(seat)))
	e.rngs[seat] = r
	return r
}

func newGameEntry(id string, g *engine.Game, tokens map[string]string, createdAt time.Time) *gameEntry {
	return &gameEntry{id: id, game: g, tokens: tokens, createdAt: createdAt, rngs: map[int]*mrand.Rand{}}
}

// randomHex returns n random bytes as lowercase hex.
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err)) // never fails on supported platforms
	}
	return hex.EncodeToString(b)
}

func validGameID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

// save writes the game file atomically (temp file + rename). Caller holds e.mu.
func (s *Server) save(e *gameEntry) error {
	if s.dataDir == "" {
		return nil
	}
	f := gameFile{
		Version:   fileVersion,
		Config:    e.game.Config(),
		Tokens:    e.tokens,
		Decisions: e.game.Decisions(),
		CreatedAt: e.createdAt,
		UpdatedAt: time.Now().UTC(),
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dataDir, "."+e.id+"-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(s.dataDir, e.id+".json"))
}

// loadAll reads every game file in the data dir and replays it. Broken files are logged and skipped.
func (s *Server) loadAll() error {
	if s.dataDir == "" {
		return nil
	}
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return err
	}
	for _, de := range entries {
		name := de.Name()
		if de.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		e, err := s.loadFile(id, filepath.Join(s.dataDir, name))
		if err != nil {
			s.log.Error("skipping game file", "file", name, "err", err)
			continue
		}
		s.games[id] = e
		for tok := range e.tokens {
			s.tokens[tok] = e
		}
	}
	s.log.Info("games loaded", "count", len(s.games), "dir", s.dataDir)
	return nil
}

func (s *Server) loadFile(id, path string) (*gameEntry, error) {
	if !validGameID(id) {
		return nil, errors.New("invalid game ID in file name")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f gameFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	if f.Version != fileVersion {
		return nil, fmt.Errorf("unsupported file version %d", f.Version)
	}
	if f.Config.GameID != id {
		return nil, fmt.Errorf("game ID mismatch: file %q, config %q", id, f.Config.GameID)
	}
	g, err := engine.Replay(f.Config, f.Decisions)
	if err != nil {
		return nil, err
	}
	if f.Tokens == nil {
		f.Tokens = map[string]string{}
	}
	return newGameEntry(id, g, f.Tokens, f.CreatedAt), nil
}
