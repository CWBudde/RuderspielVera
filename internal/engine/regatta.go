package engine

import (
	"slices"
	"strconv"

	"ruderspiel/internal/data"
)

// compState is the open competition with all modifiers fixed when the phase opened.
type compState struct {
	def         *data.Competition
	station     int
	name        string
	races       []RaceSpec
	weights     map[Stat]float64
	weather     Weather // "" for ergo cups
	krebs       int     // Krebs probability in per mille
	material    int     // Materialschaden probability in per mille
	cityBoatMod int
	guestMod    float64 // added to every guest stat level
	notes       []Msg
}

func (cs *compState) race(id string) *RaceSpec {
	for i := range cs.races {
		if cs.races[i].ID == id {
			return &cs.races[i]
		}
	}

	return nil
}

func (cs *compState) team() bool { return cs.def.Kind == string(StationJTFO) }

func raceSpec(r data.Race) RaceSpec {
	rs := RaceSpec{ID: r.ID, Name: r.Name, NameKey: r.TextKey, Class: BoatClass(r.Class), Mixed: r.Mixed, Seats: r.Seats, LoanBoat: r.LoanBoat}
	for _, a := range r.Ages {
		rs.Ages = append(rs.Ages, AgeGroup(a))
	}

	for _, gd := range r.Genders {
		rs.Genders = append(rs.Genders, Gender(gd))
	}

	return rs
}

// openCompetition rolls the weather (water only), consumes matching Ereignisse and opens the
// regatta decision phase. Weather is known before trainer choice and lineup.
func (g *Game) openCompetition() {
	st := g.currentStation()
	def := g.d.Competitions[st.Competition]

	cs := &compState{
		def: def, station: g.station, name: st.Name, weights: map[Stat]float64{},
		krebs: g.d.Rules.Incidents.Krebs, material: g.d.Rules.Incidents.Material, notes: []Msg{},
	}
	for i := range def.Notes {
		cs.notes = append(cs.notes, dataMsg("competition", def.ID, "notes", strconv.Itoa(i)))
	}
	for _, r := range g.d.RacesFor(def, len(g.players)) {
		cs.races = append(cs.races, raceSpec(r))
	}

	for _, s := range AllStats {
		if w, ok := def.Weights[string(s)]; ok {
			cs.weights[s] = w
		}
	}

	g.applyCompetitionEreignisse(cs)

	prompt := msg("prompt.regatta")
	if cs.weather != "" {
		prompt = msg("prompt.regattaWeather", "weather", weatherMsg(cs.weather))
	}

	p := g.newPhase(PhaseRegatta, stageNone, g.allPlayerIDs(), dataMsg("station", st.ID, "name"), prompt)
	p.comp = cs
}

// applyCompetitionEreignisse consumes every pending competition modifier whose scope matches this
// competition (oldest first) and rolls the weather of a water competition unless an Ereignis
// forces it (the most recent one wins). RNG: at most one d6 for the weather.
func (g *Game) applyCompetitionEreignisse(cs *compState) {
	var mods []*data.Ereignis

	kept := g.pending[:0:0]
	for _, e := range g.pending {
		if data.CompetitionEffect(e.Effect) && data.ScopeMatches(e.Scope, cs.def) {
			mods = append(mods, e)
		} else {
			kept = append(kept, e)
		}
	}

	g.pending = kept

	if cs.def.Water {
		var forced *data.Ereignis

		for _, e := range mods {
			if e.Effect == data.EffForceWeather {
				forced = e
			}
		}

		if forced != nil {
			cs.weather = Weather(forced.Weather)
			cs.notes = append(cs.notes, msg("note.weatherFixed", "name", dataMsg("ereignis", forced.ID, "name")))
		} else {
			cs.weather = Weather(g.d.Rules.WeatherTable[rollD(g.rng, 6)-1])
		}

		g.addLog("info", "", msg("log.weather", "competition", dataMsg("station", g.d.Calendar[cs.station].ID, "name"),
			"weather", weatherMsg(cs.weather)))
	}

	for _, mod := range mods {
		switch mod.Effect {
		case data.EffKrebsChance:
			cs.krebs = mod.Value
			cs.notes = append(cs.notes, msg("note.krebsChance", "name", dataMsg("ereignis", mod.ID, "name"),
				"percent", float64(mod.Value)/10))
		case data.EffMaterialChance:
			cs.material = mod.Value
			cs.notes = append(cs.notes, msg("note.materialChance", "name", dataMsg("ereignis", mod.ID, "name"),
				"percent", float64(mod.Value)/10))
		case data.EffCityBoatMod:
			cs.cityBoatMod += mod.Value
			cs.notes = append(cs.notes, msg("note.cityBoats", "name", dataMsg("ereignis", mod.ID, "name"),
				"value", signed(mod.Value)))
		case data.EffGuestMod:
			cs.guestMod += float64(mod.Value)
			cs.notes = append(cs.notes, msg("note.guests", "name", dataMsg("ereignis", mod.ID, "name"),
				"value", signed(mod.Value)))
		case data.EffWeightMod:
			cs.notes = append(cs.notes, msg("note.weights", "name", dataMsg("ereignis", mod.ID, "name"),
				"weights", list(g.applyWeightMod(cs, mod.Weights))))
		}
	}
}

