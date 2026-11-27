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
	stationTypes  = []string{"setup", "training", "event", "ereignis", "regatta", "ergocup", "selection", "camp", "jtfo"}
	seatsByClass  = map[string]int{"1x": 1, "2x": 2, "4x": 4, "8x": 8, "ergo": 1}
)

// rankRanges are the rank (stat sum) ranges per age group, Regelwerk §5.3 / PoC decision.
var rankRanges = map[string][2]int{"a": {86, 100}, "b": {59, 85}, "kinder": {46, 58}}

// Validate checks the internal consistency of the data set and returns all problems found.
func (d *Data) Validate() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	// Rowers.
	if len(d.Rowers) != 55 {
		add("rowers: want 55, got %d", len(d.Rowers))
	}
	ids := map[string]bool{}
	for _, r := range d.Rowers {
		if r.ID == "" || ids[r.ID] {
			add("rower %q: empty or duplicate id", r.ID)
		}
		ids[r.ID] = true
		if !slices.Contains(validAges, r.Age) || !slices.Contains(validGenders, r.Gender) {
			add("rower %s: bad age %q or gender %q", r.ID, r.Age, r.Gender)
		}
		sum := 0
		for _, s := range StatNames {
			v, ok := r.Stats[s]
			if !ok || v < 5 {
				add("rower %s: stat %s missing or < 5 (%d)", r.ID, s, v)
			}
			sum += v
		}
		if len(r.Stats) != len(StatNames) {
			add("rower %s: unexpected stats %v", r.ID, r.Stats)
		}
		if sum != r.Rank {
			add("rower %s: stats sum %d != rank %d", r.ID, sum, r.Rank)
		}
		if rr, ok := rankRanges[r.Age]; ok && (r.Rank < rr[0] || r.Rank > rr[1]) {
			add("rower %s: rank %d outside %v for age %s", r.ID, r.Rank, rr, r.Age)
		}
	}

	// Boats.
	boatIDs := map[string]bool{}
	shared := 0
	for _, b := range d.Boats {
		if b.ID == "" || boatIDs[b.ID] || ids[b.ID] {
			add("boat %q: empty or duplicate id", b.ID)
		}
		boatIDs[b.ID] = true
		if _, ok := seatsByClass[b.Class]; !ok || b.Class == "ergo" {
			add("boat %s: bad class %q", b.ID, b.Class)
		}
		if b.Value < 0 || b.Value > 10 {
			add("boat %s: value %d outside 0..10", b.ID, b.Value)
		}
		if b.Shared {
			shared++
		}
	}
	if shared != 1 {
		add("boats: want exactly one shared eight, got %d", shared)
	}
	for _, cls := range []string{"1x", "2x", "4x"} {
		if !slices.ContainsFunc(d.Boats, func(b Boat) bool { return b.City && b.Class == cls }) {
			add("boats: no city boat for class %s", cls)
		}
	}

	// Trainers.
	if len(d.Trainers) != 10 {
		add("trainers: want 10, got %d", len(d.Trainers))
	}
	for _, t := range d.Trainers {
		if t.ID == "" || boatIDs[t.ID] || ids[t.ID] {
			add("trainer %q: empty or colliding id", t.ID)
		}
		if len(t.Effects) == 0 {
			add("trainer %s: no effects", t.ID)
		}
		for _, e := range t.Effects {
			if e.Weather != "" && !slices.Contains(validWeather, e.Weather) {
				add("trainer %s: bad weather %q", t.ID, e.Weather)
			}
			if !slices.Contains(validContexts, e.Context) {
				add("trainer %s: bad context %q", t.ID, e.Context)
			}
		}
	}

	// Events, Ereignisse, missions.
	if len(d.Events) != 12 {
		add("events: want 12, got %d", len(d.Events))
	}
	for _, e := range d.Events {
		if e.Weight <= 0 {
			add("event %s: weight must be > 0", e.Type)
		}
		if e.Stat != "" && !slices.Contains(StatNames, e.Stat) {
			add("event %s: bad stat %q", e.Type, e.Stat)
		}
	}
	for _, t := range []string{"wetter", "wachstum", "phoenix", "tausch", "verdopplung", "seelenzwilling", "mission"} {
		if d.EventType(t) == nil {
			add("events: missing type %s", t)
		}
	}
	if len(d.Ereignisse) == 0 {
		add("ereignisse: empty")
	}
	for _, e := range d.Ereignisse {
		if e.Weight <= 0 || !ereignisEffects[e.Effect] {
			add("ereignis %s: bad weight or effect %q", e.ID, e.Effect)
		}
		if e.Effect == EffForceWeather && !slices.Contains(validWeather, e.Weather) {
			add("ereignis %s: bad weather %q", e.ID, e.Weather)
		}
	}
	if len(d.Missions) == 0 {
		add("missions: empty")
	}
	for _, m := range d.Missions {
		c := m.Condition
		if c.Type != "medal" && c.Type != "starts" {
			add("mission %s: bad condition type %q", m.ID, c.Type)
		}
		if c.Type == "medal" && c.Medal != "gold" && c.Medal != "any" {
			add("mission %s: bad medal %q", m.ID, c.Medal)
		}
		if c.Type == "starts" && c.Count <= 0 {
			add("mission %s: starts count must be > 0", m.ID)
		}
		for _, cid := range []string{c.Competition, c.Before} {
			if cid != "" && d.Competitions[cid] == nil {
				add("mission %s: unknown competition %q", m.ID, cid)
			}
		}
		if m.Reward.Type != "stat" && m.Reward.Type != "points" {
			add("mission %s: bad reward type %q", m.ID, m.Reward.Type)
		}
	}

	// Calendar and competitions.
	if len(d.Calendar) != 52 {
		add("calendar: want 52 stations, got %d", len(d.Calendar))
	}
	stIDs := map[string]bool{}
	events, prevSection := 0, 1
	for i, s := range d.Calendar {
		if stIDs[s.ID] {
			add("station %s: duplicate id", s.ID)
		}
		stIDs[s.ID] = true
		if !slices.Contains(stationTypes, s.Type) {
			add("station %s: bad type %q", s.ID, s.Type)
		}
		if s.Section < prevSection || s.Section > 12 {
			add("station %s: section %d out of order", s.ID, s.Section)
		}
		prevSection = s.Section
		switch s.Type {
		case "regatta", "ergocup", "jtfo":
			c := d.Competitions[s.Competition]
			if c == nil {
				add("station %s: no competition entry %q", s.ID, s.Competition)
			} else if c.Kind != s.Type {
				add("station %s: competition kind %q != station type %q", s.ID, c.Kind, s.Type)
			}
		case "selection":
			sel, ok := d.Rules.Selections[s.Selection]
			if !ok || sel.Count <= 0 || len(sel.Effect) == 0 {
				add("station %s: bad selection %q", s.ID, s.Selection)
			}
		case "event":
			events++
		}
		if i == 0 && s.Type != "setup" {
			add("calendar must start with setup")
		}
	}
	if len(d.Calendar) == 0 {
		add("calendar: empty")
	} else if last := d.Calendar[len(d.Calendar)-1]; last.Type != "regatta" && last.Type != "ergocup" && last.Type != "jtfo" {
		add("calendar must end with a competition")
	}
	if len(d.Rules.EventCardsPerStation) != events {
		add("rules: eventCardsPerStation has %d entries for %d event stations", len(d.Rules.EventCardsPerStation), events)
	}

	for id, c := range d.Competitions {
		if len(c.Weights) == 0 {
			add("competition %s: no weights", id)
		}
		for s := range c.Weights {
			if !slices.Contains(StatNames, s) {
				add("competition %s: bad weight stat %q", id, s)
			}
		}
		if !slices.Contains(validContexts, c.Context) {
			add("competition %s: bad context %q", id, c.Context)
		}
		raceSets := [][]Race{c.Races}
		if len(c.Variants) > 0 {
			raceSets = nil
			for _, n := range []string{"2", "3", "4"} {
				v, ok := c.Variants[d.Rules.JTFOVariants[n]]
				if !ok {
					add("competition %s: no variant for %s players", id, n)
				}
				raceSets = append(raceSets, v)
			}
		}
		for _, races := range raceSets {
			if len(races) == 0 {
				add("competition %s: no races", id)
			}
			raceIDs := map[string]bool{}
			for _, r := range races {
				if raceIDs[r.ID] {
					add("competition %s: duplicate race %s", id, r.ID)
				}
				raceIDs[r.ID] = true
				if err := d.checkRace(c, r); err != nil {
					add("competition %s race %s: %v", id, r.ID, err)
				}
			}
		}
	}

	// Rules sanity.
	r := d.Rules
	if r.TrainingPoints <= 0 || r.Camp.Points <= 0 || r.Camp.MaxRowers <= 0 {
		add("rules: training/camp points must be > 0")
	}
	if r.Incidents.Kentern+r.Incidents.Material+r.Incidents.Krebs > 100 {
		add("rules: incident probabilities exceed 100 %%")
	}
	if len(r.WeatherTable) != 6 {
		add("rules: weatherTable needs 6 faces")
	}
	for _, w := range r.WeatherTable {
		if !slices.Contains(validWeather, w) {
			add("rules: bad weather %q", w)
		}
	}
	if r.Deal.RowersPerAgeGroup <= 0 || r.Deal.Trainers <= 0 {
		add("rules: bad deal sizes")
	}
	return errors.Join(errs...)
}

