// Package engine is the pure, deterministic Ruderspiel rules engine.
//
// This file is the shared CONTRACT between engine, API and frontend. The JSON shapes here are what
// the web client receives (GameView) and sends (Decision). TS types are generated from this file
// (tygo -> web/src/api/types.gen.ts). Change it deliberately and update docs/api.md alongside.
package engine

// ---------- enums (string-typed for readable JSON) ----------

type Stat string

const (
	StatKraft      Stat = "kraft"
	StatAusdauer   Stat = "ausdauer"
	StatSpeed      Stat = "geschwindigkeit"
	StatTechnik    Stat = "technik"
	StatMotivation Stat = "motivation"
)

// AllStats is the canonical display order.
var AllStats = []Stat{StatKraft, StatAusdauer, StatSpeed, StatTechnik, StatMotivation}

type AgeGroup string

const (
	AgeKinder AgeGroup = "kinder"
	AgeB      AgeGroup = "b"
	AgeA      AgeGroup = "a"
)

type Gender string

const (
	GenderF Gender = "w" // Mädchen
	GenderM Gender = "m" // Jungen
)

type BoatClass string

const (
	Boat1x   BoatClass = "1x"
	Boat2x   BoatClass = "2x"
	Boat4x   BoatClass = "4x"
	Boat8x   BoatClass = "8x"   // only RVB-Endspurt, shared identical boat
	BoatErgo BoatClass = "ergo" // ergometer, no boat card
)

type Weather string

const (
	WeatherRegen  Weather = "regen"
	WeatherHitze  Weather = "hitze"
	WeatherWind   Weather = "wind"
	WeatherKaelte Weather = "kaelte"
	WeatherNormal Weather = "normal"
)

type StationType string

const (
	StationSetup     StationType = "setup" // Spielvorbereitung
	StationTraining  StationType = "training"
	StationEvent     StationType = "event"
	StationEreignis  StationType = "ereignis"
	StationRegatta   StationType = "regatta" // water regatta (incl. Hildesheim, RVH, RVB)
	StationErgoCup   StationType = "ergocup"
	StationSelection StationType = "selection" // WR5, Basketball, Wanderfahrt: pick N rowers
	StationCamp      StationType = "camp"      // Trainingslager (Hankensbüttel, Kassel)
	StationJTFO      StationType = "jtfo"
)

// PhaseKind tells the client which decision UI to show.
type PhaseKind string

const (
	PhaseAck       PhaseKind = "ack"       // just confirm ("Weiter"), e.g. setup, ereignis reveal, results
	PhaseTraining  PhaseKind = "training"  // distribute Points (training or camp)
	PhaseSelection PhaseKind = "selection" // choose exactly Count rowers
	PhaseEventSwap PhaseKind = "eventswap" // Tausch-Event: swap two stats of one rower (or skip)
	PhaseRegatta   PhaseKind = "regatta"   // trainer + entries (regatta, ergocup, jtfo)
	PhaseDraft     PhaseKind = "draft"     // pick one card from the open offer
	PhaseGameOver  PhaseKind = "gameover"
)

type IncidentKind string

const (
	IncidentNone     IncidentKind = ""
	IncidentKentern  IncidentKind = "kentern"
	IncidentMaterial IncidentKind = "material"
	IncidentKrebs    IncidentKind = "krebs"
)

type Medal string

const (
	MedalNone   Medal = ""
	MedalGold   Medal = "gold"
	MedalSilver Medal = "silber"
)

// ---------- text ----------

// Msg is player-facing text as a catalog key plus params. The engine never builds sentences: the
// frontend renders a Msg per language (web/src/i18n: messages.*.ts for engine keys, ui.*.ts for
// shared vocabulary like "stat.kraft", data.gen.ts for "data.*" keys from internal/data).
//
// Args values are one of:
//   - string: proper names (players, rowers) and preformatted signed numbers ("+2")
//   - int / float64: numbers, formatted by the frontend for the language
//   - Msg: a translatable reference, e.g. a stat name or a data text
//   - []any of the above: rendered as a natural-language list ("A, B und C")
type Msg struct {
	Key  string         `json:"key"`
	Args map[string]any `json:"args,omitempty"`
}

// ---------- card views ----------

type Stats map[Stat]int

