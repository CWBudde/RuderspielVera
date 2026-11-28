package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"ruderspiel/internal/data"
)

func newGame(tb testing.TB, seed uint64) *Game {
	tb.Helper()

	g, err := New(threePlayers(seed))
	if err != nil {
		tb.Fatal(err)
	}

	return g
}

func TestNewDeal(t *testing.T) {
	t.Parallel()

	g := newGame(t, 1)
	seen := map[string]bool{}

	for _, pl := range g.players {
		if len(pl.rowers) != 18 {
			t.Errorf("%s: %d rowers, want 18", pl.id, len(pl.rowers))
		}

		perAge := map[AgeGroup]map[Gender]int{}

		for _, r := range pl.rowers {
			if seen[r.id] {
				t.Errorf("rower %s dealt twice", r.id)
			}

			seen[r.id] = true
			if perAge[r.age] == nil {
				perAge[r.age] = map[Gender]int{}
			}

			perAge[r.age][r.gender]++
		}

		for _, age := range ageOrder {
			want := map[AgeGroup]int{AgeKinder: 4, AgeB: 9, AgeA: 5}[age]
			if perAge[age][GenderF]+perAge[age][GenderM] != want || perAge[age][GenderF] == 0 || perAge[age][GenderM] == 0 {
				t.Errorf("%s: age %s split %v", pl.id, age, perAge[age])
			}
		}

		if len(pl.boats) != 4 || len(pl.trainers) != 2 {
			t.Errorf("%s: %d boats, %d trainers", pl.id, len(pl.boats), len(pl.trainers))
		}
	}

	rowers, boats, trainers := 0, 0, 0

	for _, c := range g.pool {
		switch c.kind() {
		case "rower":
			rowers++
		case "boat":
			boats++
		case "trainer":
			trainers++
		}
	}

	if rowers != 1 || boats != 14 || trainers != 4 {
		t.Errorf("pool = %d rowers, %d boats, %d trainers; want 1/14/4", rowers, boats, trainers)
	}

	if g.CurrentPhase("p1").Kind != PhaseAck {
		t.Errorf("first phase should be the setup ack")
	}
}

// TestDealCityBoats checks that every player gets one named city boat per class (value 1/2/4)
// plus the eight Kraftwerk (8),
// that a short class is filled with copies, and that undealt city boats join the draft pool.
func TestDealCityBoats(t *testing.T) {
	t.Parallel()

	values := map[BoatClass]int{Boat1x: 1, Boat2x: 2, Boat4x: 4, Boat8x: 8}

	for _, tc := range []struct{ players, poolCity int }{{2, 4}, {3, 1}, {4, 0}} {
		cfg := Config{Seed: 11}
		for i := range tc.players {
			cfg.Players = append(cfg.Players, PlayerConfig{ID: fmt.Sprintf("p%d", i+1)})
		}

		g, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}

		ids := map[string]bool{}

		for _, pl := range g.players {
			got := map[BoatClass]int{}

			for _, b := range pl.boats {
				got[b.class]++

				if b.city != (b.class != Boat8x) || b.value != values[b.class] || b.defID == "" || ids[b.id] {
					t.Errorf("%d players, %s: bad boat %+v", tc.players, pl.id, *b)
				}

				ids[b.id] = true
			}

			if !maps.Equal(got, map[BoatClass]int{Boat1x: 1, Boat2x: 1, Boat4x: 1, Boat8x: 1}) {
				t.Errorf("%d players, %s: boats per class %v", tc.players, pl.id, got)
			}
		}

		city := 0

		for _, c := range g.pool {
			if c.boat != nil && c.boat.city {
				city++
			}
		}

		if city != tc.poolCity {
			t.Errorf("%d players: %d city boats in the pool, want %d", tc.players, city, tc.poolCity)
		}
	}
}

func TestDealByPlayerCount(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		players    int
		perAge     map[AgeGroup]int
		poolRowers int
	}{
		{2, map[AgeGroup]int{AgeKinder: 6, AgeB: 13, AgeA: 7}, 3},
		{3, map[AgeGroup]int{AgeKinder: 4, AgeB: 9, AgeA: 5}, 1},
		{4, map[AgeGroup]int{AgeKinder: 3, AgeB: 4, AgeA: 3}, 15},
	} {
		cfg := Config{Seed: 7}
		for i := range tc.players {
			cfg.Players = append(cfg.Players, PlayerConfig{ID: fmt.Sprintf("p%d", i+1)})
		}

		g, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}

		for _, pl := range g.players {
			got := map[AgeGroup]int{}
			for _, r := range pl.rowers {
				got[r.age]++
			}

			if !maps.Equal(got, tc.perAge) {
				t.Errorf("%d players, %s: rowers per age %v, want %v", tc.players, pl.id, got, tc.perAge)
			}
		}

		rowers := 0

		for _, c := range g.pool {
			if c.kind() == "rower" {
				rowers++
			}
		}

		if rowers != tc.poolRowers {
			t.Errorf("%d players: %d rowers in the pool, want %d", tc.players, rowers, tc.poolRowers)
		}
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	t.Parallel()

	for _, cfg := range []Config{
		{Players: []PlayerConfig{{ID: "a"}}},
		{Players: []PlayerConfig{{ID: "a"}, {ID: "a"}}},
		{Players: []PlayerConfig{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "e"}}},
	} {
		_, err := New(cfg)
		if err == nil {
			t.Errorf("New(%v) should fail", cfg.Players)
		}
	}
}

