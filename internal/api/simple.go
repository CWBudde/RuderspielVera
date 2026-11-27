package api

import (
	mrand "math/rand/v2"

	"ruderspiel/internal/engine"
)

// SimpleDecider is a minimal, always-legal decider that only uses the public view: it trains one
// random rower, picks the first rowers, never swaps, never enters races and drafts the first card.
// It is a fallback for tests and for running the server without internal/bot.
func SimpleDecider(g *engine.Game, playerID string, _ int, rng *mrand.Rand) (engine.Decision, error) {
	v := g.View(playerID)
	pv := v.Phase
	d := engine.Decision{PhaseID: pv.ID, Kind: pv.Kind}
	var rowers []engine.RowerView
	for _, p := range v.Players {
		if p.ID == playerID {
			rowers = p.Rowers
		}
	}
	switch pv.Kind {
	case engine.PhaseTraining:
		if len(rowers) > 0 && pv.Points > 0 {
			r := rowers[rng.IntN(len(rowers))]
			stat := engine.AllStats[rng.IntN(len(engine.AllStats))]
			d.Training = []engine.TrainingAlloc{{RowerID: r.ID, Stat: stat, Points: pv.Points}}
		}
	case engine.PhaseSelection:
		for _, r := range rowers[:min(pv.Count, len(rowers))] {
			d.RowerIDs = append(d.RowerIDs, r.ID)
		}
	case engine.PhaseDraft:
		if len(pv.DraftOffer) > 0 {
			d.DraftCardID = pv.DraftOffer[0].ID
		}
	}
	return d, nil
}
