package gameservice

import (
	"encoding/json"
	"errors"
	mrand "math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"ruderspiel/internal/engine"
)

type memStore struct{ saved map[string]Snapshot }

func (m *memStore) Save(s Snapshot) error {
	m.saved[s.Config.GameID] = s
	return nil
}

func newService(t *testing.T, store Store) *Service {
	t.Helper()

	s, err := New(Options{Decider: SimpleDecider, Store: store})
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func create(t *testing.T, s *Service, seed uint64) CreateResponse {
	t.Helper()

	bots := 2

	cr, err := s.Create(CreateRequest{PlayerName: " Vera ", Seed: &seed, Bots: &bots})
	if err != nil {
		t.Fatal(err)
	}

	return cr
}

func humanDecision(t *testing.T, s *Service, gameID string, rng *mrand.Rand) engine.Decision {
	t.Helper()

	var d engine.Decision

	err := s.WithGame(gameID, func(g *engine.Game) error {
		var err error
		d, err = SimpleDecider(g, "p1", 0, rng)

		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	return d
}

func wantStatus(t *testing.T, err error, status int) {
	t.Helper()

	var se *Error
	if !errors.As(err, &se) {
		t.Fatalf("want *Error with status %d, got %v", status, err)
	}

	if se.Status != status || se.Msg.Key == "" {
		t.Fatalf("want status %d with message, got %d %q", status, se.Status, se.Msg)
	}
}

func TestFullGameViaService(t *testing.T) {
	t.Parallel()

	store := &memStore{saved: map[string]Snapshot{}}
	s := newService(t, store)

	cr := create(t, s, 7)
	if cr.PlayerID != "p1" || len(cr.Token) != 32 || cr.View.Players[0].Name != "Vera" {
		t.Fatalf("unexpected create response: %+v", cr)
	}

	rng := mrand.New(mrand.NewPCG(7, 1))

	view := cr.View
	for steps := 0; view.Phase.Kind != engine.PhaseGameOver; steps++ {
		if steps > 2000 {
			t.Fatalf("no game over after %d decisions (phase %s)", steps, view.Phase.ID)
		}

		if view.Phase.Waiting {
			t.Fatalf("view waits on others in phase %s", view.Phase.ID)
		}

		next, err := s.Submit(cr.GameID, cr.Token, humanDecision(t, s, cr.GameID, rng))
		if err != nil {
			t.Fatalf("submit in phase %s: %v", view.Phase.ID, err)
		}

		view = next
	}

	if len(view.Winners) == 0 {
		t.Fatal("game over without winners")
	}

	lr, err := s.Log(cr.GameID, cr.Token)
	if err != nil || len(lr.Log) == 0 || len(lr.Results) == 0 {
		t.Fatalf("log: %v, %d entries, %d results", err, len(lr.Log), len(lr.Results))
	}

	_, err = s.Submit(cr.GameID, cr.Token, engine.Decision{PhaseID: view.Phase.ID, Kind: engine.PhaseAck})
	wantStatus(t, err, StatusBadRequest)

	snap, ok := store.saved[cr.GameID]
	if !ok || len(snap.Decisions) == 0 || snap.Tokens[cr.Token] != "p1" {
		t.Fatalf("store not updated: %+v", snap.Config)
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()

	s := newService(t, nil)
	cr := create(t, s, 3)

	_, err := s.View(cr.GameID, "")
	wantStatus(t, err, StatusUnauthorized)
	_, err = s.View(cr.GameID, strings.Repeat("0", 32))
	wantStatus(t, err, StatusUnauthorized)
	_, err = s.View("deadbeef", cr.Token)
	wantStatus(t, err, StatusNotFound)
	_, err = s.Submit(cr.GameID, cr.Token, engine.Decision{PhaseID: "x", Kind: cr.View.Phase.Kind})
	wantStatus(t, err, StatusConflict)

	wrongKind := engine.PhaseDraft
	if cr.View.Phase.Kind == engine.PhaseDraft {
		wrongKind = engine.PhaseAck
	}

	_, err = s.Submit(cr.GameID, cr.Token, engine.Decision{PhaseID: cr.View.Phase.ID, Kind: wrongKind})
	wantStatus(t, err, StatusBadRequest)

	nine := 9
	_, err = s.Create(CreateRequest{Bots: &nine})
	wantStatus(t, err, StatusBadRequest)

	c, err := s.Catalog()
	if err != nil || len(c.Rowers) == 0 {
		t.Fatalf("catalog: %v", err)
	}

	got := AsError(errBoom)
	if got.Status != StatusInternal {
		t.Fatalf("AsError(plain) = %d", got.Status)
	}
}

var errBoom = errors.New("boom")

func TestExportImportRoundTrip(t *testing.T) {
	t.Parallel()

	s1 := newService(t, nil)
	cr := create(t, s1, 11)

	rng := mrand.New(mrand.NewPCG(11, 1))
	for range 25 {
		_, err := s1.Submit(cr.GameID, cr.Token, humanDecision(t, s1, cr.GameID, rng))
		if err != nil {
			t.Fatal(err)
		}
	}

	before, _ := s1.View(cr.GameID, cr.Token)

	snap, err := s1.Export(cr.GameID)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(snap) // through JSON, like localStorage
	if err != nil {
		t.Fatal(err)
	}

	var back Snapshot

	err = json.Unmarshal(raw, &back)
	if err != nil {
		t.Fatal(err)
	}

	s2 := newService(t, nil)

	id, err := s2.Import(back)
	if err != nil || id != cr.GameID {
		t.Fatalf("import: %q, %v", id, err)
	}

	after, err := s2.View(cr.GameID, cr.Token)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatalf("view differs after import (phase %s vs %s)", before.Phase.ID, after.Phase.ID)
	}
	// The imported game keeps playing.
	_, err = s2.Submit(cr.GameID, cr.Token, humanDecision(t, s2, cr.GameID, rng))
	if err != nil {
		t.Fatal(err)
	}

	_, err = s2.Import(Snapshot{Version: 99})
	wantStatus(t, err, StatusBadRequest)

	old := back
	old.Version = 1
	_, err = s2.Import(old)
	wantStatus(t, err, StatusBadRequest)

	if !strings.Contains(err.Error(), "err.snapshotOld") {
		t.Errorf("old save: %v", err)
	}

	bad := back
	bad.Config.GameID = "not-hex"
	_, err = s2.Import(bad)
	wantStatus(t, err, StatusBadRequest)
	_, err = s2.Export("deadbeef")
	wantStatus(t, err, StatusNotFound)
}

func TestCleanName(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                              "Du",
		"   ":                           "Du",
		"  Vera  ":                      "Vera",
		"Anna   Lena":                   "Anna Lena",
		"Ruderin mit sehr langem Namen": "Ruderin mit sehr langem",
		"ÄÖÜäöüßÄÖÜäöüßÄÖÜäöüßÄÖÜ": "ÄÖÜäöüßÄÖÜäöüßÄÖÜäöüßÄÖÜ",
	}
	for in, want := range cases {
		if got := CleanName(in); got != want {
			t.Errorf("CleanName(%q) = %q, want %q", in, got, want)
		}
	}
}