func TestRankEntries(t *testing.T) {
	t.Parallel()

	type in struct {
		total    int
		incident IncidentKind
	}
	type want struct {
		place int
		medal Medal
	}

	tests := []struct {
		name string
		in   []in
		want []want // in sorted output order
	}{
		{
			"clear order",
			[]in{{90, ""}, {100, ""}, {80, ""}},
			[]want{{1, MedalGold}, {2, MedalSilver}, {3, ""}},
		},
		{
			"two golds skip silver",
			[]in{{100, ""}, {100, ""}, {90, ""}},
			[]want{{1, MedalGold}, {1, MedalGold}, {3, ""}},
		},
		{
			"shared silver",
			[]in{{100, ""}, {90, ""}, {90, ""}},
			[]want{{1, MedalGold}, {2, MedalSilver}, {2, MedalSilver}},
		},
		{
			"krebs still finishes",
			[]in{{95, IncidentKrebs}, {90, ""}},
			[]want{{1, MedalGold}, {2, MedalSilver}},
		},
		{
			"kentern last, no medal",
			[]in{{200, IncidentKentern}, {90, ""}, {80, ""}},
			[]want{{1, MedalGold}, {2, MedalSilver}, {3, ""}},
		},
		{
			"two capsized share last",
			[]in{{200, IncidentKentern}, {150, IncidentKentern}, {80, ""}},
			[]want{{1, MedalGold}, {2, ""}, {2, ""}},
		},
		{
			"material is DNF",
			[]in{{200, IncidentMaterial}, {90, ""}},
			[]want{{1, MedalGold}, {0, ""}},
		},
		{
			"single capsized starter gets nothing",
			[]in{{100, IncidentKentern}},
			[]want{{1, ""}},
		},
		{
			"single finisher gets gold",
			[]in{{10, ""}},
			[]want{{1, MedalGold}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var es []EntryResult
			for i, x := range tt.in {
				es = append(es, EntryResult{PlayerID: string(rune('a' + i)), Total: x.total, Incident: x.incident})
			}

			rankEntries(es)

			for i, w := range tt.want {
				if es[i].Place != w.place || es[i].Medal != w.medal {
					t.Errorf("entry %d (total %d %s): place %d medal %q, want %d %q",
						i, es[i].Total, es[i].Incident, es[i].Place, es[i].Medal, w.place, w.medal)
				}
			}
		})
	}
}

func TestAwardMedal(t *testing.T) {
	t.Parallel()

	g := newGame(t, 2)
	pl := g.players[0]
	crew := []string{pl.rowers[0].id, pl.rowers[1].id}
	before := []int{pl.rowers[0].stats[StatMotivation], pl.rowers[1].stats[StatMotivation]}
	out := map[string]rowerOutcome{}
	g.awardMedal(EntryResult{PlayerID: pl.id, RowerIDs: crew, Medal: MedalGold}, Boat2x, out)

	if pl.gold != 1 || g.points(pl) != 2 {
		t.Errorf("gold %d points %d, want 1/2", pl.gold, g.points(pl))
	}

	g.awardMedal(EntryResult{PlayerID: pl.id, RowerIDs: crew[:1], Medal: MedalSilver}, Boat1x, out)

	if pl.silver != 1 || g.points(pl) != 3 {
		t.Errorf("silver must add 1 point")
	}

	for i := range crew {
		if got := pl.rowers[i].stats[StatMotivation]; got != before[i] {
			t.Errorf("crew %d: medal changed the permanent motivation %d -> %d", i, before[i], got)
		}
	}

	if out[crew[0]].medal != MedalSilver || out[crew[1]].class != Boat2x {
		t.Errorf("outcomes not recorded: %v", out)
	}
}

// TestWinMotivation plays sequences of competitions (the rower's medal at each, "" = no winning
// placement or no start) and checks the Siegesmotivation afterwards.
func TestWinMotivation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		medals    []Medal
		wantBonus int
		wantLeft  int
	}{
		{"gold gives +1 for 3 competitions", []Medal{MedalGold}, 1, 3},
		{"silver also triggers", []Medal{MedalSilver}, 1, 3},
		{"no medal, no bonus", []Medal{MedalNone}, 0, 0},
		{"two golds do not stack", []Medal{MedalGold, MedalGold}, 1, 3},
		{"gold then silver does not stack", []Medal{MedalGold, MedalSilver}, 1, 3},
		{"one competition used", []Medal{MedalGold, MedalNone}, 1, 2},
		{"two competitions used", []Medal{MedalGold, MedalNone, MedalNone}, 1, 1},
		{"expires after 3 competitions", []Medal{MedalGold, MedalNone, MedalNone, MedalNone}, 0, 0},
		{"stays expired", []Medal{MedalSilver, MedalNone, MedalNone, MedalNone, MedalNone}, 0, 0},
		{"win while active refreshes", []Medal{MedalGold, MedalNone, MedalNone, MedalSilver}, 1, 3},
		{"refresh restarts the countdown", []Medal{MedalGold, MedalNone, MedalGold, MedalNone, MedalNone}, 1, 1},
		{"refreshed bonus expires too", []Medal{MedalGold, MedalNone, MedalGold, MedalNone, MedalNone, MedalNone}, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := newGame(t, 3)
			rw, other := g.players[0].rowers[0], g.players[0].rowers[1]
			base := rw.stats[StatMotivation]

			for _, m := range tt.medals {
				out := map[string]rowerOutcome{}
				if m != MedalNone {
					out[rw.id] = rowerOutcome{medal: m, class: Boat1x}
				}

				g.updateWinMotivation(out)
			}

			if rw.winBonus != tt.wantBonus || rw.winLeft != tt.wantLeft {
				t.Errorf("bonus %d left %d, want %d / %d", rw.winBonus, rw.winLeft, tt.wantBonus, tt.wantLeft)
			}

			if rw.stats[StatMotivation] != base {
				t.Errorf("permanent motivation changed: %d -> %d", base, rw.stats[StatMotivation])
			}

			if got, want := rw.stat(StatMotivation), base+tt.wantBonus; got != want {
				t.Errorf("effective motivation %d, want %d", got, want)
			}

			v := rowerView(rw, true)
			if v.Stats[StatMotivation] != base+tt.wantBonus || v.WinMotivation != tt.wantBonus ||
				v.WinMotivationLeft != tt.wantLeft {
				t.Errorf("view motivation %d (+%d, %d left), want %d (+%d, %d left)", v.Stats[StatMotivation],
					v.WinMotivation, v.WinMotivationLeft, base+tt.wantBonus, tt.wantBonus, tt.wantLeft)
			}

			if other.winBonus != 0 || other.winLeft != 0 {
				t.Errorf("rower without a medal got a bonus")
			}
		})
	}
}

// TestWinMotivationScoring checks that an active Siegesmotivation counts in scoring and in
// EstimateEntry exactly like Motivation (weight × bonus).
func TestWinMotivationScoring(t *testing.T) {
	t.Parallel()

	g := newGame(t, 4)
	pl := g.players[0]
	cs := compFor(g, "rvh", WeatherNormal)
	race := cs.race("1x-open-w")

	var rw *rower

	for _, x := range pl.rowers {
		if x.gender == GenderF {
			rw = x
			break
		}
	}

	if rw == nil {
		t.Fatal("no girl")
	}

	tests := []struct {
		name  string
		bonus int
		left  int
	}{
		{"inactive", 0, 0},
		{"active", 1, 2},
	}

	totals := make([]int, 0, len(tests))

	for _, tt := range tests {
		rw.winBonus, rw.winLeft = tt.bonus, tt.left
		totals = append(totals, g.scoreEntry(cs, *race, raceEntry{pl: pl, rowers: []*rower{rw}}).Crew)
	}

	if want := totals[0] + int(cs.weights[StatMotivation]); totals[1] != want {
		t.Errorf("crew with Siegesmotivation %d, want %d (without: %d)", totals[1], want, totals[0])
	}

	// EstimateEntry (bots, UI preview) sees the bonus of the open competition.
	g.phase = &phase{comp: cs}
	rw.winBonus, rw.winLeft = 0, 0
	plain := EstimateEntry(g, pl.id, race.ID, []string{rw.id}, pl.boats[0].id, "")
	rw.winBonus, rw.winLeft = 1, 1

	if got := EstimateEntry(g, pl.id, race.ID, []string{rw.id}, pl.boats[0].id, ""); got != plain+1 {
		t.Errorf("EstimateEntry with Siegesmotivation %d, want %d", got, plain+1)
	}
}

