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
func (pr *problem) buildCrew(r int, used []bool) []int {
	race := pr.races[r]
	var c []int
	for x := range pr.rowers {
		if !used[x] && pr.elig[r][x] {
			c = append(c, x)
		}
	}
	if len(c) < race.Seats {
		return nil
	}
	slices.SortStableFunc(c, func(a, b int) int {
		switch {
		case pr.indiv[r][a] > pr.indiv[r][b]:
			return -1
		case pr.indiv[r][a] < pr.indiv[r][b]:
			return 1
		}
		return 0
	})
	crew := slices.Clone(c[:race.Seats])
	if !race.Mixed {
		return crew
	}
	rest := c[race.Seats:]
	for tries := 0; tries < 4; tries++ {
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
		for r := range pr.races {
			if pl.crew[r] != nil {
				continue
			}
			crew := pr.buildCrew(r, ur)
			if crew == nil {
				continue
			}
			boat, ok := pr.bestBoat(r, ub)
			if !ok {
				continue
			}
			s := pr.score(r, crew, boat)
			pl.crew[r], pl.boat[r], pl.score[r] = crew, boat, s
			gain := pr.value(pl) - cur
			pl.crew[r], pl.boat[r], pl.score[r] = nil, -1, 0
			if gain > bestGain {
				bestR, bestGain, bestCrew, bestBoat, bestScore = r, gain, crew, boat, s
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
	cur := pr.value(pl)
	accept := func() bool {
		v := pr.value(pl)
		if v > cur+1e-9 {
			cur = v
			return true
		}
		return false
	}
	rescore := func(r int) {
		if pl.crew[r] != nil {
			pl.score[r] = pr.score(r, pl.crew[r], pl.boat[r])
		}
	}
	for pass := 0; pass < maxPasses; pass++ {
		improved := false
		ur, ub := pr.used(pl)
		// 1. replace a rower by an unused one.
		for r := range pr.races {
			for seat := range pl.crew[r] {
				for x := range pr.rowers {
					if ur[x] || !pr.elig[r][x] {
						continue
					}
					old, oldS := pl.crew[r][seat], pl.score[r]
					pl.crew[r][seat] = x
					if pr.crewOK(r, pl.crew[r]) {
						rescore(r)
						if accept() {
							ur[old], ur[x], improved = false, true, true
							continue
						}
					}
					pl.crew[r][seat], pl.score[r] = old, oldS
				}
			}
		}
		// 2. swap rowers between two entries.
		for r1 := range pr.races {
			for r2 := r1 + 1; r2 < len(pr.races); r2++ {
				for s1 := range pl.crew[r1] {
					for s2 := range pl.crew[r2] {
						a, b := pl.crew[r1][s1], pl.crew[r2][s2]
						if !pr.elig[r2][a] || !pr.elig[r1][b] {
							continue
						}
						o1, o2 := pl.score[r1], pl.score[r2]
						pl.crew[r1][s1], pl.crew[r2][s2] = b, a
						if pr.crewOK(r1, pl.crew[r1]) && pr.crewOK(r2, pl.crew[r2]) {
							rescore(r1)
							rescore(r2)
							if accept() {
								improved = true
								continue
							}
						}
						pl.crew[r1][s1], pl.crew[r2][s2] = a, b
						pl.score[r1], pl.score[r2] = o1, o2
					}
				}
			}
		}
		// 3. boat exchanges: unused boat or between entries of the same class.
		for r := range pr.races {
			if pl.crew[r] == nil || !needsBoat(pr.races[r]) {
				continue
			}
			for i, bt := range pr.boats {
				if bt.Class != pr.races[r].Class || i == pl.boat[r] {
					continue
				}
				other := -1
				if ub[i] {
					for r2 := range pr.races {
						if pl.crew[r2] != nil && pl.boat[r2] == i {
							other = r2
						}
					}
				}
				old, oldS := pl.boat[r], pl.score[r]
				var oS float64
				pl.boat[r] = i
				if other >= 0 {
					oS = pl.score[other]
					pl.boat[other] = old
					if old < 0 && !pr.races[other].LoanBoat {
						pl.boat[r], pl.boat[other] = old, i
						continue
					}
					rescore(other)
				}
				rescore(r)
				if accept() {
					if other < 0 {
						ub[i] = true
						if old >= 0 {
							ub[old] = false
						}
					}
					improved = true
					continue
				}
				pl.boat[r], pl.score[r] = old, oldS
				if other >= 0 {
					pl.boat[other], pl.score[other] = i, oS
				}
			}
		}
		// 4. drop an entry, 5. add an entry.
		for r := range pr.races {
			if pl.crew[r] == nil {
				continue
			}
			c, bt, s := pl.crew[r], pl.boat[r], pl.score[r]
			pl.crew[r], pl.boat[r], pl.score[r] = nil, -1, 0
			if accept() {
				improved = true
				continue
			}
			pl.crew[r], pl.boat[r], pl.score[r] = c, bt, s
		}
		ur, ub = pr.used(pl)
		for r := range pr.races {
			if pl.crew[r] != nil {
				continue
			}
			crew := pr.buildCrew(r, ur)
			if crew == nil {
				continue
			}
			bt, ok := pr.bestBoat(r, ub)
			if !ok {
				continue
			}
			pl.crew[r], pl.boat[r] = crew, bt
			rescore(r)
			if accept() {
				for _, x := range crew {
					ur[x] = true
				}
				if bt >= 0 {
					ub[bt] = true
				}
				improved = true
				continue
			}
			pl.crew[r], pl.boat[r], pl.score[r] = nil, -1, 0
		}
		if !improved {
			return
		}
	}
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
