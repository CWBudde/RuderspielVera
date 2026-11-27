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
	for k := 0; k < n; k++ {
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
func (g *Game) applyEvent(pl *player, r *rower, def *data.EventType) string {
	card := &eventCard{def: def}
	r.event = card
	switch def.Type {
	case "phoenix":
		s := g.pickExtreme(r.stats, false)
		r.addStat(s, def.Value)
		return fmt.Sprintf("Phönix: %s erhält +%d %s.", r.name, def.Value, statName(s))
	case "verdopplung":
		s := g.pickExtreme(r.stats, true)
		gain := r.stats[s]*def.Value - r.stats[s]
		r.addStat(s, gain)
		return fmt.Sprintf("Verdopplung: %s von %s steigt auf %d.", statName(s), r.name, r.stats[s])
	case "tausch":
		g.swapQueue[pl.id] = append(g.swapQueue[pl.id], r.id)
		return fmt.Sprintf("Tausch: Du darfst zwei Eigenschaften von %s tauschen.", r.name)
	case "seelenzwilling":
		twin := &rower{
			id: r.id + "-zwilling", name: r.name + " (Zwilling)", age: r.age, gender: r.gender,
			artKey: r.artKey, isTwin: true, stats: Stats{},
		}
		for _, s := range AllStats {
			twin.stats[s] = (r.stats[s]*def.Value + 50) / 100
		}
		twin.base = twin.stats.clone()
		pl.rowers = append(pl.rowers, twin)
		return fmt.Sprintf("Seelenzwilling: %s kommt neu ins Team.", twin.name)
	case "mission":
		ms := g.possibleMissions()
		m := ms[g.rng.IntN(len(ms))]
		card.mission = &missionState{def: m, text: strings.ReplaceAll(m.Text, "{name}", r.name)}
		return fmt.Sprintf("Geheime Mission: %s (Belohnung: %s)", card.mission.text, m.RewardText)
	case "wetter", "wachstum":
		return fmt.Sprintf("%s: %s – %s", def.Name, r.name, def.Description)
	}
	if def.Stat != "" {
		s := Stat(def.Stat)
		r.addStat(s, def.Value)
		return fmt.Sprintf("%s: %s erhält +%d %s.", def.Name, r.name, def.Value, statName(s))
	}
	return fmt.Sprintf("%s: %s.", def.Name, r.name)
}

// pickExtreme returns the highest (best) or lowest stat; ties are drawn at random.
func (g *Game) pickExtreme(s Stats, best bool) Stat {
	var cands []Stat
	for _, st := range AllStats {
		switch {
		case len(cands) == 0:
			cands = []Stat{st}
		case s[st] == s[cands[0]]:
			cands = append(cands, st)
		case (best && s[st] > s[cands[0]]) || (!best && s[st] < s[cands[0]]):
			cands = []Stat{st}
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
		prompts[pl.id] = fmt.Sprintf("Tausch-Event: Du darfst zwei Eigenschaften von %s miteinander tauschen – oder darauf verzichten.",
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
	g.addLog("event", pl.id, fmt.Sprintf("%s tauscht bei %s %s und %s.", pl.name, r.name, statName(sw[0]), statName(sw[1])))
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
	c := m.Condition
	if c.Before != "" && g.competitionIndex(c.Before) <= g.station {
		return false
	}
	remaining := 0
	for j := g.station + 1; j < len(g.d.Calendar); j++ {
		st := &g.d.Calendar[j]
		comp := g.d.Competitions[st.Competition]
		if comp == nil {
			continue
		}
		if c.Before != "" && comp.ID == c.Before {
			break
		}
		if g.competitionMatches(c, comp) {
			remaining++
		}
	}
	if c.Type == "starts" {
		return startsSoFar+remaining >= c.Count
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

func (g *Game) competitionMatches(c data.MissionCondition, comp *data.Competition) bool {
	if c.Competition != "" && comp.ID != c.Competition {
		return false
	}
	if c.Kind != "" && comp.Kind != c.Kind {
		return false
	}
	if c.BadWeather && !comp.Water {
		return false
	}
	if c.Class != "" {
		found := false
		for _, r := range g.d.RacesFor(comp, len(g.players)) {
			if r.Class == c.Class && comp.Kind != "jtfo" {
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
		for _, r := range pl.rowers {
			if r.event == nil || r.event.mission == nil {
				continue
			}
			ms := r.event.mission
			if ms.done || ms.failed {
				continue
			}
			c := ms.def.Condition
			if c.Before != "" && c.Before == cs.def.ID {
				ms.failed = true
				g.addLog("mission", pl.id, fmt.Sprintf("Mission gescheitert: %s", ms.text))
				continue
			}
			out, started := outcomes[r.id]
			switch c.Type {
			case "starts":
				if started {
					ms.starts++
				}
				ms.done = ms.starts >= c.Count
			case "medal":
				ms.done = started && g.medalMatches(c, cs, out)
			}
			if ms.done {
				g.rewardMission(pl, r, ms)
				continue
			}
			if !g.missionPossible(ms.def, ms.starts) {
				ms.failed = true
				g.addLog("mission", pl.id, fmt.Sprintf("Mission gescheitert: %s", ms.text))
			}
		}
	}
}

func (g *Game) medalMatches(c data.MissionCondition, cs *compState, out rowerOutcome) bool {
	switch c.Medal {
	case "gold":
		if out.medal != MedalGold {
			return false
		}
	case "any":
		if out.medal == MedalNone {
			return false
		}
	}
	if c.Class != "" && string(out.class) != c.Class {
		return false
	}
	if c.Kind != "" && cs.def.Kind != c.Kind {
		return false
	}
	if c.Competition != "" && cs.def.ID != c.Competition {
		return false
	}
	if c.BadWeather && (cs.weather == "" || cs.weather == WeatherNormal) {
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