// TestWinMotivationInSeason plays a season with bots and checks that a crew with a winning
// placement carries the bonus afterwards and that it is never above the rule value.
func TestWinMotivationInSeason(t *testing.T) {
	t.Parallel()

	g := newGame(t, 11)
	wm := g.d.Rules.WinMotivation
	rng := rand.New(rand.NewPCG(11, 77))
	seen := false

	for steps := 0; !g.Over() && steps < 5000; steps++ {
		done := len(g.results)

		for _, pid := range g.Pending() {
			err := g.Submit(pid, autoDecision(g, pid, rng))
			if err != nil {
				t.Fatal(err)
			}
		}

		if len(g.results) == done {
			continue
		}

		// At JTFO only the team row awards medals; its rows per boat carry none.
		for _, rr := range g.results[done].Races {
			for _, e := range rr.Entries {
				pl := g.playerByID(e.PlayerID)
				if pl == nil || e.Medal == MedalNone {
					continue
				}

				for _, id := range e.RowerIDs {
					if r := pl.rower(id); r.winBonus != wm.Value || r.winLeft != wm.Competitions {
						t.Errorf("%s: %s has bonus %d left %d after %s", g.results[done].Name, id, r.winBonus, r.winLeft, e.Medal)
					}

					seen = true
				}
			}
		}

		for _, pl := range g.players {
			for _, r := range pl.rowers {
				if r.winBonus > wm.Value || r.winLeft > wm.Competitions || (r.winBonus > 0) != (r.winLeft > 0) {
					t.Fatalf("%s: bad Siegesmotivation +%d / %d", r.id, r.winBonus, r.winLeft)
				}
			}
		}
	}

	if !seen {
		t.Error("no winning placement in a whole season")
	}
}

func TestIncidentDistribution(t *testing.T) {
	t.Parallel()

	const n = 400000
	counts := map[IncidentKind]int{}

	for seed := range uint64(20) {
		rng := newRNG(seed)
		for range n / 20 {
			counts[rollIncident(rng, 20, 30, 150)]++
		}
	}

	want := map[IncidentKind]float64{
		IncidentKentern: 0.02, IncidentMaterial: 0.03, IncidentKrebs: 0.15, IncidentNone: 0.80,
	}
	for k, p := range want {
		got := float64(counts[k]) / n
		if got < p-0.003 || got > p+0.003 {
			t.Errorf("%q: %.4f, want %.3f ± 0.003", k, got, p)
		}
	}
}

func TestIncidentRules(t *testing.T) {
	t.Parallel()

	inc := newGame(t, 3).d.Rules.Incidents
	want := map[int]int{1: 20, 2: 10, 4: 5, 8: 2}

	for seats, k := range want {
		if got := inc.Kentern(seats); got != k {
			t.Errorf("Kentern(%d) = %d ‰, want %d ‰", seats, got, k)
		}
	}

	if inc.Material != 30 || inc.Krebs != 75 {
		t.Errorf("material %d ‰, krebs %d ‰, want 30 / 75", inc.Material, inc.Krebs)
	}

	ks := inc.KrebsScaling
	if ks.TechnikPerSeat != 14 || ks.Min != 10 || ks.Max != 225 || ks.SeatPercent[1] != 100 || ks.SeatPercent[8] != 150 {
		t.Errorf("krebsScaling %+v, want perSeat 14, min 10, max 225, 1x 100 %%, 8x 150 %%", ks)
	}
}

func TestKrebsChance(t *testing.T) {
	t.Parallel()

	ks := data.KrebsScaling{SeatPercent: map[int]int{1: 100, 2: 115, 4: 130, 8: 150}, TechnikPerSeat: 14, Min: 20, Max: 450}
	tests := []struct {
		name                       string
		base, seats, technik, want int
	}{
		{"1x at reference keeps base", 150, 1, 14, 150},
		{"1x half reference doubles", 150, 1, 7, 300},
		{"1x double reference halves", 150, 1, 28, 75},
		{"8x at reference scales by seats", 150, 8, 112, 225},
		{"4x at reference", 150, 4, 56, 195},
		{"8x strong crew", 150, 8, 224, 113},
		{"sum 0 hits max", 150, 1, 0, 450},
		{"clamped to min", 50, 1, 60, 20},
		{"Ereignis base with weak crew clamped to max", 300, 1, 8, 450},
	}

	for _, tt := range tests {
		if got := krebsChance(tt.base, tt.seats, tt.technik, ks); got != tt.want {
			t.Errorf("%s: krebsChance(%d, %d, %d) = %d, want %d", tt.name, tt.base, tt.seats, tt.technik, got, tt.want)
		}
	}

	// Equally skilled crews: the bigger boat catches more crabs.
	if one, eight := krebsChance(150, 1, 14, ks), krebsChance(150, 8, 112, ks); eight <= one {
		t.Errorf("8x %d ‰ should exceed 1x %d ‰", eight, one)
	}
}

// compFor builds an open competition state for validation tests.
func compFor(g *Game, id string, weather Weather) *compState {
	def := g.d.Competitions[id]

	cs := &compState{def: def, name: def.Name, weights: map[Stat]float64{}, weather: weather}
	for _, r := range g.d.RacesFor(def, len(g.players)) {
		cs.races = append(cs.races, raceSpec(r))
	}

	for _, s := range AllStats {
		if w, ok := def.Weights[string(s)]; ok {
			cs.weights[s] = w
		}
	}

	return cs
}

