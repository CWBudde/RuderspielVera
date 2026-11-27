package engine

import (
	"fmt"
	"strings"

	"ruderspiel/internal/data"
)

// runEventStation deals the personal event cards of the current event station and applies their
// immediate effects. It returns a German summary per player for the reveal phase.
func (g *Game) runEventStation() (map[string]string, map[string][]string) {
	idx := g.eventStations
	g.eventStations++
	perStation := g.d.Rules.EventCardsPerStation
	n := perStation[min(idx, len(perStation)-1)]
	lines := map[string][]string{}
	dealt := map[string][]string{}

	for range n {
		for _, pl := range g.players {
			eligible := eventEligible(pl)
			if len(eligible) == 0 {
				continue
			}

			r := eligible[g.rng.IntN(len(eligible))]
			def := g.drawEventType()
			text := g.applyEvent(pl, r, def)
			lines[pl.id] = append(lines[pl.id], text)
			dealt[pl.id] = append(dealt[pl.id], r.id)
			public := fmt.Sprintf("%s: %s erhält %s.", pl.name, r.name, def.Name)
			g.addLog("event", pl.id, public)
		}
	}

	prompts := map[string]string{}

	for _, pl := range g.players {
		if len(lines[pl.id]) == 0 {
			prompts[pl.id] = "Keiner deiner Ruderer kann noch eine Eventkarte erhalten."
			continue
		}

		prompts[pl.id] = "Deine Eventkarten: " + strings.Join(lines[pl.id], " ")
	}

	return prompts, dealt
}

// eventEligible lists own rowers without an event card (twins never get one).
func eventEligible(pl *player) []*rower {
	var out []*rower

	for _, r := range pl.rowers {
		if r.event == nil && !r.isTwin {
			out = append(out, r)
		}
	}

	return out
}

func (g *Game) drawEventType() *data.EventType {
	weights := make([]int, len(g.d.Events))
	for i, e := range g.d.Events {
		weights[i] = e.Weight
		if e.Type == "mission" && len(g.possibleMissions()) == 0 {
			weights[i] = 0
		}
	}

	return &g.d.Events[pickWeighted(g.rng, weights)]
}

// applyEvent attaches the card to the rower and applies its immediate effect.
func (g *Game) applyEvent(pl *player, rw *rower, def *data.EventType) string {
	card := &eventCard{def: def}
	rw.event = card

	switch def.Type {
	case "phoenix":
		s := g.pickExtreme(rw.stats, false)
		rw.addStat(s, def.Value)

		return fmt.Sprintf("Phönix: %s erhält +%d %s.", rw.name, def.Value, statName(s))
	case "verdopplung":
		s := g.pickExtreme(rw.stats, true)
		gain := rw.stats[s]*def.Value - rw.stats[s]
		rw.addStat(s, gain)

		return fmt.Sprintf("Verdopplung: %s von %s steigt auf %d.", statName(s), rw.name, rw.stats[s])
	case "tausch":
		g.swapQueue[pl.id] = append(g.swapQueue[pl.id], rw.id)
		return fmt.Sprintf("Tausch: Du darfst zwei Eigenschaften von %s tauschen.", rw.name)
	case "seelenzwilling":
		twin := &rower{
			id: rw.id + "-zwilling", name: rw.name + " (Zwilling)", age: rw.age, gender: rw.gender,
			artKey: rw.artKey, isTwin: true, stats: Stats{},
		}
		for _, s := range AllStats {
			twin.stats[s] = (rw.stats[s]*def.Value + 50) / 100
		}

		twin.base = twin.stats.clone()
		pl.rowers = append(pl.rowers, twin)

		return fmt.Sprintf("Seelenzwilling: %s kommt neu ins Team.", twin.name)
	case "mission":
		ms := g.possibleMissions()
		m := ms[g.rng.IntN(len(ms))]
		card.mission = &missionState{def: m, text: strings.ReplaceAll(m.Text, "{name}", rw.name)}

		return fmt.Sprintf("Geheime Mission: %s (Belohnung: %s)", card.mission.text, m.RewardText)
	case "wetter", "wachstum":
		return fmt.Sprintf("%s: %s – %s", def.Name, rw.name, def.Description)
	}

	if def.Stat != "" {
		s := Stat(def.Stat)
		rw.addStat(s, def.Value)

		return fmt.Sprintf("%s: %s erhält +%d %s.", def.Name, rw.name, def.Value, statName(s))
	}

	return fmt.Sprintf("%s: %s.", def.Name, rw.name)
}

// pickExtreme returns the highest (best) or lowest stat; ties are drawn at random.
func (g *Game) pickExtreme(s Stats, best bool) Stat {
	cands := make([]Stat, 0, len(AllStats))

	for _, st := range AllStats {
		switch {
		case len(cands) == 0:
			cands = append(cands, st)
		case s[st] == s[cands[0]]:
			cands = append(cands, st)
		case (best && s[st] > s[cands[0]]) || (!best && s[st] < s[cands[0]]):
			cands = append(cands[:0], st)
		}
	}

	if len(cands) == 1 {
		return cands[0]
	}

	return cands[g.rng.IntN(len(cands))]
}

// ---------- Tausch ----------

func (g *Game) openSwapPhase() bool {
	var required []string
	swap := map[string]string{}
	prompts := map[string]string{}

	for _, pl := range g.players {
		q := g.swapQueue[pl.id]
		if len(q) == 0 {
			continue
		}

		swap[pl.id] = q[0]
		g.swapQueue[pl.id] = q[1:]
		required = append(required, pl.id)
		prompts[pl.id] = fmt.Sprintf(
			"Tausch-Event: Du darfst zwei Eigenschaften von %s miteinander tauschen – oder darauf verzichten.",
			pl.rower(q[0]).name)
	}

	if len(required) == 0 {
		return false
	}

	p := g.newPhase(PhaseEventSwap, stageNone, required, "Tausch-Event", "Ein Mitspieler darf Eigenschaften tauschen.")
	p.swap, p.prompts = swap, prompts

	return true
}

