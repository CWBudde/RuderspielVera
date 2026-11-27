// Package bot implements the computer opponents of Ruderspiel.
//
// A bot sees exactly what a human in its seat sees: the GameView of its player (own cards,
// public cards of the opponents, the open phase) plus the public rule data and EstimateEntry for
// the open competition. It never reads other players' pending decisions. All randomness comes
// from the rng passed by the caller, so the game's own RNG (and thus replay determinism) is
// untouched; bot decisions end up in the decision log like any human decision.
package bot

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

	"ruderspiel/internal/data"
	"ruderspiel/internal/engine"
)

// Personality selects the playing style of a bot.
type Personality string

const (
	// Taktiker plays sharp: small perception noise, full lineup optimisation.
	Taktiker Personality = "taktiker"
	// Bauchgefuehl plays by gut feeling: noisy perception of strengths, simpler lineups.
	Bauchgefuehl Personality = "bauchgefuehl"
)

// PersonalityFor returns the default personality of a seat: odd seats are Taktiker, even seats
// Bauchgefühl (seat 0 is usually the human, so a 1-human/2-bot game gets one of each).
func PersonalityFor(seat int) Personality {
	if seat%2 == 1 {
		return Taktiker
	}

	return Bauchgefuehl
}

func (p Personality) noise() float64 {
	if p == Taktiker {
		return 0.02
	}

	return 0.07
}

// passes is the number of local-search passes over a lineup (0 = greedy only).
func (p Personality) passes() int {
	if p == Taktiker {
		return 8
	}

	return 1
}

// Errors returned by Decide.
var (
	ErrGameOver      = errors.New("bot: game is over")
	ErrNotPending    = errors.New("bot: player has nothing to decide")
	ErrUnknownPlayer = errors.New("bot: unknown player")
	ErrUnknownPhase  = errors.New("bot: cannot decide phase kind")
)

// Decide returns a legal decision for the given player in the open phase.
func Decide(g *engine.Game, playerID string, p Personality, rng *rand.Rand) (engine.Decision, error) {
	if g.Over() {
		return engine.Decision{}, ErrGameOver
	}

	if !slices.Contains(g.Pending(), playerID) {
		return engine.Decision{}, fmt.Errorf("%w: %s", ErrNotPending, playerID)
	}

	d, err := data.Load()
	if err != nil {
		return engine.Decision{}, fmt.Errorf("bot: %w", err)
	}

	b := newBrain(g, playerID, p, rng, d)
	if b.me == nil {
		return engine.Decision{}, fmt.Errorf("%w: %s", ErrUnknownPlayer, playerID)
	}

	pv := b.view.Phase

	dec := engine.Decision{PhaseID: pv.ID, Kind: pv.Kind}
	switch pv.Kind {
	case engine.PhaseAck:
	case engine.PhaseTraining:
		dec.Training = b.training(pv.Points, pv.MaxRowers)
	case engine.PhaseSelection:
		dec.RowerIDs = b.selection(pv.Count)
	case engine.PhaseEventSwap:
		dec.Swap = b.swap(pv.SwapRowerID)
	case engine.PhaseRegatta:
		dec.TrainerID, dec.Entries = b.regatta()
	case engine.PhaseDraft:
		dec.DraftCardID = b.draft(pv.DraftOffer)
	default:
		return engine.Decision{}, fmt.Errorf("%w %q", ErrUnknownPhase, pv.Kind)
	}

	return dec, nil
}

// brain holds the per-decision context.
type brain struct {
	g     *engine.Game
	d     *data.Data
	pid   string
	p     Personality
	rng   *rand.Rand
	view  engine.GameView
	me    *engine.PlayerSummary
	opps  []*engine.PlayerSummary
	noise map[string]float64 // perceived relative strength error per own rower, fixed per decision
}

func newBrain(g *engine.Game, pid string, p Personality, rng *rand.Rand, d *data.Data) *brain {
	b := &brain{g: g, d: d, pid: pid, p: p, rng: rng, view: g.View(pid), noise: map[string]float64{}}
	for i := range b.view.Players {
		ps := &b.view.Players[i]
		if ps.ID == pid {
			b.me = ps
		} else {
			b.opps = append(b.opps, ps)
		}
	}

	if b.me != nil {
		for _, r := range b.me.Rowers {
			b.noise[r.ID] = rng.NormFloat64() * p.noise()
		}
	}

	return b
}

// ---------- shared helpers ----------

func teamOf(ps *engine.PlayerSummary) *team {
	return &team{id: ps.ID, rowers: ps.Rowers, boats: ps.Boats, trainers: ps.Trainers}
}

// upcoming returns the competitions strictly after the current station (at most n).
func (b *brain) upcoming(n int, includeCurrent bool) []*compModel {
	var out []*compModel

	start := b.view.Current + 1
	if includeCurrent {
		start = b.view.Current
	}

	for i := start; i < len(b.d.Calendar) && len(out) < n; i++ {
		st := b.d.Calendar[i]
		if st.Competition == "" {
			continue
		}

		out = append(out, b.futureComp(i))
	}

	return out
}

// lookahead weights for the k-th upcoming competition.
//
//nolint:gochecknoglobals // read-only weight table
var lookahead = []float64{1, 0.6, 0.35, 0.2}

func weighted(stats engine.Stats, w map[engine.Stat]float64) float64 {
	s := 0.0
	for st, x := range w {
		s += x * float64(stats[st])
	}

	return s
}

func phi(x float64) float64 { return 0.5 * (1 + math.Erf(x/math.Sqrt2)) }

func sigma(x float64) float64 { return 0.05*math.Abs(x) + 4 }

// expMedals is the expected medal points of a score against competitor scores (normal model).
func (b *brain) expMedals(me float64, opps []float64) float64 {
	gold, silver := float64(b.d.Rules.MedalPoints.Gold), float64(b.d.Rules.MedalPoints.Silver)
	sm := sigma(me)
	pg := 1.0

	ps := make([]float64, len(opps))
	for i, o := range opps {
		so := sigma(o)
		ps[i] = phi((me - o) / math.Sqrt(sm*sm+so*so))
		pg *= ps[i]
	}

	p2 := 0.0

	for i := range opps {
		x := 1 - ps[i]
		for j := range opps {
			if j != i {
				x *= ps[j]
			}
		}

		p2 += x
	}

	return gold*pg + silver*p2
}
