package engine

import (
	"fmt"
	"strings"

	"ruderspiel/internal/data"
)

// runEreignisStation draws one global Ereignis. Immediate effects are applied now; the others are
// stored as pending modifiers until the next matching station consumes them. It returns the card
// name and the description (with per-player details).
func (g *Game) runEreignisStation() (string, string) {
	weights := make([]int, len(g.d.Ereignisse))
	for i, e := range g.d.Ereignisse {
		weights[i] = e.Weight
	}

	ev := &g.d.Ereignisse[pickWeighted(g.rng, weights)]

	desc := ev.Description

	switch ev.Effect {
	case data.EffAllMotivation:
		for _, pl := range g.players {
			for _, r := range pl.rowers {
				r.addStat(StatMotivation, ev.Value)
			}
		}
	case data.EffRandomRowerMot:
		var details strings.Builder

		for _, pl := range g.players {
			if len(pl.rowers) == 0 {
				continue
			}

			r := pl.rowers[g.rng.IntN(len(pl.rowers))]
			r.addStat(StatMotivation, ev.Value)
			fmt.Fprintf(&details, " %s: %s (%+d Motivation).", pl.name, r.name, ev.Value)
		}

		desc += details.String()
	case data.EffNone:
	default:
		g.pending = append(g.pending, ev)
	}

	g.addLog("ereignis", "", ev.Name+": "+desc)

	return ev.Name, desc
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
		if !g.ereignisAhead(e.Effect) {
			continue // no station left that would consume it
		}

		out = append(out, fmt.Sprintf("%s: %s", e.Name, e.Description))
	}

	return out
}

// ereignisAhead reports whether a station after the current one would still consume a pending
// Ereignis with this effect. Pending modifiers without such a station stay in the state (replay
// compatibility) but are no longer shown as active.
func (g *Game) ereignisAhead(effect string) bool {
	if g.over {
		return false
	}

	cal := g.d.Calendar
	for i := g.station + 1; i < len(cal); i++ {
		st := cal[i]

		var comp *data.Competition
		if c, ok := g.d.Competitions[st.Competition]; ok {
			comp = c
		}

		switch effect {
		case data.EffTrainingPoints:
			if StationType(st.Type) == StationTraining {
				return true
			}
		case data.EffSponsorPick:
			// Consumed by the draft after a competition; there is none after the last station.
			if comp != nil && i < len(cal)-1 {
				return true
			}
		case data.EffForceWeather, data.EffKrebsChance, data.EffCityBoatMod:
			if comp != nil && comp.Water {
				return true
			}
		case data.EffErgoKraftWeight:
			if StationType(st.Type) == StationErgoCup {
				return true
			}
		default:
			return true
		}
	}

	return false
}
