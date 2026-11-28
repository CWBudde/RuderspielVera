package engine

import (
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"

	"ruderspiel/internal/data"
)

// Config starts a game. The game is fully determined by Config plus the ordered decision log.
type Config struct {
	GameID  string         `json:"gameId,omitempty"`
	Seed    uint64         `json:"seed"`
	Players []PlayerConfig `json:"players"` // seat order; 2–4 players
}

type PlayerConfig struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	IsBot bool   `json:"isBot"`
}

type rower struct {
	id, name string
	age      AgeGroup
	gender   Gender
	artKey   string
	base     Stats // starting values
	stats    Stats // current values
	event    *eventCard
	// Siegesmotivation: temporary Motivation bonus after a winning placement (rules.json
	// winMotivation). It is not part of stats, so training, Tausch and stat-based events never
	// see it; winLeft is the number of competitions it still lasts (0 = inactive).
	winBonus int
	winLeft  int
}

type eventCard struct {
	def     *data.EventType
	mission *missionState
	// Seelenzwilling: the linked teammate. At the next training (or camp) in which the card holder
	// or the partner gets points, each also receives the other's points; then the card is spent.
	partner *rower
	spent   bool
}

type missionState struct {
	def    *data.Mission
	rower  any // rowerName of the card holder, the {name} of the mission text
	starts int // competitions started since the mission was given
	done   bool
	failed bool
}

type boat struct {
	id, name string
	defID    string // boats.json id
	class    BoatClass
	value    int
	city     bool
	artKey   string
}

type trainer struct {
	def *data.Trainer
}

type player struct {
	id, name string
	isBot    bool
	seat     int
	rowers   []*rower
	boats    []*boat
	trainers []*trainer
	gold     int
	silver   int
	bonus    int // mission medal points
}

// poolCard is a card in the draft pool: exactly one of the pointers is set.
type poolCard struct {
	rower   *rower
	boat    *boat
	trainer *trainer
}

func (c poolCard) id() string {
	switch {
	case c.rower != nil:
		return "rower:" + c.rower.id
	case c.boat != nil:
		return "boat:" + c.boat.id
	default:
		return "trainer:" + c.trainer.def.ID
	}
}

func (c poolCard) kind() string {
	switch {
	case c.rower != nil:
		return "rower"
	case c.boat != nil:
		return "boat"
	default:
		return "trainer"
	}
}

// nameMsg is the card's name as a Msg arg (rower names are proper names).
func (c poolCard) nameMsg() any {
	switch {
	case c.rower != nil:
		return rowerName(c.rower)
	case c.boat != nil:
		return dataMsg("boat", c.boat.defID, "name")
	default:
		return dataMsg("trainer", c.trainer.def.ID, "name")
	}
}

// Game is the complete game state. It is not safe for concurrent use.
type Game struct {
	cfg       Config
	d         *data.Data
	rng       *rand.Rand
	players   []*player
	station   int
	phase     *phase
	phaseSeq  int
	pending   []*data.Ereignis // drawn global modifiers not consumed yet
	pool      []poolCard
	log       []LogEntry
	results   []CompetitionResult
	decisions []SubmittedDecision
	over      bool
	winners   []string

	eventStations int                 // event stations entered so far
	swapQueue     map[string][]string // player ID -> rower IDs with an unresolved Tausch card
	draft         *draftState
	mood          map[string]int // trainer ID -> current mood (Laune)
	prevMood      map[string]int // trainer ID -> mood before the last drift
}

// Config errors returned by New.
var (
	ErrPlayerCount = errors.New("2 bis 4 Spieler erforderlich")
	ErrBadPlayerID = errors.New("leere oder doppelte Spieler-ID")
)

// New deals the cards and opens the first phase.
func New(cfg Config) (*Game, error) {
	d, err := data.Load()
	if err != nil {
		return nil, fmt.Errorf("rule data: %w", err)
	}

	if n := len(cfg.Players); n < 2 || n > 4 {
		return nil, fmt.Errorf("%w, nicht %d", ErrPlayerCount, n)
	}

	g := &Game{
		cfg:       cfg,
		d:         d,
		rng:       newRNG(cfg.Seed),
		swapQueue: map[string][]string{},
	}

	seen := map[string]bool{}
	for i, pc := range cfg.Players {
		if pc.ID == "" || seen[pc.ID] {
			return nil, fmt.Errorf("%w: %q", ErrBadPlayerID, pc.ID)
		}

		seen[pc.ID] = true

		name := pc.Name
		if name == "" {
			name = pc.ID
		}

		g.players = append(g.players, &player{id: pc.ID, name: name, isBot: pc.IsBot, seat: i})
	}

	g.initMood()
	g.deal()
	g.enterStation(0)

	return g, nil
}

