package bot

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"testing"

	"ruderspiel/internal/data"
	"ruderspiel/internal/engine"
)

// decider is any player brain used by the tests.
type decider func(g *engine.Game, pid string, rng *rand.Rand) (engine.Decision, error)

func botDecider(p Personality) decider {
	return func(g *engine.Game, pid string, rng *rand.Rand) (engine.Decision, error) {
		return Decide(g, pid, p, rng)
	}
}

// randomDecider is a random but legal player: random training, selections and draft picks,
// random trainer and random eligible crews with a random free boat.
func randomDecider(g *engine.Game, pid string, rng *rand.Rand) (engine.Decision, error) {
	v := g.View(pid)
	pv := v.Phase
	d := engine.Decision{PhaseID: pv.ID, Kind: pv.Kind}
	var me *engine.PlayerSummary
	for i := range v.Players {
		if v.Players[i].ID == pid {
			me = &v.Players[i]
		}
	}
	switch pv.Kind {
	case engine.PhaseTraining:
		n := len(me.Rowers)
		if pv.MaxRowers > 0 {
			n = min(n, pv.MaxRowers)
		}
		perm := rng.Perm(len(me.Rowers))[:n]
		for range pv.Points {
			d.Training = append(d.Training, engine.TrainingAlloc{
				RowerID: me.Rowers[perm[rng.IntN(n)]].ID, Stat: engine.AllStats[rng.IntN(5)], Points: 1,
			})
		}
	case engine.PhaseSelection:
		for _, i := range rng.Perm(len(me.Rowers))[:min(pv.Count, len(me.Rowers))] {
			d.RowerIDs = append(d.RowerIDs, me.Rowers[i].ID)
		}
	case engine.PhaseRegatta:
		if len(me.Trainers) > 0 {
			d.TrainerID = me.Trainers[rng.IntN(len(me.Trainers))].ID
		}
		t := teamOf(me)
		noise := map[string]float64{}
		pr := newProblem(pv.Competition.Races, t, map[engine.Stat]float64{}, noise)
		for r := range pr.indiv {
			for x := range pr.indiv[r] {
				pr.indiv[r][x] = rng.Float64()
			}
		}
		pr.score = func(int, []int, int) float64 { return rng.Float64() }
		pr.value = func(pl *plan) float64 {
			s := 0.0
			for r, c := range pl.crew {
				if c != nil {
					s += pl.score[r] + 1
				}
			}
			return s
		}
		d.Entries = pr.entries(pr.greedy())
	case engine.PhaseDraft:
		d.DraftCardID = pv.DraftOffer[rng.IntN(len(pv.DraftOffer))].ID
	}
	return d, nil
}

func playSeason(t testing.TB, seed uint64, ds []decider) *engine.Game {
	t.Helper()
	cfg := engine.Config{Seed: seed}
	for i := range ds {
		cfg.Players = append(cfg.Players, engine.PlayerConfig{ID: fmt.Sprintf("p%d", i), IsBot: true})
	}
	g, err := engine.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rngs := make([]*rand.Rand, len(ds))
	for i := range rngs {
		rngs[i] = rand.New(rand.NewPCG(seed, uint64(i)))
	}
	for steps := 0; !g.Over(); steps++ {
		if steps > 10000 {
			t.Fatalf("seed %d: no game over", seed)
		}
		for _, pid := range g.Pending() {
			var seat int
			fmt.Sscanf(pid, "p%d", &seat)
			d, err := ds[seat](g, pid, rngs[seat])
			if err != nil {
				t.Fatalf("seed %d, phase %s, %s: decide: %v", seed, g.PhaseID(), pid, err)
			}
			if err := g.Submit(pid, d); err != nil {
				t.Fatalf("seed %d, phase %s, %s: rejected: %v", seed, g.PhaseID(), pid, err)
			}
		}
	}
	return g
}

func TestBotsNeverRejected(t *testing.T) {
	seasons := 300
	if testing.Short() {
		seasons = 30
	}
	ds := []decider{botDecider(Bauchgefuehl), botDecider(Taktiker), botDecider(Bauchgefuehl)}
	for seed := uint64(1); seed <= uint64(seasons); seed++ {
		g := playSeason(t, seed, ds)
		if len(g.Results()) != 12 {
			t.Fatalf("seed %d: %d competitions", seed, len(g.Results()))
		}
	}
}

func TestBotsOtherPlayerCounts(t *testing.T) {
	for _, n := range []int{2, 4} {
		for seed := uint64(1); seed <= 10; seed++ {
			ds := make([]decider, n)
			for i := range ds {
				ds[i] = botDecider(PersonalityFor(i))
			}
			playSeason(t, seed, ds)
		}
	}
}

