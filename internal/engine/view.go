package engine

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"ruderspiel/internal/data"
)

// View returns the game as seen by one player. Other players' pending decisions and secret
// mission texts are never included.
func (g *Game) View(playerID string) GameView {
	gv := GameView{
		GameID:           g.cfg.GameID,
		You:              playerID,
		Current:          g.station,
		Phase:            g.CurrentPhase(playerID),
		ActiveEreignisse: g.activeEreignisse(),
		Log:              g.Log(),
		Winners:          g.Winners(),
	}
	for i, st := range g.d.Calendar {
		gv.Calendar = append(gv.Calendar, StationView{
			ID: st.ID, Index: i, Section: st.Section, Type: StationType(st.Type), Name: st.Name, Info: st.Info,
		})
	}

	for _, pl := range g.players {
		_, decided := g.phase.decisions[pl.id]

		ps := PlayerSummary{
			ID: pl.id, Name: pl.name, IsBot: pl.isBot, Gold: pl.gold, Silver: pl.silver, Points: g.points(pl),
			Decided:  decided || !g.phase.isRequired(pl.id),
			Rowers:   []RowerView{},
			Boats:    []BoatView{},
			Trainers: []TrainerView{},
		}
		for _, r := range pl.rowers {
			ps.Rowers = append(ps.Rowers, rowerView(r, pl.id == playerID))
		}

		for _, b := range pl.boats {
			ps.Boats = append(ps.Boats, boatView(b))
		}

		for _, t := range pl.trainers {
			ps.Trainers = append(ps.Trainers, g.trainerView(t.def))
		}

		gv.Players = append(gv.Players, ps)
	}

	if n := len(g.results); n > 0 {
		last := cloneResult(g.results[n-1])
		gv.LastResult = &last
	}

	fillNil(&gv)

	return gv
}

// CurrentPhase returns what the open phase asks of the given player.
func (g *Game) CurrentPhase(playerID string) PhaseView {
	p := g.phase
	_, decided := p.decisions[playerID]

	pv := PhaseView{
		ID: p.id, Kind: p.kind, Title: p.title, Prompt: p.prompt,
		Points: p.points, MaxRowers: p.maxRowers, Count: p.count,
		Waiting: p.kind != PhaseGameOver && (decided || !p.isRequired(playerID)),
	}
	if m, ok := p.prompts[playerID]; ok {
		pv.Prompt = m
	}

	if p.selection != nil {
		pv.SelectionEffect = p.selection.Label
		pv.SelectionID = p.selectionID

		for _, a := range p.selection.Ages {
			pv.SelectionAges = append(pv.SelectionAges, AgeGroup(a))
		}

		if pl := g.playerByID(playerID); pl != nil {
			pv.Count = selectionWant(pl, p)
		}
	}

	if p.swap != nil {
		pv.SwapRowerID = p.swap[playerID]
	}

	if cs := p.comp; cs != nil {
		pv.Competition = &CompetitionView{
			ID: cs.def.ID, Name: cs.name, Kind: StationType(cs.def.Kind), Weights: maps.Clone(cs.weights),
			Races: g.raceViews(cs), Weather: cs.weather, TeamEvent: cs.team(),
			Notes: slices.Clone(cs.notes), Context: cs.def.Context, CityBoatMod: cs.cityBoatMod,
		}
	}

	if p.kind == PhaseDraft && g.draft != nil {
		for _, c := range g.draft.offer {
			pv.DraftOffer = append(pv.DraftOffer, g.draftCardView(c))
		}

		pv.DraftOrder = slices.Clone(g.draft.order)
		pv.DraftTurn = g.draft.turn
	}

	pv.Stage = stageNames[p.stage]
	if pl := g.playerByID(playerID); pl != nil {
		for _, id := range p.newEvents[playerID] {
			if r := pl.rower(id); r != nil && r.event != nil {
				pv.NewEvents = append(pv.NewEvents, *rowerView(r, true).Event)
			}
		}
	}

	if p.ereignis != nil {
		e := *p.ereignis
		pv.Ereignis = &e
	}

	return pv
}

var stageNames = map[stage]PhaseStage{
	stageSetup: StageSetup, stageEventReveal: StageEvent, stageEreignisReveal: StageEreignis, stageResult: StageResult,
}

