package engine

import (
	"slices"

	"ruderspiel/internal/data"
)

// runEreignisStation draws one global Ereignis among the cards that can still have an effect
// (ereignisDrawable). Immediate effects are applied now; the others are stored as pending
// modifiers until the next matching station consumes them. It returns the card and the
// per-player details (affected rowers).
//
// RNG order: one weighted draw for the card, then (random_rower_stat only) one draw per player
// in seat order for the affected rower.
func (g *Game) runEreignisStation() (*data.Ereignis, []Msg) {
	weights := make([]int, len(g.d.Ereignisse))
	for i := range g.d.Ereignisse {
		if e := &g.d.Ereignisse[i]; g.ereignisDrawable(e) {
			weights[i] = e.Weight
		}
	}

	idx := pickWeighted(g.rng, weights)
	if idx < 0 {
		// Unreachable with valid data (a "none" card with scope any is always drawable).
		idx = slices.IndexFunc(g.d.Ereignisse, func(e data.Ereignis) bool { return e.Effect == data.EffNone })
	}

	ev := &g.d.Ereignisse[idx]

	details := []Msg{}

	switch ev.Effect {
	case data.EffAllStat:
		for _, pl := range g.players {
			for _, r := range pl.rowers {
				r.addStat(Stat(ev.Stat), ev.Value)
			}
		}
	case data.EffRandomRowerStat:
		for _, pl := range g.players {
			if len(pl.rowers) == 0 {
				continue
			}

			r := pl.rowers[g.rng.IntN(len(pl.rowers))]
			r.addStat(Stat(ev.Stat), ev.Value)
			details = append(details, msg("ereignis.detail", "player", pl.name, "rower", rowerName(r),
				"value", signed(ev.Value), "stat", statMsg(Stat(ev.Stat))))
		}
	case data.EffNone:
	default:
		g.pending = append(g.pending, ev)
	}

	g.addLog("ereignis", "", msg("log.ereignis", "name", dataMsg("ereignis", ev.ID, "name"),
		"description", dataMsg("ereignis", ev.ID, "description"), "details", list(details)))

	return ev, details
}

// nextCompetition returns the first competition after the current station (or nil).
func (g *Game) nextCompetition() *data.Competition {
	for i := g.station + 1; i < len(g.d.Calendar); i++ {
		if c := g.d.Competitions[g.d.Calendar[i].Competition]; c != nil {
			return c
		}
	}

	return nil
}

// ereignisDrawable reports whether an Ereignis may be drawn at the current station: a water or
// ergo card only when the next competition is of that kind (no Maschsee-Monster in the Ergo-Cup
// winter), and a modifier only when a later station would still consume it.
func (g *Game) ereignisDrawable(e *data.Ereignis) bool {
	if e.Scope != data.ScopeAny {
		next := g.nextCompetition()
		if next == nil || !data.ScopeMatches(e.Scope, next) {
			return false
		}
	}

	return data.ImmediateEffect(e.Effect) || g.ereignisAhead(e)
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
		g.addLog("ereignis", "", msg("log.trainingPoints", "name", dataMsg("ereignis", e.ID, "name"), "points", e.Value))
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

// activeEreignisse returns all pending modifiers that a later station still consumes.
func (g *Game) activeEreignisse() []Msg {
	out := []Msg{}

	for _, e := range g.pending {
		if !g.ereignisAhead(e) {
			continue // no station left that would consume it
		}

		out = append(out, msg("ereignis.active", "name", dataMsg("ereignis", e.ID, "name"),
			"description", dataMsg("ereignis", e.ID, "description")))
	}

	return out
}

// ereignisAhead reports whether a station after the current one would still consume a pending
// Ereignis with this effect. Pending modifiers without such a station stay in the state (replay
// compatibility) but are no longer shown as active.
func (g *Game) ereignisAhead(card *data.Ereignis) bool {
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

		switch {
		case card.Effect == data.EffTrainingPoints:
			if StationType(st.Type) == StationTraining {
				return true
			}
		case card.Effect == data.EffSponsorPick:
			// Consumed by the draft after a competition; there is none after the last station.
			if comp != nil && i < len(cal)-1 {
				return true
			}
		case data.CompetitionEffect(card.Effect):
			if comp != nil && data.ScopeMatches(card.Scope, comp) {
				return true
			}
		default:
			return true
		}
	}

	return false
}