func validateSwap(sw *[2]Stat) error {
	if sw == nil {
		return nil
	}

	if !validStat(sw[0]) || !validStat(sw[1]) {
		return invalid("Unbekannte Eigenschaft beim Tausch.")
	}

	if sw[0] == sw[1] {
		return invalid("Wähle zwei verschiedene Eigenschaften zum Tauschen.")
	}

	return nil
}

func (g *Game) applySwap(pl *player, rowerID string, sw *[2]Stat) {
	r := pl.rower(rowerID)
	if sw == nil {
		g.addLog("event", pl.id, fmt.Sprintf("%s verzichtet auf den Tausch bei %s.", pl.name, r.name))
		return
	}

	r.stats[sw[0]], r.stats[sw[1]] = r.stats[sw[1]], r.stats[sw[0]]
	g.addLog("event", pl.id, fmt.Sprintf("%s tauscht bei %s %s und %s.",
		pl.name, r.name, statName(sw[0]), statName(sw[1])))
}

// ---------- missions ----------

// possibleMissions lists missions that can still be fulfilled from the next station on.
func (g *Game) possibleMissions() []*data.Mission {
	var out []*data.Mission

	for i := range g.d.Missions {
		if g.missionPossible(&g.d.Missions[i], 0) {
			out = append(out, &g.d.Missions[i])
		}
	}

	return out
}

// missionPossible reports whether enough matching competitions remain after the current station.
// startsSoFar counts starts already achieved for "starts" missions.
func (g *Game) missionPossible(m *data.Mission, startsSoFar int) bool {
	cond := m.Condition
	if cond.Before != "" && g.competitionIndex(cond.Before) <= g.station {
		return false
	}

	remaining := 0

	for j := g.station + 1; j < len(g.d.Calendar); j++ {
		st := &g.d.Calendar[j]

		comp := g.d.Competitions[st.Competition]
		if comp == nil {
			continue
		}

		if cond.Before != "" && comp.ID == cond.Before {
			break
		}

		if g.competitionMatches(cond, comp) {
			remaining++
		}
	}

	if cond.Type == "starts" {
		return startsSoFar+remaining >= cond.Count
	}

	return remaining > 0
}

// competitionIndex returns the calendar index of a competition (or -1).
func (g *Game) competitionIndex(id string) int {
	for j, st := range g.d.Calendar {
		if st.Competition == id {
			return j
		}
	}

	return -1
}

func (g *Game) competitionMatches(cond data.MissionCondition, comp *data.Competition) bool {
	if cond.Competition != "" && comp.ID != cond.Competition {
		return false
	}

	if cond.Kind != "" && comp.Kind != cond.Kind {
		return false
	}

	if cond.BadWeather && !comp.Water {
		return false
	}

	if cond.Class != "" {
		found := false

		for _, r := range g.d.RacesFor(comp, len(g.players)) {
			if r.Class == cond.Class && comp.Kind != "jtfo" {
				found = true
			}
		}

		return found
	}

	return true
}

// rowerOutcome is what a rower achieved in one competition (for missions).
type rowerOutcome struct {
	medal Medal
	class BoatClass // "" for the JTFO team result
}

// checkMissions updates all open missions after a competition.
func (g *Game) checkMissions(cs *compState, outcomes map[string]rowerOutcome) {
	for _, pl := range g.players {
		for _, rw := range pl.rowers {
			if rw.event == nil || rw.event.mission == nil {
				continue
			}

			ms := rw.event.mission
			if ms.done || ms.failed {
				continue
			}

			cond := ms.def.Condition
			if cond.Before != "" && cond.Before == cs.def.ID {
				ms.failed = true
				g.addLog("mission", pl.id, "Mission gescheitert: "+ms.text)

				continue
			}

			out, started := outcomes[rw.id]

			switch cond.Type {
			case "starts":
				if started {
					ms.starts++
				}

				ms.done = ms.starts >= cond.Count
			case "medal":
				ms.done = started && g.medalMatches(cond, cs, out)
			}

			if ms.done {
				g.rewardMission(pl, rw, ms)
				continue
			}

			if !g.missionPossible(ms.def, ms.starts) {
				ms.failed = true
				g.addLog("mission", pl.id, "Mission gescheitert: "+ms.text)
			}
		}
	}
}

func (g *Game) medalMatches(cond data.MissionCondition, cs *compState, out rowerOutcome) bool {
	switch cond.Medal {
	case "gold":
		if out.medal != MedalGold {
			return false
		}
	case "any":
		if out.medal == MedalNone {
			return false
		}
	}

	if cond.Class != "" && string(out.class) != cond.Class {
		return false
	}

	if cond.Kind != "" && cs.def.Kind != cond.Kind {
		return false
	}

	if cond.Competition != "" && cs.def.ID != cond.Competition {
		return false
	}

	if cond.BadWeather && (cs.weather == "" || cs.weather == WeatherNormal) {
		return false
	}

	return true
}

func (g *Game) rewardMission(pl *player, r *rower, ms *missionState) {
	switch ms.def.Reward.Type {
	case "stat":
		s := r.stats.bestStat()
		r.addStat(s, ms.def.Reward.Value)
	case "points":
		pl.bonus += ms.def.Reward.Value
	}

	g.addLog("mission", pl.id, fmt.Sprintf("Mission erfüllt: %s Belohnung: %s.", ms.text, ms.def.RewardText))
}