// checkRace validates a race spec and that it is fillable from the full rower set.
func (d *Data) checkRace(c *Competition, r Race) error {
	seats, ok := seatsByClass[r.Class]
	if !ok {
		return fmt.Errorf("bad class %q", r.Class)
	}
	if r.Seats != seats {
		return fmt.Errorf("seats %d != %d for class %s", r.Seats, seats, r.Class)
	}
	if (r.Class == "ergo") != (c.Kind == "ergocup") {
		return fmt.Errorf("ergo class only in ergo cups")
	}
	if len(r.Ages) == 0 || len(r.Genders) == 0 {
		return fmt.Errorf("no ages or genders")
	}
	for _, a := range r.Ages {
		if !slices.Contains(validAges, a) {
			return fmt.Errorf("bad age %q", a)
		}
	}
	for _, g := range r.Genders {
		if !slices.Contains(validGenders, g) {
			return fmt.Errorf("bad gender %q", g)
		}
	}
	if r.Mixed && (len(r.Genders) < 2 || len(r.Ages) < 2) {
		return fmt.Errorf("mixed race needs two genders and two ages")
	}
	eligible, girls, boys := 0, 0, 0
	ages := map[string]bool{}
	for _, rw := range d.Rowers {
		if slices.Contains(r.Ages, rw.Age) && slices.Contains(r.Genders, rw.Gender) {
			eligible++
			ages[rw.Age] = true
			if rw.Gender == "w" {
				girls++
			} else {
				boys++
			}
		}
	}
	if eligible < r.Seats {
		return fmt.Errorf("not fillable: %d eligible rowers for %d seats", eligible, r.Seats)
	}
	if r.Mixed && (girls == 0 || boys == 0 || len(ages) < 2) {
		return fmt.Errorf("mixed race not fillable")
	}
	return nil
}
