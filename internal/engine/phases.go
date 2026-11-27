package engine

import (
	"fmt"
	"slices"

	"ruderspiel/internal/data"
)

// stage distinguishes ack phases (and other phases) that need different follow-ups.
type stage int

const (
	stageNone stage = iota
	stageSetup
	stageEventReveal
	stageEreignisReveal
	stageResult
)

// phase is one decision round. It is resolved when every required player has decided.
type phase struct {
	id        string
	kind      PhaseKind
	stage     stage
	title     Msg
	prompt    Msg
	required  []string // player IDs in seat order
	decisions map[string]Decision

	points      int             // training / camp
	maxRowers   int             // camp
	count       int             // selection
	selection   *data.Selection // selection effect
	selectionID string
	comp        *compState // regatta
	swap        map[string]string
	prompts     map[string]Msg      // per-player prompt override
	newEvents   map[string][]string // event reveal: rower IDs that got a card, per player
	ereignis    *EreignisView
}

func (p *phase) isRequired(playerID string) bool {
	return slices.Contains(p.required, playerID)
}

func (g *Game) allPlayerIDs() []string {
	ids := make([]string, len(g.players))
	for i, p := range g.players {
		ids[i] = p.id
	}

	return ids
}

func (g *Game) newPhase(kind PhaseKind, st stage, required []string, title, prompt Msg) *phase {
	p := &phase{
		id:        fmt.Sprintf("%d-%s-%d", g.station, kind, g.phaseSeq),
		kind:      kind,
		stage:     st,
		title:     title,
		prompt:    prompt,
		required:  required,
		decisions: map[string]Decision{},
	}
	g.phaseSeq++
	g.phase = p

	return p
}

// enterStation moves the calendar figure and opens the station's first phase.
func (g *Game) enterStation(i int) {
	g.station = i

	st := g.currentStation()
	if i > 0 {
		g.addLog("station", "", dataMsg("station", st.ID, "name"))

		if st.Section != g.d.Calendar[i-1].Section {
			g.driftMood()
		}
	}

	all := g.allPlayerIDs()

	switch StationType(st.Type) {
	case StationSetup:
		g.newPhase(PhaseAck, stageSetup, all, dataMsg("station", st.ID, "name"), msg("prompt.setup"))
	case StationTraining:
		pts := g.consumeTrainingPoints()
		p := g.newPhase(PhaseTraining, stageNone, all, dataMsg("station", st.ID, "name"),
			msg("prompt.training", "points", pts))
		p.points = pts
	case StationCamp:
		c := g.d.Rules.Camp
		p := g.newPhase(PhaseTraining, stageNone, all, dataMsg("station", st.ID, "name"),
			msg("prompt.camp", "points", c.Points, "maxRowers", c.MaxRowers))
		p.points, p.maxRowers = c.Points, c.MaxRowers
	case StationSelection:
		sel := g.d.Rules.Selections[st.Selection]
		p := g.newPhase(PhaseSelection, stageNone, all, dataMsg("station", st.ID, "name"),
			msg("prompt.selection", "count", sel.Count, "effect", dataMsg("selection", st.Selection, "label")))
		p.count, p.selection, p.selectionID = sel.Count, &sel, st.Selection
	case StationEvent:
		prompts, dealt := g.runEventStation()
		p := g.newPhase(PhaseAck, stageEventReveal, all, dataMsg("station", st.ID, "name"), msg("prompt.eventsDealt"))
		p.prompts, p.newEvents = prompts, dealt
	case StationEreignis:
		ev, details := g.runEreignisStation()
		p := g.newPhase(PhaseAck, stageEreignisReveal, all, dataMsg("station", st.ID, "name"),
			msg("prompt.ereignis", "name", dataMsg("ereignis", ev.ID, "name"),
				"description", dataMsg("ereignis", ev.ID, "description"), "details", list(details)))
		p.ereignis = &EreignisView{ID: ev.ID, Name: ev.Name, Description: ev.Description, Details: details}
	case StationRegatta, StationErgoCup, StationJTFO:
		g.openCompetition()
	}
}

func (g *Game) nextStation() {
	if g.station+1 >= len(g.d.Calendar) {
		g.gameOver()
		return
	}

	g.enterStation(g.station + 1)
}

// afterPhase opens whatever follows a resolved phase.
func (g *Game) afterPhase(p *phase) {
	switch p.kind {
	case PhaseAck:
		switch p.stage {
		case stageEventReveal:
			if g.openSwapPhase() {
				return
			}
		case stageResult:
			if g.station == len(g.d.Calendar)-1 {
				g.gameOver()
				return
			}

			if g.openDraft() {
				return
			}
		}
	case PhaseEventSwap:
		if g.openSwapPhase() {
			return
		}
	case PhaseRegatta:
		g.openResultAck()
		return
	case PhaseDraft:
		if g.nextDraftPick() {
			return
		}
	}

	g.nextStation()
}

func (g *Game) openResultAck() {
	res := g.results[len(g.results)-1]
	g.newPhase(PhaseAck, stageResult, g.allPlayerIDs(),
		msg("title.result", "name", dataMsg("station", g.d.Calendar[res.StationIndex].ID, "name")), msg("prompt.result"))
}

func (g *Game) gameOver() {
	for _, pl := range g.players {
		for _, r := range pl.rowers {
			if r.event != nil && r.event.mission != nil && !r.event.mission.done && !r.event.mission.failed {
				r.event.mission.failed = true
				g.addLog("mission", pl.id, msg("log.missionUnfulfilled", "mission", r.event.mission.text()))
			}
		}
	}

	best := -1
	for _, pl := range g.players {
		if pts := g.points(pl); pts > best {
			best = pts
		}
	}
	var names []string

	for _, pl := range g.players {
		if g.points(pl) == best {
			g.winners = append(g.winners, pl.id)
			names = append(names, pl.name)
		}
	}

	g.over = true
	g.newPhase(PhaseGameOver, stageNone, nil, msg("title.gameOver"),
		msg("prompt.gameOver", "winners", list(names), "count", len(names), "points", best))
	g.addLog("info", "", g.phase.prompt)
}
