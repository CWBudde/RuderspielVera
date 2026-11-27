// Package gameservice is the transport-independent core of the Ruderspiel API: game registry,
// token check, bot loop, snapshots for persistence and the mapping of engine errors to
// {status, German message}. It is used by the HTTP server (internal/api) and by the WebAssembly
// build (cmd/wasm), so it must not import net/http or touch the file system.
package gameservice

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	mrand "math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ruderspiel/internal/engine"
)

// Decider makes the decision for a bot seat in the current phase. rng is the bot's own RNG
// (seeded from game seed and seat); it must not be shared with the game.
type Decider func(g *engine.Game, playerID string, seat int, rng *mrand.Rand) (engine.Decision, error)

// Status codes of service errors. They match HTTP so both transports report the same numbers.
const (
	StatusBadRequest   = 400
	StatusUnauthorized = 401
	StatusNotFound     = 404
	StatusConflict     = 409
	StatusInternal     = 500
)

// Error is a failed service call: Status is HTTP-like, Msg is a message key (with params) that the
// UI translates. Transports send it as {"error": "<key>", "msg": {"key": …, "args": …}}.
type Error struct {
	Status int
	Msg    engine.Msg
}

func (e *Error) Error() string { return e.Msg.String() }

// NewError builds an Error from a message key and name/value pairs (see engine.NewMsg).
func NewError(status int, key string, kv ...any) *Error {
	return &Error{Status: status, Msg: engine.NewMsg(key, kv...)}
}

func newError(status int, key string, kv ...any) *Error { return NewError(status, key, kv...) }

// AsError maps any error to a *Error (unknown errors become a generic 500).
func AsError(err error) *Error {
	if se, ok := errors.AsType[*Error](err); ok {
		return se
	}

	return newError(StatusInternal, "err.internal")
}

// Store persists a game after every change. Save is called with the game lock held.
type Store interface {
	Save(snap Snapshot) error
}

// Options configures a Service.
type Options struct {
	Decider Decider      // bot brain (required)
	Store   Store        // optional persistence
	Logger  *slog.Logger // defaults to slog.Default()
	Now     func() time.Time
}

// Service holds all games. The maps are guarded by mu; each game has its own mutex.
type Service struct {
	mu     sync.RWMutex
	games  map[string]*gameEntry
	decide Decider
	store  Store
	log    *slog.Logger
	now    func() time.Time
}

// Bot display names per bot index (seat 1..3).
//
//nolint:gochecknoglobals // read-only name table
var botNames = []string{"Ruderclub Ihme", "Leine-Ruderer", "RV Maschsee"}

const (
	maxNameLen  = 24
	defaultName = "Du"
	defaultBots = 2
	maxBotSteps = 10000
)

// ErrNoDecider is returned by New when Options.Decider is nil.
var ErrNoDecider = errors.New("gameservice: Decider is required")

// New creates an empty service.
func New(opts Options) (*Service, error) {
	if opts.Decider == nil {
		return nil, ErrNoDecider
	}

	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}

	return &Service{
		games:  map[string]*gameEntry{},
		decide: opts.Decider,
		store:  opts.Store,
		log:    opts.Logger,
		now:    opts.Now,
	}, nil
}

// CreateRequest is the body of "create game".
type CreateRequest struct {
	PlayerName string  `json:"playerName"`
	Seed       *uint64 `json:"seed,omitempty"`
	Bots       *int    `json:"bots,omitempty"`
}

// CreateResponse is the answer to "create game".
type CreateResponse struct {
	GameID   string          `json:"gameId"`
	PlayerID string          `json:"playerId"`
	Token    string          `json:"token"`
	View     engine.GameView `json:"view"`
}