func newRowerFromData(r *data.Rower) *rower {
	base := Stats{}
	for _, s := range AllStats {
		base[s] = r.Stats[string(s)]
	}

	return &rower{
		id: r.ID, name: r.Name, age: AgeGroup(r.Age), gender: Gender(r.Gender), artKey: r.ArtKey,
		base: base, stats: base.clone(),
	}
}

func (s Stats) clone() Stats {
	c := Stats{}
	maps.Copy(c, s)

	return c
}

var ageOrder = []AgeGroup{AgeKinder, AgeB, AgeA}

// deal distributes the starting cards (rowers per age group by player count with at least one girl
// and one boy per age group, city boats, 2 random trainers) and builds the draft pool from the rest.
func (g *Game) deal() {
	rules := g.d.Rules.Deal
	byAge := map[AgeGroup][]*rower{}

	for i := range g.d.Rowers {
		r := newRowerFromData(&g.d.Rowers[i])
		byAge[r.age] = append(byAge[r.age], r)
	}

	for _, age := range ageOrder {
		g.dealAgeGroup(slices.Clone(byAge[age]), g.d.DealRowers(len(g.players), string(age)), rules.MinPerGender)
	}

	for _, p := range g.players {
		sortRowers(p.rowers)
	}

	g.poolUndealtRowers(byAge)
	g.dealTrainers(rules.Trainers)
	g.dealBoats()

	g.addLog("station", "", msg("log.setup", "rowers", len(g.players[0].rowers), "trainers", rules.Trainers))
}

// dealAgeGroup shuffles one age group and deals perPlayer rowers to each player, first
// minPerGender girls and boys per player in turn, then whoever is left.
func (g *Game) dealAgeGroup(list []*rower, perPlayer, minPerGender int) {
	shuffle(g.rng, list)

	taken := make([]bool, len(list))
	count := map[string]int{}

	take := func(p *player, gender Gender) {
		if count[p.id] >= perPlayer {
			return
		}

		for i, r := range list {
			if !taken[i] && (gender == "" || r.gender == gender) {
				taken[i] = true
				count[p.id]++
				p.rowers = append(p.rowers, r)

				return
			}
		}
	}

	for range minPerGender {
		for _, gender := range []Gender{GenderF, GenderM} {
			for _, p := range g.players {
				take(p, gender)
			}
		}
	}

	for _, p := range g.players {
		for count[p.id] < perPlayer && count[p.id] < len(list) {
			before := count[p.id]
			take(p, "")

			if count[p.id] == before {
				break
			}
		}
	}
}

// poolUndealtRowers puts every rower nobody was dealt into the draft pool, in catalog order.
func (g *Game) poolUndealtRowers(byAge map[AgeGroup][]*rower) {
	owned := map[string]bool{}

	for _, p := range g.players {
		for _, r := range p.rowers {
			owned[r.id] = true
		}
	}

	for i := range g.d.Rowers {
		if id := g.d.Rowers[i].ID; !owned[id] {
			for _, r := range byAge[AgeGroup(g.d.Rowers[i].Age)] {
				if r.id == id {
					g.pool = append(g.pool, poolCard{rower: r})
				}
			}
		}
	}
}

// dealTrainers deals perPlayer random trainers to each player; the rest go to the draft pool.
func (g *Game) dealTrainers(perPlayer int) {
	trainers := make([]*trainer, len(g.d.Trainers))
	for i := range g.d.Trainers {
		trainers[i] = &trainer{def: &g.d.Trainers[i]}
	}

	shuffle(g.rng, trainers)

	next := 0
	for _, p := range g.players {
		for k := 0; k < perPlayer && next < len(trainers); k++ {
			p.trainers = append(p.trainers, trainers[next])
			next++
		}
	}
	// Remaining trainers go to the pool in catalog order (draws are random anyway).
	for i := range g.d.Trainers {
		if !slices.ContainsFunc(trainers[:next], func(t *trainer) bool { return t.def.ID == g.d.Trainers[i].ID }) {
			g.pool = append(g.pool, poolCard{trainer: &trainer{def: &g.d.Trainers[i]}})
		}
	}
}

