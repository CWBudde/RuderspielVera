package bot

import (
	"maps"
	"math"
	"slices"

	"ruderspiel/internal/data"
	"ruderspiel/internal/engine"
)

// team is the card set of one player as the bot sees it (possibly with a hypothetical card).
type team struct {
	id       string
	rowers   []engine.RowerView
	boats    []engine.BoatView
	trainers []engine.TrainerView
}

// compModel is a competition as the bot models it: the open one (exact, with weather) or a
// future one (from the public rule data, weather unknown).
type compModel struct {
	station int
	def     *data.Competition
	races   []engine.RaceSpec
	weights map[engine.Stat]float64
	weather engine.Weather
	current bool
	ahead   int // future only: competitions between now and this one
}

// at returns the team as it will be at c: a rower's Siegesmotivation (part of the view's
// motivation) is left out where it will have expired. The open competition uses the view as is.
func (c *compModel) at(tm *team) *team {
	if c.current {
		return tm
	}

	var rowers []engine.RowerView

	for i, rw := range tm.rowers {
		if rw.WinMotivation == 0 || rw.WinMotivationLeft > c.ahead {
			continue
		}

		if rowers == nil {
			rowers = slices.Clone(tm.rowers)
		}

		st := maps.Clone(rw.Stats)
		st[engine.StatMotivation] -= rw.WinMotivation
		rowers[i].Stats = st
	}

	if rowers == nil {
		return tm
	}

	return &team{id: tm.id, rowers: rowers, boats: tm.boats, trainers: tm.trainers}
}

func (c *compModel) team() bool { return c.def.Kind == string(engine.StationJTFO) }

func raceSpec(r data.Race) engine.RaceSpec {
	rs := engine.RaceSpec{
		ID: r.ID, Name: r.Name, Class: engine.BoatClass(r.Class),
		Mixed: r.Mixed, Seats: r.Seats, LoanBoat: r.LoanBoat,
	}
	for _, a := range r.Ages {
		rs.Ages = append(rs.Ages, engine.AgeGroup(a))
	}

	for _, g := range r.Genders {
		rs.Genders = append(rs.Genders, engine.Gender(g))
	}

	return rs
}

func (b *brain) futureComp(station int) *compModel {
	def := b.d.Competitions[b.d.Calendar[station].Competition]

	c := &compModel{station: station, def: def, weights: map[engine.Stat]float64{}}
	for _, r := range b.d.RacesFor(def, len(b.view.Players)) {
		c.races = append(c.races, raceSpec(r))
	}

	for k, v := range def.Weights {
		c.weights[engine.Stat(k)] = v
	}

	return c
}

func (b *brain) currentComp() *compModel {
	cv := b.view.Phase.Competition

	return &compModel{
		station: b.view.Current, def: b.d.Competitions[b.d.Calendar[b.view.Current].Competition],
		races: cv.Races, weights: cv.Weights, weather: cv.Weather, current: true,
	}
}

// ---------- scoring ----------

// trainerEffect mirrors the engine: per-seat effects whose weather and context match. An age bonus
// counts per matching rower; it is estimated per seat by the share of the team's rowers it covers.
func trainerEffect(tm *team, t engine.TrainerView, weather engine.Weather, context string) float64 {
	total := 0.0

	for _, eff := range t.Effects {
		if eff.Weather != "" && eff.Weather != weather {
			continue
		}

		if eff.Context != "" && eff.Context != context {
			continue
		}

		if len(eff.Ages) == 0 {
			total += float64(eff.Value)

			continue
		}

		if len(tm.rowers) == 0 {
			continue
		}

		n := 0

		for _, r := range tm.rowers {
			if slices.Contains(eff.Ages, r.Age) {
				n++
			}
		}

		total += float64(eff.Value*n) / float64(len(tm.rowers))
	}

	return total
}

