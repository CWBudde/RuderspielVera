package gameservice

import (
	"crypto/rand"
	"fmt"
	mrand "math/rand/v2"
	"slices"
	"strings"

	"ruderspiel/internal/engine"
)

// Multiplayer lobby: the host opens a game with 2..4 seats and gets a short join code. Others
// join with the code while the lobby is open; each joiner gets the next seat (p2, p3, …) and a
// token of their own. The host starts the game; seats nobody took are filled with bots. Until
// the start there is no engine.Game, only the seed and the joined players (gameEntry.cfg).

// Lobby is the multiplayer part of a game entry (persisted in the snapshot).
type Lobby struct {
	Code    string `json:"code"`    // join code, e.g. "K7QM2X"
	Seats   int    `json:"seats"`   // total seats (humans + bots), 2..4
	Started bool   `json:"started"` // the host started the game
}

const (
	minSeats   = 2
	maxSeats   = 4
	codeLength = 6
	// codeAlphabet leaves out look-alikes (0/O, 1/I/L) so a code can be read out at the table.
	codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	hostID       = "p1"
)

// CreateLobbyRequest is the body of "open a multiplayer lobby".
type CreateLobbyRequest struct {
	PlayerName string  `json:"playerName"`
	Seats      int     `json:"seats"`
	Seed       *uint64 `json:"seed,omitempty"`
}

// JoinLobbyRequest is the body of "join a lobby".
type JoinLobbyRequest struct {
	PlayerName string `json:"playerName"`
}

// LobbyResponse answers create and join: the caller's seat, token and the lobby.
type LobbyResponse struct {
	GameID   string    `json:"gameId"`
	PlayerID string    `json:"playerId"`
	Token    string    `json:"token"`
	Lobby    LobbyView `json:"lobby"`
}

// LobbyView is the lobby as one player sees it.
type LobbyView struct {
	GameID  string        `json:"gameId"`
	Code    string        `json:"code"`
	Seats   int           `json:"seats"`
	Players []LobbyPlayer `json:"players"` // humans in join order; after the start also the bots
	You     string        `json:"you"`
	Host    bool          `json:"host"` // the viewer may start the game
	Started bool          `json:"started"`
}

// LobbyPlayer is one taken seat.
type LobbyPlayer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	IsBot bool   `json:"isBot"`
}

// CreateLobby opens a lobby with the caller as host (p1).
func (s *Service) CreateLobby(req CreateLobbyRequest) (LobbyResponse, error) {
	if req.Seats < minSeats || req.Seats > maxSeats {
		return LobbyResponse{}, newError(StatusBadRequest, "err.seatCount", "min", minSeats, "max", maxSeats)
	}

	seed := mrand.Uint64()
	if req.Seed != nil {
		seed = *req.Seed
	}

	id := randomHex(8)
	token := randomHex(16)

	e := newGameEntry(id, nil, map[string]string{token: hostID}, s.now())
	e.cfg = engine.Config{GameID: id, Seed: seed, Players: []engine.PlayerConfig{
		{ID: hostID, Name: CleanName(req.PlayerName)},
	}}

	e.mu.Lock()
	defer e.mu.Unlock()

	s.mu.Lock()

	code := randomCode()
	for s.codes[code] != "" {
		code = randomCode()
	}

	e.lobby = &Lobby{Code: code, Seats: req.Seats}
	s.games[id] = e
	s.codes[code] = id
	s.mu.Unlock()

	err := s.save(e)
	if err != nil {
		return LobbyResponse{}, err
	}

	s.log.Info("lobby created", "game", id, "code", code, "seats", req.Seats)

	return LobbyResponse{GameID: id, PlayerID: hostID, Token: token, Lobby: lobbyView(e, hostID)}, nil
}

