package data

import (
	"errors"
	"fmt"
	"slices"
)

var (
	validAges     = []string{"kinder", "b", "a"}
	validGenders  = []string{"w", "m"}
	validWeather  = []string{"regen", "hitze", "wind", "kaelte", "normal"}
	validContexts = []string{"", "ergo", "sprint", "langstrecke"}
	stationTypes  = []string{
		"setup", "training", "event", "ereignis", "regatta", kindErgoCup, "selection", "camp", kindJTFO,
	}
	competitionKinds = []string{"regatta", kindErgoCup, kindJTFO}
	seatsByClass     = map[string]int{"1x": 1, "2x": 2, "4x": 4, "8x": 8, classErgo: 1}
)

const (
	classErgo   = "ergo"    // boat class of ergometer races
	kindErgoCup = "ergocup" // competition kind / station type of ergometer cups
	kindJTFO    = "jtfo"    // competition kind / station type of Jugend trainiert für Olympia
)

// rankRanges are the rank (stat sum) ranges per age group, Regelwerk §5.3 / PoC decision.
var rankRanges = map[string][2]int{"a": {86, 100}, "b": {59, 85}, "kinder": {46, 58}}

// problems collects validation findings.
type problems []error

// dataError is one validation finding.
type dataError string

func (e dataError) Error() string { return string(e) }

func (p *problems) addf(format string, args ...any) {
	*p = append(*p, problemf(format, args...))
}

// Validate checks the internal consistency of the data set and returns all problems found.
func (d *Data) Validate() error {
	var p problems

	ids := d.validateRowers(&p)
	boatIDs := d.validateBoats(&p, ids)
	d.validateTrainers(&p, ids, boatIDs)
	d.validateEvents(&p)
	d.validateMissions(&p)
	d.validateBalancing(&p)
	d.validateCalendar(&p)
	d.validateCompetitions(&p)
	d.validateRules(&p)

	return errors.Join(p...)
}

// validateRowers checks the rower cards and returns their IDs.
func (d *Data) validateRowers(p *problems) map[string]bool {
	if len(d.Rowers) != 55 {
		p.addf("rowers: want 55, got %d", len(d.Rowers))
	}

	ids := map[string]bool{}
	for _, rw := range d.Rowers {
		if rw.ID == "" || ids[rw.ID] {
			p.addf("rower %q: empty or duplicate id", rw.ID)
		}

		ids[rw.ID] = true
		if !slices.Contains(validAges, rw.Age) || !slices.Contains(validGenders, rw.Gender) {
			p.addf("rower %s: bad age %q or gender %q", rw.ID, rw.Age, rw.Gender)
		}

		sum := 0

		for _, s := range StatNames {
			v, ok := rw.Stats[s]
			if !ok || v < 5 {
				p.addf("rower %s: stat %s missing or < 5 (%d)", rw.ID, s, v)
			}

			sum += v
		}

		if len(rw.Stats) != len(StatNames) {
			p.addf("rower %s: unexpected stats %v", rw.ID, rw.Stats)
		}

		if sum != rw.Rank {
			p.addf("rower %s: stats sum %d != rank %d", rw.ID, sum, rw.Rank)
		}

		if rr, ok := rankRanges[rw.Age]; ok && (rw.Rank < rr[0] || rw.Rank > rr[1]) {
			p.addf("rower %s: rank %d outside %v for age %s", rw.ID, rw.Rank, rr, rw.Age)
		}
	}

	return ids
}

// validateBoats checks the boat cards and returns their IDs.
func (d *Data) validateBoats(p *problems, ids map[string]bool) map[string]bool {
	boatIDs := map[string]bool{}
	shared := 0

	for _, b := range d.Boats {
		if b.ID == "" || boatIDs[b.ID] || ids[b.ID] {
			p.addf("boat %q: empty or duplicate id", b.ID)
		}

		boatIDs[b.ID] = true
		if _, ok := seatsByClass[b.Class]; !ok || b.Class == classErgo {
			p.addf("boat %s: bad class %q", b.ID, b.Class)
		}

		if b.Value < 0 || b.Value > 10 {
			p.addf("boat %s: value %d outside 0..10", b.ID, b.Value)
		}

		if b.Shared {
			shared++
		}
	}

	if shared != 1 {
		p.addf("boats: want exactly one shared eight, got %d", shared)
	}

	for _, cls := range []string{"1x", "2x", "4x"} {
		if !slices.ContainsFunc(d.Boats, func(b Boat) bool { return b.City && b.Class == cls }) {
			p.addf("boats: no city boat for class %s", cls)
		}
	}

	return boatIDs
}