type RowerView struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Age    AgeGroup       `json:"age"`
	Gender Gender         `json:"gender"`
	Stats  Stats          `json:"stats"`  // current effective values (base + training + effects + winMotivation)
	Base   Stats          `json:"base"`   // starting values, for "+N trained" display
	ArtKey string         `json:"artKey"` // portrait key, e.g. "rower-b-w"
	Event  *EventCardView `json:"event,omitempty"`
	// WinMotivation (Siegesmotivation) is the temporary Motivation bonus after a gold or silver
	// placement; it is already included in stats.motivation. It is not trainable or swappable.
	WinMotivation int `json:"winMotivation,omitempty"`
	// WinMotivationLeft is the number of scored competitions (regatta, Ergo-Cup, JTFO) the bonus
	// still counts in, the next one included; it expires after the last of them.
	WinMotivationLeft int `json:"winMotivationLeft,omitempty"`
}

type BoatView struct {
	ID string `json:"id"`
	// DefID is the boat's id in boats.json (city boats get a per-player suffix in ID); the UI
	// looks up the translated name as "data.boat.<defId>.name".
	DefID  string    `json:"defId"`
	Name   string    `json:"name"` // German fallback
	Class  BoatClass `json:"class"`
	Value  int       `json:"value"`
	City   bool      `json:"city"` // städtisches Boot
	ArtKey string    `json:"artKey"`
}

type TrainerEffect struct {
	Weather Weather `json:"weather,omitempty"` // empty = applies regardless of weather
	Context string  `json:"context,omitempty"` // "", "ergo", "sprint", "langstrecke"
	Value   int     `json:"value"`             // per seat (rules.json trainerPerSeat) or per boat
}

type TrainerView struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`        // German fallback of "data.trainer.<id>.name"
	Ability     string          `json:"ability"`     // German fallback of "data.trainer.<id>.ability"
	Description string          `json:"description"` // German fallback of "data.trainer.<id>.description"
	Effects     []TrainerEffect `json:"effects"`
	ArtKey      string          `json:"artKey"`
	// Mood (Laune, public) is added to the trainer's per-seat effect in every entry, like an
	// effect (× RaceSpec.trainerScale). It drifts between rules.json trainerMood min and max at
	// every new section. 0 in the catalog.
	Mood int `json:"mood"`
	// MoodTrend is the change of Mood at the last section start (negative = got grumpier).
	MoodTrend int `json:"moodTrend"`
}

type EventCardView struct {
	Type        string       `json:"type"`        // e.g. "phoenix", "verdopplung", "mission"
	Name        string       `json:"name"`        // German fallback of "data.event.<type>.name"
	Description string       `json:"description"` // German fallback of "data.event.<type>.description"
	RowerID     string       `json:"rowerId"`
	Mission     *MissionView `json:"mission,omitempty"`
	// Seelenzwilling: the linked teammate. At the next training or camp in which either of the two
	// gets points, each also receives the other's points (mirrored per stat); then Spent is true.
	PartnerID   string `json:"partnerId,omitempty"`
	PartnerName string `json:"partnerName,omitempty"`
	Spent       bool   `json:"spent,omitempty"`
}

type MissionView struct {
	// ID is the mission id; the UI renders "data.mission.<id>.text" with {name} = RowerName and
	// "data.mission.<id>.reward".
	ID        string `json:"id"`
	RowerName string `json:"rowerName"`
	Text      string `json:"text"`   // German fallback
	Reward    string `json:"reward"` // German fallback
	Done      bool   `json:"done"`
	Failed    bool   `json:"failed"`
}

// ---------- game view ----------

type StationView struct {
	ID      string      `json:"id"`      // calendar.json id; text keys "data.station.<id>.name|info"
	Index   int         `json:"index"`   // 0-based position in calendar
	Section int         `json:"section"` // 1..12
	Type    StationType `json:"type"`
	Name    string      `json:"name"` // German fallback, e.g. "Hildesheimer Regatta"
	Info    string      `json:"info"` // German fallback: what happens here
}

type PlayerSummary struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	IsBot    bool          `json:"isBot"`
	Gold     int           `json:"gold"`
	Silver   int           `json:"silver"`
	Points   int           `json:"points"`  // 2*gold + silver (+ mission bonus points)
	Decided  bool          `json:"decided"` // has submitted a decision for the current phase
	Rowers   []RowerView   `json:"rowers"`  // public table cards
	Boats    []BoatView    `json:"boats"`
	Trainers []TrainerView `json:"trainers"`
}

// RaceSpec describes one race inside a competition.
type RaceSpec struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`    // German fallback
	NameKey string     `json:"nameKey"` // catalog key of the name (race IDs repeat across JTFO variants)
	Class   BoatClass  `json:"class"`
	Ages    []AgeGroup `json:"ages"`    // allowed ages (any of)
	Genders []Gender   `json:"genders"` // allowed genders (any of)
	Mixed   bool       `json:"mixed"`   // requires >=1 girl, >=1 boy, >=2 age groups
	Seats   int        `json:"seats"`
	// LoanBoat: boatId may be empty; a loan boat (rules.json loanBoatValue) is used instead (JTFO,
	// Hildesheim). A loan boat is not an own boat, so it never counts against one start per boat.
	LoanBoat bool `json:"loanBoat,omitempty"`

	// Scoring hints for client-side estimates (set in competition views):
	// boat part = (boat value [+ cityBoatMod for city boats], or FixedBoatValue) × BoatScale;
	// trainer part = matching trainer effects × TrainerScale.
	BoatScale    int `json:"boatScale"`
	TrainerScale int `json:"trainerScale"`
	// FixedBoatValue is the base value used without an own boat: the shared eight (8x) or the loan
	// boat (loanBoat races). 0 for ergo.
	FixedBoatValue int `json:"fixedBoatValue,omitempty"`
	// GuestScore is the guest crew's (Gastmannschaft) total in this race, 0 if there is none.
	GuestScore int `json:"guestScore,omitempty"`
}

