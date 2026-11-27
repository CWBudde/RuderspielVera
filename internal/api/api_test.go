package api

import (
	"bytes"
	"encoding/json"
	"io"
	mrand "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ruderspiel/internal/engine"
)

type client struct {
	t   *testing.T
	srv *httptest.Server
}

func newTestServer(t *testing.T, dataDir string) (*Server, *client) {
	t.Helper()

	s, err := New(Options{DataDir: dataDir, Decider: SimpleDecider})
	if err != nil {
		t.Fatal(err)
	}

	hs := httptest.NewServer(s.Handler(nil))
	t.Cleanup(hs.Close)

	return s, &client{t: t, srv: hs}
}

// do sends a request and decodes the JSON response into out (if non-nil); returns the status.
func (c *client) do(method, path, token string, body, out any) int {
	c.t.Helper()

	var rd io.Reader

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}

		rd = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(c.t.Context(), method, c.srv.URL+path, rd)
	if err != nil {
		c.t.Fatal(err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatal(err)
	}

	if out != nil {
		err := json.Unmarshal(raw, out)
		if err != nil {
			c.t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
		}
	}

	return res.StatusCode
}

func (c *client) create(seed uint64) createResponse {
	c.t.Helper()

	var cr createResponse

	body := map[string]any{"playerName": "  Vera  ", "seed": seed, "bots": 2}
	if st := c.do("POST", "/api/games", "", body, &cr); st != http.StatusCreated {
		c.t.Fatalf("create: status %d", st)
	}

	return cr
}