// validateTrainers checks the trainer cards; IDs must not collide with rowers or boats.
func (d *Data) validateTrainers(p *problems, ids, boatIDs map[string]bool) {
	if len(d.Trainers) != 10 {
		p.addf("trainers: want 10, got %d", len(d.Trainers))
	}

	for _, t := range d.Trainers {
		if t.ID == "" || boatIDs[t.ID] || ids[t.ID] {
			p.addf("trainer %q: empty or colliding id", t.ID)
		}

		if len(t.Effects) == 0 {
			p.addf("trainer %s: no effects", t.ID)
		}

		for _, e := range t.Effects {
			if e.Weather != "" && !slices.Contains(validWeather, e.Weather) {
				p.addf("trainer %s: bad weather %q", t.ID, e.Weather)
			}

			if !slices.Contains(validContexts, e.Context) {
				p.addf("trainer %s: bad context %q", t.ID, e.Context)
			}
		}
	}
}

// validateEvents checks the event cards and the Ereignisse.
func (d *Data) validateEvents(p *problems) {
	if len(d.Events) != 12 {
		p.addf("events: want 12, got %d", len(d.Events))
	}

	for _, e := range d.Events {
		if e.Weight <= 0 {
			p.addf("event %s: weight must be > 0", e.Type)
		}

		if e.Stat != "" && !slices.Contains(StatNames, e.Stat) {
			p.addf("event %s: bad stat %q", e.Type, e.Stat)
		}
	}

	for _, t := range []string{"wetter", "wachstum", "phoenix", "tausch", "verdopplung", "seelenzwilling", "mission"} {
		if d.EventType(t) == nil {
			p.addf("events: missing type %s", t)
		}
	}

	if len(d.Ereignisse) == 0 {
		p.addf("ereignisse: empty")
	}

	for _, e := range d.Ereignisse {
		if e.Weight <= 0 || !ereignisEffects[e.Effect] {
			p.addf("ereignis %s: bad weight or effect %q", e.ID, e.Effect)
		}

		if e.Effect == EffForceWeather && !slices.Contains(validWeather, e.Weather) {
			p.addf("ereignis %s: bad weather %q", e.ID, e.Weather)
		}
	}
}

// validateMissions checks the mission cards.
func (d *Data) validateMissions(p *problems) {
	if len(d.Missions) == 0 {
		p.addf("missions: empty")
	}

	for _, ms := range d.Missions {
		c := ms.Condition
		if c.Type != "medal" && c.Type != "starts" {
			p.addf("mission %s: bad condition type %q", ms.ID, c.Type)
		}

		if c.Type == "medal" && c.Medal != "gold" && c.Medal != "any" {
			p.addf("mission %s: bad medal %q", ms.ID, c.Medal)
		}

		if c.Type == "starts" && c.Count <= 0 {
			p.addf("mission %s: starts count must be > 0", ms.ID)
		}

		for _, cid := range []string{c.Competition, c.Before} {
			if cid != "" && d.Competitions[cid] == nil {
				p.addf("mission %s: unknown competition %q", ms.ID, cid)
			}
		}

		if ms.Reward.Type != "stat" && ms.Reward.Type != "points" {
			p.addf("mission %s: bad reward type %q", ms.ID, ms.Reward.Type)
		}
	}
}

// validateBalancing checks the balancing rules (factors, guest crews).
func (d *Data) validateBalancing(p *problems) {
	if d.Rules.BoatValueFactor < 0 || d.Rules.TrainerFactor < 0 {
		p.addf("rules: boatValueFactor and trainerFactor must be >= 0")
	}

	if gs := d.Rules.Guests; gs.Enabled {
		if gs.Name == "" {
			p.addf("rules: guests need a name")
		}

		for _, a := range validAges {
			if gs.StatLevel[a] <= 0 {
				p.addf("rules: guests statLevel for %s must be > 0", a)
			}
		}

		if gs.GrowthPerStation < 0 || gs.BoatValue < 0 {
			p.addf("rules: guests growthPerStation and boatValue must be >= 0")
		}
	}
}