// boatPoints mirrors the engine's boat value for a future competition (Ereignis modifiers unknown).
func (b *brain) boatPoints(race engine.RaceSpec, bt *engine.BoatView) float64 {
	rules := b.d.Rules
	val := 0

	switch {
	case race.Class == engine.BoatErgo:
		return 0
	case bt == nil:
		if race.LoanBoat {
			val = rules.LoanBoatValue
		}
	default:
		val = bt.Value
	}

	return float64(val * rules.BoatScale(race.Seats))
}

// bestEight returns the team's most valuable eight (nil if none).
func bestEight(tm *team) *engine.BoatView {
	var best *engine.BoatView

	for i := range tm.boats {
		if bt := &tm.boats[i]; bt.Class == engine.Boat8x && (best == nil || bt.Value > best.Value) {
			best = bt
		}
	}

	return best
}

// expectedTrainer is the expected per-entry bonus of the team's best trainer at a future
// competition, averaged over the weather table (water) or for no weather (ergo).
func (b *brain) expectedTrainer(tm *team, c *compModel) float64 {
	if len(tm.trainers) == 0 {
		return 0
	}

	weathers := []engine.Weather{""}
	if c.def.Water {
		weathers = weathers[:0]
		for _, w := range b.d.Rules.WeatherTable {
			weathers = append(weathers, engine.Weather(w))
		}
	}

	// The mood is known for a competition of the current section; later it has drifted (and
	// reverts towards 0), so it is left out.
	sameSection := b.d.Calendar[c.station].Section == b.d.Calendar[b.view.Current].Section

	sum := 0.0

	for _, w := range weathers {
		best := math.Inf(-1)

		for _, tr := range tm.trainers {
			v := trainerEffect(tm, tr, w, c.def.Context)
			if sameSection {
				v += float64(tr.Mood)
			}

			best = max(best, v)
		}

		sum += best
	}

	return sum / float64(len(weathers))
}

// expectedEventBonus mirrors the engine's event card bonus of a rower at a future competition,
// averaged over the weather table when the card needs certain weathers.
func (b *brain) expectedEventBonus(r engine.RowerView, c *compModel) float64 {
	if r.Event == nil || r.Event.Spent {
		return 0
	}

	def := b.d.EventType(r.Event.Type)
	if def == nil || def.Bonus == 0 || !data.ScopeMatches(def.Scope, c.def) {
		return 0
	}

	if def.Context != "" && def.Context != c.def.Context {
		return 0
	}

	if len(def.Weathers) == 0 {
		return float64(def.Bonus)
	}

	table := b.d.Rules.WeatherTable
	if !c.def.Water || len(table) == 0 {
		return 0
	}

	hits := 0

	for _, w := range table {
		if slices.Contains(def.Weathers, w) {
			hits++
		}
	}

	return float64(def.Bonus*hits) / float64(len(table))
}

// bestTrainer returns the trainer with the best effect plus mood at the open competition (or "").
func (b *brain) bestTrainer(t *team, c *compModel, jitter float64) string {
	best, bestV := "", math.Inf(-1)

	for _, tr := range t.trainers {
		v := trainerEffect(t, tr, c.weather, c.def.Context) + float64(tr.Mood)
		if jitter > 0 {
			v += b.rng.NormFloat64() * jitter
		}

		if v > bestV {
			best, bestV = tr.ID, v
		}
	}

	return best
}

// scorer returns the estimated total of an entry of team t at competition c.
func (b *brain) scorer(
	pr *problem, tm *team, c *compModel, trainerID string, noise map[string]float64,
) func(int, []int, int) float64 {
	if c.current {
		return func(r int, crew []int, boat int) float64 {
			ids := make([]string, len(crew))
			pert := 0.0

			for i, x := range crew {
				ids[i] = tm.rowers[x].ID
				pert += noise[ids[i]] * weighted(tm.rowers[x].Stats, c.weights)
			}

			boatID := ""
			if boat >= 0 {
				boatID = tm.boats[boat].ID
			}

			return float64(engine.EstimateEntry(b.g, tm.id, c.races[r].ID, ids, boatID, trainerID)) + pert
		}
	}

	tb := b.expectedTrainer(tm, c)

	return func(r int, crew []int, boat int) float64 {
		race := c.races[r]

		s := 0.0
		for _, x := range crew {
			s += pr.indiv[r][x] + b.expectedEventBonus(tm.rowers[x], c)
		}

		var bt *engine.BoatView
		if boat >= 0 {
			bt = &tm.boats[boat]
		} else if race.Class == engine.Boat8x {
			bt = bestEight(tm) // eights are not chosen: the best own eight rows
		}

		s += b.boatPoints(race, bt)
		s += tb * float64(b.d.Rules.TrainerScale(race.Seats))

		return s
	}
}