// raceViews returns the open competition's races with the scoring hints for client estimates.
func (g *Game) raceViews(cs *compState) []RaceSpec {
	races := slices.Clone(cs.races)
	for i := range races {
		r := &races[i]
		r.Ages, r.Genders = slices.Clone(r.Ages), slices.Clone(r.Genders)
		r.BoatScale = g.d.Rules.BoatScale(r.Seats)

		r.TrainerScale = g.d.Rules.TrainerScale(r.Seats)
		switch {
		case r.Class == BoatErgo:
			r.BoatScale = 0
		case r.Class == Boat8x || r.LoanBoat:
			r.FixedBoatValue = g.boatValue(cs, *r, nil)
		}

		if gs, ok := guestScore(g.d, cs.def, cs.station, *r, cs.guestMod); ok && !cs.team() {
			r.GuestScore = int(gs)
		}
	}

	return races
}

func rowerView(r *rower, owner bool) RowerView {
	v := RowerView{
		ID: r.id, Name: r.name, Age: r.age, Gender: r.gender, Stats: r.effectiveStats(), Base: r.base.clone(),
		ArtKey: r.artKey, WinMotivation: r.winBonus, WinMotivationLeft: r.winLeft,
	}
	if r.event != nil {
		ev := &EventCardView{
			Type: r.event.def.Type, Name: r.event.def.Name, Description: r.event.def.Description, RowerID: r.id,
			Spent: r.event.spent,
		}
		if p := r.event.partner; p != nil {
			ev.PartnerID, ev.PartnerName = p.id, p.name
		}
		if ms := r.event.mission; ms != nil && (owner || ms.done || ms.failed) {
			ev.Mission = &MissionView{
				ID: ms.def.ID, RowerName: r.name, Text: strings.ReplaceAll(ms.def.Text, "{name}", r.name),
				Reward: ms.def.RewardText, Done: ms.done, Failed: ms.failed,
			}
		}

		v.Event = ev
	}

	return v
}

func boatView(b *boat) BoatView {
	return BoatView{ID: b.id, DefID: b.defID, Name: b.name, Class: b.class, Value: b.value, City: b.city, ArtKey: b.artKey}
}

// trainerView is a trainer card with its current mood (Laune) in this game.
func (g *Game) trainerView(t *data.Trainer) TrainerView {
	v := trainerView(t)
	v.Mood = g.mood[t.ID]
	v.MoodTrend = g.mood[t.ID] - g.prevMood[t.ID]

	return v
}

// trainerView is the static trainer card (mood 0, e.g. in the catalog).
func trainerView(t *data.Trainer) TrainerView {
	v := TrainerView{
		ID: t.ID, Name: t.Name, Ability: t.Ability, Description: t.Description, ArtKey: t.ArtKey,
		Effects: []TrainerEffect{},
	}
	for _, e := range t.Effects {
		v.Effects = append(v.Effects, TrainerEffect{Weather: Weather(e.Weather), Context: e.Context, Value: e.Value})
	}

	return v
}

func (g *Game) draftCardView(c poolCard) DraftCard {
	dc := DraftCard{ID: c.id(), Kind: c.kind()}
	switch {
	case c.rower != nil:
		rv := rowerView(c.rower, false)
		dc.Rower = &rv
	case c.boat != nil:
		bv := boatView(c.boat)
		dc.Boat = &bv
	default:
		tv := g.trainerView(c.trainer.def)
		dc.Trainer = &tv
	}

	return dc
}

// Catalog returns the static card data (GET /api/catalog).
func Catalog() (CatalogView, error) {
	d, err := data.Load()
	if err != nil {
		return CatalogView{}, fmt.Errorf("rule data: %w", err)
	}

	var cv CatalogView
	for i := range d.Rowers {
		cv.Rowers = append(cv.Rowers, rowerView(newRowerFromData(&d.Rowers[i]), false))
	}

	for _, b := range d.Boats {
		cv.Boats = append(cv.Boats, BoatView{
			ID: b.ID, DefID: b.ID, Name: b.Name, Class: BoatClass(b.Class), Value: b.Value,
			City: b.City, ArtKey: b.ArtKey,
		})
	}

	for i := range d.Trainers {
		cv.Trainers = append(cv.Trainers, trainerView(&d.Trainers[i]))
	}

	for _, e := range d.Events {
		cv.Events = append(cv.Events, EventTypeView{Type: e.Type, Name: e.Name, Description: e.Description})
	}

	fillNil(&cv)

	return cv, nil
}
