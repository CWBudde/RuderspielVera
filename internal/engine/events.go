package engine

import (
	"ruderspiel/internal/data"
)

// runEventStation deals the personal event cards of the current event station and applies their
// immediate effects. It returns a summary per player for the reveal phase.
func (g *Game) runEventStation() (map[string]Msg, map[string][]string) {
	idx := g.eventStations
	g.eventStations++
	perStation := g.d.Rules.EventCardsPerStation
	n := perStation[min(idx, len(perStation)-1)]
	lines := map[string][]any{}
	dealt := map[string][]string{}
	trailing := g.trailingPlayers()

	for range n {
		for _, pl := range g.players {
			eligible := eventEligible(pl)
			if len(eligible) == 0 {
				continue
			}

			r := eligible[g.rng.IntN(len(eligible))]
			def := g.drawEventType(pl)
			text := g.applyEvent(pl, r, def)
			lines[pl.id] = append(lines[pl.id], text)
			dealt[pl.id] = append(dealt[pl.id], r.id)
			g.addLog("event", pl.id, msg("log.eventDealt", "player", pl.name, "rower", rowerName(r),
				"card", dataMsg("event", def.Type, "name")))

			if boost := g.d.Rules.ComebackBoost; boost > 0 && trailing[pl.id] {
				s := g.pickWeakestTrainable(r.stats)
				r.addStat(s, boost)
				m := msg("event.comeback", "rower", rowerName(r), "value", signed(boost), "stat", statMsg(s))
				lines[pl.id] = append(lines[pl.id], m)
				g.addLog("event", pl.id, m)
			}
		}
	}

	prompts := map[string]Msg{}

	for _, pl := range g.players {
		if len(lines[pl.id]) == 0 {
			prompts[pl.id] = msg("prompt.noEventCards")
			continue
		}

		prompts[pl.id] = msg("prompt.eventCards", "cards", lines[pl.id])
	}

	return prompts, dealt
}

// trailingPlayers returns the players in sole or shared last place by medal points; nobody when
// all players are level (e.g. before the first competition).
func (g *Game) trailingPlayers() map[string]bool {
	lo, hi := g.points(g.players[0]), g.points(g.players[0])
	for _, pl := range g.players[1:] {
		lo, hi = min(lo, g.points(pl)), max(hi, g.points(pl))
	}

	out := map[string]bool{}
	if lo == hi {
		return out
	}

	for _, pl := range g.players {
		if g.points(pl) == lo {
			out[pl.id] = true
		}
	}

	return out
}

// pickWeakestTrainable returns the lowest trainable stat (not Motivation); ties are drawn at random.
func (g *Game) pickWeakestTrainable(s Stats) Stat {
	var cands []Stat

	for _, st := range TrainableStats {
		switch {
		case len(cands) == 0 || s[st] < s[cands[0]]:
			cands = append(cands[:0], st)
		case s[st] == s[cands[0]]:
			cands = append(cands, st)
		}
	}

	return cands[g.rng.IntN(len(cands))]
}

// eventEligible lists own rowers without an event card.
func eventEligible(pl *player) []*rower {
	var out []*rower

	for _, r := range pl.rowers {
		if r.event == nil {
			out = append(out, r)
		}
	}

	return out
}

// drawEventType draws a card type among those that can still matter for pl (eventDrawable).
func (g *Game) drawEventType(pl *player) *data.EventType {
	weights := make([]int, len(g.d.Events))
	for i := range g.d.Events {
		if e := &g.d.Events[i]; g.eventDrawable(pl, e) {
			weights[i] = e.Weight
		}
	}

	return &g.d.Events[pickWeighted(g.rng, weights)]
}

// eventDrawable reports whether an event card type may be dealt to pl now: a mission only while
// one is achievable, a Seelenzwilling only to a team with a second rower to link, a water / ergo /
// context card only while a matching competition is still ahead (no Ergo-Bestzeit-Fieber after
// the last Ergo-Cup).
func (g *Game) eventDrawable(pl *player, e *data.EventType) bool {
	switch e.Type {
	case "mission":
		return len(g.possibleMissions()) > 0
	case "seelenzwilling":
		return len(pl.rowers) > 1
	}

	if e.Scope == data.ScopeAny && e.Context == "" {
		return true
	}

	for j := g.station + 1; j < len(g.d.Calendar); j++ {
		c := g.d.Competitions[g.d.Calendar[j].Competition]
		if c != nil && data.ScopeMatches(e.Scope, c) && (e.Context == "" || e.Context == c.Context) {
			return true
		}
	}

	return false
}

