package bot

import (
	"slices"

	"ruderspiel/internal/engine"
)

// problem is one lineup optimisation: assign rowers and boats of one team to the races of one
// competition. Every rower and boat is used at most once, at most one entry per race, and all
// eligibility rules of the engine hold (age, gender, mixed, boat class, loan boats).
type problem struct {
	races  []engine.RaceSpec
	rowers []engine.RowerView
	boats  []engine.BoatView
	elig   [][]bool    // race x rower
	indiv  [][]float64 // race x rower: perceived single-rower strength, for crew ordering
	score  func(r int, crew []int, boat int) float64
	value  func(pl *plan) float64
}

// plan is a lineup: crew[r] == nil means no entry in race r; boat[r] == -1 means no own boat.
type plan struct {
	crew  [][]int
	boat  []int
	score []float64
}

func newProblem(races []engine.RaceSpec, t *team, w map[engine.Stat]float64, noise map[string]float64) *problem {
	pr := &problem{races: races, rowers: t.rowers, boats: t.boats}
	pr.elig = make([][]bool, len(races))

	pr.indiv = make([][]float64, len(races))
	for r, race := range races {
		pr.elig[r] = make([]bool, len(t.rowers))

		pr.indiv[r] = make([]float64, len(t.rowers))
		for x, rw := range t.rowers {
			pr.elig[r][x] = slices.Contains(race.Ages, rw.Age) && slices.Contains(race.Genders, rw.Gender)
			pr.indiv[r][x] = weighted(rw.Stats, w) * (1 + noise[rw.ID])
		}
	}

	return pr
}

func (pr *problem) newPlan() *plan {
	n := len(pr.races)

	pl := &plan{crew: make([][]int, n), boat: make([]int, n), score: make([]float64, n)}
	for r := range pl.boat {
		pl.boat[r] = -1
	}

	return pl
}

func needsBoat(race engine.RaceSpec) bool {
	return race.Class != engine.BoatErgo && race.Class != engine.Boat8x
}

func (pr *problem) mixedMissing(crew []int) int {
	girls, boys := false, false
	ages := map[engine.AgeGroup]bool{}

	for _, x := range crew {
		rw := pr.rowers[x]
		girls = girls || rw.Gender == engine.GenderF
		boys = boys || rw.Gender == engine.GenderM
		ages[rw.Age] = true
	}

	m := 0
	if !girls {
		m++
	}

	if !boys {
		m++
	}

	if len(ages) < 2 {
		m++
	}

	return m
}

func (pr *problem) crewOK(r int, crew []int) bool {
	return !pr.races[r].Mixed || pr.mixedMissing(crew) == 0
}

// buildCrew picks the strongest eligible unused rowers for race r (repairing mixed crews).
func (pr *problem) buildCrew(ri int, used []bool) []int {
	race := pr.races[ri]
	var cands []int

	for x := range pr.rowers {
		if !used[x] && pr.elig[ri][x] {
			cands = append(cands, x)
		}
	}

	if len(cands) < race.Seats {
		return nil
	}

	slices.SortStableFunc(cands, func(a, b int) int {
		switch {
		case pr.indiv[ri][a] > pr.indiv[ri][b]:
			return -1
		case pr.indiv[ri][a] < pr.indiv[ri][b]:
			return 1
		}

		return 0
	})

	crew := slices.Clone(cands[:race.Seats])
	if !race.Mixed {
		return crew
	}

	return pr.repairMixed(crew, cands[race.Seats:])
}

// repairMixed swaps rowers from rest into a mixed crew until it is valid; nil if that fails.
func (pr *problem) repairMixed(crew, rest []int) []int {
	for range 4 {
		miss := pr.mixedMissing(crew)
		if miss == 0 {
			return crew
		}

		fixed := false
		for yi, y := range rest {
			for k := len(crew) - 1; k >= 0 && !fixed; k-- {
				old := crew[k]

				crew[k] = y
				if pr.mixedMissing(crew) < miss {
					rest[yi] = old
					fixed = true
				} else {
					crew[k] = old
				}
			}

			if fixed {
				break
			}
		}

		if !fixed {
			return nil
		}
	}

	if pr.mixedMissing(crew) == 0 {
		return crew
	}

	return nil
}

// bestBoat returns the most valuable unused own boat for race r, or -1. ok is false when the
// race cannot be entered without a boat.
func (pr *problem) bestBoat(r int, usedB []bool) (int, bool) {
	race := pr.races[r]
	if !needsBoat(race) {
		return -1, true
	}

	best := -1
	for i, b := range pr.boats {
		if !usedB[i] && b.Class == race.Class && (best < 0 || b.Value > pr.boats[best].Value) {
			best = i
		}
	}

	if best < 0 {
		return -1, race.LoanBoat
	}

	return best, true
}