// dealBoats deals every player one random city boat per class and a copy of every starter boat,
// and puts the undealt city boats and the other boats (except humanOffer boats) into the draft
// pool. If a class has fewer city boats
// than players, the shortfall is dealt as copies of random city boats of that class. Dealt boats
// get a per-player ID suffix so copies stay unique.
func (g *Game) dealBoats() {
	dealt := map[string]bool{}

	for _, cls := range []BoatClass{Boat1x, Boat2x, Boat4x} {
		var defs []*data.Boat

		for i := range g.d.Boats {
			if b := &g.d.Boats[i]; b.City && BoatClass(b.Class) == cls {
				defs = append(defs, b)
			}
		}

		if len(defs) == 0 {
			continue
		}

		shuffle(g.rng, defs)

		for k, p := range g.players {
			var b *data.Boat
			if k < len(defs) {
				b = defs[k]
			} else {
				b = defs[g.rng.IntN(len(defs))]
			}

			dealt[b.ID] = true
			p.boats = append(p.boats, ownBoat(b, p))
		}
	}

	for i := range g.d.Boats {
		b := &g.d.Boats[i]

		switch {
		case b.Starter:
			for _, p := range g.players {
				p.boats = append(p.boats, ownBoat(b, p))
			}
		case b.HumanOffer, dealt[b.ID]:
		default:
			g.pool = append(g.pool, poolCard{boat: &boat{
				id: b.ID, defID: b.ID, name: b.Name, class: BoatClass(b.Class),
				value: b.Value, city: b.City, artKey: b.ArtKey,
			}})
		}
	}
}

// ownBoat is player p's copy of boat card b; the ID suffix keeps copies unique.
func ownBoat(b *data.Boat, p *player) *boat {
	return &boat{
		id: b.ID + "-" + p.id, defID: b.ID, name: b.Name, class: BoatClass(b.Class),
		value: b.Value, city: b.City, artKey: b.ArtKey,
	}
}

// bestBoat returns the player's most valuable boat of a class (nil if none).
func (p *player) bestBoat(cls BoatClass) *boat {
	var best *boat

	for _, b := range p.boats {
		if b.class == cls && (best == nil || b.value > best.value) {
			best = b
		}
	}

	return best
}

// sortRowers orders a roster A, B, Kinder and then by starting strength.
func sortRowers(rs []*rower) {
	rank := map[AgeGroup]int{AgeA: 0, AgeB: 1, AgeKinder: 2}

	slices.SortStableFunc(rs, func(a, b *rower) int {
		if rank[a.age] != rank[b.age] {
			return rank[a.age] - rank[b.age]
		}

		return b.base.sum() - a.base.sum()
	})
}

func (s Stats) sum() int {
	t := 0
	for _, v := range s {
		t += v
	}

	return t
}

// bestStat returns the highest current stat; ties go to the first in AllStats order.
func (s Stats) bestStat() Stat { return s.bestOf(AllStats) }

// bestOf returns the highest current stat among stats; ties go to the first in stats order.
func (s Stats) bestOf(stats []Stat) Stat {
	best := stats[0]
	for _, st := range stats[1:] {
		if s[st] > s[best] {
			best = st
		}
	}

	return best
}

func (g *Game) playerByID(id string) *player {
	for _, p := range g.players {
		if p.id == id {
			return p
		}
	}

	return nil
}

func (p *player) rower(id string) *rower {
	for _, r := range p.rowers {
		if r.id == id {
			return r
		}
	}

	return nil
}

func (p *player) boat(id string) *boat {
	for _, b := range p.boats {
		if b.id == id {
			return b
		}
	}

	return nil
}

func (p *player) trainer(id string) *trainer {
	for _, t := range p.trainers {
		if t.def.ID == id {
			return t
		}
	}

	return nil
}

func (g *Game) points(p *player) int {
	return p.gold*g.d.Rules.MedalPoints.Gold + p.silver*g.d.Rules.MedalPoints.Silver + p.bonus
}

func (g *Game) addLog(kind, playerID string, m Msg) {
	g.log = append(g.log, LogEntry{StationIndex: g.station, Kind: kind, Msg: m, PlayerID: playerID})
}

func (g *Game) currentStation() *data.Station { return &g.d.Calendar[g.station] }

// stat is the value a stat counts with in scoring: the current value plus the Siegesmotivation
// on Motivation.
func (r *rower) stat(s Stat) int {
	if s == StatMotivation {
		return r.stats[s] + r.winBonus
	}

	return r.stats[s]
}

// effectiveStats returns all stats as they count in scoring.
func (r *rower) effectiveStats() Stats {
	c := r.stats.clone()
	c[StatMotivation] = r.stat(StatMotivation)

	return c
}

// addStat changes a stat; stats never drop below 0.
func (r *rower) addStat(s Stat, delta int) {
	r.stats[s] += delta
	if r.stats[s] < 0 {
		r.stats[s] = 0
	}
}
