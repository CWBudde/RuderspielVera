package engine

import (
	"math/rand/v2"
	"strings"
	"testing"
)

// TestPaintrainOffer checks that only the human player is offered the eight Paintrain, only in the
// last draft before the RVB-Endspurt, that taking it uses the pick, and that the best own eight
// then rows the Mixed-Achter.
func TestPaintrainOffer(t *testing.T) {
	t.Parallel()

	g, err := New(threePlayers(5))
	if err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewPCG(5, 77))
	offered := 0

	for steps := 0; !g.Over(); steps++ {
		if steps > 5000 {
			t.Fatal("no game over")
		}

		for _, pid := range g.Pending() {
			dec := autoDecision(g, pid, rng)

			pv := g.CurrentPhase(pid)
			for _, c := range pv.DraftOffer {
				if !c.Personal {
					continue
				}

				if pid != "p1" || c.Boat == nil || c.Boat.DefID != "paintrain" || c.Boat.Value != 12 {
					t.Fatalf("unexpected personal card for %s: %+v", pid, c)
				}

				if !g.nextCompetitionRows(Boat8x) {
					t.Errorf("Paintrain offered at station %d, not before the Endspurt", g.station)
				}

				offered++
				dec.DraftCardID = c.ID
			}

			if pv.Kind == PhaseRegatta && pid == "p1" {
				for _, r := range pv.Competition.Races {
					if r.Class == Boat8x && r.FixedBoatValue != 12 {
						t.Errorf("p1's eight value %d, want 12 (Paintrain)", r.FixedBoatValue)
					}
				}
			}

			if err := g.Submit(pid, dec); err != nil {
				t.Fatalf("phase %s, player %s: %v", g.PhaseID(), pid, err)
			}
		}
	}

	if offered != 1 {
		t.Fatalf("Paintrain offered %d times, want 1", offered)
	}

	if b := g.playerByID("p1").bestBoat(Boat8x); b == nil || b.defID != "paintrain" {
		t.Fatalf("p1's best eight %+v", b)
	}

	final := g.results[len(g.results)-1]
	eights := 0

	for _, rr := range final.Races {
		for _, e := range rr.Entries {
			if e.PlayerID == "" || !strings.HasPrefix(e.BoatID, "kraftwerk-") && !strings.HasPrefix(e.BoatID, "paintrain-") {
				continue
			}

			eights++
			want := 8
			if e.PlayerID == "p1" {
				want = 12
			}

			if e.BoatValue != want*g.d.Rules.BoatScale(8) {
				t.Errorf("%s: eight boat part %d, want %d", e.PlayerID, e.BoatValue, want*g.d.Rules.BoatScale(8))
			}
		}
	}

	if eights == 0 {
		t.Error("no eight entry in the final regatta")
	}
}
