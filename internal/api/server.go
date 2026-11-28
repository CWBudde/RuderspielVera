// Package api is the HTTP layer of Ruderspiel. The game logic (registry, tokens, bot loop, error
// mapping) lives in internal/gameservice; this package adds routing, JSON bodies, bearer tokens
// and file persistence. The contract is docs/api.md; JSON shapes come from internal/engine/contract.go.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"ruderspiel/internal/engine"
	"ruderspiel/internal/gameservice"
)

// Decider makes the decision for a bot seat in the current phase (see gameservice.Decider).
type Decider = gameservice.Decider

// SimpleDecider is the minimal always-legal bot (see gameservice.SimpleDecider).
//
//nolint:gochecknoglobals // alias of a function value, never reassigned
var SimpleDecider Decider = gameservice.SimpleDecider

// Options configures a Server.
type Options struct {
	DataDir string       // persistence directory; empty = in-memory only
	Decider Decider      // bot brain (required)
	Logger  *slog.Logger // defaults to slog.Default()

	// TurnTimeout and LobbyMaxAge configure the service janitor (see gameservice.Options); the
	// janitor runs only if one of them is > 0. Close stops it.
	TurnTimeout time.Duration
	LobbyMaxAge time.Duration

	// CreateRate limits POST /api/games, /api/lobbies and /api/lobbies/{code}/join per client IP
	// to CreateRate requests per minute with bursts of CreateBurst (default: CreateRate).
	// 0 disables the limit.
	CreateRate  float64
	CreateBurst int
}

// Server is the HTTP front of a gameservice.Service.
type Server struct {
	svc     *gameservice.Service
	log     *slog.Logger
	limiter *ipLimiter // nil = no rate limit
}

const maxBodySize = 64 << 10

// ErrNoDecider is returned by New when Options.Decider is nil.
var ErrNoDecider = errors.New("api: Decider is required")

// New creates a server and loads all persisted games from opts.DataDir.
func New(opts Options) (*Server, error) {
	if opts.Decider == nil {
		return nil, ErrNoDecider
	}

	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	var store gameservice.Store
	if opts.DataDir != "" {
		store = fileStore{dir: opts.DataDir}
	}

	svc, err := gameservice.New(gameservice.Options{
		Decider: opts.Decider, Store: store, Logger: opts.Logger,
		TurnTimeout: opts.TurnTimeout, LobbyMaxAge: opts.LobbyMaxAge,
	})
	if err != nil {
		return nil, fmt.Errorf("api: %w", err)
	}

	srv := &Server{svc: svc, log: opts.Logger}

	if opts.CreateRate > 0 {
		burst := opts.CreateBurst
		if burst <= 0 {
			burst = max(1, int(opts.CreateRate))
		}

		srv.limiter = newIPLimiter(opts.CreateRate/60, burst, time.Now)
	}

	err = srv.loadAll(opts.DataDir)
	if err != nil {
		return nil, err
	}

	if opts.TurnTimeout > 0 || opts.LobbyMaxAge > 0 {
		svc.StartJanitor(0)
	}

	return srv, nil
}

// Close stops the background janitor. The server must not be used afterwards.
func (s *Server) Close() { s.svc.Close() }

// Service returns the underlying game service.
func (s *Server) Service() *gameservice.Service { return s.svc }

// Handler returns the full HTTP handler. static (optional) serves everything outside /api/.
func (s *Server) Handler(static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/games", s.rateLimited(s.handleCreate))
	mux.HandleFunc("GET /api/games/{id}", s.handleView)
	mux.HandleFunc("POST /api/games/{id}/decisions", s.handleDecision)
	mux.HandleFunc("GET /api/games/{id}/log", s.handleLog)
	mux.HandleFunc("GET /api/catalog", s.handleCatalog)
	mux.HandleFunc("POST /api/lobbies", s.rateLimited(s.handleCreateLobby))
	mux.HandleFunc("POST /api/lobbies/{code}/join", s.rateLimited(s.handleJoinLobby))
	mux.HandleFunc("GET /api/games/{id}/lobby", s.handleLobby)
	mux.HandleFunc("POST /api/games/{id}/start", s.handleStart)
	mux.HandleFunc("GET /api/games/{id}/events", s.handleEvents)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, gameservice.NewError(http.StatusNotFound, "err.unknownEndpoint"))
	})

	if static != nil {
		mux.Handle("/", static)
	}

	return s.logRequests(cors(recoverPanics(s.log, mux)))
}

