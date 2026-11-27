package gameservice

import (
	"bytes"
	"encoding/json"
	mrand "math/rand/v2"
	"slices"
	"testing"

	"ruderspiel/internal/engine"
)

// TestResultWithEmptyRaces guards a bug from a real exported WASM game that got stuck on the
// result screen of its first regatta: races nobody entered were serialized as "entries": null and
// crashed the client. The test plays to the first regatta (the bots of newService enter nothing),
// enters a single race, and checks that the result, also after an export/import round trip,
// has races with "entries": [] and no JSON null, and that the game keeps playing.
func TestResultWithEmptyRaces(t *testing.T) {
	t.Parallel()

	s := newService(t, nil)
	cr := create(t, s, 3)
	rng := mrand.New(mrand.NewPCG(3, 0))

	view := cr.View
	for view.Phase.Kind != engine.PhaseRegatta {
		var err error

		view, err = s.Submit(cr.GameID, cr.Token, humanDecision(t, s, cr.GameID, rng))
		if err != nil {
			t.Fatal(err)
		}
	}

	view, err := s.Submit(cr.GameID, cr.Token, oneEntry(t, view))
	if err != nil {
		t.Fatal(err)
	}

	checkEmptyRaceResult(t, view)

	snap, err := s.Export(cr.GameID)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}

	var back Snapshot

	err = json.Unmarshal(raw, &back)
	if err != nil {
		t.Fatal(err)
	}

	s2 := newService(t, nil)

	_, err = s2.Import(back)
	if err != nil {
		t.Fatal(err)
	}

	view, err = s2.View(cr.GameID, cr.Token)
	if err != nil {
		t.Fatal(err)
	}

	checkEmptyRaceResult(t, view)

	_, err = s2.Submit(cr.GameID, cr.Token, engine.Decision{PhaseID: view.Phase.ID, Kind: engine.PhaseAck})
	if err != nil {
		t.Fatalf("continue after result: %v", err)
	}
}

// oneEntry enters the first race the human can fill alone, with a loan boat if allowed.
func oneEntry(t *testing.T, view engine.GameView) engine.Decision {
	t.Helper()

	var me engine.PlayerSummary

	for _, p := range view.Players {
		if p.ID == view.You {
			me = p
		}
	}

	dec := engine.Decision{PhaseID: view.Phase.ID, Kind: engine.PhaseRegatta}
	if len(me.Trainers) > 0 {
		dec.TrainerID = me.Trainers[0].ID
	}

	for _, race := range view.Phase.Competition.Races {
		if race.Seats != 1 {
			continue
		}

		for _, r := range me.Rowers {
			if slices.Contains(race.Ages, r.Age) && slices.Contains(race.Genders, r.Gender) {
				e := engine.Entry{RaceID: race.ID, RowerIDs: []string{r.ID}}
				if !race.LoanBoat {
					for _, b := range me.Boats {
						if b.Class == race.Class {
							e.BoatID = b.ID
						}
					}
				}

				dec.Entries = []engine.Entry{e}

				return dec
			}
		}
	}

	t.Fatal("no single the human can enter")

	return dec
}

func checkEmptyRaceResult(t *testing.T, view engine.GameView) {
	t.Helper()

	if view.Phase.Stage != engine.StageResult || view.LastResult == nil {
		t.Fatalf("phase %s stage %q, lastResult %v", view.Phase.ID, view.Phase.Stage, view.LastResult != nil)
	}

	empty, entered := 0, 0

	for _, r := range view.LastResult.Races {
		if len(r.Entries) == 0 {
			empty++
		} else {
			entered++
		}
	}

	if empty == 0 || entered == 0 {
		t.Fatalf("want empty and entered races, got %d empty, %d entered", empty, entered)
	}

	js, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Contains(js, []byte("null")) {
		t.Fatalf("view JSON contains null: %s", js)
	}

	if !bytes.Contains(js, []byte(`"entries":[]`)) {
		t.Fatalf(`view JSON has no "entries":[]: %s`, js)
	}
}
