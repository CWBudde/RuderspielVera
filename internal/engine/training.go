package engine

import (
	"fmt"
	"slices"
	"strings"
)

func validStat(s Stat) bool { return slices.Contains(AllStats, s) }

func (g *Game) validateTraining(pl *player, p *phase, allocs []TrainingAlloc) error {
	total := 0
	rowers := map[string]bool{}
	for _, a := range allocs {
		if pl.rower(a.RowerID) == nil {
			return invalid("Der Ruderer %q gehört nicht zu deinem Team.", a.RowerID)
		}
		if !validStat(a.Stat) {
			return invalid("Unbekannte Eigenschaft %q.", a.Stat)
		}
		if a.Points <= 0 {
			return invalid("Trainingspunkte müssen positiv sein.")
		}
		total += a.Points
		rowers[a.RowerID] = true
	}
	if total != p.points {
		return invalid("Du musst genau %d Trainingspunkte verteilen (verteilt: %d).", p.points, total)
	}
	if p.maxRowers > 0 && len(rowers) > p.maxRowers {
		return invalid("Im Trainingslager dürfen höchstens %d Ruderer trainieren (gewählt: %d).", p.maxRowers, len(rowers))
	}
	return nil
}

// applyTraining adds the points and then triggers Wachstum cards (every training incl. camps).
func (g *Game) applyTraining(pl *player, allocs []TrainingAlloc) {
	for _, a := range allocs {
		pl.rower(a.RowerID).addStat(a.Stat, a.Points)
	}
	for _, r := range pl.rowers {
		if r.event != nil && r.event.def.Type == "wachstum" {
			s := r.stats.bestStat()
			r.addStat(s, r.event.def.Value)
			g.addLog("event", pl.id, fmt.Sprintf("Wachstum: %s erhält +%d %s.", r.name, r.event.def.Value, statName(s)))
		}
	}
}

func validateSelection(pl *player, p *phase, ids []string) error {
	want := min(p.count, len(pl.rowers))
	seen := map[string]bool{}
	for _, id := range ids {
		if pl.rower(id) == nil {
			return invalid("Der Ruderer %q gehört nicht zu deinem Team.", id)
		}
		if seen[id] {
			return invalid("Jeder Ruderer darf nur einmal ausgewählt werden.")
		}
		seen[id] = true
	}
	if len(ids) != want {
		return invalid("Wähle genau %d Ruderer aus (gewählt: %d).", want, len(ids))
	}
	return nil
}

func (g *Game) applySelection(pl *player, p *phase, ids []string) {
	var names []string
	for _, id := range ids {
		r := pl.rower(id)
		for _, s := range AllStats {
			if v := p.selection.Effect[string(s)]; v != 0 {
				r.addStat(s, v)
			}
		}
		names = append(names, r.name)
	}
	g.addLog("info", pl.id, fmt.Sprintf("%s: %s erhalten %s.", pl.name, strings.Join(names, ", "), p.selection.Label))
}

var statNames = map[Stat]string{
	StatKraft: "Kraft", StatAusdauer: "Ausdauer", StatSpeed: "Geschwindigkeit", StatTechnik: "Technik",
	StatMotivation: "Motivation",
}

func statName(s Stat) string { return statNames[s] }