// applyWeightMod adds Ereignis weight deltas to the competition weights (in stat display order)
// and returns the new weights as notes ("Kraft zählt 3-fach").
func (g *Game) applyWeightMod(cs *compState, deltas map[string]float64) []Msg {
	var parts []Msg

	for _, s := range AllStats {
		d, ok := deltas[string(s)]
		if !ok {
			continue
		}

		cs.weights[s] = max(cs.weights[s]+d, 0)
		parts = append(parts, msg("note.weight", "stat", statMsg(s), "weight", cs.weights[s]))
	}

	return parts
}

func (g *Game) validateEntries(pl *player, cs *compState, trainerID string, entries []Entry) error {
	if trainerID != "" && pl.trainer(trainerID) == nil {
		return invalid("err.trainerNotOwned")
	}

	if len(entries) > 0 && trainerID == "" && len(pl.trainers) > 0 {
		return invalid("err.trainerRequired")
	}

	usedRowers := map[string]bool{}
	usedBoats := map[string]bool{}
	usedRaces := map[string]bool{}

	for _, en := range entries {
		race := cs.race(en.RaceID)
		if race == nil {
			return invalid("err.unknownRace", "race", en.RaceID)
		}

		if usedRaces[en.RaceID] {
			return invalid("err.raceTwice", "race", raceName(race))
		}

		usedRaces[en.RaceID] = true
		if len(en.RowerIDs) != race.Seats {
			return invalid("err.raceSeats", "race", raceName(race), "seats", race.Seats, "count", len(en.RowerIDs))
		}

		err := validateCrew(pl, race, en.RowerIDs, usedRowers)
		if err != nil {
			return err
		}

		err = validateBoat(pl, race, en.BoatID, usedBoats)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateCrew checks the rowers of one entry; usedRowers tracks starts across the competition.
func validateCrew(pl *player, race *RaceSpec, rowerIDs []string, usedRowers map[string]bool) error {
	rowers := make([]*rower, 0, len(rowerIDs))
	for _, id := range rowerIDs {
		rw := pl.rower(id)
		if rw == nil {
			return invalid("err.rowerNotOwned", "rower", id)
		}

		if usedRowers[id] {
			return invalid("err.rowerTwice", "rower", rowerName(rw))
		}

		usedRowers[id] = true

		if !slices.Contains(race.Ages, rw.age) {
			return invalid("err.rowerAge", "rower", rowerName(rw), "race", raceName(race))
		}

		if !slices.Contains(race.Genders, rw.gender) {
			return invalid("err.rowerGender", "rower", rowerName(rw), "race", raceName(race))
		}

		rowers = append(rowers, rw)
	}

	if race.Mixed && !mixedOK(rowers) {
		return invalid("err.mixed", "race", raceName(race))
	}

	return nil
}

// validateBoat checks the boat of one entry; usedBoats tracks boats across the competition.
func validateBoat(pl *player, race *RaceSpec, boatID string, usedBoats map[string]bool) error {
	switch race.Class {
	case BoatErgo, Boat8x:
		if boatID != "" {
			return invalid("err.noOwnBoat", "race", raceName(race))
		}

		return nil
	}

	if boatID == "" {
		if !race.LoanBoat {
			return invalid("err.boatRequired", "race", raceName(race))
		}

		return nil
	}

	b := pl.boat(boatID)
	if b == nil {
		return invalid("err.boatNotOwned", "boat", boatID)
	}

	if b.class != race.Class {
		return invalid("err.boatClass", "race", raceName(race), "class", msg("boatClass."+string(race.Class)),
			"boat", dataMsg("boat", b.defID, "name"), "boatClass", msg("boatClass."+string(b.class)))
	}

	if usedBoats[b.id] {
		return invalid("err.boatTwice", "boat", dataMsg("boat", b.defID, "name"))
	}

	usedBoats[b.id] = true

	return nil
}

func mixedOK(rs []*rower) bool {
	girls, boys := false, false
	ages := map[AgeGroup]bool{}

	for _, r := range rs {
		girls = girls || r.gender == GenderF
		boys = boys || r.gender == GenderM
		ages[r.age] = true
	}

	return girls && boys && len(ages) >= 2
}

// raceEntry is a validated entry with its resolved cards.
type raceEntry struct {
	pl      *player
	rowers  []*rower
	boat    *boat
	trainer *trainer
}

func (g *Game) buildEntry(pl *player, trainerID string, e Entry) raceEntry {
	re := raceEntry{pl: pl, boat: pl.boat(e.BoatID), trainer: pl.trainer(trainerID)}
	for _, id := range e.RowerIDs {
		re.rowers = append(re.rowers, pl.rower(id))
	}

	return re
}

// raceEntries resolves the entries of all required players for one race, in player order.
func (g *Game) raceEntries(p *phase, raceID string) []raceEntry {
	var entries []raceEntry

	for _, pid := range p.required {
		d := p.decisions[pid]
		for _, e := range d.Entries {
			if e.RaceID == raceID {
				entries = append(entries, g.buildEntry(g.playerByID(pid), d.TrainerID, e))
			}
		}
	}

	return entries
}

// teamAcc accumulates one player's boats into a team score (JTFO).
type teamAcc struct {
	entry EntryResult
	found bool
}

// addToTeam adds a ranked race entry to its player's team score.
func addToTeam(team map[string]*teamAcc, er EntryResult) {
	acc := team[er.PlayerID]
	if acc == nil {
		acc = &teamAcc{entry: EntryResult{PlayerID: er.PlayerID, TrainerID: er.TrainerID}}
		team[er.PlayerID] = acc
	}

	acc.found = true

	acc.entry.RowerIDs = append(acc.entry.RowerIDs, er.RowerIDs...)
	if er.Incident == IncidentKentern || er.Incident == IncidentMaterial {
		return // this boat contributes nothing to the team score
	}

	acc.entry.Crew += er.Crew
	acc.entry.BoatValue += er.BoatValue
	acc.entry.Trainer += er.Trainer
	acc.entry.EventBonus += er.EventBonus
	acc.entry.Penalty += er.Penalty
	acc.entry.Total += er.Total
}

// guestTeamEntry sums the guest crews of all races into one team entry, if there are any.
func (g *Game) guestTeamEntry(cs *compState) (EntryResult, bool) {
	guest, found := EntryResult{Guest: g.d.Rules.Guests.Name}, false

	for _, race := range cs.races {
		if ge, ok := g.guestEntry(cs, race); ok {
			guest.Crew += ge.Crew
			guest.Total += ge.Total
			found = true
		}
	}

	return guest, found
}

// scoreRace scores all entries of one race, rolls their incidents (water only), adds the guest
// crew (not in team events) and ranks the result.
func (g *Game) scoreRace(p *phase, race RaceSpec) RaceResult {
	cs := p.comp
	rr := RaceResult{RaceID: race.ID, Name: race.Name, NameKey: race.NameKey}

	for _, re := range g.raceEntries(p, race.ID) {
		er := g.scoreEntry(cs, race, re)
		if cs.def.Water {
			inc := g.d.Rules.Incidents

			technik := 0
			for _, r := range re.rowers {
				technik += r.stat(StatTechnik)
			}

			krebs := krebsChance(cs.krebs, race.Seats, technik, inc.KrebsScaling)
			er.Incident = rollIncident(g.rng, inc.Kentern(race.Seats), cs.material, krebs)
			if er.Incident == IncidentKrebs {
				er.Penalty = g.d.Rules.KrebsPenalty
				er.Total -= er.Penalty
			}
		}

		rr.Entries = append(rr.Entries, er)
	}

	if len(rr.Entries) > 0 && !cs.team() {
		if ge, ok := g.guestEntry(cs, race); ok {
			rr.Entries = append(rr.Entries, ge)
		}
	}

	rankEntries(rr.Entries)

	return rr
}

// resolveCompetition scores all entries, rolls incidents, ranks, awards medals and updates the
// Siegesmotivation (count down first, then grant the new wins).
func (g *Game) resolveCompetition(p *phase) {
	cs := p.comp
	res := CompetitionResult{StationIndex: cs.station, Name: cs.name, Weather: cs.weather}
	outcomes := map[string]rowerOutcome{}
	team := map[string]*teamAcc{}

	for _, race := range cs.races {
		rr := g.scoreRace(p, race)

		if cs.team() {
			for i := range rr.Entries {
				rr.Entries[i].Medal = MedalNone
				addToTeam(team, rr.Entries[i])
			}
		} else {
			for _, er := range rr.Entries {
				g.awardMedal(er, race.Class, outcomes)
			}
		}

		res.Races = append(res.Races, rr)
	}

	if cs.team() {
		tr := RaceResult{RaceID: "team", Name: "Teamwertung", NameKey: "race.team"}

		for _, pid := range p.required {
			if acc := team[pid]; acc != nil && acc.found {
				tr.Entries = append(tr.Entries, acc.entry)
			}
		}

		if len(tr.Entries) > 0 {
			if guest, ok := g.guestTeamEntry(cs); ok {
				tr.Entries = append(tr.Entries, guest)
			}
		}

		rankEntries(tr.Entries)

		for _, er := range tr.Entries {
			g.awardMedal(er, "", outcomes)
		}

		res.Races = append(res.Races, tr)
	}

	g.results = append(g.results, res)
	g.logResult(res)
	g.updateWinMotivation(outcomes)
	g.checkMissions(cs, outcomes)
}

// updateWinMotivation counts the Siegesmotivation of every rower down by one competition (it was
// just used in scoring) and then grants it to every rower of a crew with a winning placement:
// winBonus = value (never more, no stacking) and the full duration again (refresh).
func (g *Game) updateWinMotivation(outcomes map[string]rowerOutcome) {
	wm := g.d.Rules.WinMotivation

	for _, pl := range g.players {
		for _, r := range pl.rowers {
			if r.winLeft > 0 {
				r.winLeft--
			}

			if r.winLeft == 0 {
				r.winBonus = 0
			}

			if out, ok := outcomes[r.id]; ok && wm.Value > 0 && slices.Contains(wm.Medals, string(out.medal)) {
				r.winBonus, r.winLeft = wm.Value, wm.Competitions
			}
		}
	}
}

// awardMedal credits the medal to the player and records rower outcomes for missions and the
// Siegesmotivation.
func (g *Game) awardMedal(er EntryResult, class BoatClass, outcomes map[string]rowerOutcome) {
	pl := g.playerByID(er.PlayerID)
	if pl == nil {
		return // guest crew: takes the place, scores nothing
	}

	switch er.Medal {
	case MedalGold:
		pl.gold++
	case MedalSilver:
		pl.silver++
	}

	for _, id := range er.RowerIDs {
		outcomes[id] = rowerOutcome{medal: er.Medal, class: class}
	}
}

func (g *Game) logResult(res CompetitionResult) {
	comp := dataMsg("station", g.d.Calendar[res.StationIndex].ID, "name")

	for _, rr := range res.Races {
		var parts []Msg

		for _, er := range rr.Entries {
			var name any = dataMsg("guest", "name")
			if pl := g.playerByID(er.PlayerID); pl != nil {
				name = pl.name
			}

			m := msg("log.resultEntry", "name", name, "total", er.Total)
			if er.Medal != MedalNone {
				m = msg("log.resultEntryMedal", "medal", msg("medal."+string(er.Medal)), "name", name, "total", er.Total)
			}

			if er.Incident != IncidentNone {
				m = msg("log.resultIncident", "entry", m, "incident", msg("incident."+string(er.Incident)))
			}

			parts = append(parts, m)
		}

		if len(parts) == 0 {
			continue
		}

		g.addLog("result", "", msg("log.result", "competition", comp, "race", Msg{Key: rr.NameKey}, "entries", list(parts)))
	}
}
