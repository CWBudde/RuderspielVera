// Package api is the HTTP layer of Ruderspiel: game creation, views, decisions and persistence.
// The contract is docs/api.md; JSON shapes come from internal/engine/contract.go.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	mrand "math/rand/v2"
	"net/http"
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

// Options configures a Server.
type Options struct {
	DataDir string       // persistence directory; empty = in-memory only
	Decider Decider      // bot brain (required)
	Logger  *slog.Logger // defaults to slog.Default()
}

// Server holds all games. The maps are guarded by mu; each game has its own mutex.
type Server struct {
	mu      sync.RWMutex
	games   map[string]*gameEntry
	tokens  map[string]*gameEntry
	dataDir string
	decide  Decider
	log     *slog.Logger
}

// Bot display names per bot index (seat 1..3).
var botNames = []string{"Ruderclub Ihme", "Leine-Ruderer", "RV Maschsee"}

const (
	maxNameLen  = 24
	defaultName = "Du"
	defaultBots = 2
	maxBodySize = 64 << 10
	maxBotSteps = 10000
)

// New creates a server and loads all persisted games from opts.DataDir.
func New(opts Options) (*Server, error) {
	if opts.Decider == nil {
		return nil, errors.New("api: Decider is required")
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	s := &Server{
		games:   map[string]*gameEntry{},
		tokens:  map[string]*gameEntry{},
		dataDir: opts.DataDir,
		decide:  opts.Decider,
		log:     opts.Logger,
	}
	if err := s.loadAll(); err != nil {
		return nil, err
	}
	return s, nil
}

// Handler returns the full HTTP handler. static (optional) serves everything outside /api/.
func (s *Server) Handler(static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/games", s.handleCreate)
	mux.HandleFunc("GET /api/games/{id}", s.handleView)
	mux.HandleFunc("POST /api/games/{id}/decisions", s.handleDecision)
	mux.HandleFunc("GET /api/games/{id}/log", s.handleLog)
	mux.HandleFunc("GET /api/catalog", s.handleCatalog)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "Unbekannter Endpunkt.")
	})
	if static != nil {
		mux.Handle("/", static)
	}
	return s.logRequests(cors(recoverPanics(s.log, mux)))
}

// ---------- handlers ----------

type createRequest struct {
	PlayerName string  `json:"playerName"`
	Seed       *uint64 `json:"seed,omitempty"`
	Bots       *int    `json:"bots,omitempty"`
}

type createResponse struct {
	GameID   string          `json:"gameId"`
	PlayerID string          `json:"playerId"`
	Token    string          `json:"token"`
	View     engine.GameView `json:"view"`
}

type logResponse struct {
	Log     []engine.LogEntry          `json:"log"`
	Results []engine.CompetitionResult `json:"results"`
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !decodeBody(w, r, &req) {
		return
	}
	bots := defaultBots
	if req.Bots != nil {
		bots = *req.Bots
	}
	if bots < 1 || bots > len(botNames) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Es sind 1 bis %d Bots möglich.", len(botNames)))
		return
	}
	seed := mrand.Uint64()
	if req.Seed != nil {
		seed = *req.Seed
	}
	id := randomHex(8)
	cfg := engine.Config{GameID: id, Seed: seed, Players: []engine.PlayerConfig{
		{ID: "p1", Name: cleanName(req.PlayerName)},
	}}
	for i := range bots {
		cfg.Players = append(cfg.Players, engine.PlayerConfig{
			ID: fmt.Sprintf("p%d", i+2), Name: botNames[i], IsBot: true,
		})
	}
	g, err := engine.New(cfg)
	if err != nil {
		s.log.Error("engine.New failed", "err", err)
		writeError(w, http.StatusInternalServerError, "Das Spiel konnte nicht erstellt werden.")
		return
	}
	token := randomHex(16)
	e := newGameEntry(id, g, map[string]string{token: "p1"}, time.Now().UTC())

	e.mu.Lock()
	defer e.mu.Unlock()
	botErr := s.runBots(e)
	if err := s.save(e); err != nil {
		s.log.Error("save failed", "game", id, "err", err)
		writeError(w, http.StatusInternalServerError, "Das Spiel konnte nicht gespeichert werden.")
		return
	}
	s.mu.Lock()
	s.games[id] = e
	s.tokens[token] = e
	s.mu.Unlock()
	if botErr != nil {
		writeError(w, http.StatusInternalServerError, "Ein Bot konnte nicht entscheiden.")
		return
	}
	s.log.Info("game created", "game", id, "seed", seed, "bots", bots)
	writeJSON(w, http.StatusCreated, createResponse{GameID: id, PlayerID: "p1", Token: token, View: g.View("p1")})
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	e, pid, ok := s.authorize(w, r)
	if !ok {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	writeJSON(w, http.StatusOK, e.game.View(pid))
}

