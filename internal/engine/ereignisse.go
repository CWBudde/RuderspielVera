package engine

import (
	"fmt"

	"ruderspiel/internal/data"
)

// runEreignisStation draws one global Ereignis. Immediate effects are applied now; the others are
// stored as pending modifiers until the next matching station consumes them. It returns the card
// name and the description (with per-player details).
func (g *Game) runEreignisStation() (name, desc string) {
	weights := make([]int, len(g.d.Ereignisse))
	for i, e := range g.d.Ereignisse {
		weights[i] = e.Weight
	}
	e := &g.d.Ereignisse[pickWeighted(g.rng, weights)]
	desc = e.Description
	switch e.Effect {
	case data.EffAllMotivation:
		for _, pl := range g.players {
			for _, r := range pl.rowers {
				r.addStat(StatMotivation, e.Value)
			}
		}
	case data.EffRandomRowerMot:
		for _, pl := range g.players {
			if len(pl.rowers) == 0 {
				continue
			}
			r := pl.rowers[g.rng.IntN(len(pl.rowers))]
			r.addStat(StatMotivation, e.Value)
			desc += fmt.Sprintf(" %s: %s (%+d Motivation).", pl.name, r.name, e.Value)
		}
	case data.EffNone:
	default:
		g.pending = append(g.pending, e)
	}
	g.addLog("ereignis", "", e.Name+": "+desc)
	return e.Name, desc
}

// takePending removes and returns all pending Ereignisse with the given effect, oldest first.
func (g *Game) takePending(effect string) []*data.Ereignis {
	var taken, kept []*data.Ereignis
	for _, e := range g.pending {
		if e.Effect == effect {
			taken = append(taken, e)
		} else {
			kept = append(kept, e)
		}
	}
	g.pending = kept
	return taken
}

// consumeTrainingPoints returns the points of the next regular training (Trainerstreik,
// Doppeltraining; the most recent modifier wins).
func (g *Game) consumeTrainingPoints() int {
	pts := g.d.Rules.TrainingPoints
	for _, e := range g.takePending(data.EffTrainingPoints) {
		pts = e.Value
		g.addLog("ereignis", "", fmt.Sprintf("%s wirkt: Dieses Training bringt %d Punkte.", e.Name, e.Value))
	}
	return pts
}

// sponsorPicks consumes pending Sponsor cards and returns the number of extra picks.
func (g *Game) sponsorPicks() int {
	n := 0
	for _, e := range g.takePending(data.EffSponsorPick) {
		n += e.Value
	}
	return n
}

// activeEreignisse returns German texts of all pending modifiers.
func (g *Game) activeEreignisse() []string {
	out := []string{}
	for _, e := range g.pending {
		out = append(out, fmt.Sprintf("%s: %s", e.Name, e.Description))
	}
	return out
}
