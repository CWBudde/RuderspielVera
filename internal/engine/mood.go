package engine

import (
	"strconv"
)

// initMood gives every trainer of the catalog the starting mood (rules.json trainerMood.start).
func (g *Game) initMood() {
	start := g.d.Rules.TrainerMood.Start
	g.mood = map[string]int{}

	g.prevMood = map[string]int{}
	for _, t := range g.d.Trainers {
		g.mood[t.ID] = start
		g.prevMood[t.ID] = start
	}
}

// driftMood moves every trainer's mood one step of the bounded random walk (rules.json
// trainerMood). It runs when the calendar figure enters a new section (sections 2–12), before the
// station's own draws.
//
// RNG order: one weighted draw per trainer, in catalog (trainers.json) order, for every trainer
// whether it is on a team or still in the draft pool.
func (g *Game) driftMood() {
	cfg := g.d.Rules.TrainerMood

	parts := make([]Msg, 0, len(g.d.Trainers))
	for _, t := range g.d.Trainers {
		cur := g.mood[t.ID]
		g.prevMood[t.ID] = cur

		next := cur
		if i := pickWeighted(g.rng, cfg.Weights[strconv.Itoa(cur)]); i >= 0 {
			next = min(max(cur+cfg.Steps[i], cfg.Min), cfg.Max)
		}

		g.mood[t.ID] = next
		parts = append(parts, msg("log.moodEntry", "trainer", dataMsg("trainer", t.ID, "name"), "mood", signed(next)))
	}

	g.addLog("info", "", msg("log.mood", "entries", list(parts)))
}
