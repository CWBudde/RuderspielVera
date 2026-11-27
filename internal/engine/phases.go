package engine

import (
	"fmt"
	"slices"
	"strings"

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
	title     string
	prompt    string
	required  []string // player IDs in seat order
	decisions map[string]Decision

	points    int             // training / camp
	maxRowers int             // camp
	count     int             // selection
	selection *data.Selection // selection effect
	comp      *compState      // regatta
	swap      map[string]string
	prompts   map[string]string   // per-player prompt override
	newEvents map[string][]string // event reveal: rower IDs that got a card, per player
	ereignis  *EreignisView
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

func (g *Game) newPhase(kind PhaseKind, st stage, required []string, title, prompt string) *phase {
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
		g.addLog("station", "", st.Name)
	}

	all := g.allPlayerIDs()

	switch StationType(st.Type) {
	case StationSetup:
		g.newPhase(PhaseAck, stageSetup, all, "Spielvorbereitung",
			"Deine Startmannschaft steht: Ruderer, städtische Boote und Trainer. Los geht die Saison!")
	case StationTraining:
		pts := g.consumeTrainingPoints()
		p := g.newPhase(PhaseTraining, stageNone, all, "Training",
			fmt.Sprintf("Verteile genau %d Trainingspunkte auf deine Ruderer. Motivation lässt sich nicht trainieren.", pts))
		p.points = pts
	case StationCamp:
		c := g.d.Rules.Camp
		p := g.newPhase(PhaseTraining, stageNone, all, st.Name,
			fmt.Sprintf("Trainingslager: Verteile genau %d Trainingspunkte auf höchstens %d Ruderer. "+
				"Motivation lässt sich nicht trainieren.", c.Points, c.MaxRowers))
		p.points, p.maxRowers = c.Points, c.MaxRowers
	case StationSelection:
		sel := g.d.Rules.Selections[st.Selection]
		p := g.newPhase(PhaseSelection, stageNone, all, st.Name,
			fmt.Sprintf("Wähle genau %d Ruderer aus. Jede ausgewählte Person erhält %s.", sel.Count, sel.Label))
		p.count, p.selection = sel.Count, &sel
	case StationEvent:
		prompts, dealt := g.runEventStation()
		p := g.newPhase(PhaseAck, stageEventReveal, all, "Event", "Die Eventkarten wurden verteilt.")
		p.prompts, p.newEvents = prompts, dealt
	case StationEreignis:
		name, desc := g.runEreignisStation()
		p := g.newPhase(PhaseAck, stageEreignisReveal, all, "Ereignis", name+": "+desc)
		p.ereignis = &EreignisView{Name: name, Description: desc}
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
	g.newPhase(PhaseAck, stageResult, g.allPlayerIDs(), "Ergebnis: "+res.Name,
		"Die Ergebnisse stehen fest. Medaillen und Motivationspunkte wurden vergeben.")
}

func (g *Game) gameOver() {
	for _, pl := range g.players {
		for _, r := range pl.rowers {
			if r.event != nil && r.event.mission != nil && !r.event.mission.done && !r.event.mission.failed {
				r.event.mission.failed = true
				g.addLog("mission", pl.id, "Mission nicht erfüllt: "+r.event.mission.text)
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
	g.newPhase(PhaseGameOver, stageNone, nil, "Saisonende",
		fmt.Sprintf("Die Saison ist vorbei. Gewonnen hat %s mit %d Medaillenpunkten.", strings.Join(names, " und "), best))
	g.addLog("info", "", g.phase.prompt)
}