// ---------- opponents and objectives ----------

// field is the expected competition in each race: opponents' estimated scores and the guest crew.
type field struct {
	race [][]float64 // per race: competitor scores
	team []float64   // team competitions: competitor team totals
}

// estimateField predicts every opponent's lineup (greedy, maximising its own total) at c.
func (b *brain) estimateField(c *compModel) *field {
	fd := &field{race: make([][]float64, len(c.races))}

	for _, op := range b.opps {
		t := c.at(teamOf(op))
		pr := newProblem(c.races, t, c.weights, nil)

		trainer := ""
		if c.current {
			trainer = b.bestTrainer(t, c, 0)
		}

		pr.score = b.scorer(pr, t, c, trainer, nil)
		pr.value = func(pl *plan) float64 {
			s := 0.0

			for r, cr := range pl.crew {
				if cr != nil {
					s += pl.score[r]
				}
			}

			return s
		}
		pl := pr.greedy()
		total, found := 0.0, false

		for r, cr := range pl.crew {
			if cr != nil {
				fd.race[r] = append(fd.race[r], pl.score[r])
				total += pl.score[r]
				found = true
			}
		}

		if found {
			fd.team = append(fd.team, total)
		}
	}

	for r, race := range c.races {
		if g, ok := b.guestScore(c, race); ok {
			fd.race[r] = append(fd.race[r], g)
		}
	}

	if c.team() {
		if g, ok := b.guestTeam(c); ok {
			fd.team = append(fd.team, g)
		}
	}

	return fd
}

// guestScore is the strength of the race's guest crew (Gastmannschaft), if the rules have one.
func (b *brain) guestScore(c *compModel, race engine.RaceSpec) (float64, bool) {
	if c.team() {
		return 0, false
	}

	if c.current {
		// The view's value includes Ereignis guest modifiers.
		return float64(race.GuestScore), race.GuestScore > 0
	}

	return engine.GuestScore(b.d, c.def, c.station, race)
}

func (b *brain) guestTeam(c *compModel) (float64, bool) {
	sum, found := 0.0, false

	for _, race := range c.races {
		if g, ok := engine.GuestScore(b.d, c.def, c.station, race); ok {
			sum += g
			found = true
		}
	}

	return sum, found
}

const rawEps = 0.002 // tie-break: prefer stronger lineups when medal chances are equal

// objective returns the expected medal points of a plan against the field.
func (b *brain) objective(c *compModel, fd *field) func(pl *plan) float64 {
	if c.team() {
		return func(pl *plan) float64 {
			total, found := 0.0, false

			for r, cr := range pl.crew {
				if cr != nil {
					total += pl.score[r]
					found = true
				}
			}

			if !found {
				return 0
			}

			return b.expMedals(total, fd.team) + rawEps*total
		}
	}

	return func(pl *plan) float64 {
		v := 0.0

		for r, cr := range pl.crew {
			if cr != nil {
				v += b.expMedals(pl.score[r], fd.race[r]) + rawEps*pl.score[r]
			}
		}

		return v
	}
}

// lineup optimises team t at competition c against field f.
func (b *brain) lineup(t *team, c *compModel, f *field, trainerID string, passes int) (*problem, *plan) {
	t = c.at(t)
	pr := newProblem(c.races, t, c.weights, b.noise)
	pr.score = b.scorer(pr, t, c, trainerID, b.noise)
	pr.value = b.objective(c, f)

	pl := pr.greedy()
	if passes > 0 {
		pr.climb(pl, passes)
	}

	return pr, pl
}
