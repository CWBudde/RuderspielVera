package engine

import (
	"math"
	"slices"

	"ruderspiel/internal/data"
)

// scoreEntry computes an entry without incident:
// total = round(Σ weight·stat over the crew) + boat + (trainer effect + trainer mood) × scale +
// event bonuses.
func (g *Game) scoreEntry(cs *compState, race RaceSpec, re raceEntry) EntryResult {
	er := EntryResult{PlayerID: re.pl.id}
	crew := 0.0

	for _, r := range re.rowers {
		er.RowerIDs = append(er.RowerIDs, r.id)
		for _, s := range AllStats {
			crew += cs.weights[s] * float64(r.stat(s))
		}

		if r.event != nil {
			er.EventBonus += eventBonus(r.event.def, cs)
		}
	}

	er.Crew = int(math.Round(crew))

	bt := re.boat
	if race.Class == Boat8x {
		bt = re.pl.bestBoat(Boat8x) // eights are not chosen: the best own eight rows
	}

	er.BoatValue = g.boatValue(cs, race, bt) * g.d.Rules.BoatScale(race.Seats)
	if bt != nil {
		er.BoatID = bt.id
	}

	if re.trainer != nil {
		er.TrainerID = re.trainer.def.ID
		er.Trainer = (trainerEffect(re.trainer, cs) + g.mood[re.trainer.def.ID]) * g.d.Rules.TrainerScale(race.Seats)
	}

	er.Total = er.Crew + er.BoatValue + er.Trainer + er.EventBonus

	return er
}

// eventBonus is a rower's event card bonus at a competition: the card's Bonus when the competition
// matches its scope, context and (if any) weathers.
func eventBonus(def *data.EventType, cs *compState) int {
	if def.Bonus == 0 || !data.ScopeMatches(def.Scope, cs.def) {
		return 0
	}

	if def.Context != "" && def.Context != cs.def.Context {
		return 0
	}

	if len(def.Weathers) > 0 && !slices.Contains(def.Weathers, string(cs.weather)) {
		return 0
	}

	return def.Bonus
}

func (g *Game) boatValue(cs *compState, race RaceSpec, bt *boat) int {
	switch {
	case race.Class == BoatErgo:
		return 0
	case bt == nil:
		if race.LoanBoat {
			return g.d.Rules.LoanBoatValue
		}

		return 0
	}

	v := bt.value
	if bt.city {
		v += cs.cityBoatMod
	}

	return max(v, 0)
}

// trainerEffect sums the trainer's effects matching the competition's weather and context.
// It applies once per boat / ergo entry.
func trainerEffect(t *trainer, cs *compState) int {
	total := 0

	for _, e := range t.def.Effects {
		if e.Weather != "" && Weather(e.Weather) != cs.weather {
			continue
		}

		if e.Context != "" && e.Context != cs.def.Context {
			continue
		}

		total += e.Value
	}

	return total
}

// rankEntries assigns places and medals in place and sorts the entries by place.
// Finishers (clean or Krebs) are ranked by total with competition ranking (1,1,3). Capsized
// crews share the place after all finishers and get no medal. Material damage = DNF (place 0).
func rankEntries(es []EntryResult) {
	var finishers, capsized, dnf []EntryResult

	for _, e := range es {
		switch e.Incident {
		case IncidentKentern:
			capsized = append(capsized, e)
		case IncidentMaterial:
			dnf = append(dnf, e)
		default:
			finishers = append(finishers, e)
		}
	}

	slices.SortStableFunc(finishers, func(a, b EntryResult) int { return b.Total - a.Total })

	for i := range finishers {
		if i > 0 && finishers[i].Total == finishers[i-1].Total {
			finishers[i].Place = finishers[i-1].Place
		} else {
			finishers[i].Place = i + 1
		}

		switch finishers[i].Place {
		case 1:
			finishers[i].Medal = MedalGold
		case 2:
			finishers[i].Medal = MedalSilver
		default:
			finishers[i].Medal = MedalNone
		}
	}

	for i := range capsized {
		capsized[i].Place = len(finishers) + 1
		capsized[i].Medal = MedalNone
	}

	for i := range dnf {
		dnf[i].Place = 0
		dnf[i].Medal = MedalNone
	}

	out := append(append(finishers, capsized...), dnf...)
	copy(es, out)
}

// EstimateEntry returns the expected score of a hypothetical entry in the currently open
// competition (no randomness, no incident). It returns 0 if no competition is open or the
// player, race, rowers, boat or trainer are unknown. Eligibility is not checked; use it for
// lineup estimates in bots and the UI.
func EstimateEntry(g *Game, playerID, raceID string, rowerIDs []string, boatID, trainerID string) int {
	if g.phase == nil || g.phase.comp == nil {
		return 0
	}

	cs := g.phase.comp
	pl := g.playerByID(playerID)

	race := cs.race(raceID)
	if pl == nil || race == nil {
		return 0
	}

	re := raceEntry{pl: pl, trainer: pl.trainer(trainerID)}
	if boatID != "" {
		if re.boat = pl.boat(boatID); re.boat == nil {
			return 0
		}
	}

	if trainerID != "" && re.trainer == nil {
		return 0
	}

	for _, id := range rowerIDs {
		r := pl.rower(id)
		if r == nil {
			return 0
		}

		re.rowers = append(re.rowers, r)
	}

	return g.scoreEntry(cs, *race, re).Total
}

// ageRank ranks age groups from youngest to oldest.
var ageRank = map[AgeGroup]int{AgeKinder: 0, AgeB: 1, AgeA: 2}

// GuestScore returns the total of the guest crew (Gastmannschaft) of a race, or false when the
// rules have no guest crews. A guest rower has every stat at the level of the race's oldest
// allowed age group plus season growth; its boat is valued like a player boat. The score is
// deterministic, so bots and the UI can compute it in advance.
func GuestScore(d *data.Data, comp *data.Competition, station int, race RaceSpec) (float64, bool) {
	return guestScore(d, comp, station, race, 0)
}

// guestScore is GuestScore with an extra stat level offset (Ereignis guest modifiers).
func guestScore(d *data.Data, comp *data.Competition, station int, race RaceSpec, mod float64) (float64, bool) {
	gc := d.Rules.Guests
	if !gc.Enabled || len(race.Ages) == 0 {
		return 0, false
	}

	oldest := race.Ages[0]
	for _, a := range race.Ages {
		if ageRank[a] > ageRank[oldest] {
			oldest = a
		}
	}

	level := gc.StatLevel[string(oldest)] + gc.GrowthPerStation*float64(station) + comp.GuestOffset + mod

	wsum := 0.0
	for _, st := range data.StatNames {
		wsum += comp.Weights[st]
	}

	crew := math.Round(float64(race.Seats) * wsum * level)
	boat := 0

	switch race.Class {
	case BoatErgo:
	case Boat8x:
		if sb := d.StarterBoat(string(Boat8x)); sb != nil {
			boat = sb.Value
		}
	default:
		boat = gc.BoatValue
	}

	return crew + float64(boat*d.Rules.BoatScale(race.Seats)), true
}

// guestEntry builds the result row of a race's guest crew.
func (g *Game) guestEntry(cs *compState, race RaceSpec) (EntryResult, bool) {
	total, ok := guestScore(g.d, cs.def, cs.station, race, cs.guestMod)
	if !ok {
		return EntryResult{}, false
	}

	return EntryResult{Guest: g.d.Rules.Guests.Name, Crew: int(total), Total: int(total)}, true
}