// testPlayer replaces player 0's roster with a controlled one.
func testPlayer(g *Game) *player {
	pl := g.players[0]
	mk := func(id string, age AgeGroup, gender Gender) *rower {
		s := Stats{StatKraft: 10, StatAusdauer: 10, StatSpeed: 10, StatTechnik: 10, StatMotivation: 10}
		return &rower{id: id, name: id, age: age, gender: gender, base: s.clone(), stats: s}
	}
	pl.rowers = []*rower{
		mk("aw1", AgeA, GenderF), mk("aw2", AgeA, GenderF), mk("am1", AgeA, GenderM),
		mk("bw1", AgeB, GenderF), mk("bm1", AgeB, GenderM), mk("bm2", AgeB, GenderM),
		mk("kw1", AgeKinder, GenderF), mk("km1", AgeKinder, GenderM), mk("kw2", AgeKinder, GenderF),
	}

	return pl
}

func TestValidateEntries(t *testing.T) {
	t.Parallel()

	g := newGame(t, 3)
	pl := testPlayer(g)
	tr := pl.trainers[0].def.ID
	b1, b2, b4 := pl.boats[0].id, pl.boats[1].id, pl.boats[2].id
	hild := compFor(g, "hildesheim", WeatherNormal)
	rvb := compFor(g, "rvb", WeatherNormal)
	ergo := compFor(g, "mindoor", "")
	jtfo := compFor(g, "jtfo", WeatherNormal)
	schiller := compFor(g, "schillerschule", WeatherNormal)

	tests := []struct {
		name    string
		cs      *compState
		trainer string
		entries []Entry
		errPart string // "" = valid
	}{
		{"valid single", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1}}, ""},
		{"valid two races", hild, tr, []Entry{
			{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1},
			{RaceID: "2x-a-w", RowerIDs: []string{"aw2", "bw1"}, BoatID: b2},
		}, ""},
		{"no entries no trainer", hild, "", nil, ""},
		{"trainer required", hild, "", []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1}}, "err.trainerRequired"},
		{"foreign trainer", hild, "nobody", nil, "err.trainerNotOwned"},
		{"unknown race", hild, tr, []Entry{{RaceID: "nope", RowerIDs: []string{"aw1"}, BoatID: b1}}, "err.unknownRace"},
		{"wrong age", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"bw1"}, BoatID: b1}}, "err.rowerAge"},
		{"row up in crew boat", hild, tr, []Entry{{RaceID: "2x-a-w", RowerIDs: []string{"aw1", "bw1"}, BoatID: b2}}, ""},
		{"wrong gender", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"am1"}, BoatID: b1}}, "err.rowerGender"},
		{"wrong seats", hild, tr, []Entry{{RaceID: "2x-a-w", RowerIDs: []string{"aw1"}, BoatID: b2}}, "err.raceSeats{count=1 race=data.race.hildesheim.2x-a-w.name seats=2}"},
		{"rower twice", hild, tr, []Entry{
			{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1},
			{RaceID: "2x-a-w", RowerIDs: []string{"aw1", "aw2"}, BoatID: b2},
		}, "err.rowerTwice"},
		{"boat twice", hild, tr, []Entry{
			{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1},
			{RaceID: "1x-a-m", RowerIDs: []string{"am1"}, BoatID: b1},
		}, "err.boatTwice"},
		{"race twice", hild, tr, []Entry{
			{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1},
			{RaceID: "1x-a-w", RowerIDs: []string{"aw2"}, BoatID: b1},
		}, "err.raceTwice"},
		{"wrong boat class", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b2}}, "err.boatClass"},
		{"missing boat", schiller, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}}}, "err.boatRequired"},
		{"hildesheim loan boats", hild, tr, []Entry{
			{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1},
			{RaceID: "1x-a-m", RowerIDs: []string{"am1"}},
			{RaceID: "2x-a-w", RowerIDs: []string{"aw2", "bw1"}, BoatID: b2},
			{RaceID: "2x-b-m", RowerIDs: []string{"bm1", "km1"}},
		}, ""},
		{"foreign boat", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: "ice-man"}}, "NotOwned"},
		{"foreign rower", hild, tr, []Entry{
			{RaceID: "1x-a-w", RowerIDs: []string{"michel-a"}, BoatID: b1},
		}, "NotOwned"},
		{"ergo no boat", ergo, tr, []Entry{{RaceID: "ergo-kinder-w", RowerIDs: []string{"kw1"}}}, ""},
		{"ergo with boat", ergo, tr, []Entry{
			{RaceID: "ergo-kinder-w", RowerIDs: []string{"kw1"}, BoatID: b1},
		}, "err.noOwnBoat"},
		{"mixed ok", rvb, tr, []Entry{
			{RaceID: "mixed-4x", RowerIDs: []string{"aw1", "am1", "bw1", "kw1"}, BoatID: b4},
		}, ""},
		{"mixed one gender", rvb, tr, []Entry{
			{RaceID: "mixed-4x", RowerIDs: []string{"aw1", "aw2", "bw1", "kw1"}, BoatID: b4},
		}, "err.mixed"},
		{"mixed one age", rvb, tr, []Entry{
			{RaceID: "mixed-4x", RowerIDs: []string{"bw1", "bm1", "bm2", "bm1"}, BoatID: b4},
		}, "err.rowerTwice"},
		{"mixed same age", compFor(g, "rvb", WeatherNormal), tr, []Entry{
			{RaceID: "mixed-4x", RowerIDs: []string{"aw1", "aw2", "am1", "am1"}, BoatID: b4},
		}, "err.rowerTwice"},
		{"eight without boat", rvb, tr, []Entry{{
			RaceID:   "mixed-8x",
			RowerIDs: []string{"aw1", "aw2", "am1", "bw1", "bm1", "bm2", "kw1", "km1"},
		}}, ""},
		{"jtfo loan boats", jtfo, tr, []Entry{
			{RaceID: "jtfo-1-2x", RowerIDs: []string{"aw1", "am1"}, BoatID: b2},
			{RaceID: "jtfo-3-2x", RowerIDs: []string{"bw1", "bm1"}},
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := g.validateEntries(pl, tt.cs, tt.trainer, tt.entries)
			if tt.errPart == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				return
			}

			var ve *ValidationError
			if err == nil || !errors.As(err, &ve) || !strings.Contains(err.Error(), tt.errPart) {
				t.Fatalf("error %v, want ValidationError containing %q", err, tt.errPart)
			}
		})
	}
	// Mixed with two genders but a single age group.
	pl.rowers = append(pl.rowers,
		&rower{id: "bw2", name: "bw2", age: AgeB, gender: GenderF, stats: Stats{}, base: Stats{}})

	err := g.validateEntries(pl, rvb, tr, []Entry{
		{RaceID: "mixed-4x", RowerIDs: []string{"bw1", "bw2", "bm1", "bm2"}, BoatID: b4},
	})
	if err == nil || !strings.Contains(err.Error(), "err.mixed") {
		t.Errorf("single-age mixed crew accepted: %v", err)
	}
}