type CompetitionView struct {
	ID      string           `json:"id"`   // competitions.json key ("bremen")
	Name    string           `json:"name"` // German fallback (the station name)
	Kind    StationType      `json:"kind"` // regatta | ergocup | jtfo
	Weights map[Stat]float64 `json:"weights"`
	Races   []RaceSpec       `json:"races"`
	Weather Weather          `json:"weather,omitempty"` // revealed before decisions (water only)
	// Context is matched against TrainerEffect.Context ("", "ergo", "sprint", "langstrecke").
	Context string `json:"context,omitempty"`
	// CityBoatMod is added to the value of städtische Boote (Ereignis modifier, may be negative).
	CityBoatMod int   `json:"cityBoatMod,omitempty"`
	TeamEvent   bool  `json:"teamEvent"` // JTFO: all races form one team score
	Notes       []Msg `json:"notes"`     // rule hints (static competition notes, Ereignis modifiers)
}

// PhaseView is what the current station asks of the viewing player.
type PhaseView struct {
	ID              string           `json:"id"` // must be echoed in the decision (stale-decision guard)
	Kind            PhaseKind        `json:"kind"`
	Title           Msg              `json:"title"`
	Prompt          Msg              `json:"prompt"`                    // instruction text
	Points          int              `json:"points,omitempty"`          // training/camp
	MaxRowers       int              `json:"maxRowers,omitempty"`       // camp: at most N rowers
	Count           int              `json:"count,omitempty"`           // selection: exactly N rowers
	SelectionEffect string           `json:"selectionEffect,omitempty"` // German fallback, e.g. "+10 Motivation"
	SelectionID     string           `json:"selectionId,omitempty"`     // rules.json selection key; text "data.selection.<id>.label"
	Competition     *CompetitionView `json:"competition,omitempty"`
	DraftOffer      []DraftCard      `json:"draftOffer,omitempty"`
	DraftOrder      []string         `json:"draftOrder,omitempty"` // player IDs
	SwapRowerID     string           `json:"swapRowerId,omitempty"`
	Waiting         bool             `json:"waiting"` // viewer already decided, waiting for others

	// Stage refines ack phases: "setup", "event" (event cards dealt), "ereignis" (global event
	// revealed) or "result" (lastResult of this station is shown). Empty for other phases.
	Stage PhaseStage `json:"stage,omitempty"`
	// NewEvents: the viewer's event cards dealt at this event station (stage "event").
	NewEvents []EventCardView `json:"newEvents,omitempty"`
	// Ereignis: the drawn global event (stage "ereignis").
	Ereignis *EreignisView `json:"ereignis,omitempty"`
	// DraftTurn: index into DraftOrder of the pick that is open now (draft phases; 0 is omitted).
	// DraftOrder may name a player twice (Sponsor extra pick), so this is an index, not an ID.
	DraftTurn int `json:"draftTurn,omitempty"`
}

type PhaseStage string

const (
	StageNone     PhaseStage = ""
	StageSetup    PhaseStage = "setup"
	StageEvent    PhaseStage = "event"
	StageEreignis PhaseStage = "ereignis"
	StageResult   PhaseStage = "result"
)

