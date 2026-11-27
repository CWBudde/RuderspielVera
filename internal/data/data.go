// Package data holds the embedded, hand-editable rule data of Ruderspiel (cards, calendar,
// competitions and all tunable numbers) and loads it into validated, typed structs.
//
// The structs use plain strings for enums so this package does not depend on the engine; the
// engine converts them into its contract types.
package data

import (
	"embed"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
)

//go:embed *.json
var files embed.FS

// StatNames are the stat keys used in stat maps.
var StatNames = []string{"kraft", "ausdauer", "geschwindigkeit", "technik", "motivation"}

// Rower is one starting rower card.
type Rower struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Age       string         `json:"age"`    // kinder | b | a
	Gender    string         `json:"gender"` // w | m
	Archetype string         `json:"archetype"`
	Rank      int            `json:"rank"` // sum of the five starting stats
	Stats     map[string]int `json:"stats"`
	ArtKey    string         `json:"artKey"`
}

// Boat is a boat card. City boats are templates; the engine gives every player one copy each.
type Boat struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Class  string `json:"class"` // 1x | 2x | 4x | 8x
	Value  int    `json:"value"`
	Group  string `json:"group"` // city | standard | premium | shared
	City   bool   `json:"city,omitempty"`
	Shared bool   `json:"shared,omitempty"` // the identical RVB eight
	ArtKey string `json:"artKey"`
}

// TrainerEffect applies per boat / ergo entry when weather and context match (empty = any).
type TrainerEffect struct {
	Weather string `json:"weather,omitempty"`
	Context string `json:"context,omitempty"`
	Value   int    `json:"value"`
}

type Trainer struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Gender      string          `json:"gender"`
	Ability     string          `json:"ability"`
	Description string          `json:"description"`
	Effects     []TrainerEffect `json:"effects"`
	ArtKey      string          `json:"artKey"`
}

// EventType is a personal event card type (Regelwerk §12.2 plus PoC additions).
//
// Besides the hard-wired types (phoenix, tausch, …) a card can be fully data-driven: Stats are
// applied once when the card is dealt, Bonus is added to the rower's entry at every competition
// that matches Scope, Context and (if set) one of Weathers.
type EventType struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`
	Stat        string `json:"stat,omitempty"` // stat events only
	Value       int    `json:"value"`
	// Scope says where the card can still matter: "water", "ergo" or "any". A water or ergo card is
	// only dealt while a matching competition (and Context, if set) is still ahead.
	Scope    string         `json:"scope"`
	Stats    map[string]int `json:"stats,omitempty"`    // immediate stat changes
	Bonus    int            `json:"bonus,omitempty"`    // per-rower entry bonus at matching competitions
	Weathers []string       `json:"weathers,omitempty"` // bonus only in one of these weathers
	Context  string         `json:"context,omitempty"`  // bonus only at competitions with this context
}

// Ereignis is a global modifier card.
type Ereignis struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`
	Effect      string `json:"effect"`
	// Scope ("water", "ergo", "any") says which competition the card is meant for: a water or ergo
	// card is only drawn when the next competition is of that kind, and its competition modifier
	// is consumed by the next competition of that kind.
	Scope   string             `json:"scope"`
	Weather string             `json:"weather,omitempty"` // force_weather
	Stat    string             `json:"stat,omitempty"`    // all_stat, random_rower_stat
	Weights map[string]float64 `json:"weights,omitempty"` // weight_mod: added to the competition weights
	Value   int                `json:"value"`
}

// Ereignis effect kinds.
const (
	EffTrainingPoints  = "training_points"
	EffKrebsChance     = "krebs_chance"    // value: Krebs probability in per mille (‰)
	EffMaterialChance  = "material_chance" // value: Materialschaden probability in per mille (‰)
	EffRandomRowerStat = "random_rower_stat"
	EffAllStat         = "all_stat"
	EffWeightMod       = "weight_mod"
	EffGuestMod        = "guest_mod" // value: added to every guest stat level
	EffSponsorPick     = "sponsor_pick"
	EffForceWeather    = "force_weather"
	EffNone            = "none"
	EffCityBoatMod     = "city_boat_mod"
	ScopeWater         = "water"
	ScopeErgo          = "ergo"
	ScopeAny           = "any"
)