func TestValidateTraining(t *testing.T) {
	t.Parallel()

	g := newGame(t, 4)
	pl := testPlayer(g)
	training := &phase{points: 10}
	camp := &phase{points: 30, maxRowers: 5}
	alloc := func(id string, s Stat, p int) TrainingAlloc { return TrainingAlloc{RowerID: id, Stat: s, Points: p} }

	tests := []struct {
		name    string
		p       *phase
		allocs  []TrainingAlloc
		errPart string
	}{
		{"exact", training, []TrainingAlloc{
			alloc("aw1", StatKraft, 4), alloc("bw1", StatTechnik, 3), alloc("kw1", StatSpeed, 3),
		}, ""},
		{"too few", training, []TrainingAlloc{alloc("aw1", StatKraft, 9)}, "err.trainTotal{points=10"},
		{"too many", training, []TrainingAlloc{alloc("aw1", StatKraft, 11)}, "err.trainTotal{points=10"},
		{"negative", training, []TrainingAlloc{alloc("aw1", StatKraft, 12), alloc("aw2", StatKraft, -2)}, "err.trainPositive"},
		{"bad stat", training, []TrainingAlloc{alloc("aw1", "magie", 10)}, "err.unknownStat"},
		{"motivation", training, []TrainingAlloc{
			alloc("aw1", StatKraft, 9), alloc("aw1", StatMotivation, 1),
		}, "err.trainMotivation"},
		{"camp motivation", camp, []TrainingAlloc{alloc("aw1", StatMotivation, 30)}, "err.trainMotivation"},
		{"foreign rower", training, []TrainingAlloc{alloc("michel-a", StatKraft, 10)}, "NotOwned"},
		{"camp five rowers", camp, []TrainingAlloc{
			alloc("aw1", StatKraft, 6), alloc("aw2", StatKraft, 6), alloc("am1", StatKraft, 6),
			alloc("bw1", StatKraft, 6), alloc("bm1", StatKraft, 6),
		}, ""},
		{"camp six rowers", camp, []TrainingAlloc{
			alloc("aw1", StatKraft, 5), alloc("aw2", StatKraft, 5), alloc("am1", StatKraft, 5),
			alloc("bw1", StatKraft, 5), alloc("bm1", StatKraft, 5), alloc("bm2", StatKraft, 5),
		}, "err.campMaxRowers{count=6 maxRowers=5}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := g.validateTraining(pl, tt.p, tt.allocs)
			if (err == nil) != (tt.errPart == "") || (err != nil && !strings.Contains(err.Error(), tt.errPart)) {
				t.Fatalf("err = %v, want %q", err, tt.errPart)
			}
		})
	}

	g.applyTraining(pl, []TrainingAlloc{alloc("aw1", StatKraft, 4), alloc("aw1", StatKraft, 2)})

	if pl.rower("aw1").stats[StatKraft] != 16 {
		t.Errorf("training not applied: %d", pl.rower("aw1").stats[StatKraft])
	}
}