type EreignisView struct {
	ID          string `json:"id"`          // ereignisse.json id; text "data.ereignis.<id>.name|description"
	Name        string `json:"name"`        // German fallback
	Description string `json:"description"` // German fallback of the card text
	Details     []Msg  `json:"details"`     // per-player details (affected rowers)
}

type DraftCard struct {
	ID      string       `json:"id"`
	Kind    string       `json:"kind"` // "rower" | "boat" | "trainer"
	Rower   *RowerView   `json:"rower,omitempty"`
	Boat    *BoatView    `json:"boat,omitempty"`
	Trainer *TrainerView `json:"trainer,omitempty"`
}

// ---------- results / log ----------

type EntryResult struct {
	PlayerID   string       `json:"playerId"`        // empty for a guest crew
	Guest      string       `json:"guest,omitempty"` // German name of a guest crew; the UI shows "data.guest.name"
	RowerIDs   []string     `json:"rowerIds"`
	BoatID     string       `json:"boatId,omitempty"`
	TrainerID  string       `json:"trainerId,omitempty"`
	Crew       int          `json:"crew"` // summed weighted rower score (rounded)
	BoatValue  int          `json:"boatValue"`
	Trainer    int          `json:"trainer"`
	EventBonus int          `json:"eventBonus"`
	Penalty    int          `json:"penalty"`
	Total      int          `json:"total"`
	Incident   IncidentKind `json:"incident,omitempty"`
	Place      int          `json:"place"` // 1-based, ties share; 0 = DNF
	Medal      Medal        `json:"medal,omitempty"`
}

type RaceResult struct {
	RaceID  string        `json:"raceId"`
	Name    string        `json:"name"`    // German fallback
	NameKey string        `json:"nameKey"` // catalog key of the name ("race.team" for the JTFO team score)
	Entries []EntryResult `json:"entries"` // sorted by place
}

type CompetitionResult struct {
	StationIndex int          `json:"stationIndex"`
	Name         string       `json:"name"` // German fallback of the station name
	Weather      Weather      `json:"weather,omitempty"`
	Races        []RaceResult `json:"races"`
}

type LogEntry struct {
	StationIndex int    `json:"stationIndex"`
	Kind         string `json:"kind"` // "station", "event", "ereignis", "result", "draft", "mission", "info"
	Msg          Msg    `json:"msg"`
	PlayerID     string `json:"playerId,omitempty"`
}

type GameView struct {
	GameID           string             `json:"gameId"`
	You              string             `json:"you"` // viewing player ID
	Calendar         []StationView      `json:"calendar"`
	Current          int                `json:"current"` // index into Calendar
	Phase            PhaseView          `json:"phase"`
	Players          []PlayerSummary    `json:"players"`              // includes you; your entry has full detail
	LastResult       *CompetitionResult `json:"lastResult,omitempty"` // most recent competition
	ActiveEreignisse []Msg              `json:"activeEreignisse"`     // pending modifiers
	Log              []LogEntry         `json:"log"`                  // newest last; may be truncated by API
	Winners          []string           `json:"winners,omitempty"`    // set at game over (ties possible)
}

// ---------- catalog (static card data, GET /api/catalog) ----------

type EventTypeView struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CatalogView struct {
	Rowers   []RowerView     `json:"rowers"`
	Boats    []BoatView      `json:"boats"`
	Trainers []TrainerView   `json:"trainers"`
	Events   []EventTypeView `json:"events"`
}

// ---------- decisions (client -> server) ----------

type TrainingAlloc struct {
	RowerID string `json:"rowerId"`
	Stat    Stat   `json:"stat"`
	Points  int    `json:"points"`
}

type Entry struct {
	RaceID   string   `json:"raceId"`
	RowerIDs []string `json:"rowerIds"`         // len == race seats
	BoatID   string   `json:"boatId,omitempty"` // empty for ergo and 8x (shared eight)
}

// Decision is a tagged union by Kind (must match Phase.Kind). Only the matching fields are read.
type Decision struct {
	PhaseID     string          `json:"phaseId"`
	Kind        PhaseKind       `json:"kind"`
	Training    []TrainingAlloc `json:"training,omitempty"`
	RowerIDs    []string        `json:"rowerIds,omitempty"`  // selection
	Swap        *[2]Stat        `json:"swap,omitempty"`      // eventswap; nil = skip
	TrainerID   string          `json:"trainerId,omitempty"` // regatta (ignored for ergocup if none)
	Entries     []Entry         `json:"entries,omitempty"`   // regatta; omitted race = no entry
	DraftCardID string          `json:"draftCardId,omitempty"`
}