// applyEvent attaches the card to the rower and applies its immediate effect.
func (g *Game) applyEvent(pl *player, rw *rower, def *data.EventType) Msg {
	card := &eventCard{def: def}
	rw.event = card

	switch def.Type {
	case "phoenix":
		s := g.pickExtreme(rw.stats, false)
		rw.addStat(s, def.Value)

		return msg("event.gain", "card", dataMsg("event", def.Type, "name"), "rower", rowerName(rw),
			"value", signed(def.Value), "stat", statMsg(s))
	case "verdopplung":
		s := g.pickExtreme(rw.stats, true)
		gain := rw.stats[s]*def.Value - rw.stats[s]
		rw.addStat(s, gain)

		return msg("event.doubled", "card", dataMsg("event", def.Type, "name"), "rower", rowerName(rw),
			"stat", statMsg(s), "value", rw.stats[s])
	case "tausch":
		g.swapQueue[pl.id] = append(g.swapQueue[pl.id], rw.id)
		return msg("event.swap", "card", dataMsg("event", def.Type, "name"), "rower", rowerName(rw))
	case "seelenzwilling":
		card.partner = g.pickTwinPartner(pl, rw)

		return msg("event.twin", "card", dataMsg("event", def.Type, "name"), "rower", rowerName(rw),
			"partner", rowerName(card.partner))
	case "mission":
		ms := g.possibleMissions()
		m := ms[g.rng.IntN(len(ms))]
		card.mission = &missionState{def: m, rower: rowerName(rw)}

		return msg("event.mission", "mission", card.mission.text(), "reward", card.mission.reward())
	case "wachstum":
		return msg("event.described", "card", dataMsg("event", def.Type, "name"), "rower", rowerName(rw),
			"description", dataMsg("event", def.Type, "description"))
	}

	if len(def.Stats) > 0 || def.Bonus != 0 {
		for _, s := range AllStats {
			if v, ok := def.Stats[string(s)]; ok {
				rw.addStat(s, v)
			}
		}

		return msg("event.described", "card", dataMsg("event", def.Type, "name"), "rower", rowerName(rw),
			"description", dataMsg("event", def.Type, "description"))
	}

	if def.Stat != "" {
		s := Stat(def.Stat)
		rw.addStat(s, def.Value)

		return msg("event.gain", "card", dataMsg("event", def.Type, "name"), "rower", rowerName(rw),
			"value", signed(def.Value), "stat", statMsg(s))
	}

	return msg("event.plain", "card", dataMsg("event", def.Type, "name"), "rower", rowerName(rw))
}

// pickTwinPartner draws the Seelenzwilling partner of rw among the other own rowers, preferring
// the same gender and age group, then the same gender, then the same age group, then anyone.
func (g *Game) pickTwinPartner(pl *player, rw *rower) *rower {
	tiers := [4][]*rower{}

	for _, r := range pl.rowers {
		if r == rw {
			continue
		}

		tier := 3

		switch {
		case r.gender == rw.gender && r.age == rw.age:
			tier = 0
		case r.gender == rw.gender:
			tier = 1
		case r.age == rw.age:
			tier = 2
		}

		tiers[tier] = append(tiers[tier], r)
	}

	for _, cands := range tiers {
		if len(cands) > 0 {
			return cands[g.rng.IntN(len(cands))]
		}
	}

	return nil
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
	prompts := map[string]Msg{}

	for _, pl := range g.players {
		q := g.swapQueue[pl.id]
		if len(q) == 0 {
			continue
		}

		swap[pl.id] = q[0]
		g.swapQueue[pl.id] = q[1:]
		required = append(required, pl.id)
		prompts[pl.id] = msg("prompt.swap", "rower", rowerName(pl.rower(q[0])))
	}

	if len(required) == 0 {
		return false
	}

	p := g.newPhase(PhaseEventSwap, stageNone, required, msg("title.swap"), msg("prompt.swapOther"))
	p.swap, p.prompts = swap, prompts

	return true
}

func validateSwap(sw *[2]Stat) error {
	if sw == nil {
		return nil
	}

	if !validStat(sw[0]) || !validStat(sw[1]) {
		return invalid("err.swapUnknownStat")
	}

	if sw[0] == sw[1] {
		return invalid("err.swapSameStat")
	}

	if !Trainable(sw[0]) || !Trainable(sw[1]) {
		return invalid("err.swapMotivation")
	}

	return nil
}

func (g *Game) applySwap(pl *player, rowerID string, sw *[2]Stat) {
	r := pl.rower(rowerID)
	if sw == nil {
		g.addLog("event", pl.id, msg("log.swapSkipped", "player", pl.name, "rower", rowerName(r)))
		return
	}

	r.stats[sw[0]], r.stats[sw[1]] = r.stats[sw[1]], r.stats[sw[0]]
	g.addLog("event", pl.id, msg("log.swapped", "player", pl.name, "rower", rowerName(r),
		"a", statMsg(sw[0]), "b", statMsg(sw[1])))
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
				g.addLog("mission", pl.id, msg("log.missionFailed", "mission", ms.text()))

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
				g.addLog("mission", pl.id, msg("log.missionFailed", "mission", ms.text()))
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

	g.addLog("mission", pl.id, msg("log.missionDone", "mission", ms.text(), "reward", ms.reward()))
}