func (pr *problem) used(pl *plan) ([]bool, []bool) {
	ur := make([]bool, len(pr.rowers))
	ub := make([]bool, len(pr.boats))

	for r, c := range pl.crew {
		for _, x := range c {
			ur[x] = true
		}

		if c != nil && pl.boat[r] >= 0 {
			ub[pl.boat[r]] = true
		}
	}

	return ur, ub
}

// greedy fills races one at a time, always taking the entry with the largest objective gain.
func (pr *problem) greedy() *plan {
	pl := pr.newPlan()
	ur := make([]bool, len(pr.rowers))
	ub := make([]bool, len(pr.boats))
	cur := pr.value(pl)

	for {
		bestR, bestGain := -1, 1e-9
		var bestCrew []int
		bestBoat := -1
		bestScore := 0.0

		for ri := range pr.races {
			if pl.crew[ri] != nil {
				continue
			}

			crew := pr.buildCrew(ri, ur)
			if crew == nil {
				continue
			}

			boat, ok := pr.bestBoat(ri, ub)
			if !ok {
				continue
			}

			s := pr.score(ri, crew, boat)
			pl.crew[ri], pl.boat[ri], pl.score[ri] = crew, boat, s
			gain := pr.value(pl) - cur

			pl.crew[ri], pl.boat[ri], pl.score[ri] = nil, -1, 0
			if gain > bestGain {
				bestR, bestGain, bestCrew, bestBoat, bestScore = ri, gain, crew, boat, s
			}
		}

		if bestR < 0 {
			return pl
		}

		pl.crew[bestR], pl.boat[bestR], pl.score[bestR] = bestCrew, bestBoat, bestScore
		for _, x := range bestCrew {
			ur[x] = true
		}

		if bestBoat >= 0 {
			ub[bestBoat] = true
		}

		cur += bestGain
	}
}

// climb improves a plan by first-improvement local search over rower replacements, rower swaps
// between entries, dropping/adding entries and boat exchanges.
func (pr *problem) climb(pl *plan, maxPasses int) {
	c := &climber{pr: pr, pl: pl, cur: pr.value(pl)}

	for range maxPasses {
		ur, ub := pr.used(pl)
		improved := c.replaceRowers(ur)
		improved = c.swapRowers() || improved
		improved = c.exchangeBoats(ub) || improved
		improved = c.dropEntries() || improved

		ur, ub = pr.used(pl)
		improved = c.addEntries(ur, ub) || improved

		if !improved {
			return
		}
	}
}

// climber is the state of one local search: the plan being improved and its current value.
type climber struct {
	pr  *problem
	pl  *plan
	cur float64
}

// accept keeps the current change if it improves the plan value.
func (c *climber) accept() bool {
	v := c.pr.value(c.pl)
	if v > c.cur+1e-9 {
		c.cur = v
		return true
	}

	return false
}

func (c *climber) rescore(r int) {
	if c.pl.crew[r] != nil {
		c.pl.score[r] = c.pr.score(r, c.pl.crew[r], c.pl.boat[r])
	}
}

// replaceRowers tries to replace each seated rower by an unused one (ur = used rowers).
func (c *climber) replaceRowers(ur []bool) bool {
	pr, pl := c.pr, c.pl
	improved := false

	for ri := range pr.races {
		for seat := range pl.crew[ri] {
			for x := range pr.rowers {
				if ur[x] || !pr.elig[ri][x] {
					continue
				}

				old, oldS := pl.crew[ri][seat], pl.score[ri]

				pl.crew[ri][seat] = x
				if pr.crewOK(ri, pl.crew[ri]) {
					c.rescore(ri)

					if c.accept() {
						ur[old], ur[x], improved = false, true, true
						continue
					}
				}

				pl.crew[ri][seat], pl.score[ri] = old, oldS
			}
		}
	}

	return improved
}

// swapRowers tries to swap rowers between two entries.
func (c *climber) swapRowers() bool {
	improved := false

	for r1 := range c.pr.races {
		for r2 := r1 + 1; r2 < len(c.pr.races); r2++ {
			improved = c.swapBetween(r1, r2) || improved
		}
	}

	return improved
}

