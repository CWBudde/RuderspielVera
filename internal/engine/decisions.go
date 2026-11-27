package engine

import (
	"errors"
	"fmt"
	"slices"
)

// ErrStalePhase is returned when a decision names a phase that is no longer open (HTTP 409).
// The UI shows the message key "err.stalePhase".
var ErrStalePhase = errors.New("err.stalePhase")

// ValidationError is an invalid decision (HTTP 400); Msg is shown in the UI.
type ValidationError struct{ Msg Msg }

func (e *ValidationError) Error() string { return e.Msg.String() }

// invalid builds a ValidationError from a message key and name/value pairs (see NewMsg).
func invalid(key string, kv ...any) error {
	return &ValidationError{Msg: msg(key, kv...)}
}

// SubmittedDecision is one entry of the persisted decision log.
type SubmittedDecision struct {
	PlayerID string   `json:"playerId"`
	Decision Decision `json:"decision"`
}

// Submit validates and records a decision. When the last required decision of the phase arrives,
// the phase is resolved and the game advances to the next phase.
func (g *Game) Submit(playerID string, dec Decision) error {
	if g.over {
		return invalid("err.gameOver")
	}

	pl := g.playerByID(playerID)
	if pl == nil {
		return invalid("err.unknownPlayer")
	}

	p := g.phase
	if dec.PhaseID != p.id {
		return ErrStalePhase
	}

	if !p.isRequired(playerID) {
		return invalid("err.notYourTurn")
	}

	if _, done := p.decisions[playerID]; done {
		return invalid("err.alreadyDecided")
	}

	if dec.Kind != p.kind {
		return invalid("err.wrongKind", "want", string(p.kind), "got", string(dec.Kind))
	}

	err := g.validate(pl, p, dec)
	if err != nil {
		return err
	}

	dec = cloneDecision(dec)
	p.decisions[playerID] = dec

	g.decisions = append(g.decisions, SubmittedDecision{PlayerID: playerID, Decision: dec})
	if len(p.decisions) == len(p.required) {
		g.resolve(p)
		g.afterPhase(p)
	}

	return nil
}

func (g *Game) validate(pl *player, p *phase, d Decision) error {
	switch p.kind {
	case PhaseAck:
		return nil
	case PhaseTraining:
		return g.validateTraining(pl, p, d.Training)
	case PhaseSelection:
		return validateSelection(pl, p, d.RowerIDs)
	case PhaseEventSwap:
		return validateSwap(d.Swap)
	case PhaseRegatta:
		return g.validateEntries(pl, p.comp, d.TrainerID, d.Entries)
	case PhaseDraft:
		if g.draft.offerIndex(d.DraftCardID) < 0 {
			return invalid("err.cardNotOffered")
		}

		return nil
	}

	return invalid("err.noDecision")
}

// resolve applies all decisions of a phase in seat order (deterministic RNG order).
func (g *Game) resolve(p *phase) {
	switch p.kind {
	case PhaseTraining:
		for _, id := range p.required {
			g.applyTraining(g.playerByID(id), p.decisions[id].Training)
		}
	case PhaseSelection:
		for _, id := range p.required {
			g.applySelection(g.playerByID(id), p, p.decisions[id].RowerIDs)
		}
	case PhaseEventSwap:
		for _, id := range p.required {
			g.applySwap(g.playerByID(id), p.swap[id], p.decisions[id].Swap)
		}
	case PhaseRegatta:
		g.resolveCompetition(p)
	case PhaseDraft:
		for _, id := range p.required {
			g.applyDraftPick(g.playerByID(id), p.decisions[id].DraftCardID)
		}
	}
}

func cloneDecision(d Decision) Decision {
	cp := d
	cp.Training = slices.Clone(d.Training)

	cp.RowerIDs = slices.Clone(d.RowerIDs)
	if d.Swap != nil {
		s := *d.Swap
		cp.Swap = &s
	}

	cp.Entries = make([]Entry, len(d.Entries))
	for i, e := range d.Entries {
		cp.Entries[i] = Entry{RaceID: e.RaceID, RowerIDs: slices.Clone(e.RowerIDs), BoatID: e.BoatID}
	}

	if len(d.Entries) == 0 {
		cp.Entries = nil
	}

	return cp
}

// Replay rebuilds a game from its config and decision log. Any rejected decision is an error.
func Replay(cfg Config, log []SubmittedDecision) (*Game, error) {
	g, err := New(cfg)
	if err != nil {
		return nil, err
	}

	for i, sd := range log {
		err := g.Submit(sd.PlayerID, sd.Decision)
		if err != nil {
			return nil, fmt.Errorf("replay decision %d (%s, %s): %w", i, sd.PlayerID, sd.Decision.PhaseID, err)
		}
	}

	return g, nil
}

// Decisions returns a copy of the accepted decision log (for persistence).
func (g *Game) Decisions() []SubmittedDecision { return slices.Clone(g.decisions) }

// Config returns the configuration the game was created with.
func (g *Game) Config() Config { return g.cfg }

// Over reports whether the season has ended.
func (g *Game) Over() bool { return g.over }

// Winners returns the winning player IDs (several on a tie); empty before game over.
func (g *Game) Winners() []string { return slices.Clone(g.winners) }

// Log returns the full resolution log.
func (g *Game) Log() []LogEntry { return append([]LogEntry{}, g.log...) }

// Results returns all competition results so far.
func (g *Game) Results() []CompetitionResult {
	out := make([]CompetitionResult, len(g.results))
	for i, r := range g.results {
		out[i] = cloneResult(r)
		fillNil(&out[i])
	}

	return out
}

// PhaseID returns the ID of the currently open phase.
func (g *Game) PhaseID() string { return g.phase.id }

// Pending returns the IDs of players whose decision the current phase still needs.
func (g *Game) Pending() []string {
	var ids []string

	for _, id := range g.phase.required {
		if _, ok := g.phase.decisions[id]; !ok {
			ids = append(ids, id)
		}
	}

	return ids
}