var ereignisEffects = map[string]bool{
	EffTrainingPoints: true, EffKrebsChance: true, EffMaterialChance: true, EffRandomRowerStat: true,
	EffAllStat: true, EffWeightMod: true, EffGuestMod: true, EffSponsorPick: true, EffForceWeather: true,
	EffNone: true, EffCityBoatMod: true,
}

// waterOnlyEffects are Ereignis effects that only exist at water regattas (weather, incidents,
// boats); their cards must have scope water.
var waterOnlyEffects = map[string]bool{
	EffKrebsChance: true, EffMaterialChance: true, EffForceWeather: true, EffCityBoatMod: true,
}

// CompetitionEffect reports whether an Ereignis effect is a modifier that the next matching
// competition consumes (as opposed to immediate effects and training / draft modifiers).
func CompetitionEffect(effect string) bool {
	return waterOnlyEffects[effect] || effect == EffWeightMod || effect == EffGuestMod
}

// ImmediateEffect reports whether an Ereignis effect is applied when the card is drawn.
func ImmediateEffect(effect string) bool {
	return effect == EffNone || effect == EffAllStat || effect == EffRandomRowerStat
}

// ScopeMatches reports whether a competition is of the kind a scope names: water regattas (incl.
// JTFO) for "water", Ergo-Cups for "ergo", every competition for "any".
func ScopeMatches(scope string, c *Competition) bool {
	switch scope {
	case ScopeWater:
		return c.Water
	case ScopeErgo:
		return c.Kind == kindErgoCup
	default:
		return true
	}
}

type MissionCondition struct {
	Type        string `json:"type"`                  // medal | starts
	Medal       string `json:"medal,omitempty"`       // gold | any
	Class       string `json:"class,omitempty"`       // boat class filter
	Kind        string `json:"kind,omitempty"`        // competition kind filter (regatta|ergocup|jtfo)
	Competition string `json:"competition,omitempty"` // competition id filter
	BadWeather  bool   `json:"badWeather,omitempty"`  // weather must not be normal
	Count       int    `json:"count,omitempty"`       // starts
	Before      string `json:"before,omitempty"`      // competition id deadline (exclusive)
}

type MissionReward struct {
	Type  string `json:"type"` // stat | points
	Value int    `json:"value"`
}

type Mission struct {
	ID         string           `json:"id"`
	Text       string           `json:"text"` // "{name}" is replaced by the rower's name
	RewardText string           `json:"rewardText"`
	Condition  MissionCondition `json:"condition"`
	Reward     MissionReward    `json:"reward"`
}