func (c *climber) swapBetween(r1, r2 int) bool {
	pr, pl := c.pr, c.pl
	improved := false

	for s1 := range pl.crew[r1] {
		for s2 := range pl.crew[r2] {
			ra, rb := pl.crew[r1][s1], pl.crew[r2][s2]
			if !pr.elig[r2][ra] || !pr.elig[r1][rb] {
				continue
			}

			o1, o2 := pl.score[r1], pl.score[r2]

			pl.crew[r1][s1], pl.crew[r2][s2] = rb, ra
			if pr.crewOK(r1, pl.crew[r1]) && pr.crewOK(r2, pl.crew[r2]) {
				c.rescore(r1)
				c.rescore(r2)

				if c.accept() {
					improved = true
					continue
				}
			}

			pl.crew[r1][s1], pl.crew[r2][s2] = ra, rb
			pl.score[r1], pl.score[r2] = o1, o2
		}
	}

	return improved
}

// exchangeBoats tries an unused boat or a boat of another entry of the same class for every
// entry (ub = used boats).
func (c *climber) exchangeBoats(ub []bool) bool {
	pr, pl := c.pr, c.pl
	improved := false

	for r := range pr.races {
		if pl.crew[r] == nil || !needsBoat(pr.races[r]) {
			continue
		}

		for i, bt := range pr.boats {
			if bt.Class != pr.races[r].Class || i == pl.boat[r] {
				continue
			}

			improved = c.tryBoat(r, i, ub) || improved
		}
	}

	return improved
}

// tryBoat puts boat i into entry r (swapping with the entry that uses it, if any) and keeps the
// change if it improves the plan.
func (c *climber) tryBoat(ri, bi int, ub []bool) bool {
	pr, pl := c.pr, c.pl
	other := -1

	if ub[bi] {
		for r2 := range pr.races {
			if pl.crew[r2] != nil && pl.boat[r2] == bi {
				other = r2
			}
		}
	}

	old, oldS := pl.boat[ri], pl.score[ri]
	var oS float64

	pl.boat[ri] = bi
	if other >= 0 {
		oS = pl.score[other]

		pl.boat[other] = old
		if old < 0 && !pr.races[other].LoanBoat {
			pl.boat[ri], pl.boat[other] = old, bi
			return false
		}

		c.rescore(other)
	}

	c.rescore(ri)

	if c.accept() {
		if other < 0 {
			ub[bi] = true
			if old >= 0 {
				ub[old] = false
			}
		}

		return true
	}

	pl.boat[ri], pl.score[ri] = old, oldS
	if other >= 0 {
		pl.boat[other], pl.score[other] = bi, oS
	}

	return false
}

// dropEntries tries to leave out each entry.
func (c *climber) dropEntries() bool {
	pl := c.pl
	improved := false

	for r := range c.pr.races {
		if pl.crew[r] == nil {
			continue
		}

		crew, bt, s := pl.crew[r], pl.boat[r], pl.score[r]
		pl.crew[r], pl.boat[r], pl.score[r] = nil, -1, 0

		if c.accept() {
			improved = true
			continue
		}

		pl.crew[r], pl.boat[r], pl.score[r] = crew, bt, s
	}

	return improved
}

// addEntries tries to enter each empty race with the best available crew and boat.
func (c *climber) addEntries(ur, ub []bool) bool {
	pr, pl := c.pr, c.pl
	improved := false

	for ri := range pr.races {
		if pl.crew[ri] != nil {
			continue
		}

		crew := pr.buildCrew(ri, ur)
		if crew == nil {
			continue
		}

		bt, ok := pr.bestBoat(ri, ub)
		if !ok {
			continue
		}

		pl.crew[ri], pl.boat[ri] = crew, bt
		c.rescore(ri)

		if c.accept() {
			for _, x := range crew {
				ur[x] = true
			}

			if bt >= 0 {
				ub[bt] = true
			}

			improved = true

			continue
		}

		pl.crew[ri], pl.boat[ri], pl.score[ri] = nil, -1, 0
	}

	return improved
}

// entries converts a plan into engine entries in race order.
func (pr *problem) entries(pl *plan) []engine.Entry {
	var out []engine.Entry

	for r, c := range pl.crew {
		if c == nil {
			continue
		}

		e := engine.Entry{RaceID: pr.races[r].ID}
		if pl.boat[r] >= 0 {
			e.BoatID = pr.boats[pl.boat[r]].ID
		}

		for _, x := range c {
			e.RowerIDs = append(e.RowerIDs, pr.rowers[x].ID)
		}

		out = append(out, e)
	}

	return out
}
