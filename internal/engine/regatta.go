package engine

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

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
	notes       []string
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
	rs := RaceSpec{ID: r.ID, Name: r.Name, Class: BoatClass(r.Class), Mixed: r.Mixed, Seats: r.Seats, LoanBoat: r.LoanBoat}
	for _, a := range r.Ages {
		rs.Ages = append(rs.Ages, AgeGroup(a))
	}

	for _, gd := range r.Genders {
		rs.Genders = append(rs.Genders, Gender(gd))
	}

	return rs
}

var weatherNames = map[Weather]string{
	WeatherRegen: "Regen", WeatherHitze: "Hitze", WeatherWind: "Wind", WeatherKaelte: "Kälte", WeatherNormal: "Normal",
}

// openCompetition rolls the weather (water only), consumes matching Ereignisse and opens the
// regatta decision phase. Weather is known before trainer choice and lineup.
func (g *Game) openCompetition() {
	st := g.currentStation()
	def := g.d.Competitions[st.Competition]

	cs := &compState{
		def: def, station: g.station, name: st.Name, weights: map[Stat]float64{},
		krebs: g.d.Rules.Incidents.Krebs, material: g.d.Rules.Incidents.Material, notes: slices.Clone(def.Notes),
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

	prompt := "Wähle einen Trainer und melde deine Mannschaften. Jeder Ruderer und jedes Boot darf nur einmal starten."
	if cs.weather != "" {
		prompt = fmt.Sprintf("Wetter: %s. %s", weatherNames[cs.weather], prompt)
	}

	p := g.newPhase(PhaseRegatta, stageNone, g.allPlayerIDs(), st.Name, prompt)
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
			cs.notes = append(cs.notes, forced.Name+": Das Wetter steht fest.")
		} else {
			cs.weather = Weather(g.d.Rules.WeatherTable[rollD(g.rng, 6)-1])
		}

		g.addLog("info", "", fmt.Sprintf("Wetter bei %s: %s.", cs.name, weatherNames[cs.weather]))
	}

	for _, mod := range mods {
		switch mod.Effect {
		case data.EffKrebsChance:
			cs.krebs = mod.Value
			cs.notes = append(cs.notes, fmt.Sprintf("%s: Krebs-Wahrscheinlichkeit %s %%.", mod.Name, perMilleText(mod.Value)))
		case data.EffMaterialChance:
			cs.material = mod.Value
			cs.notes = append(cs.notes, fmt.Sprintf("%s: Materialschaden-Wahrscheinlichkeit %s %%.",
				mod.Name, perMilleText(mod.Value)))
		case data.EffCityBoatMod:
			cs.cityBoatMod += mod.Value
			cs.notes = append(cs.notes, fmt.Sprintf("%s: Städtische Boote %+d.", mod.Name, mod.Value))
		case data.EffGuestMod:
			cs.guestMod += float64(mod.Value)
			cs.notes = append(cs.notes, fmt.Sprintf("%s: Gastmannschaften %+d auf jede Eigenschaft.", mod.Name, mod.Value))
		case data.EffWeightMod:
			cs.notes = append(cs.notes, mod.Name+": "+g.applyWeightMod(cs, mod.Weights))
		}
	}
}

// applyWeightMod adds Ereignis weight deltas to the competition weights (in stat display order)
// and returns a German note like "Kraft zählt 3-fach.".
func (g *Game) applyWeightMod(cs *compState, deltas map[string]float64) string {
	var parts []string

	for _, s := range AllStats {
		d, ok := deltas[string(s)]
		if !ok {
			continue
		}

		cs.weights[s] = max(cs.weights[s]+d, 0)
		parts = append(parts, fmt.Sprintf("%s zählt %s-fach", statName(s), weightText(cs.weights[s])))
	}

	return strings.Join(parts, ", ") + "."
}

// weightText formats a weight German style: 3 → "3", 1.5 → "1,5".
func weightText(w float64) string {
	return strings.Replace(strconv.FormatFloat(w, 'f', -1, 64), ".", ",", 1)
}