func TestTaktikerBeatsRandom(t *testing.T) {
	games := 60
	wins := 0.0
	for seed := uint64(1); seed <= uint64(games); seed++ {
		g := playSeason(t, seed, []decider{botDecider(Taktiker), randomDecider, randomDecider})
		for _, w := range g.Winners() {
			if w == "p0" {
				wins += 1 / float64(len(g.Winners()))
			}
		}
	}
	rate := wins / float64(games)
	t.Logf("Taktiker win rate vs two random players: %.0f%%", 100*rate)
	if rate < 0.75 {
		t.Errorf("Taktiker won %.0f%% against two random players, want >= 75%%", 100*rate)
	}
}

func TestDecideIsDeterministic(t *testing.T) {
	ds := []decider{botDecider(Taktiker), botDecider(Bauchgefuehl), botDecider(Taktiker)}
	a := playSeason(t, 7, ds)
	b := playSeason(t, 7, ds)
	ja, _ := json.Marshal(a.Decisions())
	jb, _ := json.Marshal(b.Decisions())
	if string(ja) != string(jb) {
		t.Fatal("same seeds produced different bot decisions")
	}
	// The decision log replays to the same game (bots do not touch the game RNG).
	r, err := engine.Replay(a.Config(), a.Decisions())
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(r.Winners()) != fmt.Sprint(a.Winners()) {
		t.Fatal("replay differs")
	}
}

func TestDecideRejectsNonPending(t *testing.T) {
	g, err := engine.New(engine.Config{Seed: 1, Players: []engine.PlayerConfig{{ID: "a"}, {ID: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(1, 1))
	d, err := Decide(g, "a", Taktiker, rng)
	if err != nil || d.Kind != engine.PhaseAck {
		t.Fatalf("setup ack: %+v %v", d, err)
	}
	if err := g.Submit("a", d); err != nil {
		t.Fatal(err)
	}
	if _, err := Decide(g, "a", Taktiker, rng); err == nil {
		t.Error("decision for a player that already decided")
	}
}

func TestPersonalityFor(t *testing.T) {
	if PersonalityFor(1) != Taktiker || PersonalityFor(2) != Bauchgefuehl || PersonalityFor(0) != Bauchgefuehl {
		t.Error("unexpected seat personalities")
	}
}

// TestFutureScoreMatchesEngine keeps the bot's offline scoring model in sync with the engine for
// the open competition (no trainer, no weather events).
func TestFutureScoreMatchesEngine(t *testing.T) {
	g := playUntilKind(t, 3, engine.PhaseRegatta, 3)
	pid := "p0"
	b := newBrain(g, pid, Taktiker, rand.New(rand.NewPCG(1, 1)), mustData(t))
	b.noise = map[string]float64{}
	cur := b.currentComp()
	fut := b.futureComp(b.view.Current)
	me := teamOf(b.me)
	pr := newProblem(cur.races, me, cur.weights, nil)
	exact := b.scorer(pr, me, cur, "", nil)
	approx := b.scorer(pr, &team{id: me.id, rowers: me.rowers, boats: me.boats}, fut, "", nil)
	checked := 0
	for r := range cur.races {
		crew := pr.buildCrew(r, make([]bool, len(me.rowers)))
		if crew == nil {
			continue
		}
		boat, ok := pr.bestBoat(r, make([]bool, len(me.boats)))
		if !ok {
			continue
		}
		e, a := exact(r, crew, boat), approx(r, crew, boat)
		if d := e - a; d > 0.51 || d < -0.51 {
			t.Errorf("race %s: engine %.1f, bot model %.1f", cur.races[r].ID, e, a)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no race checked")
	}
}

// playUntilKind plays bots until the n-th phase of the given kind opens.
func playUntilKind(t *testing.T, seed uint64, kind engine.PhaseKind, n int) *engine.Game {
	t.Helper()
	g, err := engine.New(engine.Config{Seed: seed, Players: []engine.PlayerConfig{{ID: "p0"}, {ID: "p1"}, {ID: "p2"}}})
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(seed, 9))
	seen, last := 0, ""
	for !g.Over() {
		if pv := g.CurrentPhase("p0"); pv.Kind == kind && pv.ID != last {
			last = pv.ID
			if seen++; seen == n {
				return g
			}
		}
		for _, pid := range g.Pending() {
			d, err := Decide(g, pid, Taktiker, rng)
			if err != nil {
				t.Fatal(err)
			}
			if err := g.Submit(pid, d); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Fatalf("no %d-th %s phase", n, kind)
	return nil
}

func mustData(t *testing.T) *data.Data {
	t.Helper()
	d, err := data.Load()
	if err != nil {
		t.Fatal(err)
	}
	return d
}