// validateCalendar checks the station sequence.
func (d *Data) validateCalendar(p *problems) {
	if len(d.Calendar) != 52 {
		p.addf("calendar: want 52 stations, got %d", len(d.Calendar))
	}

	stIDs := map[string]bool{}
	events, prevSection := 0, 1

	for si, st := range d.Calendar {
		if stIDs[st.ID] {
			p.addf("station %s: duplicate id", st.ID)
		}

		stIDs[st.ID] = true
		if !slices.Contains(stationTypes, st.Type) {
			p.addf("station %s: bad type %q", st.ID, st.Type)
		}

		if st.Section < prevSection || st.Section > 12 {
			p.addf("station %s: section %d out of order", st.ID, st.Section)
		}

		prevSection = st.Section
		if st.Type == "event" {
			events++
		}

		d.checkStationRefs(p, st)

		if si == 0 && st.Type != "setup" {
			p.addf("calendar must start with setup")
		}
	}

	if len(d.Calendar) == 0 {
		p.addf("calendar: empty")
	} else if last := d.Calendar[len(d.Calendar)-1]; !slices.Contains(competitionKinds, last.Type) {
		p.addf("calendar must end with a competition")
	}

	if len(d.Rules.EventCardsPerStation) != events {
		p.addf("rules: eventCardsPerStation has %d entries for %d event stations",
			len(d.Rules.EventCardsPerStation), events)
	}
}

// checkStationRefs checks that a station's competition or selection exists and fits.
func (d *Data) checkStationRefs(p *problems, s Station) {
	switch {
	case slices.Contains(competitionKinds, s.Type):
		c := d.Competitions[s.Competition]
		if c == nil {
			p.addf("station %s: no competition entry %q", s.ID, s.Competition)
		} else if c.Kind != s.Type {
			p.addf("station %s: competition kind %q != station type %q", s.ID, c.Kind, s.Type)
		}
	case s.Type == "selection":
		sel, ok := d.Rules.Selections[s.Selection]
		if !ok || sel.Count <= 0 || len(sel.Effect) == 0 {
			p.addf("station %s: bad selection %q", s.ID, s.Selection)
		}
	}
}

// validateCompetitions checks every competition and its races.
func (d *Data) validateCompetitions(p *problems) {
	for id, comp := range d.Competitions {
		if len(comp.Weights) == 0 {
			p.addf("competition %s: no weights", id)
		}

		if comp.Weights["motivation"] <= 0 {
			p.addf("competition %s: Motivation must count in every competition (weight > 0)", id)
		}

		for s := range comp.Weights {
			if !slices.Contains(StatNames, s) {
				p.addf("competition %s: bad weight stat %q", id, s)
			}
		}

		if !slices.Contains(validContexts, comp.Context) {
			p.addf("competition %s: bad context %q", id, comp.Context)
		}

		for _, races := range d.raceSets(p, id, comp) {
			d.checkRaces(p, id, comp, races)
		}
	}
}

// raceSets returns the race lists of a competition: its races, or one variant per player count.
func (d *Data) raceSets(p *problems, id string, c *Competition) [][]Race {
	if len(c.Variants) == 0 {
		return [][]Race{c.Races}
	}

	raceSets := make([][]Race, 0, 3)

	for _, n := range []string{"2", "3", "4"} {
		v, ok := c.Variants[d.Rules.JTFOVariants[n]]
		if !ok {
			p.addf("competition %s: no variant for %s players", id, n)
		}

		raceSets = append(raceSets, v)
	}

	return raceSets
}

// checkRaces checks one race list of competition id.
func (d *Data) checkRaces(p *problems, id string, c *Competition, races []Race) {
	if len(races) == 0 {
		p.addf("competition %s: no races", id)
	}

	raceIDs := map[string]bool{}
	for _, r := range races {
		if raceIDs[r.ID] {
			p.addf("competition %s: duplicate race %s", id, r.ID)
		}

		raceIDs[r.ID] = true

		err := d.checkRace(c, r)
		if err != nil {
			p.addf("competition %s race %s: %v", id, r.ID, err)
		}
	}
}