// ---------- handlers ----------

type (
	createRequest  = gameservice.CreateRequest
	createResponse = gameservice.CreateResponse
	logResponse    = gameservice.LogResponse
)

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !decodeBody(w, r, &req) {
		return
	}

	resp, err := s.svc.Create(req)
	reply(w, http.StatusCreated, resp, err)
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	v, err := s.svc.View(r.PathValue("id"), bearer(r))
	reply(w, http.StatusOK, v, err)
}

func (s *Server) handleDecision(w http.ResponseWriter, r *http.Request) {
	id, token := r.PathValue("id"), bearer(r)
	// Check game and token before reading the body, so a bad token is a 401 even with a bad body.
	err := s.svc.CheckToken(id, token)
	if err != nil {
		reply(w, 0, nil, err)
		return
	}

	var d engine.Decision
	if !decodeBody(w, r, &d) {
		return
	}

	v, err := s.svc.Submit(id, token, d)
	reply(w, http.StatusOK, v, err)
}

func (s *Server) handleLog(w http.ResponseWriter, r *http.Request) {
	lr, err := s.svc.Log(r.PathValue("id"), bearer(r))
	reply(w, http.StatusOK, lr, err)
}

func (s *Server) handleCatalog(w http.ResponseWriter, _ *http.Request) {
	c, err := s.svc.Catalog()
	reply(w, http.StatusOK, c, err)
}

func (s *Server) handleCreateLobby(w http.ResponseWriter, r *http.Request) {
	var req gameservice.CreateLobbyRequest
	if !decodeBody(w, r, &req) {
		return
	}

	resp, err := s.svc.CreateLobby(req)
	reply(w, http.StatusCreated, resp, err)
}

func (s *Server) handleJoinLobby(w http.ResponseWriter, r *http.Request) {
	var req gameservice.JoinLobbyRequest
	if !decodeBody(w, r, &req) {
		return
	}

	resp, err := s.svc.JoinLobby(r.PathValue("code"), req)
	reply(w, http.StatusCreated, resp, err)
}

func (s *Server) handleLobby(w http.ResponseWriter, r *http.Request) {
	v, err := s.svc.Lobby(r.PathValue("id"), bearer(r))
	reply(w, http.StatusOK, v, err)
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	v, err := s.svc.StartLobby(r.PathValue("id"), bearer(r))
	reply(w, http.StatusOK, v, err)
}

// bearer extracts the token from "Authorization: Bearer <token>" ("" if missing).
func bearer(r *http.Request) string {
	token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !found {
		return ""
	}

	return strings.TrimSpace(token)
}

// ---------- JSON helpers ----------

// reply writes v with status ok, or the service error as {"error": key, "msg": …} with its status.
func reply(w http.ResponseWriter, ok int, v any, err error) {
	if err != nil {
		writeError(w, gameservice.AsError(err))

		return
	}

	writeJSON(w, ok, v)
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		writeError(w, gameservice.NewError(http.StatusBadRequest, "err.badRequestBody"))
		return false
	}

	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		// Headers are already sent; all we can do is note the truncated body.
		slog.Default().Warn("encode response", "err", err)
	}
}

// writeError sends {"error": "<message key>", "msg": {"key": …, "args": …}}; the UI translates msg.
func writeError(w http.ResponseWriter, se *gameservice.Error) {
	writeJSON(w, se.Status, ErrorBody{Error: se.Msg.Key, Msg: se.Msg})
}

// ErrorBody is the JSON body of every failed API call.
type ErrorBody struct {
	Error string     `json:"error"` // message key, e.g. "err.gameNotFound"
	Msg   engine.Msg `json:"msg"`
}
