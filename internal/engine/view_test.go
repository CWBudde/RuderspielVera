package engine

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"testing"
)

// TestPhaseViewHints walks a full game and checks the explicit UI hints of the phase view:
// stage, new event cards, the Ereignis card, the draft turn and the race scoring hints.
func TestPhaseViewHints(t *testing.T) {
	g, err := New(threePlayers(7))
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(7, 77))
	seen := map[PhaseStage]bool{}
	sawDraft, sawEvents, sawRaces := false, false, false
	for steps := 0; !g.Over(); steps++ {
		if steps > 5000 {
			t.Fatal("no game over")
		}
		pv := g.CurrentPhase("p1")
		st := g.d.Calendar[g.station]
		seen[pv.Stage] = true
		switch pv.Stage {
		case StageEvent:
			for _, ev := range pv.NewEvents {
				if ev.RowerID == "" || ev.Name == "" {
					t.Fatalf("incomplete new event card %+v", ev)
				}
				sawEvents = true
			}
		case StageEreignis:
			if pv.Ereignis == nil || pv.Ereignis.Name == "" || pv.Ereignis.Description == "" {
				t.Fatalf("ereignis stage without card: %+v", pv.Ereignis)
			}
		case StageResult:
			if v := g.View("p1"); v.LastResult == nil || v.LastResult.StationIndex != g.station {
				t.Fatal("result stage without lastResult of this station")
			}
		}
		if pv.Kind == PhaseAck && pv.Stage == StageNone {
			t.Fatalf("ack phase at %s (%s) without stage", st.Name, st.Type)
		}
		if pv.Kind == PhaseDraft {
			if pv.DraftTurn == "" || pv.DraftTurn != g.Pending()[0] {
				t.Fatalf("draft turn %q, pending %v", pv.DraftTurn, g.Pending())
			}
			sawDraft = true
		}
		if c := pv.Competition; c != nil {
			for _, r := range c.Races {
				if r.Class != BoatErgo && r.BoatScale == 0 || r.TrainerScale == 0 {
					t.Fatalf("race %s without scales: %+v", r.ID, r)
				}
				if !c.TeamEvent && r.GuestScore == 0 {
					t.Fatalf("race %s without guest score", r.ID)
				}
			}
			sawRaces = true
		}
		for _, pid := range g.Pending() {
			if err := g.Submit(pid, autoDecision(g, pid, rng)); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, s := range []PhaseStage{StageSetup, StageEvent, StageEreignis, StageResult} {
		if !seen[s] {
			t.Errorf("stage %q never seen", s)
		}
	}
	if !sawDraft || !sawEvents || !sawRaces {
		t.Errorf("draft %v, events %v, races %v", sawDraft, sawEvents, sawRaces)
	}
}

// TestViewsHaveNoNull serializes every player's view in every phase of a full bot season (plus
// the results and the catalog) and fails on any JSON null: the TS types declare arrays/objects.
func TestViewsHaveNoNull(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3} {
		g, err := New(threePlayers(seed))
		if err != nil {
			t.Fatal(err)
		}
		rng := rand.New(rand.NewPCG(seed, 77))
		check := func(what string, v any) {
			t.Helper()
			raw, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			var tree any
			if err := json.Unmarshal(raw, &tree); err != nil {
				t.Fatal(err)
			}
			if path := findNull(tree, "$"); path != "" {
				t.Fatalf("seed %d, %s: null at %s", seed, what, path)
			}
		}
		for steps := 0; ; steps++ {
			if steps > 5000 {
				t.Fatal("no game over")
			}
			for _, pid := range []string{"p1", "p2", "p3"} {
				check("view "+g.PhaseID()+" "+pid, g.View(pid))
			}
			if g.Over() {
				break
			}
			for _, pid := range g.Pending() {
				d := autoDecision(g, pid, rng)
				if d.Kind == PhaseRegatta && pid == "p1" && seed == 1 {
					d.Entries, d.TrainerID = nil, "" // races without any entry
				}
				if err := g.Submit(pid, d); err != nil {
					t.Fatal(err)
				}
			}
		}
		check("results", g.Results())
		check("log", g.Log())
	}
	c, err := Catalog()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(c)
	var tree any
	_ = json.Unmarshal(raw, &tree)
	if path := findNull(tree, "$"); path != "" {
		t.Fatalf("catalog: null at %s", path)
	}
}

func findNull(v any, path string) string {
	switch x := v.(type) {
	case nil:
		return path
	case map[string]any:
		for k, e := range x {
			if p := findNull(e, path+"."+k); p != "" {
				return p
			}
		}
	case []any:
		for i, e := range x {
			if p := findNull(e, fmt.Sprintf("%s[%d]", path, i)); p != "" {
				return p
			}
		}
	}
	return ""
}
