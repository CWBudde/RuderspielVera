package engine

import (
	"math"
	"slices"
)

// scoreEntry computes an entry without incident:
// total = round(Σ weight·stat over the crew) + boat + trainer effect + event bonuses.
func (g *Game) scoreEntry(cs *compState, race RaceSpec, re raceEntry) EntryResult {
	er := EntryResult{PlayerID: re.pl.id}
	crew := 0.0
	for _, r := range re.rowers {
		er.RowerIDs = append(er.RowerIDs, r.id)
		for _, s := range AllStats {
			crew += cs.weights[s] * float64(r.stats[s])
		}
		if r.event != nil && r.event.def.Type == "wetter" && cs.def.Water &&
			cs.weather != "" && cs.weather != WeatherNormal {
			er.EventBonus += r.event.def.Value
		}
	}
	er.Crew = int(math.Round(crew))
	er.BoatValue = g.boatValue(cs, race, re.boat)
	if re.boat != nil {
		er.BoatID = re.boat.id
	}
	if re.trainer != nil {
		er.TrainerID = re.trainer.def.ID
		er.Trainer = trainerEffect(re.trainer, cs)
	}
	er.Total = er.Crew + er.BoatValue + er.Trainer + er.EventBonus
	return er
}

func (g *Game) boatValue(cs *compState, race RaceSpec, b *boat) int {
	switch {
	case race.Class == BoatErgo:
		return 0
	case race.Class == Boat8x:
		if sb := g.d.SharedBoat(); sb != nil {
			return sb.Value
		}
		return 0
	case b == nil:
		if race.LoanBoat {
			return g.d.Rules.LoanBoatValue
		}
		return 0
	}
	v := b.value
	if b.city {
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
