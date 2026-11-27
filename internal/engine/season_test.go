package engine

import (
	"maps"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"ruderspiel/internal/data"
)

// stationIndex returns the calendar index of the n-th (0-based) station of a type.
func stationIndex(tb testing.TB, g *Game, typ StationType, n int) int {
	tb.Helper()

	for i, st := range g.d.Calendar {
		if StationType(st.Type) == typ {
			if n == 0 {
				return i
			}

			n--
		}
	}

	tb.Fatalf("no %s station #%d", typ, n)

	return -1
}

// TestEreignisScope checks the season-aware Ereignis deck: water cards are never drawable while
// the next competition is an Ergo-Cup, ergo cards never before a water regatta, and modifiers
// without a station left to consume them are not drawn at all.
//
// Subtests share one game and run in order on purpose.
//
//nolint:paralleltest,tparallel // see above
func TestEreignisScope(t *testing.T) {
	t.Parallel()

	g := newGame(t, 11)

	for idx, st := range g.d.Calendar {
		if StationType(st.Type) != StationEreignis {
			continue
		}

		g.station = idx
		next := g.nextCompetition()

		for k := range g.d.Ereignisse {
			e := &g.d.Ereignisse[k]
			got := g.ereignisDrawable(e)

			switch {
			case e.Scope == data.ScopeWater && !next.Water:
				if got {
					t.Errorf("station %d (%s next): water card %s drawable", idx, next.Name, e.ID)
				}
			case e.Scope == data.ScopeErgo && next.Kind != string(StationErgoCup):
				if got {
					t.Errorf("station %d (%s next): ergo card %s drawable", idx, next.Name, e.ID)
				}
			case e.Effect == data.EffNone && !got:
				t.Errorf("station %d: fallback card %s not drawable", idx, e.ID)
			}
		}
	}

	tests := []struct {
		name    string
		station int // calendar index of the Ereignis station
		id      string
		want    bool
	}{
		{"monster before the first Ergo-Cup", stationIndex(t, g, StationEreignis, 0), "maschsee-monster", false},
		{"monster in the Ergo-Cup winter", stationIndex(t, g, StationEreignis, 2), "maschsee-monster", false},
		{"monster before Bremen", stationIndex(t, g, StationEreignis, 3), "maschsee-monster", true},
		{"ergometer before Osnabrück", stationIndex(t, g, StationEreignis, 2), "neue-ergometer", true},
		{"ergometer before Bremen", stationIndex(t, g, StationEreignis, 3), "neue-ergometer", false},
		{"livestream in the winter", stationIndex(t, g, StationEreignis, 1), "livestream-publikum", true},
		{"strike before the last training", stationIndex(t, g, StationEreignis, 9), "trainerstreik", true},
		{"strike after the last training", stationIndex(t, g, StationEreignis, 10), "trainerstreik", false},
		{"sponsor before the last regatta", stationIndex(t, g, StationEreignis, 10), "sponsor", false},
		{"strong guests any time", stationIndex(t, g, StationEreignis, 1), "starke-gaeste", true},
		{"weather before RVB", stationIndex(t, g, StationEreignis, 10), "kaiserwetter", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g.station = tc.station

			i := slices.IndexFunc(g.d.Ereignisse, func(e data.Ereignis) bool { return e.ID == tc.id })
			if i < 0 {
				t.Fatalf("no Ereignis %s", tc.id)
			}

			if got := g.ereignisDrawable(&g.d.Ereignisse[i]); got != tc.want {
				t.Errorf("drawable = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestNoWaterEreignisInErgoWinter draws many Ereignisse at the stations before the Ergo-Cups and
// checks that no water-only card comes up (and that ergo cards do).
func TestNoWaterEreignisInErgoWinter(t *testing.T) {
	t.Parallel()

	g := newGame(t, 12)
	byName := map[string]*data.Ereignis{}

	for i := range g.d.Ereignisse {
		byName[g.d.Ereignisse[i].Name] = &g.d.Ereignisse[i]
	}

	for n := range 3 {
		g.station = stationIndex(t, g, StationEreignis, n)
		if g.nextCompetition().Kind != string(StationErgoCup) {
			t.Fatalf("Ereignis station %d is not before an Ergo-Cup", n)
		}

		ergo := 0

		for range 400 {
			g.pending = nil

			name, _ := g.runEreignisStation()

			e := byName[name]
			if e.Scope == data.ScopeWater {
				t.Fatalf("water Ereignis %s drawn before %s", name, g.nextCompetition().Name)
			}

			if e.Scope == data.ScopeErgo {
				ergo++
			}
		}

		if ergo == 0 {
			t.Errorf("no ergo Ereignis in 400 draws at station %d", g.station)
		}
	}
}

// TestCompetitionEreignisse checks that pending modifiers are consumed only by a competition of
// their scope and change the competition as described.
//
// Subtests share one game and run in order on purpose.
//
//nolint:paralleltest,tparallel // see above
func TestCompetitionEreignisse(t *testing.T) {
	t.Parallel()

	g := newGame(t, 13)
	card := func(id string) *data.Ereignis {
		for i := range g.d.Ereignisse {
			if g.d.Ereignisse[i].ID == id {
				return &g.d.Ereignisse[i]
			}
		}

		t.Fatalf("no Ereignis %s", id)

		return nil
	}

	open := func(comp string, pending ...string) *compState {
		g.pending = nil
		for _, id := range pending {
			g.pending = append(g.pending, card(id))
		}

		cs := compFor(g, comp, "")
		cs.krebs, cs.material = g.d.Rules.Incidents.Krebs, g.d.Rules.Incidents.Material
		g.applyCompetitionEreignisse(cs)

		return cs
	}

	t.Run("ergo modifiers wait for an Ergo-Cup", func(t *testing.T) {
		cs := open("bremen", "neue-ergometer", "hallenluft")
		if cs.weights[StatKraft] != 0 || cs.weights[StatAusdauer] != 0 || len(g.pending) != 2 {
			t.Errorf("weights %v, pending %d", cs.weights, len(g.pending))
		}
	})
	t.Run("ergo weight modifiers add up", func(t *testing.T) {
		cs := open("drc", "neue-ergometer", "livestream-publikum", "hallenluft")

		want := map[Stat]float64{StatKraft: 3, StatMotivation: 2, StatAusdauer: 1}
		if !reflect.DeepEqual(cs.weights, want) || len(g.pending) != 0 {
			t.Errorf("weights %v, pending %d", cs.weights, len(g.pending))
		}
	})
	t.Run("water modifiers", func(t *testing.T) {
		cs := open("kassel", "treibgut", "spiegelglatter-see", "dauerregen", "bootshaus-renovierung", "rueckenwind")
		if cs.material != 100 || cs.krebs != 25 || cs.weather != WeatherRegen || cs.cityBoatMod != -1 ||
			cs.weights[StatSpeed] != 2 {
			t.Errorf("cs %+v", cs)
		}
	})
	t.Run("the most recent forced weather wins", func(t *testing.T) {
		cs := open("vr1", "hitzewelle", "kaiserwetter")
		if cs.weather != WeatherNormal {
			t.Errorf("weather %s", cs.weather)
		}
	})
	t.Run("guest modifier", func(t *testing.T) {
		race := compFor(g, "mindoor", "").races[0]
		base, _ := GuestScore(g.d, g.d.Competitions["mindoor"], 0, race)

		cs := open("mindoor", "starke-gaeste")

		got, _ := g.guestEntry(cs, race)
		if want := int(base) + 2*3; got.Total != want { // 1 seat × (Kraft 2 + Motivation 1) × 2
			t.Errorf("guest total %d, want %d", got.Total, want)
		}
	})
}

// TestEventScopeAndBonus checks the data-driven event cards: ergo cards are only dealt while an
// Ergo-Cup is ahead, and entry bonuses only apply at matching competitions and weathers.
//
// Subtests share one game and run in order on purpose.
//
//nolint:paralleltest,tparallel // see above
func TestEventScopeAndBonus(t *testing.T) {
	t.Parallel()

	g := newGame(t, 14)
	ev := func(typ string) *data.EventType {
		e := g.d.EventType(typ)
		if e == nil {
			t.Fatalf("no event type %s", typ)
		}

		return e
	}

	draw := []struct {
		name    string
		station int
		typ     string
		want    bool
	}{
		{"ergo card before the Ergo-Cups", stationIndex(t, g, StationEvent, 0), "bestzeit-fieber", true},
		{"ergo card before the last Ergo-Cup", stationIndex(t, g, StationEvent, 2), "livestream-liebling", true},
		{"ergo card after the last Ergo-Cup", stationIndex(t, g, StationEvent, 3), "bestzeit-fieber", false},
		{"sprint card before RVB", stationIndex(t, g, StationEvent, 10), "spurtstark", true},
		{"water card in the winter", stationIndex(t, g, StationEvent, 1), "regenjacke", true},
		{"stat card any time", stationIndex(t, g, StationEvent, 10), "kraftraum", true},
	}
	for _, tc := range draw {
		t.Run(tc.name, func(t *testing.T) {
			g.station = tc.station
			if got := g.eventDrawable(ev(tc.typ)); got != tc.want {
				t.Errorf("drawable = %v, want %v", got, tc.want)
			}
		})
	}

	bonus := []struct {
		typ     string
		comp    string
		weather Weather
		want    int
	}{
		{"wetter", "bremen", WeatherRegen, 4},
		{"wetter", "bremen", WeatherNormal, 0},
		{"wetter", "drc", "", 0},
		{"regenjacke", "kassel", WeatherRegen, 8},
		{"regenjacke", "kassel", WeatherWind, 0},
		{"eisbaden", "jtfo", WeatherKaelte, 8},
		{"maschsee-kenner", "vr2", WeatherNormal, 3},
		{"maschsee-kenner", "osnabrueck", "", 0},
		{"bestzeit-fieber", "mindoor", "", 6},
		{"bestzeit-fieber", "rvh", WeatherNormal, 0},
		{"spurtstark", "rvh", WeatherWind, 6},
		{"spurtstark", "bremen", WeatherNormal, 0},
		{"kraftraum", "drc", "", 0},
	}
	for _, tc := range bonus {
		t.Run(tc.typ+"@"+tc.comp+"/"+string(tc.weather), func(t *testing.T) {
			if got := eventBonus(ev(tc.typ), compFor(g, tc.comp, tc.weather)); got != tc.want {
				t.Errorf("bonus %d, want %d", got, tc.want)
			}
		})
	}

	t.Run("immediate stats", func(t *testing.T) {
		pl := testPlayer(g)
		r := pl.rower("bw1")
		g.applyEvent(pl, r, ev("kraftraum"))

		if r.stats[StatKraft] != 20 || r.stats[StatMotivation] != 8 {
			t.Errorf("stats %v", r.stats)
		}
	})
}

// TestTrainerMood plays full seasons and checks the mood walk: it starts at 0, changes only when a
// new section begins, never leaves [min, max], moves by at most the largest step, is part of the
// trainer score and is identical for the same seed.
//
// Subtests share one game and run in order on purpose.
//
//nolint:tparallel // see above
func TestTrainerMood(t *testing.T) {
	t.Parallel()

	cfg := data.MustLoad().Rules.TrainerMood
	maxStep := 0

	for _, s := range cfg.Steps {
		maxStep = max(maxStep, abs(s))
	}

	trace := func(seed uint64) []map[string]int {
		g := newGame(t, seed)
		for id, m := range g.mood {
			if m != cfg.Start {
				t.Fatalf("seed %d: %s starts with mood %d", seed, id, m)
			}
		}

		rng := rand.New(rand.NewPCG(seed, 77))
		out := []map[string]int{maps.Clone(g.mood)}
		last := g.station

		for steps := 0; !g.Over(); steps++ {
			if steps > 5000 {
				t.Fatal("no game over")
			}

			for _, pid := range g.Pending() {
				err := g.Submit(pid, autoDecision(g, pid, rng))
				if err != nil {
					t.Fatal(err)
				}
			}

			if g.station == last {
				continue
			}

			prev := out[len(out)-1]
			newSection := g.d.Calendar[g.station].Section != g.d.Calendar[last].Section
			last = g.station

			for id, m := range g.mood {
				if m < cfg.Min || m > cfg.Max {
					t.Fatalf("seed %d: %s mood %d outside [%d, %d]", seed, id, m, cfg.Min, cfg.Max)
				}

				if abs(m-prev[id]) > maxStep || (!newSection && m != prev[id]) {
					t.Fatalf("seed %d station %d: %s mood %d -> %d", seed, g.station, id, prev[id], m)
				}

				if tv := g.trainerView(&g.d.Trainers[0]); tv.MoodTrend != g.mood[tv.ID]-g.prevMood[tv.ID] {
					t.Fatalf("trend %d", tv.MoodTrend)
				}
			}

			out = append(out, maps.Clone(g.mood))
		}

		return out
	}

	moved := false

	for _, seed := range []uint64{1, 2, 3} {
		a, b := trace(seed), trace(seed)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("seed %d: mood not deterministic", seed)
		}

		for _, m := range a[len(a)-1] {
			moved = moved || m != 0
		}
	}

	if !moved {
		t.Error("no trainer mood ever moved")
	}

	t.Run("mood is added per seat", func(t *testing.T) {
		g := newGame(t, 15)
		pl := testPlayer(g)
		tr := pl.trainers[0]
		cs := compFor(g, "kassel", WeatherNormal)
		race := *cs.race("4x-a-w")
		re := raceEntry{pl: pl, trainer: tr, rowers: []*rower{
			pl.rower("aw1"), pl.rower("aw2"), pl.rower("bw1"), pl.rower("kw1"),
		}}

		base := g.scoreEntry(cs, race, re).Trainer
		g.mood[tr.def.ID] = -2

		if got := g.scoreEntry(cs, race, re).Trainer; got != base-2*g.d.Rules.TrainerScale(4) {
			t.Errorf("trainer part %d, want %d", got, base-2*g.d.Rules.TrainerScale(4))
		}
	})
}

func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}