// validateRules checks the remaining rule numbers.
func (d *Data) validateRules(p *problems) {
	rules := d.Rules
	if rules.TrainingPoints <= 0 || rules.Camp.Points <= 0 || rules.Camp.MaxRowers <= 0 {
		p.addf("rules: training/camp points must be > 0")
	}

	d.validateIncidents(p)

	if len(rules.WeatherTable) != 6 {
		p.addf("rules: weatherTable needs 6 faces")
	}

	for _, w := range rules.WeatherTable {
		if !slices.Contains(validWeather, w) {
			p.addf("rules: bad weather %q", w)
		}
	}

	if rules.Deal.RowersPerAgeGroup <= 0 || rules.Deal.Trainers <= 0 {
		p.addf("rules: bad deal sizes")
	}
}

// validateIncidents checks the per-mille incident probabilities: a capsize value for every boat
// size, and Kentern + Material + Krebs (also with a Krebs Ereignis) must fit into one d1000.
func (d *Data) validateIncidents(p *problems) {
	inc := d.Rules.Incidents
	if inc.Material < 0 || inc.Krebs < 0 {
		p.addf("rules: incident probabilities must be >= 0")
	}

	krebs := inc.Krebs

	for _, e := range d.Ereignisse {
		if e.Effect != EffKrebsChance {
			continue
		}

		if e.Value < 0 || e.Value > 1000 {
			p.addf("ereignis %s: krebs_chance %d ‰ outside 0..1000", e.ID, e.Value)
		}

		krebs = max(krebs, e.Value)
	}

	for seats := range inc.KenternBySeats {
		if !slices.Contains([]int{1, 2, 4, 8}, seats) {
			p.addf("rules: kenternBySeats has unknown boat size %d", seats)
		}
	}

	for _, seats := range []int{1, 2, 4, 8} {
		k, ok := inc.KenternBySeats[seats]
		if !ok || k < 0 {
			p.addf("rules: kenternBySeats needs a value >= 0 for %d seats", seats)
		}

		if k+inc.Material+krebs > 1000 {
			p.addf("rules: incident probabilities for %d seats exceed 1000 ‰", seats)
		}
	}
}

// checkRace validates a race spec and that it is fillable from the full rower set.
func (d *Data) checkRace(c *Competition, r Race) error {
	err := checkRaceSpec(c, r)
	if err != nil {
		return err
	}

	return d.checkRaceFillable(r)
}

// checkRaceSpec validates class, seats, age groups and genders of a race.
func checkRaceSpec(c *Competition, race Race) error {
	seats, ok := seatsByClass[race.Class]
	if !ok {
		return problemf("bad class %q", race.Class)
	}

	if race.Seats != seats {
		return problemf("seats %d != %d for class %s", race.Seats, seats, race.Class)
	}

	if (race.Class == classErgo) != (c.Kind == kindErgoCup) {
		return dataError("ergo class only in ergo cups")
	}

	if len(race.Ages) == 0 || len(race.Genders) == 0 {
		return dataError("no ages or genders")
	}

	for _, a := range race.Ages {
		if !slices.Contains(validAges, a) {
			return problemf("bad age %q", a)
		}
	}

	for _, g := range race.Genders {
		if !slices.Contains(validGenders, g) {
			return problemf("bad gender %q", g)
		}
	}

	if race.Mixed && (len(race.Genders) < 2 || len(race.Ages) < 2) {
		return dataError("mixed race needs two genders and two ages")
	}

	return nil
}

// checkRaceFillable checks that the full rower set has enough eligible rowers for the race.
func (d *Data) checkRaceFillable(race Race) error {
	eligible, girls, boys := 0, 0, 0
	ages := map[string]bool{}

	for _, rw := range d.Rowers {
		if slices.Contains(race.Ages, rw.Age) && slices.Contains(race.Genders, rw.Gender) {
			eligible++

			ages[rw.Age] = true
			if rw.Gender == "w" {
				girls++
			} else {
				boys++
			}
		}
	}

	if eligible < race.Seats {
		return problemf("not fillable: %d eligible rowers for %d seats", eligible, race.Seats)
	}

	if race.Mixed && (girls == 0 || boys == 0 || len(ages) < 2) {
		return dataError("mixed race not fillable")
	}

	return nil
}

// problemf formats a single validation finding.
func problemf(format string, args ...any) error {
	return dataError(fmt.Sprintf(format, args...))
}