// Subtests share one game and its RNG and run in order on purpose.
//
//nolint:paralleltest,tparallel // see above
func TestEventEffects(t *testing.T) {
	t.Parallel()

	g := newGame(t, 5)
	pl := testPlayer(g)
	ev := func(typ string) *data.EventType { return g.d.EventType(typ) }

	t.Run("phoenix raises the weakest", func(t *testing.T) {
		r := pl.rower("aw1")
		r.stats = Stats{StatKraft: 20, StatAusdauer: 7, StatSpeed: 15, StatTechnik: 12, StatMotivation: 9}
		g.applyEvent(pl, r, ev("phoenix"))

		if r.stats[StatAusdauer] != 12 || r.event == nil || r.event.def.Type != "phoenix" {
			t.Errorf("stats %v", r.stats)
		}
	})
	t.Run("phoenix tie picks one tied stat", func(t *testing.T) {
		r := pl.rower("aw2")
		r.stats = Stats{StatKraft: 5, StatAusdauer: 5, StatSpeed: 15, StatTechnik: 12, StatMotivation: 9}
		g.applyEvent(pl, r, ev("phoenix"))

		if (r.stats[StatKraft] != 10) != (r.stats[StatAusdauer] == 10) || r.stats.sum() != 51 {
			t.Errorf("stats %v", r.stats)
		}
	})
	t.Run("verdopplung doubles the best", func(t *testing.T) {
		r := pl.rower("am1")
		r.stats = Stats{StatKraft: 20, StatAusdauer: 7, StatSpeed: 15, StatTechnik: 12, StatMotivation: 9}
		g.applyEvent(pl, r, ev("verdopplung"))

		if r.stats[StatKraft] != 40 {
			t.Errorf("stats %v", r.stats)
		}
	})
	t.Run("stat event", func(t *testing.T) {
		r := pl.rower("bw1")
		before := r.stats[StatTechnik]
		g.applyEvent(pl, r, ev("technik"))

		if r.stats[StatTechnik] != before+8 {
			t.Errorf("technik %d", r.stats[StatTechnik])
		}
	})
	t.Run("seelenzwilling links a same-gender, same-age partner", func(t *testing.T) {
		rw := pl.rower("bm1")
		n := len(pl.rowers)
		g.applyEvent(pl, rw, ev("seelenzwilling"))

		if len(pl.rowers) != n {
			t.Fatalf("seelenzwilling added a rower")
		}

		// bm2 is the only other B-junior man in the test team.
		if p := rw.event.partner; p == nil || p.id != "bm2" {
			t.Fatalf("partner %+v, want bm2", p)
		}
	})
	t.Run("seelenzwilling mirrors one training", func(t *testing.T) {
		rw, p := pl.rower("bm1"), pl.rower("bm2")
		rw.stats = Stats{StatKraft: 10, StatAusdauer: 10, StatSpeed: 10, StatTechnik: 10, StatMotivation: 10}
		p.stats = Stats{StatKraft: 10, StatAusdauer: 10, StatSpeed: 10, StatTechnik: 10, StatMotivation: 10}

		// A training without the pair leaves the card unspent.
		g.applyTraining(pl, []TrainingAlloc{{RowerID: "aw1", Stat: StatKraft, Points: 10}})

		if rw.event.spent || p.stats[StatKraft] != 10 {
			t.Fatalf("card spent %v, partner kraft %d", rw.event.spent, p.stats[StatKraft])
		}

		// Training the partner also trains the holder, and vice versa.
		g.applyTraining(pl, []TrainingAlloc{
			{RowerID: "bm2", Stat: StatKraft, Points: 4},
			{RowerID: "bm1", Stat: StatTechnik, Points: 6},
		})

		want := Stats{StatKraft: 14, StatAusdauer: 10, StatSpeed: 10, StatTechnik: 16, StatMotivation: 10}
		if !reflect.DeepEqual(rw.stats, want) || !reflect.DeepEqual(p.stats, want) {
			t.Fatalf("holder %v, partner %v, want both %v", rw.stats, p.stats, want)
		}

		if !rw.event.spent {
			t.Fatalf("card not spent")
		}

		// Only one training is mirrored.
		g.applyTraining(pl, []TrainingAlloc{{RowerID: "bm1", Stat: StatKraft, Points: 5}})

		if p.stats[StatKraft] != 14 {
			t.Errorf("second training mirrored: partner kraft %d", p.stats[StatKraft])
		}
	})
	t.Run("seelenzwilling falls back to the same gender", func(t *testing.T) {
		g2 := newGame(t, 5)
		pl2 := testPlayer(g2)
		rw := pl2.rower("km1")
		g2.applyEvent(pl2, rw, g2.d.EventType("seelenzwilling"))

		if p := rw.event.partner; p == nil || p.gender != rw.gender || p == rw {
			t.Errorf("partner %+v, want another man", p)
		}
	})
	t.Run("wachstum grows the best stat at training", func(t *testing.T) {
		r := pl.rower("km1")
		r.stats = Stats{StatKraft: 8, StatAusdauer: 10, StatSpeed: 9, StatTechnik: 12, StatMotivation: 9}
		g.applyEvent(pl, r, ev("wachstum"))
		g.applyTraining(pl, []TrainingAlloc{{RowerID: "kw1", Stat: StatKraft, Points: 10}})

		if r.stats[StatTechnik] != 14 {
			t.Errorf("wachstum: technik %d, want 14", r.stats[StatTechnik])
		}
	})
	t.Run("wachstum skips motivation", func(t *testing.T) {
		r := pl.rower("kw1")
		r.stats = Stats{StatKraft: 8, StatAusdauer: 10, StatSpeed: 9, StatTechnik: 12, StatMotivation: 30}
		g.applyEvent(pl, r, ev("wachstum"))
		g.applyTraining(pl, []TrainingAlloc{{RowerID: "aw1", Stat: StatKraft, Points: 10}})

		if r.stats[StatTechnik] != 14 || r.stats[StatMotivation] != 30 {
			t.Errorf("wachstum: technik %d, motivation %d, want 14 / 30", r.stats[StatTechnik], r.stats[StatMotivation])
		}
	})
	t.Run("tausch queues a swap phase", func(t *testing.T) {
		rw := pl.rower("kw2")
		rw.stats = Stats{StatKraft: 8, StatAusdauer: 10, StatSpeed: 9, StatTechnik: 12, StatMotivation: 9}
		g.applyEvent(pl, rw, ev("tausch"))

		if !g.openSwapPhase() {
			t.Fatal("no swap phase")
		}

		pv := g.CurrentPhase(pl.id)
		if pv.Kind != PhaseEventSwap || pv.SwapRowerID != "kw2" || pv.Waiting {
			t.Fatalf("phase %+v", pv)
		}

		if other := g.CurrentPhase(g.players[1].id); !other.Waiting {
			t.Errorf("other player must wait")
		}

		err := g.Submit(pl.id, Decision{PhaseID: pv.ID, Kind: PhaseEventSwap, Swap: &[2]Stat{StatKraft, StatKraft}})
		if err == nil {
			t.Errorf("same-stat swap accepted")
		}

		err = g.Submit(pl.id, Decision{PhaseID: pv.ID, Kind: PhaseEventSwap, Swap: &[2]Stat{StatMotivation, StatTechnik}})
		if err == nil || !strings.Contains(err.Error(), "Motivation") {
			t.Errorf("motivation swap: %v", err)
		}

		err = g.Submit(pl.id, Decision{PhaseID: pv.ID, Kind: PhaseEventSwap, Swap: &[2]Stat{StatKraft, StatTechnik}})
		if err != nil {
			t.Fatal(err)
		}

		if rw.stats[StatKraft] != 12 || rw.stats[StatTechnik] != 8 {
			t.Errorf("swap not applied: %v", rw.stats)
		}
	})
	t.Run("wetter event bonus only in bad weather", func(t *testing.T) {
		r := pl.rower("aw1")
		r.event = &eventCard{def: ev("wetter")}

		for _, w := range []Weather{WeatherNormal, WeatherRegen} {
			cs := compFor(g, "hildesheim", w)
			er := g.scoreEntry(cs, *cs.race("1x-a-w"), raceEntry{pl: pl, rowers: []*rower{r}})

			want := 0
			if w == WeatherRegen {
				want = 4
			}

			if er.EventBonus != want {
				t.Errorf("weather %s: bonus %d, want %d", w, er.EventBonus, want)
			}
		}
	})
}

func TestMissions(t *testing.T) {
	t.Parallel()

	g := newGame(t, 6)
	pl := testPlayer(g)
	rw := pl.rower("aw1")
	var ms *data.Mission

	for i := range g.d.Missions {
		if g.d.Missions[i].ID == "gold-einer" {
			ms = &g.d.Missions[i]
		}
	}

	rw.event = &eventCard{def: g.d.EventType("mission"), mission: &missionState{def: ms, rower: rw.name}}
	g.station = 2 // Hildesheim
	cs := compFor(g, "hildesheim", WeatherNormal)
	best := rw.stats.bestStat()
	before := rw.stats[best]

	g.checkMissions(cs, map[string]rowerOutcome{"aw1": {medal: MedalGold, class: Boat2x}})

	if rw.event.mission.done {
		t.Fatal("2x gold must not fulfil a 1x mission")
	}

	g.checkMissions(cs, map[string]rowerOutcome{"aw1": {medal: MedalGold, class: Boat1x}})

	if !rw.event.mission.done || rw.stats[best] != before+10 {
		t.Errorf("mission not rewarded: done=%v stat %d", rw.event.mission.done, rw.stats[best])
	}
	// Other players never see the mission text before it resolves.
	r2 := pl.rower("aw2")

	r2.event = &eventCard{def: g.d.EventType("mission"), mission: &missionState{def: ms, rower: r2.name}}
	for _, ps := range g.View(g.players[1].id).Players[0].Rowers {
		if ps.ID == "aw2" && ps.Event.Mission != nil {
			t.Errorf("secret mission visible to opponent")
		}
	}

	for _, ps := range g.View(pl.id).Players[0].Rowers {
		if ps.ID == "aw2" && (ps.Event.Mission == nil || ps.Event.Mission.ID != ms.ID) {
			t.Errorf("owner must see the mission")
		}
	}
}

