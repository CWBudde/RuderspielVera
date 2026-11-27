package engine

import (
	"slices"

	"ruderspiel/internal/data"
)

func validStat(s Stat) bool { return slices.Contains(AllStats, s) }

// TrainableStats are the stats that training and camps may raise and that Tausch may swap.
// Motivation is not trainable: it comes only from selections, events, Ereignisse and the
// temporary Siegesmotivation.
//
//nolint:gochecknoglobals // read-only stat list, like AllStats
var TrainableStats = []Stat{StatKraft, StatAusdauer, StatSpeed, StatTechnik}

// Trainable reports whether s may be trained (and swapped by Tausch).
func Trainable(s Stat) bool { return slices.Contains(TrainableStats, s) }

func (g *Game) validateTraining(pl *player, p *phase, allocs []TrainingAlloc) error {
	total := 0
	rowers := map[string]bool{}

	for _, alloc := range allocs {
		if pl.rower(alloc.RowerID) == nil {
			return invalid("err.rowerNotOwned", "rower", alloc.RowerID)
		}

		if !validStat(alloc.Stat) {
			return invalid("err.unknownStat", "stat", string(alloc.Stat))
		}

		if !Trainable(alloc.Stat) {
			return invalid("err.trainMotivation")
		}

		if alloc.Points <= 0 {
			return invalid("err.trainPositive")
		}

		total += alloc.Points
		rowers[alloc.RowerID] = true
	}

	if total != p.points {
		return invalid("err.trainTotal", "points", p.points, "total", total)
	}

	if p.maxRowers > 0 && len(rowers) > p.maxRowers {
		return invalid("err.campMaxRowers", "maxRowers", p.maxRowers, "count", len(rowers))
	}

	return nil
}

// applyTraining adds the points, mirrors them across Seelenzwilling pairs and then triggers
// Wachstum cards (every training incl. camps). Wachstum raises the best trainable stat: it
// piggybacks on training, so never Motivation.
func (g *Game) applyTraining(pl *player, allocs []TrainingAlloc) {
	for _, a := range allocs {
		pl.rower(a.RowerID).addStat(a.Stat, a.Points)
	}

	g.applyTwinTraining(pl, allocs)

	for _, r := range pl.rowers {
		if r.event != nil && r.event.def.Type == "wachstum" {
			s := r.stats.bestOf(TrainableStats)
			r.addStat(s, r.event.def.Value)
			g.addLog("event", pl.id, msg("event.gain", "card", dataMsg("event", r.event.def.Type, "name"),
				"rower", rowerName(r), "value", signed(r.event.def.Value), "stat", statMsg(s)))
		}
	}
}

// applyTwinTraining handles unspent Seelenzwilling cards: if the holder or the partner got points
// in this training, each also gets the other's points (from allocs, so a pair trained on both
// sides ends with the sum on both), and the card is spent. The mirrored points do not count
// against a camp's maxRowers.
func (g *Game) applyTwinTraining(pl *player, allocs []TrainingAlloc) {
	for _, r := range pl.rowers {
		card := r.event
		if card == nil || card.def.Type != "seelenzwilling" || card.spent || card.partner == nil {
			continue
		}

		p := card.partner
		fromR, fromP := trainedPoints(allocs, r.id), trainedPoints(allocs, p.id)

		if len(fromR) == 0 && len(fromP) == 0 {
			continue
		}

		card.spent = true

		for _, s := range TrainableStats {
			p.addStat(s, fromR[s])
			r.addStat(s, fromP[s])
		}

		g.addLog("event", pl.id, msg("log.twinTraining", "card", dataMsg("event", card.def.Type, "name"),
			"rower", rowerName(r), "partner", rowerName(p)))
	}
}

// trainedPoints sums the allocated points per stat for one rower.
func trainedPoints(allocs []TrainingAlloc, rowerID string) Stats {
	out := Stats{}

	for _, a := range allocs {
		if a.RowerID == rowerID {
			out[a.Stat] += a.Points
		}
	}

	return out
}

// selectionAllowed reports whether r may be picked for the selection (rules.json ages; empty =
// everyone, e.g. the Wanderfahrt is for Junioren only).
func selectionAllowed(sel *data.Selection, r *rower) bool {
	return len(sel.Ages) == 0 || slices.Contains(sel.Ages, string(r.age))
}

// SelectionAllows reports whether a rower of age a may be picked in this selection phase.
func (pv PhaseView) SelectionAllows(a AgeGroup) bool {
	return len(pv.SelectionAges) == 0 || slices.Contains(pv.SelectionAges, a)
}

// selectionWant is the number of rowers pl must pick: the selection count, capped by the rowers
// allowed to take part.
func selectionWant(pl *player, p *phase) int {
	n := 0

	for _, r := range pl.rowers {
		if selectionAllowed(p.selection, r) {
			n++
		}
	}

	return min(p.count, n)
}

func validateSelection(pl *player, p *phase, ids []string) error {
	want := selectionWant(pl, p)
	seen := map[string]bool{}

	for _, id := range ids {
		r := pl.rower(id)
		if r == nil {
			return invalid("err.rowerNotOwned", "rower", id)
		}

		if !selectionAllowed(p.selection, r) {
			return invalid("err.selectAge", "rower", r.name)
		}

		if seen[id] {
			return invalid("err.selectTwice")
		}

		seen[id] = true
	}

	if len(ids) != want {
		return invalid("err.selectCount", "want", want, "count", len(ids))
	}

	return nil
}

func (g *Game) applySelection(pl *player, p *phase, ids []string) {
	names := make([]any, 0, len(ids))

	for _, id := range ids {
		r := pl.rower(id)

		for _, s := range AllStats {
			if v := p.selection.Effect[string(s)]; v != 0 {
				r.addStat(s, v)
			}
		}

		names = append(names, rowerName(r))
	}

	g.addLog("info", pl.id, msg("log.selection", "player", pl.name, "rowers", names,
		"effect", dataMsg("selection", p.selectionID, "label")))
}