// JoinLobby takes the next free seat in the lobby with the given code.
func (s *Service) JoinLobby(code string, req JoinLobbyRequest) (LobbyResponse, error) {
	code = NormalizeCode(code)

	s.mu.RLock()
	e := s.games[s.codes[code]]
	s.mu.RUnlock()

	if e == nil {
		return LobbyResponse{}, newError(StatusNotFound, "err.lobbyNotFound")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	switch {
	case e.lobby.Started:
		return LobbyResponse{}, newError(StatusConflict, "err.lobbyStarted")
	case len(e.cfg.Players) >= e.lobby.Seats:
		return LobbyResponse{}, newError(StatusConflict, "err.lobbyFull")
	}

	pid := fmt.Sprintf("p%d", len(e.cfg.Players)+1)
	token := randomHex(16)

	e.cfg.Players = append(e.cfg.Players, engine.PlayerConfig{ID: pid, Name: CleanName(req.PlayerName)})

	e.tokMu.Lock()
	e.tokens[token] = pid
	e.tokMu.Unlock()

	err := s.save(e)
	if err != nil {
		// Roll back so memory matches the last saved state.
		e.cfg.Players = e.cfg.Players[:len(e.cfg.Players)-1]

		e.tokMu.Lock()
		delete(e.tokens, token)
		e.tokMu.Unlock()

		return LobbyResponse{}, err
	}

	s.log.Info("lobby joined", "game", e.id, "player", pid)

	return LobbyResponse{GameID: e.id, PlayerID: pid, Token: token, Lobby: lobbyView(e, pid)}, nil
}

// Lobby returns the lobby as seen by the token's player (also after the start).
func (s *Service) Lobby(gameID, token string) (LobbyView, error) {
	e, pid, err := s.authorize(gameID, token)
	if err != nil {
		return LobbyView{}, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.lobby == nil {
		return LobbyView{}, newError(StatusNotFound, "err.lobbyNotFound")
	}

	return lobbyView(e, pid), nil
}

// StartLobby (host only) fills the free seats with bots, deals and runs the bots until a human
// is on turn. Starting twice is an error.
func (s *Service) StartLobby(gameID, token string) (engine.GameView, error) {
	e, pid, err := s.authorize(gameID, token)
	if err != nil {
		return engine.GameView{}, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	switch {
	case e.lobby == nil:
		return engine.GameView{}, newError(StatusNotFound, "err.lobbyNotFound")
	case pid != hostID:
		return engine.GameView{}, newError(StatusForbidden, "err.notHost")
	case e.lobby.Started:
		return engine.GameView{}, newError(StatusConflict, "err.lobbyStarted")
	}

	cfg := e.cfg
	cfg.Players = slices.Clone(e.cfg.Players)

	for i := 0; len(cfg.Players) < e.lobby.Seats; i++ {
		cfg.Players = append(cfg.Players, engine.PlayerConfig{
			ID: fmt.Sprintf("p%d", len(cfg.Players)+1), Name: botNames[i], IsBot: true,
		})
	}

	g, err := engine.New(cfg)
	if err != nil {
		s.log.Error("engine.New failed", "game", e.id, "err", err)
		return engine.GameView{}, newError(StatusInternal, "err.createFailed")
	}

	e.game = g
	e.lobby.Started = true

	botErr := s.runBots(e)

	err = s.save(e)
	if err != nil {
		return engine.GameView{}, err
	}

	if botErr != nil {
		return engine.GameView{}, botErr
	}

	s.log.Info("lobby started", "game", e.id, "players", len(cfg.Players))

	return e.game.View(pid), nil
}

// lobbyView builds the lobby view for player pid. Caller holds e.mu.
func lobbyView(e *gameEntry, pid string) LobbyView {
	players := e.cfg.Players
	if e.game != nil {
		players = e.game.Config().Players
	}

	v := LobbyView{
		GameID:  e.id,
		Code:    e.lobby.Code,
		Seats:   e.lobby.Seats,
		Players: make([]LobbyPlayer, 0, len(players)),
		You:     pid,
		Host:    pid == hostID,
		Started: e.lobby.Started,
	}
	for _, p := range players {
		v.Players = append(v.Players, LobbyPlayer{ID: p.ID, Name: p.Name, IsBot: p.IsBot})
	}

	return v
}

// validLobby checks a persisted open lobby: seat count, code and joined humans.
func validLobby(l *Lobby, cfg engine.Config) bool {
	if l.Seats < minSeats || l.Seats > maxSeats || NormalizeCode(l.Code) != l.Code || len(l.Code) != codeLength {
		return false
	}

	return len(cfg.Players) >= 1 && len(cfg.Players) <= l.Seats
}

// NormalizeCode upper-cases a join code and drops spaces and dashes ("k7q m2x" -> "K7QM2X").
func NormalizeCode(code string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == ' ' || r == '-':
			return -1
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		default:
			return r
		}
	}, code)
}

// randomCode returns a fresh join code (31^6 ≈ 890 million combinations).
func randomCode() string {
	b := make([]byte, codeLength)

	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err)) // never fails on supported platforms
	}

	for i, x := range b {
		b[i] = codeAlphabet[int(x)%len(codeAlphabet)] // modulo bias is irrelevant for join codes
	}

	return string(b)
}