func TestJTFOTeamScore(t *testing.T) {
	t.Parallel()

	g := newGame(t, 7)
	g.station = g.competitionIndex("jtfo")
	g.openCompetition()

	p := g.phase
	if p.comp == nil || !p.comp.team() || len(p.comp.races) != 3 || p.comp.races[1].Class != Boat4x {
		t.Fatalf("JTFO for 3 players must be variant B (2x+4x+2x), got %+v", p.comp.races)
	}

	rng := rand.New(rand.NewPCG(1, 2))
	for _, pid := range g.Pending() {
		err := g.Submit(pid, autoDecision(g, pid, rng))
		if err != nil {
			t.Fatal(err)
		}
	}

	res := g.results[len(g.results)-1]

	team := res.Races[len(res.Races)-1]
	if team.RaceID != "team" {
		t.Fatalf("last race should be the team result, got %s", team.RaceID)
	}

	golds := 0

	for _, te := range team.Entries {
		if te.Guest != "" {
			want := 0

			for _, race := range p.comp.races {
				ge, _ := g.guestEntry(p.comp, race)
				want += ge.Total
			}

			if te.Total != want || te.PlayerID != "" {
				t.Errorf("guest team %+v, want total %d", te, want)
			}

			if te.Medal == MedalGold {
				golds++
			}

			continue
		}

		sum := 0

		for _, rr := range res.Races[:len(res.Races)-1] {
			for _, e := range rr.Entries {
				if e.Medal != MedalNone {
					t.Errorf("single JTFO boats must not get medals")
				}

				if e.PlayerID == te.PlayerID && e.Incident != IncidentKentern && e.Incident != IncidentMaterial {
					sum += e.Total
				}
			}
		}

		if te.Total != sum {
			t.Errorf("%s: team total %d, want %d", te.PlayerID, te.Total, sum)
		}

		if te.Medal == MedalGold {
			golds++
		}
	}

	if golds == 0 {
		t.Errorf("no team gold awarded")
	}

	medals := 0
	for _, pl := range g.players {
		medals += pl.gold + pl.silver
	}

	if medals > len(g.players) {
		t.Errorf("more than one medal per team: %d", medals)
	}
}