// LogResponse is the full game log plus all competition results.
type LogResponse struct {
	Log     []engine.LogEntry          `json:"log"`
	Results []engine.CompetitionResult `json:"results"`
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

func newGameEntry(id string, g *engine.Game, tokens map[string]string, createdAt time.Time) *gameEntry {
	return &gameEntry{id: id, game: g, tokens: tokens, createdAt: createdAt, rngs: map[int]*mrand.Rand{}}
}

// botRNG returns the bot's private RNG for a seat, seeded from (game seed, seat). After a reload
// the stream starts over; that is fine because past bot decisions are in the log.
func (e *gameEntry) botRNG(seat int) *mrand.Rand {
	if r, ok := e.rngs[seat]; ok {
		return r
	}

	stream := 0xB0700 + uint64(seat) //nolint:gosec // seat is a small non-negative index
	r := mrand.New(mrand.NewPCG(e.game.Config().Seed, stream))
	e.rngs[seat] = r

	return r
}

// Create starts a new game for one human (p1) and req.Bots bots, and runs the bots until the
// human is on turn.
func (s *Service) Create(req CreateRequest) (CreateResponse, error) {
	bots := defaultBots
	if req.Bots != nil {
		bots = *req.Bots
	}

	if bots < 1 || bots > len(botNames) {
		return CreateResponse{}, newError(StatusBadRequest, "err.botCount", "max", len(botNames))
	}

	seed := mrand.Uint64()
	if req.Seed != nil {
		seed = *req.Seed
	}

	id := randomHex(8)

	cfg := engine.Config{GameID: id, Seed: seed, Players: []engine.PlayerConfig{
		{ID: "p1", Name: CleanName(req.PlayerName)},
	}}
	for i := range bots {
		cfg.Players = append(cfg.Players, engine.PlayerConfig{
			ID: fmt.Sprintf("p%d", i+2), Name: botNames[i], IsBot: true,
		})
	}

	g, err := engine.New(cfg)
	if err != nil {
		s.log.Error("engine.New failed", "err", err)
		return CreateResponse{}, newError(StatusInternal, "err.createFailed")
	}

	token := randomHex(16)
	e := newGameEntry(id, g, map[string]string{token: "p1"}, s.now())

	e.mu.Lock()
	defer e.mu.Unlock()

	botErr := s.runBots(e)

	err = s.save(e)
	if err != nil {
		return CreateResponse{}, err
	}

	s.mu.Lock()
	s.games[id] = e
	s.mu.Unlock()

	if botErr != nil {
		return CreateResponse{}, botErr
	}

	s.log.Info("game created", "game", id, "seed", seed, "bots", bots)

	return CreateResponse{GameID: id, PlayerID: "p1", Token: token, View: g.View("p1")}, nil
}

// View returns the game as seen by the token's player.
func (s *Service) View(gameID, token string) (engine.GameView, error) {
	e, pid, err := s.authorize(gameID, token)
	if err != nil {
		return engine.GameView{}, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	return e.game.View(pid), nil
}

// Submit records the human's decision, lets the bots catch up, persists and returns the new view.
func (s *Service) Submit(gameID, token string, d engine.Decision) (engine.GameView, error) {
	e, pid, err := s.authorize(gameID, token)
	if err != nil {
		return engine.GameView{}, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	err = e.game.Submit(pid, d)
	if err != nil {
		var ve *engine.ValidationError
		switch {
		case errors.Is(err, engine.ErrStalePhase):
			return engine.GameView{}, newError(StatusConflict, "err.stalePhase")
		case errors.As(err, &ve):
			return engine.GameView{}, &Error{Status: StatusBadRequest, Msg: ve.Msg}
		default:
			s.log.Error("submit failed", "game", e.id, "player", pid, "err", err)
			return engine.GameView{}, newError(StatusInternal, "err.submitFailed")
		}
	}

	botErr := s.runBots(e)

	err = s.save(e)
	if err != nil {
		return engine.GameView{}, err
	}

	if botErr != nil {
		return engine.GameView{}, botErr
	}

	return e.game.View(pid), nil
}

// Log returns the game log and all competition results (never nil slices).
func (s *Service) Log(gameID, token string) (LogResponse, error) {
	e, _, err := s.authorize(gameID, token)
	if err != nil {
		return LogResponse{}, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	resp := LogResponse{Log: e.game.Log(), Results: e.game.Results()}
	if resp.Log == nil {
		resp.Log = []engine.LogEntry{}
	}

	if resp.Results == nil {
		resp.Results = []engine.CompetitionResult{}
	}

	return resp, nil
}

// Catalog returns the static card catalog.
func (s *Service) Catalog() (engine.CatalogView, error) {
	c, err := engine.Catalog()
	if err != nil {
		s.log.Error("catalog failed", "err", err)
		return engine.CatalogView{}, newError(StatusInternal, "err.catalogFailed")
	}

	return c, nil
}

// WithGame runs fn on the live game under its lock. Meant for tests and tools, not transports.
func (s *Service) WithGame(gameID string, fn func(*engine.Game) error) error {
	e := s.lookup(gameID)
	if e == nil {
		return newError(StatusNotFound, "err.gameNotFound")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	return fn(e.game)
}

// CheckToken reports (as *Error) whether the game exists and token belongs to one of its players.
func (s *Service) CheckToken(gameID, token string) error {
	_, _, err := s.authorize(gameID, token)
	return err
}

// runBots lets bots decide while the game is running and a pending player is a bot, so the view
// returned afterwards always waits on a human (or shows game over). Caller holds e.mu.
func (s *Service) runBots(e *gameEntry) error {
	cfg := e.game.Config()
	for steps := 0; !e.game.Over(); steps++ {
		if steps > maxBotSteps {
			s.log.Error("bots did not hand back control", "game", e.id, "steps", maxBotSteps)
			return newError(StatusInternal, "err.botFailed")
		}

		seat := -1

		for _, pid := range e.game.Pending() {
			i := slices.IndexFunc(cfg.Players, func(p engine.PlayerConfig) bool { return p.ID == pid })
			if i >= 0 && cfg.Players[i].IsBot {
				seat = i
				break
			}
		}

		if seat < 0 {
			return nil
		}

		pid := cfg.Players[seat].ID

		d, err := s.decide(e.game, pid, seat, e.botRNG(seat))
		if err == nil {
			err = e.game.Submit(pid, d)
		}

		if err != nil {
			s.log.Error("bot failed", "game", e.id, "player", pid, "phase", e.game.PhaseID(), "err", err)
			return newError(StatusInternal, "err.botFailed")
		}
	}

	return nil
}

func (s *Service) lookup(gameID string) *gameEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.games[gameID]
}

// authorize resolves the game and the player behind token.
func (s *Service) authorize(gameID, token string) (*gameEntry, string, error) {
	e := s.lookup(gameID)
	if e == nil {
		return nil, "", newError(StatusNotFound, "err.gameNotFound")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, "", newError(StatusUnauthorized, "err.noToken")
	}

	pid, ok := e.tokens[token] // immutable after creation, no lock needed
	if !ok {
		return nil, "", newError(StatusUnauthorized, "err.badToken")
	}

	return e, pid, nil
}

// CleanName normalises a player name: collapses whitespace, defaults to "Du", caps the length.
func CleanName(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return defaultName
	}

	if utf8.RuneCountInString(name) > maxNameLen {
		name = strings.TrimSpace(string([]rune(name)[:maxNameLen]))
	}

	return name
}

// randomHex returns n random bytes as lowercase hex.
func randomHex(n int) string {
	b := make([]byte, n)

	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err)) // never fails on supported platforms
	}

	return hex.EncodeToString(b)
}

// ValidGameID reports whether id looks like a game ID (non-empty hex, at most 64 chars).
func ValidGameID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}

	_, err := hex.DecodeString(id)

	return err == nil
}