func (s *Server) handleDecision(w http.ResponseWriter, r *http.Request) {
	e, pid, ok := s.authorize(w, r)
	if !ok {
		return
	}
	var d engine.Decision
	if !decodeBody(w, r, &d) {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.game.Submit(pid, d); err != nil {
		var ve *engine.ValidationError
		switch {
		case errors.Is(err, engine.ErrStalePhase):
			writeError(w, http.StatusConflict, err.Error())
		case errors.As(err, &ve):
			writeError(w, http.StatusBadRequest, ve.Msg)
		default:
			s.log.Error("submit failed", "game", e.id, "player", pid, "err", err)
			writeError(w, http.StatusInternalServerError, "Die Entscheidung konnte nicht verarbeitet werden.")
		}
		return
	}
	botErr := s.runBots(e)
	if err := s.save(e); err != nil {
		s.log.Error("save failed", "game", e.id, "err", err)
		writeError(w, http.StatusInternalServerError, "Das Spiel konnte nicht gespeichert werden.")
		return
	}
	if botErr != nil {
		writeError(w, http.StatusInternalServerError, "Ein Bot konnte nicht entscheiden.")
		return
	}
	writeJSON(w, http.StatusOK, e.game.View(pid))
}

func (s *Server) handleLog(w http.ResponseWriter, r *http.Request) {
	e, _, ok := s.authorize(w, r)
	if !ok {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	resp := logResponse{Log: e.game.Log(), Results: e.game.Results()}
	if resp.Log == nil {
		resp.Log = []engine.LogEntry{}
	}
	if resp.Results == nil {
		resp.Results = []engine.CompetitionResult{}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	c, err := engine.Catalog()
	if err != nil {
		s.log.Error("catalog failed", "err", err)
		writeError(w, http.StatusInternalServerError, "Der Kartenkatalog konnte nicht geladen werden.")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ---------- game logic glue ----------

// runBots lets bots decide while the game is running and a pending player is a bot, so the view
// returned afterwards always waits on a human (or shows game over). Caller holds e.mu.
func (s *Server) runBots(e *gameEntry) error {
	cfg := e.game.Config()
	for steps := 0; !e.game.Over(); steps++ {
		if steps > maxBotSteps {
			return fmt.Errorf("game %s: bots did not hand back control after %d steps", e.id, maxBotSteps)
		}
		seat := -1
		for _, pid := range e.game.Pending() {
			if i := slices.IndexFunc(cfg.Players, func(p engine.PlayerConfig) bool { return p.ID == pid }); i >= 0 && cfg.Players[i].IsBot {
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
			return err
		}
	}
	return nil
}

// authorize resolves the game from the path and the player from the bearer token.
func (s *Server) authorize(w http.ResponseWriter, r *http.Request) (*gameEntry, string, bool) {
	id := r.PathValue("id")
	s.mu.RLock()
	e := s.games[id]
	s.mu.RUnlock()
	if e == nil {
		writeError(w, http.StatusNotFound, "Dieses Spiel gibt es nicht.")
		return nil, "", false
	}
	auth := r.Header.Get("Authorization")
	token, found := strings.CutPrefix(auth, "Bearer ")
	token = strings.TrimSpace(token)
	if !found || token == "" {
		writeError(w, http.StatusUnauthorized, "Anmeldung fehlt. Bitte starte ein neues Spiel.")
		return nil, "", false
	}
	pid, ok := e.tokens[token] // immutable after creation, no lock needed
	if !ok {
		writeError(w, http.StatusUnauthorized, "Ungültige Anmeldung für dieses Spiel.")
		return nil, "", false
	}
	return e, pid, true
}

func cleanName(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return defaultName
	}
	if utf8.RuneCountInString(name) > maxNameLen {
		name = strings.TrimSpace(string([]rune(name)[:maxNameLen]))
	}
	return name
}

// ---------- JSON helpers ----------

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage: Die Daten konnten nicht gelesen werden.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