func (g *Game) validateEntries(pl *player, cs *compState, trainerID string, entries []Entry) error {
	if trainerID != "" && pl.trainer(trainerID) == nil {
		return invalid("Dieser Trainer gehört nicht zu deinem Team.")
	}

	if len(entries) > 0 && trainerID == "" && len(pl.trainers) > 0 {
		return invalid("Wähle einen Trainer für diesen Wettkampf.")
	}

	usedRowers := map[string]bool{}
	usedBoats := map[string]bool{}
	usedRaces := map[string]bool{}

	for _, en := range entries {
		race := cs.race(en.RaceID)
		if race == nil {
			return invalid("Unbekanntes Rennen %q.", en.RaceID)
		}

		if usedRaces[en.RaceID] {
			return invalid("Pro Rennen ist nur eine Meldung erlaubt (%s).", race.Name)
		}

		usedRaces[en.RaceID] = true
		if len(en.RowerIDs) != race.Seats {
			return invalid("%s braucht genau %d Ruderer (gemeldet: %d).", race.Name, race.Seats, len(en.RowerIDs))
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
			return invalid("Der Ruderer %q gehört nicht zu deinem Team.", id)
		}

		if usedRowers[id] {
			return invalid("%s darf pro Wettkampf nur einmal starten.", rw.name)
		}

		usedRowers[id] = true

		if !slices.Contains(race.Ages, rw.age) {
			return invalid("%s ist in %s nicht startberechtigt (Altersklasse).", rw.name, race.Name)
		}

		if !slices.Contains(race.Genders, rw.gender) {
			return invalid("%s ist in %s nicht startberechtigt (Geschlecht).", rw.name, race.Name)
		}

		rowers = append(rowers, rw)
	}

	if race.Mixed && !mixedOK(rowers) {
		return invalid("%s: Mixed-Boote brauchen mindestens ein Mädchen, einen Jungen und zwei Altersgruppen.",
			race.Name)
	}

	return nil
}

// validateBoat checks the boat of one entry; usedBoats tracks boats across the competition.
func validateBoat(pl *player, race *RaceSpec, boatID string, usedBoats map[string]bool) error {
	switch race.Class {
	case BoatErgo, Boat8x:
		if boatID != "" {
			return invalid("Für %s wird kein eigenes Boot gemeldet.", race.Name)
		}

		return nil
	}

	if boatID == "" {
		if !race.LoanBoat {
			return invalid("Wähle ein Boot für %s.", race.Name)
		}

		return nil
	}

	b := pl.boat(boatID)
	if b == nil {
		return invalid("Das Boot %q gehört nicht zu deinem Team.", boatID)
	}

	if b.class != race.Class {
		return invalid("%s braucht ein Boot der Klasse %s, %s ist ein %s.", race.Name, race.Class, b.name, b.class)
	}

	if usedBoats[b.id] {
		return invalid("Das Boot %s darf pro Wettkampf nur einmal eingesetzt werden.", b.name)
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
	rr := RaceResult{RaceID: race.ID, Name: race.Name}

	for _, re := range g.raceEntries(p, race.ID) {
		er := g.scoreEntry(cs, race, re)
		if cs.def.Water {
			inc := g.d.Rules.Incidents

			er.Incident = rollIncident(g.rng, inc.Kentern(race.Seats), cs.material, cs.krebs)
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

// resolveCompetition scores all entries, rolls incidents, ranks, awards medals and Gold motivation.
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
		tr := RaceResult{RaceID: "team", Name: "Teamwertung"}

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
	g.checkMissions(cs, outcomes)
}

// awardMedal credits the medal to the player, applies Gold motivation to every crew member and
// records rower outcomes for missions.
func (g *Game) awardMedal(er EntryResult, class BoatClass, outcomes map[string]rowerOutcome) {
	pl := g.playerByID(er.PlayerID)
	if pl == nil {
		return // guest crew: takes the place, scores nothing
	}

	switch er.Medal {
	case MedalGold:
		pl.gold++
		for _, id := range er.RowerIDs {
			pl.rower(id).addStat(StatMotivation, g.d.Rules.GoldMotivation)
		}
	case MedalSilver:
		pl.silver++
	}

	for _, id := range er.RowerIDs {
		outcomes[id] = rowerOutcome{medal: er.Medal, class: class}
	}
}

var (
	medalNames    = map[Medal]string{MedalGold: "Gold", MedalSilver: "Silber"}
	incidentNames = map[IncidentKind]string{
		IncidentKentern: "gekentert", IncidentMaterial: "Materialschaden", IncidentKrebs: "Krebs gefangen",
	}
)

func (g *Game) logResult(res CompetitionResult) {
	for _, rr := range res.Races {
		var parts []string

		for _, er := range rr.Entries {
			name := er.Guest
			if pl := g.playerByID(er.PlayerID); pl != nil {
				name = pl.name
			}

			s := fmt.Sprintf("%s %d", name, er.Total)
			if er.Medal != MedalNone {
				s = medalNames[er.Medal] + ": " + s
			}

			if er.Incident != IncidentNone {
				s += " (" + incidentNames[er.Incident] + ")"
			}

			parts = append(parts, s)
		}

		if len(parts) == 0 {
			continue
		}

		g.addLog("result", "", fmt.Sprintf("%s – %s: %s", res.Name, rr.Name, strings.Join(parts, ", ")))
	}
}