// Station is one calendar step.
type Station struct {
	ID          string `json:"id"`
	Section     int    `json:"section"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Info        string `json:"info"`
	Competition string `json:"competition,omitempty"`
	Selection   string `json:"selection,omitempty"`
}

type Race struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Class    string   `json:"class"`
	Ages     []string `json:"ages"`
	Genders  []string `json:"genders"`
	Mixed    bool     `json:"mixed,omitempty"`
	Seats    int      `json:"seats"`
	LoanBoat bool     `json:"loanBoat,omitempty"` // a loan boat is provided when no own boat is given
}

type Competition struct {
	ID       string             `json:"-"`
	Name     string             `json:"name"`
	Kind     string             `json:"kind"` // regatta | ergocup | jtfo
	Context  string             `json:"context,omitempty"`
	Water    bool               `json:"water"`
	Weights  map[string]float64 `json:"weights"`
	Races    []Race             `json:"races,omitempty"`
	Variants map[string][]Race  `json:"variants,omitempty"` // JTFO: variant key -> races
	Notes    []string           `json:"notes,omitempty"`
	// GuestOffset is added to the guest crews' stat level at this competition.
	GuestOffset float64 `json:"guestOffset,omitempty"`
}

type Selection struct {
	Count  int            `json:"count"`
	Effect map[string]int `json:"effect"`
	Label  string         `json:"label"`
}

type Rules struct {
	TrainingPoints int `json:"trainingPoints"`
	Camp           struct {
		Points    int `json:"points"`
		MaxRowers int `json:"maxRowers"`
	} `json:"camp"`
	Selections           map[string]Selection `json:"selections"`
	Incidents            Incidents            `json:"incidents"`
	KrebsPenalty         int                  `json:"krebsPenalty"`
	WeatherTable         []string             `json:"weatherTable"` // d6 faces
	EventCardsPerStation []int                `json:"eventCardsPerStation"`
	Deal                 struct {
		RowersPerAgeGroup int `json:"rowersPerAgeGroup"`
		MinPerGender      int `json:"minPerGender"`
		Trainers          int `json:"trainers"`
	} `json:"deal"`
	Draft struct {
		ExtraOffer          int  `json:"extraOffer"`
		TrainersFromSection int  `json:"trainersFromSection"`
		AtLeastOneBoat      bool `json:"atLeastOneBoat"`
	} `json:"draft"`
	MedalPoints struct {
		Gold   int `json:"gold"`
		Silver int `json:"silver"`
	} `json:"medalPoints"`
	WinMotivation WinMotivation     `json:"winMotivation"`
	LoanBoatValue int               `json:"loanBoatValue"`
	JTFOVariants  map[string]string `json:"jtfoVariants"` // player count -> variant key
	// BoatValuePerSeat: a boat's value counts once per seat (1x ×1, 2x ×2, 4x ×4) instead of once.
	BoatValuePerSeat bool `json:"boatValuePerSeat"`
	// TrainerPerSeat: a trainer effect counts once per seat instead of once per boat.
	TrainerPerSeat bool `json:"trainerPerSeat"`
	// BoatValueFactor and TrainerFactor multiply boat values and trainer effects (0 means 1).
	BoatValueFactor int         `json:"boatValueFactor"`
	TrainerFactor   int         `json:"trainerFactor"`
	Guests          Guests      `json:"guests"`
	TrainerMood     TrainerMood `json:"trainerMood"`
}

// WinMotivation is the temporary Motivation bonus (Siegesmotivation) of every rower of a crew
// that wins one of Medals. It never stacks: a new win while it is active only resets the
// duration to Competitions. Every scored competition of the calendar (regatta, Ergo-Cup, JTFO)
// counts down one, whether or not the rower started.
type WinMotivation struct {
	Value        int      `json:"value"`
	Competitions int      `json:"competitions"`
	Medals       []string `json:"medals"` // "gold", "silber"
}

// TrainerMood is the bounded random walk of every trainer's mood (Laune). At the start of every
// section from the second on, each trainer (catalog order) draws one step from Steps with the
// weights of its current mood (Weights["-1"] etc.; this is where mean reversion lives), and the
// result is clamped to [Min, Max]. The mood is added to the trainer's per-seat effect.
type TrainerMood struct {
	Start   int              `json:"start"`
	Min     int              `json:"min"`
	Max     int              `json:"max"`
	Steps   []int            `json:"steps"`
	Weights map[string][]int `json:"weights"` // current mood -> weight per step
}

// Incidents are the probabilities of the mutually exclusive water incidents in per mille (‰).
// One d1000 per boat entry: Kentern first, then Materialschaden, then Krebs, else clean.
type Incidents struct {
	// KenternBySeats is the capsize probability by the boat's number of seats (1, 2, 4, 8):
	// bigger crews capsize less often.
	KenternBySeats map[int]int `json:"kenternBySeats"`
	Material       int         `json:"material"`
	Krebs          int         `json:"krebs"`
}

// Kentern is the capsize probability (‰) of a boat with the given number of seats.
func (i Incidents) Kentern(seats int) int { return i.KenternBySeats[seats] }

// BoatScale is how often a boat value counts in a race with the given number of seats.
func (r Rules) BoatScale(seats int) int { return scale(r.BoatValuePerSeat, r.BoatValueFactor, seats) }

// TrainerScale is how often a trainer effect counts in a race with the given number of seats.
func (r Rules) TrainerScale(seats int) int { return scale(r.TrainerPerSeat, r.TrainerFactor, seats) }

func scale(perSeat bool, factor, seats int) int {
	if factor == 0 {
		factor = 1
	}

	if perSeat {
		return factor * seats
	}

	return factor
}

// Guests configures the guest crews (Gastmannschaften): one fixed-strength crew per contested
// race that takes a place like a player but never scores medal points.
type Guests struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
	// StatLevel is the value of every stat of a guest rower by the race's oldest allowed age group.
	StatLevel map[string]float64 `json:"statLevel"`
	// GrowthPerStation is added to every stat level per calendar station (season progress).
	GrowthPerStation float64 `json:"growthPerStation"`
	BoatValue        int     `json:"boatValue"` // value of the guest boat (scaled like player boats)
}

// Data is the complete, validated rule data set. Treat it as read-only.
type Data struct {
	Rowers       []Rower
	Boats        []Boat
	Trainers     []Trainer
	Events       []EventType
	Ereignisse   []Ereignis
	Missions     []Mission
	Calendar     []Station
	Competitions map[string]*Competition
	Rules        Rules
}

var (
	loadOnce sync.Once
	loaded   *Data
	errLoad  error
)

// Load parses and validates the embedded data once and returns the shared, read-only result.
func Load() (*Data, error) {
	loadOnce.Do(func() { loaded, errLoad = parse() })
	return loaded, errLoad
}

// MustLoad is Load for tests and tools; it panics on invalid data.
func MustLoad() *Data {
	d, err := Load()
	if err != nil {
		panic(err)
	}

	return d
}

func parse() (*Data, error) {
	d := &Data{}

	targets := []struct {
		file string
		v    any
	}{
		{"rowers.json", &d.Rowers},
		{"boats.json", &d.Boats},
		{"trainers.json", &d.Trainers},
		{"events.json", &d.Events},
		{"ereignisse.json", &d.Ereignisse},
		{"missions.json", &d.Missions},
		{"calendar.json", &d.Calendar},
		{"competitions.json", &d.Competitions},
		{"rules.json", &d.Rules},
	}
	for _, t := range targets {
		raw, err := files.ReadFile(t.file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", t.file, err)
		}

		err = json.Unmarshal(raw, t.v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.file, err)
		}
	}

	for id, c := range d.Competitions {
		c.ID = id
	}

	err := d.Validate()
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Boat returns the boat with the given ID or nil.
func (d *Data) Boat(id string) *Boat {
	for i := range d.Boats {
		if d.Boats[i].ID == id {
			return &d.Boats[i]
		}
	}

	return nil
}

// SharedBoat returns the shared RVB eight (or nil).
func (d *Data) SharedBoat() *Boat {
	for i := range d.Boats {
		if d.Boats[i].Shared {
			return &d.Boats[i]
		}
	}

	return nil
}

// EventType returns the event type definition or nil.
func (d *Data) EventType(t string) *EventType {
	for i := range d.Events {
		if d.Events[i].Type == t {
			return &d.Events[i]
		}
	}

	return nil
}

// RacesFor returns the races of a competition for the given number of players (JTFO variants).
func (d *Data) RacesFor(c *Competition, players int) []Race {
	if len(c.Variants) == 0 {
		return c.Races
	}

	return c.Variants[d.Rules.JTFOVariants[strconv.Itoa(players)]]
}