// humanDecision builds a legal decision for the human from the server's live game
// (under the game lock), using only public engine API.
func humanDecision(t *testing.T, s *Server, gameID, pid string, rng *mrand.Rand) engine.Decision {
	t.Helper()
	var d engine.Decision

	err := s.Service().WithGame(gameID, func(g *engine.Game) error {
		var err error
		d, err = SimpleDecider(g, pid, 0, rng)

		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	return d
}

func assertWaitsOnHuman(t *testing.T, v engine.GameView) {
	t.Helper()

	if v.Phase.Kind == engine.PhaseGameOver {
		return
	}

	if v.Phase.Waiting {
		t.Fatalf("view waits on others in phase %s (%s), want the human to be on turn", v.Phase.ID, v.Phase.Kind)
	}
}

func TestCreateGameWaitsOnHuman(t *testing.T) {
	t.Parallel()

	_, cl := newTestServer(t, "")

	cr := cl.create(42)
	if cr.GameID == "" || cr.PlayerID != "p1" || len(cr.Token) != 32 {
		t.Fatalf("unexpected create response: %+v", cr)
	}

	if len(cr.View.Players) != 3 || cr.View.Players[0].Name != "Vera" || !cr.View.Players[1].IsBot {
		t.Fatalf("players: %+v", cr.View.Players)
	}

	if cr.View.Players[1].Name != "Ruderclub Ihme" || cr.View.Players[2].Name != "Leine-Ruderer" {
		t.Fatalf("bot names: %q, %q", cr.View.Players[1].Name, cr.View.Players[2].Name)
	}

	assertWaitsOnHuman(t, cr.View)

	var v engine.GameView
	if st := cl.do("GET", "/api/games/"+cr.GameID, cr.Token, nil, &v); st != http.StatusOK {
		t.Fatalf("view: status %d", st)
	}

	if v.Phase.ID != cr.View.Phase.ID || v.You != "p1" {
		t.Fatalf("view mismatch: %s vs %s", v.Phase.ID, cr.View.Phase.ID)
	}
}

func TestPlayFullSeason(t *testing.T) {
	t.Parallel()

	s, cl := newTestServer(t, "")
	cr := cl.create(7)
	rng := mrand.New(mrand.NewPCG(7, 1))

	view := cr.View
	for steps := 0; view.Phase.Kind != engine.PhaseGameOver; steps++ {
		if steps > 2000 {
			t.Fatalf("no game over after %d decisions (phase %s)", steps, view.Phase.ID)
		}

		assertWaitsOnHuman(t, view)
		d := humanDecision(t, s, cr.GameID, "p1", rng)

		var next engine.GameView
		if st := cl.do("POST", "/api/games/"+cr.GameID+"/decisions", cr.Token, d, &next); st != http.StatusOK {
			t.Fatalf("decision in phase %s (%s): status %d", d.PhaseID, d.Kind, st)
		}

		view = next
	}

	if len(view.Winners) == 0 {
		t.Fatal("game over without winners")
	}

	var lr logResponse
	if st := cl.do("GET", "/api/games/"+cr.GameID+"/log", cr.Token, nil, &lr); st != http.StatusOK {
		t.Fatalf("log: status %d", st)
	}

	if len(lr.Log) == 0 || len(lr.Results) == 0 {
		t.Fatalf("log %d entries, %d results", len(lr.Log), len(lr.Results))
	}
	// Further decisions after game over are rejected with a German message.
	var er map[string]string

	ack := engine.Decision{PhaseID: view.Phase.ID, Kind: engine.PhaseAck}
	if st := cl.do("POST", "/api/games/"+cr.GameID+"/decisions", cr.Token, ack, &er); st != http.StatusBadRequest ||
		er["error"] == "" {
		t.Fatalf("after game over: status %d, %v", st, er)
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()

	_, cl := newTestServer(t, "")
	cr := cl.create(3)
	base := "/api/games/" + cr.GameID
	var er map[string]string

	if st := cl.do("GET", base, "", nil, &er); st != http.StatusUnauthorized || er["error"] == "" {
		t.Errorf("missing token: status %d, %v", st, er)
	}

	if st := cl.do("GET", base, strings.Repeat("0", 32), nil, &er); st != http.StatusUnauthorized {
		t.Errorf("bad token: status %d", st)
	}

	if st := cl.do("GET", "/api/games/deadbeef", cr.Token, nil, &er); st != http.StatusNotFound {
		t.Errorf("unknown game: status %d", st)
	}

	stale := engine.Decision{PhaseID: "x-does-not-exist", Kind: cr.View.Phase.Kind}
	if st := cl.do("POST", base+"/decisions", cr.Token, stale, &er); st != http.StatusConflict || er["error"] == "" {
		t.Errorf("stale phase: status %d, %v", st, er)
	}

	wrongKind := engine.PhaseDraft
	if cr.View.Phase.Kind == engine.PhaseDraft {
		wrongKind = engine.PhaseAck
	}

	er = nil

	bad := engine.Decision{PhaseID: cr.View.Phase.ID, Kind: wrongKind}
	if st := cl.do("POST", base+"/decisions", cr.Token, bad, &er); st != http.StatusBadRequest ||
		!strings.Contains(er["error"], "Entscheidungstyp") {
		t.Errorf("invalid decision: status %d, %v", st, er)
	}

	if st := cl.do("POST", "/api/games", "", map[string]any{"bots": 9}, &er); st != http.StatusBadRequest {
		t.Errorf("too many bots: status %d", st)
	}

	if st := cl.do("GET", "/api/nope", "", nil, &er); st != http.StatusNotFound {
		t.Errorf("unknown endpoint: status %d", st)
	}

	var cat engine.CatalogView
	if st := cl.do("GET", "/api/catalog", "", nil, &cat); st != http.StatusOK || len(cat.Rowers) == 0 {
		t.Errorf("catalog: status %d, %d rowers", st, len(cat.Rowers))
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s1, c1 := newTestServer(t, dir)
	cr := c1.create(11)

	rng := mrand.New(mrand.NewPCG(11, 1))
	for range 25 {
		d := humanDecision(t, s1, cr.GameID, "p1", rng)
		if st := c1.do("POST", "/api/games/"+cr.GameID+"/decisions", cr.Token, d, nil); st != http.StatusOK {
			t.Fatalf("decision: status %d", st)
		}
	}
	var before engine.GameView
	c1.do("GET", "/api/games/"+cr.GameID, cr.Token, nil, &before)

	_, err := os.Stat(filepath.Join(dir, cr.GameID+".json"))
	if err != nil {
		t.Fatalf("game file missing: %v", err)
	}

	if m, _ := filepath.Glob(filepath.Join(dir, "*.tmp")); len(m) > 0 {
		t.Fatalf("leftover temp files: %v", m)
	}

	_, c2 := newTestServer(t, dir)

	var after engine.GameView
	if st := c2.do("GET", "/api/games/"+cr.GameID, cr.Token, nil, &after); st != http.StatusOK {
		t.Fatalf("view after reload: status %d", st)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatalf("view differs after reload (phase %s vs %s)", before.Phase.ID, after.Phase.ID)
	}
}