func TestEstimateMatchesScore(t *testing.T) {
	t.Parallel()

	g := newGame(t, 8)
	for g.phase.kind != PhaseRegatta {
		rng := rand.New(rand.NewPCG(3, 4))
		for _, pid := range g.Pending() {
			err := g.Submit(pid, autoDecision(g, pid, rng))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	pl := g.players[0]
	cs := g.phase.comp
	rw := pl.rowers[0]
	var race *RaceSpec

	for i := range cs.races {
		if cs.races[i].Class == Boat1x && cs.races[i].Ages[0] == rw.age && cs.races[i].Genders[0] == rw.gender {
			race = &cs.races[i]
		}
	}

	if race == nil {
		t.Skip("no matching single for the top rower")
	}

	b := pl.boats[0]
	tr := pl.trainers[0]

	want := g.scoreEntry(cs, *race, raceEntry{pl: pl, rowers: []*rower{rw}, boat: b, trainer: tr}).Total
	if got := EstimateEntry(g, pl.id, race.ID, []string{rw.id}, b.id, tr.def.ID); got != want || got == 0 {
		t.Errorf("estimate %d, want %d", got, want)
	}

	if EstimateEntry(g, "nobody", race.ID, []string{rw.id}, b.id, "") != 0 {
		t.Errorf("unknown player must estimate 0")
	}
}

func TestStalePhaseAndWrongKind(t *testing.T) {
	t.Parallel()

	g := newGame(t, 9)

	id := g.PhaseID()

	err := g.Submit("p1", Decision{PhaseID: "bogus", Kind: PhaseAck})
	if !errors.Is(err, ErrStalePhase) {
		t.Errorf("want ErrStalePhase, got %v", err)
	}

	err = g.Submit("p1", Decision{PhaseID: id, Kind: PhaseTraining})
	if err == nil {
		t.Errorf("wrong kind accepted")
	}

	err = g.Submit("p1", Decision{PhaseID: id, Kind: PhaseAck})
	if err != nil {
		t.Fatal(err)
	}

	err = g.Submit("p1", Decision{PhaseID: id, Kind: PhaseAck})
	if err == nil {
		t.Errorf("double decision accepted")
	}

	if !g.CurrentPhase("p1").Waiting || g.CurrentPhase("p2").Waiting {
		t.Errorf("waiting flags wrong")
	}

	for _, pid := range []string{"p2", "p3"} {
		err := g.Submit(pid, Decision{PhaseID: id, Kind: PhaseAck})
		if err != nil {
			t.Fatal(err)
		}
	}

	err = g.Submit("p1", Decision{PhaseID: id, Kind: PhaseAck})
	if !errors.Is(err, ErrStalePhase) {
		t.Errorf("old phase id must be stale after advancing, got %v", err)
	}

	if pv := g.CurrentPhase("p1"); pv.Kind != PhaseTraining || pv.Points != 10 {
		t.Errorf("second phase = %+v, want training with 10 points", pv)
	}
}

func TestDraftOrderCatchUp(t *testing.T) {
	t.Parallel()

	g := newGame(t, 10)
	g.station = g.competitionIndex("bremen")
	g.players[0].gold = 3

	g.players[1].silver = 1
	if !g.openDraft() {
		t.Fatal("no draft")
	}

	want := []string{"p3", "p2", "p1"}
	if !reflect.DeepEqual(g.draft.order, want) {
		t.Errorf("order %v, want %v", g.draft.order, want)
	}

	if len(g.draft.offer) != 4 {
		t.Errorf("offer size %d, want 4", len(g.draft.offer))
	}

	pv := g.CurrentPhase("p3")
	if pv.Kind != PhaseDraft || pv.Waiting || !g.CurrentPhase("p1").Waiting {
		t.Fatalf("draft phase view wrong: %+v", pv)
	}

	poolBefore := len(g.pool)

	card := pv.DraftOffer[0].ID

	err := g.Submit("p1", Decision{PhaseID: pv.ID, Kind: PhaseDraft, DraftCardID: card})
	if err == nil {
		t.Errorf("out-of-turn pick accepted")
	}

	err = g.Submit("p3", Decision{PhaseID: pv.ID, Kind: PhaseDraft, DraftCardID: card})
	if err != nil {
		t.Fatal(err)
	}

	if len(g.pool) != poolBefore-1 || len(g.CurrentPhase("p2").DraftOffer) != 3 {
		t.Errorf("pick not removed from pool/offer")
	}
}

func TestReplayIsIdentical(t *testing.T) {
	t.Parallel()

	for _, seed := range []uint64{1, 42, 2026} {
		g := newGame(t, seed)
		playToEnd(t, g, seed)

		rg, err := Replay(g.Config(), g.Decisions())
		if err != nil {
			t.Fatal(err)
		}

		for _, pid := range []string{"p1", "p2", "p3"} {
			a := mustJSON(t, g.View(pid))

			b := mustJSON(t, rg.View(pid))
			if a != b {
				t.Fatalf("seed %d: replayed view for %s differs", seed, pid)
			}
		}

		a := mustJSON(t, g.Results())

		b := mustJSON(t, rg.Results())
		if a != b {
			t.Fatalf("seed %d: results differ", seed)
		}
		// The decision log survives a JSON round trip (what the API persists).
		raw := mustJSON(t, g.Decisions())

		var log []SubmittedDecision

		err = json.Unmarshal([]byte(raw), &log)
		if err != nil {
			t.Fatal(err)
		}

		_, err = Replay(g.Config(), log)
		if err != nil {
			t.Fatalf("seed %d: replay from JSON: %v", seed, err)
		}
	}
	// Same seed, same deal; different seed, different deal.
	a := mustJSON(t, newGame(t, 5).View("p1"))
	b := mustJSON(t, newGame(t, 5).View("p1"))

	c := mustJSON(t, newGame(t, 6).View("p1"))
	if a != b || a == c {
		t.Errorf("deal must depend on the seed only")
	}
}

func TestFullSeasonSmoke(t *testing.T) {
	t.Parallel()

	seeds := 200
	if testing.Short() {
		seeds = 20
	}

	for seed := uint64(1); seed <= uint64(seeds); seed++ {
		g := newGame(t, seed)
		playToEnd(t, g, seed)

		if len(g.Results()) != 12 {
			t.Fatalf("seed %d: %d competitions, want 12", seed, len(g.Results()))
		}

		if len(g.Winners()) == 0 {
			t.Fatalf("seed %d: no winner", seed)
		}

		v := g.View("p1")
		if v.Phase.Kind != PhaseGameOver || len(v.Winners) == 0 || v.Current != 51 {
			t.Fatalf("seed %d: end view %+v", seed, v.Phase)
		}

		for _, pid := range []string{"p1", "p2", "p3"} {
			err := g.Submit(pid, Decision{PhaseID: v.Phase.ID, Kind: PhaseGameOver})
			if err == nil {
				t.Fatalf("decision after game over accepted")
			}
		}
	}
}

func TestOtherPlayerCounts(t *testing.T) {
	t.Parallel()

	for _, np := range []int{2, 4} {
		cfg := Config{Seed: 11}
		for i := range np {
			cfg.Players = append(cfg.Players, PlayerConfig{ID: string(rune('a' + i))})
		}

		g, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}

		playToEnd(t, g, 11)
		jtfo := g.Results()[9]

		wantBoats := map[int]int{2: 3, 4: 3}[np]
		if len(jtfo.Races) != wantBoats+1 {
			t.Errorf("%d players: JTFO has %d races", np, len(jtfo.Races))
		}
	}
}

func TestGuestCrews(t *testing.T) {
	t.Parallel()

	g := newGame(t, 3)
	g.station = g.competitionIndex("bremen")
	g.openCompetition()
	cs := g.phase.comp
	race := *cs.race("4x-b-w")

	gc := g.d.Rules.Guests
	if !gc.Enabled {
		t.Skip("guest crews disabled in rules.json")
	}

	got, ok := GuestScore(g.d, cs.def, cs.station, race)

	wsum := 0.0
	for _, w := range cs.def.Weights {
		wsum += w
	}

	level := gc.StatLevel["b"] + gc.GrowthPerStation*float64(cs.station) + cs.def.GuestOffset

	want := math.Round(4*wsum*level) + float64(gc.BoatValue*g.d.Rules.BoatScale(4))
	if !ok || got != want {
		t.Fatalf("guest score %v (%v), want %v", got, ok, want)
	}
	// A lone entry far below the guest crew gets silver, the guest takes first place but no points.
	pl := g.players[0]
	before := g.points(pl)
	ge, _ := g.guestEntry(cs, race)
	es := []EntryResult{{PlayerID: pl.id, Total: int(want) - 50}, ge}
	rankEntries(es)

	for _, e := range es {
		g.awardMedal(e, race.Class, map[string]rowerOutcome{})
	}

	if es[0].Guest == "" || es[0].Place != 1 || es[1].Medal != MedalSilver ||
		g.points(pl) != before+g.d.Rules.MedalPoints.Silver {
		t.Errorf("ranking with guest: %+v, points %d -> %d", es, before, g.points(pl))
	}
}

// mustJSON marshals v for comparisons and fails the test on error.
func mustJSON(tb testing.TB, v any) string {
	tb.Helper()

	raw, err := json.Marshal(v)
	if err != nil {
		tb.Fatal(err)
	}

	return string(raw)
}

func TestSelectionAges(t *testing.T) {
	t.Parallel()

	g := newGame(t, 5)
	pl := testPlayer(g) // 6 Junioren (A/B) and 3 Kinder
	sel := g.d.Rules.Selections["wanderfahrt"]
	p := &phase{count: sel.Count, selection: &sel}

	if err := validateSelection(pl, p, []string{"aw1", "aw2", "am1", "bw1", "kw1"}); err == nil ||
		!strings.Contains(err.Error(), "err.selectAge") {
		t.Errorf("Kind accepted for the Wanderfahrt: %v", err)
	}

	if err := validateSelection(pl, p, []string{"aw1", "aw2", "am1", "bw1", "bm1"}); err != nil {
		t.Errorf("five Junioren rejected: %v", err)
	}

	// With only three Junioren, exactly those three must go.
	pl.rowers = pl.rowers[3:] // bw1, bm1, bm2, kw1, km1, kw2

	if got := selectionWant(pl, p); got != 3 {
		t.Errorf("want %d rowers, expected 3", got)
	}

	if err := validateSelection(pl, p, []string{"bw1", "bm1", "bm2"}); err != nil {
		t.Errorf("all three Junioren rejected: %v", err)
	}
}
