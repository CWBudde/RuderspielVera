package engine

import (
	"encoding/json"
	"errors"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"ruderspiel/internal/data"
)

func newGame(t testing.TB, seed uint64) *Game {
	t.Helper()
	g, err := New(threePlayers(seed))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestNewDeal(t *testing.T) {
	g := newGame(t, 1)
	seen := map[string]bool{}
	for _, pl := range g.players {
		if len(pl.rowers) != 9 {
			t.Errorf("%s: %d rowers, want 9", pl.id, len(pl.rowers))
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
			if perAge[age][GenderF]+perAge[age][GenderM] != 3 || perAge[age][GenderF] == 0 || perAge[age][GenderM] == 0 {
				t.Errorf("%s: age %s split %v", pl.id, age, perAge[age])
			}
		}
		if len(pl.boats) != 3 || len(pl.trainers) != 2 {
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
	if rowers != 28 || boats != 21 || trainers != 4 {
		t.Errorf("pool = %d rowers, %d boats, %d trainers; want 28/21/4", rowers, boats, trainers)
	}
	if g.CurrentPhase("p1").Kind != PhaseAck {
		t.Errorf("first phase should be the setup ack")
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	for _, cfg := range []Config{
		{Players: []PlayerConfig{{ID: "a"}}},
		{Players: []PlayerConfig{{ID: "a"}, {ID: "a"}}},
		{Players: []PlayerConfig{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "e"}}},
	} {
		if _, err := New(cfg); err == nil {
			t.Errorf("New(%v) should fail", cfg.Players)
		}
	}
}

func TestRankEntries(t *testing.T) {
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
		{"clear order", []in{{90, ""}, {100, ""}, {80, ""}},
			[]want{{1, MedalGold}, {2, MedalSilver}, {3, ""}}},
		{"two golds skip silver", []in{{100, ""}, {100, ""}, {90, ""}},
			[]want{{1, MedalGold}, {1, MedalGold}, {3, ""}}},
		{"shared silver", []in{{100, ""}, {90, ""}, {90, ""}},
			[]want{{1, MedalGold}, {2, MedalSilver}, {2, MedalSilver}}},
		{"krebs still finishes", []in{{95, IncidentKrebs}, {90, ""}},
			[]want{{1, MedalGold}, {2, MedalSilver}}},
		{"kentern last, no medal", []in{{200, IncidentKentern}, {90, ""}, {80, ""}},
			[]want{{1, MedalGold}, {2, MedalSilver}, {3, ""}}},
		{"two capsized share last", []in{{200, IncidentKentern}, {150, IncidentKentern}, {80, ""}},
			[]want{{1, MedalGold}, {2, ""}, {2, ""}}},
		{"material is DNF", []in{{200, IncidentMaterial}, {90, ""}},
			[]want{{1, MedalGold}, {0, ""}}},
		{"single capsized starter gets nothing", []in{{100, IncidentKentern}},
			[]want{{1, ""}}},
		{"single finisher gets gold", []in{{10, ""}},
			[]want{{1, MedalGold}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestGoldMotivation(t *testing.T) {
	g := newGame(t, 2)
	pl := g.players[0]
	crew := []string{pl.rowers[0].id, pl.rowers[1].id}
	before := []int{pl.rowers[0].stats[StatMotivation], pl.rowers[1].stats[StatMotivation]}
	other := pl.rowers[2].stats[StatMotivation]
	out := map[string]rowerOutcome{}
	g.awardMedal(EntryResult{PlayerID: pl.id, RowerIDs: crew, Medal: MedalGold}, Boat2x, out)
	for i := range crew {
		if got := pl.rowers[i].stats[StatMotivation]; got != before[i]+1 {
			t.Errorf("crew %d motivation %d, want %d", i, got, before[i]+1)
		}
	}
	if pl.rowers[2].stats[StatMotivation] != other {
		t.Errorf("non-crew rower changed")
	}
	if pl.gold != 1 || g.points(pl) != 2 {
		t.Errorf("gold %d points %d, want 1/2", pl.gold, g.points(pl))
	}
	g.awardMedal(EntryResult{PlayerID: pl.id, RowerIDs: crew[:1], Medal: MedalSilver}, Boat1x, out)
	if pl.silver != 1 || g.points(pl) != 3 || pl.rowers[0].stats[StatMotivation] != before[0]+1 {
		t.Errorf("silver must add 1 point and no motivation")
	}
	if out[crew[0]].medal != MedalSilver || out[crew[1]].class != Boat2x {
		t.Errorf("outcomes not recorded: %v", out)
	}
}

func TestIncidentDistribution(t *testing.T) {
	const n = 200000
	counts := map[IncidentKind]int{}
	for seed := uint64(0); seed < 20; seed++ {
		rng := newRNG(seed)
		for i := 0; i < n/20; i++ {
			counts[rollIncident(rng, 5, 10, 15)]++
		}
	}
	want := map[IncidentKind]float64{IncidentKentern: 0.05, IncidentMaterial: 0.10, IncidentKrebs: 0.15, IncidentNone: 0.70}
	for k, p := range want {
		got := float64(counts[k]) / n
		if got < p-0.005 || got > p+0.005 {
			t.Errorf("%q: %.4f, want %.2f ± 0.005", k, got, p)
		}
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
	g := newGame(t, 3)
	pl := testPlayer(g)
	tr := pl.trainers[0].def.ID
	b1, b2, b4 := pl.boats[0].id, pl.boats[1].id, pl.boats[2].id
	hild := compFor(g, "hildesheim", WeatherNormal)
	rvb := compFor(g, "rvb", WeatherNormal)
	ergo := compFor(g, "mindor", "")
	jtfo := compFor(g, "jtfo", WeatherNormal)

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
			{RaceID: "2x-a-w", RowerIDs: []string{"aw2", "bw1"}, BoatID: b2}}, ""},
		{"no entries no trainer", hild, "", nil, ""},
		{"trainer required", hild, "", []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1}}, "Trainer"},
		{"foreign trainer", hild, "nobody", nil, "Trainer"},
		{"unknown race", hild, tr, []Entry{{RaceID: "nope", RowerIDs: []string{"aw1"}, BoatID: b1}}, "Unbekanntes Rennen"},
		{"wrong age", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"bw1"}, BoatID: b1}}, "Altersklasse"},
		{"row up in crew boat", hild, tr, []Entry{{RaceID: "2x-a-w", RowerIDs: []string{"aw1", "bw1"}, BoatID: b2}}, ""},
		{"wrong gender", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"am1"}, BoatID: b1}}, "Geschlecht"},
		{"wrong seats", hild, tr, []Entry{{RaceID: "2x-a-w", RowerIDs: []string{"aw1"}, BoatID: b2}}, "genau 2"},
		{"rower twice", hild, tr, []Entry{
			{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1},
			{RaceID: "2x-a-w", RowerIDs: []string{"aw1", "aw2"}, BoatID: b2}}, "nur einmal starten"},
		{"boat twice", hild, tr, []Entry{
			{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1},
			{RaceID: "1x-a-m", RowerIDs: []string{"am1"}, BoatID: b1}}, "nur einmal eingesetzt"},
		{"race twice", hild, tr, []Entry{
			{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b1},
			{RaceID: "1x-a-w", RowerIDs: []string{"aw2"}, BoatID: b1}}, "nur eine Meldung"},
		{"wrong boat class", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: b2}}, "Klasse 1x"},
		{"missing boat", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}}}, "Wähle ein Boot"},
		{"foreign boat", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"aw1"}, BoatID: "ice-man"}}, "gehört nicht"},
		{"foreign rower", hild, tr, []Entry{{RaceID: "1x-a-w", RowerIDs: []string{"michel-a"}, BoatID: b1}}, "gehört nicht"},
		{"ergo no boat", ergo, tr, []Entry{{RaceID: "ergo-kinder-w", RowerIDs: []string{"kw1"}}}, ""},
		{"ergo with boat", ergo, tr, []Entry{{RaceID: "ergo-kinder-w", RowerIDs: []string{"kw1"}, BoatID: b1}}, "kein eigenes Boot"},
		{"mixed ok", rvb, tr, []Entry{{RaceID: "mixed-4x", RowerIDs: []string{"aw1", "am1", "bw1", "kw1"}, BoatID: b4}}, ""},
		{"mixed one gender", rvb, tr, []Entry{{RaceID: "mixed-4x", RowerIDs: []string{"aw1", "aw2", "bw1", "kw1"}, BoatID: b4}}, "Mixed"},
		{"mixed one age", rvb, tr, []Entry{{RaceID: "mixed-4x", RowerIDs: []string{"bw1", "bm1", "bm2", "bm1"}, BoatID: b4}}, "nur einmal"},
		{"mixed same age", compFor(g, "rvb", WeatherNormal), tr, []Entry{{RaceID: "mixed-4x", RowerIDs: []string{"aw1", "aw2", "am1", "am1"}, BoatID: b4}}, "nur einmal"},
		{"eight without boat", rvb, tr, []Entry{{RaceID: "mixed-8x",
			RowerIDs: []string{"aw1", "aw2", "am1", "bw1", "bm1", "bm2", "kw1", "km1"}}}, ""},
		{"jtfo loan boats", jtfo, tr, []Entry{
			{RaceID: "jtfo-1-2x", RowerIDs: []string{"aw1", "am1"}, BoatID: b2},
			{RaceID: "jtfo-3-2x", RowerIDs: []string{"bw1", "bm1"}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	pl.rowers = append(pl.rowers, &rower{id: "bw2", name: "bw2", age: AgeB, gender: GenderF, stats: Stats{}, base: Stats{}})
	err := g.validateEntries(pl, rvb, tr, []Entry{{RaceID: "mixed-4x", RowerIDs: []string{"bw1", "bw2", "bm1", "bm2"}, BoatID: b4}})
	if err == nil || !strings.Contains(err.Error(), "Mixed") {
		t.Errorf("single-age mixed crew accepted: %v", err)
	}
}

func TestValidateTraining(t *testing.T) {
	g := newGame(t, 4)
	pl := testPlayer(g)
	training := &phase{points: 10}
	camp := &phase{points: 30, maxRowers: 5}
	a := func(id string, s Stat, p int) TrainingAlloc { return TrainingAlloc{RowerID: id, Stat: s, Points: p} }
	tests := []struct {
		name    string
		p       *phase
		allocs  []TrainingAlloc
		errPart string
	}{
		{"exact", training, []TrainingAlloc{a("aw1", StatKraft, 4), a("bw1", StatTechnik, 3), a("kw1", StatSpeed, 3)}, ""},
		{"too few", training, []TrainingAlloc{a("aw1", StatKraft, 9)}, "genau 10"},
		{"too many", training, []TrainingAlloc{a("aw1", StatKraft, 11)}, "genau 10"},
		{"negative", training, []TrainingAlloc{a("aw1", StatKraft, 12), a("aw2", StatKraft, -2)}, "positiv"},
		{"bad stat", training, []TrainingAlloc{a("aw1", "magie", 10)}, "Unbekannte Eigenschaft"},
		{"foreign rower", training, []TrainingAlloc{a("michel-a", StatKraft, 10)}, "gehört nicht"},
		{"camp five rowers", camp, []TrainingAlloc{a("aw1", StatKraft, 6), a("aw2", StatKraft, 6), a("am1", StatKraft, 6),
			a("bw1", StatKraft, 6), a("bm1", StatKraft, 6)}, ""},
		{"camp six rowers", camp, []TrainingAlloc{a("aw1", StatKraft, 5), a("aw2", StatKraft, 5), a("am1", StatKraft, 5),
			a("bw1", StatKraft, 5), a("bm1", StatKraft, 5), a("bm2", StatKraft, 5)}, "höchstens 5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.validateTraining(pl, tt.p, tt.allocs)
			if (err == nil) != (tt.errPart == "") || (err != nil && !strings.Contains(err.Error(), tt.errPart)) {
				t.Fatalf("err = %v, want %q", err, tt.errPart)
			}
		})
	}
	g.applyTraining(pl, []TrainingAlloc{a("aw1", StatKraft, 4), a("aw1", StatKraft, 2)})
	if pl.rower("aw1").stats[StatKraft] != 16 {
		t.Errorf("training not applied: %d", pl.rower("aw1").stats[StatKraft])
	}
}

func TestEventEffects(t *testing.T) {
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
		if !(r.stats[StatKraft] == 10) != (r.stats[StatAusdauer] == 10) || r.stats.sum() != 51 {
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
	t.Run("seelenzwilling adds a twin", func(t *testing.T) {
		r := pl.rower("bm1")
		r.stats = Stats{StatKraft: 20, StatAusdauer: 10, StatSpeed: 15, StatTechnik: 12, StatMotivation: 9}
		n := len(pl.rowers)
		g.applyEvent(pl, r, ev("seelenzwilling"))
		if len(pl.rowers) != n+1 {
			t.Fatalf("no twin added")
		}
		tw := pl.rowers[n]
		want := Stats{StatKraft: 16, StatAusdauer: 8, StatSpeed: 12, StatTechnik: 10, StatMotivation: 7}
		if !tw.isTwin || tw.age != r.age || tw.gender != r.gender || !reflect.DeepEqual(tw.stats, want) {
			t.Errorf("twin %+v stats %v", tw, tw.stats)
		}
		for _, e := range eventEligible(pl) {
			if e == tw || e == r {
				t.Errorf("twin or carded rower still eligible for events")
			}
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
	t.Run("tausch queues a swap phase", func(t *testing.T) {
		r := pl.rower("kw2")
		r.stats = Stats{StatKraft: 8, StatAusdauer: 10, StatSpeed: 9, StatTechnik: 12, StatMotivation: 9}
		g.applyEvent(pl, r, ev("tausch"))
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
		if err := g.Submit(pl.id, Decision{PhaseID: pv.ID, Kind: PhaseEventSwap, Swap: &[2]Stat{StatKraft, StatKraft}}); err == nil {
			t.Errorf("same-stat swap accepted")
		}
		if err := g.Submit(pl.id, Decision{PhaseID: pv.ID, Kind: PhaseEventSwap, Swap: &[2]Stat{StatKraft, StatTechnik}}); err != nil {
			t.Fatal(err)
		}
		if r.stats[StatKraft] != 12 || r.stats[StatTechnik] != 8 {
			t.Errorf("swap not applied: %v", r.stats)
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
	g := newGame(t, 6)
	pl := testPlayer(g)
	r := pl.rower("aw1")
	var m *data.Mission
	for i := range g.d.Missions {
		if g.d.Missions[i].ID == "gold-einer" {
			m = &g.d.Missions[i]
		}
	}
	r.event = &eventCard{def: g.d.EventType("mission"), mission: &missionState{def: m, text: "x"}}
	g.station = 2 // Hildesheim
	cs := compFor(g, "hildesheim", WeatherNormal)
	best := r.stats.bestStat()
	before := r.stats[best]
	g.checkMissions(cs, map[string]rowerOutcome{"aw1": {medal: MedalGold, class: Boat2x}})
	if r.event.mission.done {
		t.Fatal("2x gold must not fulfil a 1x mission")
	}
	g.checkMissions(cs, map[string]rowerOutcome{"aw1": {medal: MedalGold, class: Boat1x}})
	if !r.event.mission.done || r.stats[best] != before+10 {
		t.Errorf("mission not rewarded: done=%v stat %d", r.event.mission.done, r.stats[best])
	}
	// Other players never see the mission text before it resolves.
	r2 := pl.rower("aw2")
	r2.event = &eventCard{def: g.d.EventType("mission"), mission: &missionState{def: m, text: "geheim"}}
	for _, ps := range g.View(g.players[1].id).Players[0].Rowers {
		if ps.ID == "aw2" && ps.Event.Mission != nil {
			t.Errorf("secret mission visible to opponent")
		}
	}
	for _, ps := range g.View(pl.id).Players[0].Rowers {
		if ps.ID == "aw2" && (ps.Event.Mission == nil || ps.Event.Mission.Text != "geheim") {
			t.Errorf("owner must see the mission")
		}
	}
}

func TestJTFOTeamScore(t *testing.T) {
	g := newGame(t, 7)
	g.station = g.competitionIndex("jtfo")
	g.openCompetition()
	p := g.phase
	if p.comp == nil || !p.comp.team() || len(p.comp.races) != 3 || p.comp.races[1].Class != Boat4x {
		t.Fatalf("JTFO for 3 players must be variant B (2x+4x+2x), got %+v", p.comp.races)
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for _, pid := range g.Pending() {
		if err := g.Submit(pid, autoDecision(g, pid, rng)); err != nil {
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
	g := newGame(t, 8)
	for g.phase.kind != PhaseRegatta {
		rng := rand.New(rand.NewPCG(3, 4))
		for _, pid := range g.Pending() {
			if err := g.Submit(pid, autoDecision(g, pid, rng)); err != nil {
				t.Fatal(err)
			}
		}
	}
	pl := g.players[0]
	cs := g.phase.comp
	r := pl.rowers[0]
	var race *RaceSpec
	for i := range cs.races {
		if cs.races[i].Class == Boat1x && cs.races[i].Ages[0] == r.age && cs.races[i].Genders[0] == r.gender {
			race = &cs.races[i]
		}
	}
	if race == nil {
		t.Skip("no matching single for the top rower")
	}
	b := pl.boats[0]
	tr := pl.trainers[0]
	want := g.scoreEntry(cs, *race, raceEntry{pl: pl, rowers: []*rower{r}, boat: b, trainer: tr}).Total
	if got := EstimateEntry(g, pl.id, race.ID, []string{r.id}, b.id, tr.def.ID); got != want || got == 0 {
		t.Errorf("estimate %d, want %d", got, want)
	}
	if EstimateEntry(g, "nobody", race.ID, []string{r.id}, b.id, "") != 0 {
		t.Errorf("unknown player must estimate 0")
	}
}

func TestStalePhaseAndWrongKind(t *testing.T) {
	g := newGame(t, 9)
	id := g.PhaseID()
	if err := g.Submit("p1", Decision{PhaseID: "bogus", Kind: PhaseAck}); !errors.Is(err, ErrStalePhase) {
		t.Errorf("want ErrStalePhase, got %v", err)
	}
	if err := g.Submit("p1", Decision{PhaseID: id, Kind: PhaseTraining}); err == nil {
		t.Errorf("wrong kind accepted")
	}
	if err := g.Submit("p1", Decision{PhaseID: id, Kind: PhaseAck}); err != nil {
		t.Fatal(err)
	}
	if err := g.Submit("p1", Decision{PhaseID: id, Kind: PhaseAck}); err == nil {
		t.Errorf("double decision accepted")
	}
	if !g.CurrentPhase("p1").Waiting || g.CurrentPhase("p2").Waiting {
		t.Errorf("waiting flags wrong")
	}
	for _, pid := range []string{"p2", "p3"} {
		if err := g.Submit(pid, Decision{PhaseID: id, Kind: PhaseAck}); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.Submit("p1", Decision{PhaseID: id, Kind: PhaseAck}); !errors.Is(err, ErrStalePhase) {
		t.Errorf("old phase id must be stale after advancing, got %v", err)
	}
	if pv := g.CurrentPhase("p1"); pv.Kind != PhaseTraining || pv.Points != 10 {
		t.Errorf("second phase = %+v, want training with 10 points", pv)
	}
}

func TestDraftOrderCatchUp(t *testing.T) {
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
	if err := g.Submit("p1", Decision{PhaseID: pv.ID, Kind: PhaseDraft, DraftCardID: card}); err == nil {
		t.Errorf("out-of-turn pick accepted")
	}
	if err := g.Submit("p3", Decision{PhaseID: pv.ID, Kind: PhaseDraft, DraftCardID: card}); err != nil {
		t.Fatal(err)
	}
	if len(g.pool) != poolBefore-1 || len(g.CurrentPhase("p2").DraftOffer) != 3 {
		t.Errorf("pick not removed from pool/offer")
	}
}

func TestReplayIsIdentical(t *testing.T) {
	for _, seed := range []uint64{1, 42, 2026} {
		g := newGame(t, seed)
		playToEnd(t, g, seed)
		r, err := Replay(g.Config(), g.Decisions())
		if err != nil {
			t.Fatal(err)
		}
		for _, pid := range []string{"p1", "p2", "p3"} {
			a, _ := json.Marshal(g.View(pid))
			b, _ := json.Marshal(r.View(pid))
			if string(a) != string(b) {
				t.Fatalf("seed %d: replayed view for %s differs", seed, pid)
			}
		}
		a, _ := json.Marshal(g.Results())
		b, _ := json.Marshal(r.Results())
		if string(a) != string(b) {
			t.Fatalf("seed %d: results differ", seed)
		}
		// The decision log survives a JSON round trip (what the API persists).
		raw, _ := json.Marshal(g.Decisions())
		var log []SubmittedDecision
		if err := json.Unmarshal(raw, &log); err != nil {
			t.Fatal(err)
		}
		if _, err := Replay(g.Config(), log); err != nil {
			t.Fatalf("seed %d: replay from JSON: %v", seed, err)
		}
	}
	// Same seed, same deal; different seed, different deal.
	a, _ := json.Marshal(newGame(t, 5).View("p1"))
	b, _ := json.Marshal(newGame(t, 5).View("p1"))
	c, _ := json.Marshal(newGame(t, 6).View("p1"))
	if string(a) != string(b) || string(a) == string(c) {
		t.Errorf("deal must depend on the seed only")
	}
}

func TestFullSeasonSmoke(t *testing.T) {
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
			if err := g.Submit(pid, Decision{PhaseID: v.Phase.ID, Kind: PhaseGameOver}); err == nil {
				t.Fatalf("decision after game over accepted")
			}
		}
	}
}

func TestOtherPlayerCounts(t *testing.T) {
	for _, n := range []int{2, 4} {
		cfg := Config{Seed: 11}
		for i := 0; i < n; i++ {
			cfg.Players = append(cfg.Players, PlayerConfig{ID: string(rune('a' + i))})
		}
		g, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		playToEnd(t, g, 11)
		jtfo := g.Results()[9]
		wantBoats := map[int]int{2: 3, 4: 3}[n]
		if len(jtfo.Races) != wantBoats+1 {
			t.Errorf("%d players: JTFO has %d races", n, len(jtfo.Races))
		}
	}
}
