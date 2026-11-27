package engine

import (
	"fmt"
	"slices"
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
	krebsPct    int
	cityBoatMod int
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
		krebsPct: g.d.Rules.Incidents.Krebs, notes: slices.Clone(def.Notes),
	}
	for _, r := range g.d.RacesFor(def, len(g.players)) {
		cs.races = append(cs.races, raceSpec(r))
	}
	for _, s := range AllStats {
		if w, ok := def.Weights[string(s)]; ok {
			cs.weights[s] = w
		}
	}
	if def.Water {
		if forced := g.takePending(data.EffForceWeather); len(forced) > 0 {
			e := forced[len(forced)-1]
			cs.weather = Weather(e.Weather)
			cs.notes = append(cs.notes, fmt.Sprintf("%s: Das Wetter steht fest.", e.Name))
		} else {
			cs.weather = Weather(g.d.Rules.WeatherTable[rollD(g.rng, 6)-1])
		}
		for _, e := range g.takePending(data.EffKrebsChance) {
			cs.krebsPct = e.Value
			cs.notes = append(cs.notes, fmt.Sprintf("%s: Krebs-Wahrscheinlichkeit %d %%.", e.Name, e.Value))
		}
		for _, e := range g.takePending(data.EffCityBoatMod) {
			cs.cityBoatMod += e.Value
			cs.notes = append(cs.notes, fmt.Sprintf("%s: Städtische Boote %+d.", e.Name, e.Value))
		}
		g.addLog("info", "", fmt.Sprintf("Wetter bei %s: %s.", st.Name, weatherNames[cs.weather]))
	}
	if def.Kind == string(StationErgoCup) {
		for _, e := range g.takePending(data.EffErgoKraftWeight) {
			cs.weights[StatKraft] = float64(e.Value)
			cs.notes = append(cs.notes, fmt.Sprintf("%s: Kraft zählt %d-fach.", e.Name, e.Value))
		}
	}
	prompt := "Wähle einen Trainer und melde deine Mannschaften. Jeder Ruderer und jedes Boot darf nur einmal starten."
	if cs.weather != "" {
		prompt = fmt.Sprintf("Wetter: %s. %s", weatherNames[cs.weather], prompt)
	}
	p := g.newPhase(PhaseRegatta, stageNone, g.allPlayerIDs(), st.Name, prompt)
	p.comp = cs
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
	for _, e := range entries {
		race := cs.race(e.RaceID)
		if race == nil {
			return invalid("Unbekanntes Rennen %q.", e.RaceID)
		}
		if usedRaces[e.RaceID] {
			return invalid("Pro Rennen ist nur eine Meldung erlaubt (%s).", race.Name)
		}
		usedRaces[e.RaceID] = true
		if len(e.RowerIDs) != race.Seats {
			return invalid("%s braucht genau %d Ruderer (gemeldet: %d).", race.Name, race.Seats, len(e.RowerIDs))
		}
		rowers := make([]*rower, 0, len(e.RowerIDs))
		for _, id := range e.RowerIDs {
			r := pl.rower(id)
			if r == nil {
				return invalid("Der Ruderer %q gehört nicht zu deinem Team.", id)
			}
			if usedRowers[id] {
				return invalid("%s darf pro Wettkampf nur einmal starten.", r.name)
			}
			usedRowers[id] = true
			if !slices.Contains(race.Ages, r.age) {
				return invalid("%s ist in %s nicht startberechtigt (Altersklasse).", r.name, race.Name)
			}
			if !slices.Contains(race.Genders, r.gender) {
				return invalid("%s ist in %s nicht startberechtigt (Geschlecht).", r.name, race.Name)
			}
			rowers = append(rowers, r)
		}
		if race.Mixed && !mixedOK(rowers) {
			return invalid("%s: Mixed-Boote brauchen mindestens ein Mädchen, einen Jungen und zwei Altersgruppen.", race.Name)
		}
		switch race.Class {
		case BoatErgo, Boat8x:
			if e.BoatID != "" {
				return invalid("Für %s wird kein eigenes Boot gemeldet.", race.Name)
			}
		default:
			if e.BoatID == "" {
				if !race.LoanBoat {
					return invalid("Wähle ein Boot für %s.", race.Name)
				}
				continue
			}
			b := pl.boat(e.BoatID)
			if b == nil {
				return invalid("Das Boot %q gehört nicht zu deinem Team.", e.BoatID)
			}
			if b.class != race.Class {
				return invalid("%s braucht ein Boot der Klasse %s, %s ist ein %s.", race.Name, race.Class, b.name, b.class)
			}
			if usedBoats[b.id] {
				return invalid("Das Boot %s darf pro Wettkampf nur einmal eingesetzt werden.", b.name)
			}
			usedBoats[b.id] = true
		}
	}
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

// resolveCompetition scores all entries, rolls incidents, ranks, awards medals and Gold motivation.
func (g *Game) resolveCompetition(p *phase) {
	cs := p.comp
	res := CompetitionResult{StationIndex: cs.station, Name: cs.name, Weather: cs.weather}
	outcomes := map[string]rowerOutcome{}
	type teamAcc struct {
		entry EntryResult
		any   bool
	}
	team := map[string]*teamAcc{}

	for _, race := range cs.races {
		rr := RaceResult{RaceID: race.ID, Name: race.Name}
		var entries []raceEntry
		for _, pid := range p.required {
			d := p.decisions[pid]
			for _, e := range d.Entries {
				if e.RaceID == race.ID {
					entries = append(entries, g.buildEntry(g.playerByID(pid), d.TrainerID, e))
				}
			}
		}
		for _, re := range entries {
			er := g.scoreEntry(cs, race, re)
			if cs.def.Water {
				er.Incident = rollIncident(g.rng, g.d.Rules.Incidents.Kentern, g.d.Rules.Incidents.Material, cs.krebsPct)
				if er.Incident == IncidentKrebs {
					er.Penalty = g.d.Rules.KrebsPenalty
					er.Total -= er.Penalty
				}
			}
			rr.Entries = append(rr.Entries, er)
		}
		rankEntries(rr.Entries)
		if cs.team() {
			for i := range rr.Entries {
				rr.Entries[i].Medal = MedalNone
				er := rr.Entries[i]
				acc := team[er.PlayerID]
				if acc == nil {
					acc = &teamAcc{entry: EntryResult{PlayerID: er.PlayerID, TrainerID: er.TrainerID}}
					team[er.PlayerID] = acc
				}
				acc.any = true
				acc.entry.RowerIDs = append(acc.entry.RowerIDs, er.RowerIDs...)
				if er.Incident == IncidentKentern || er.Incident == IncidentMaterial {
					continue // this boat contributes nothing to the team score
				}
				acc.entry.Crew += er.Crew
				acc.entry.BoatValue += er.BoatValue
				acc.entry.Trainer += er.Trainer
				acc.entry.EventBonus += er.EventBonus
				acc.entry.Penalty += er.Penalty
				acc.entry.Total += er.Total
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
			if acc := team[pid]; acc != nil && acc.any {
				tr.Entries = append(tr.Entries, acc.entry)
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

var medalNames = map[Medal]string{MedalGold: "Gold", MedalSilver: "Silber"}
var incidentNames = map[IncidentKind]string{
	IncidentKentern: "gekentert", IncidentMaterial: "Materialschaden", IncidentKrebs: "Krebs gefangen",
}

func (g *Game) logResult(res CompetitionResult) {
	for _, rr := range res.Races {
		var parts []string
		for _, er := range rr.Entries {
			pl := g.playerByID(er.PlayerID)
			s := fmt.Sprintf("%s %d", pl.name, er.Total)
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
