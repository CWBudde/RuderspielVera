package engine

import (
	"slices"

	"ruderspiel/internal/data"
)

// draftState is the open offer after a competition. Picks are sequential.
type draftState struct {
	offer []poolCard
	order []string // player IDs in pick order (Sponsor may add an extra pick)
	turn  int
	// personal are extra cards only the player of the open pick may take (humanOffer boats);
	// they are not in the pool and taking one uses up the pick.
	personal []poolCard
}

func (d *draftState) offerIndex(id string) int {
	if d == nil {
		return -1
	}

	for i, c := range d.offer {
		if c.id() == id {
			return i
		}
	}

	return -1
}

func (d *draftState) personalIndex(id string) int {
	if d == nil {
		return -1
	}

	return slices.IndexFunc(d.personal, func(c poolCard) bool { return c.id() == id })
}

// valid reports whether the player of the open pick may take the card.
func (d *draftState) valid(id string) bool { return d.offerIndex(id) >= 0 || d.personalIndex(id) >= 0 }

// openDraft draws an open offer of (players + extraOffer) cards from the pool and opens the first
// pick. With an empty pool it still opens when a human player has a personal offer (Paintrain).
// Pick order: fewest medal points first (catch-up), then fewer golds, then seat order.
func (g *Game) openDraft() bool {
	rules := g.d.Rules.Draft
	section := g.currentStation().Section
	var eligible []int

	for i, c := range g.pool {
		if c.trainer != nil && section < rules.TrainersFromSection {
			continue
		}

		eligible = append(eligible, i)
	}

	extra := g.sponsorPicks()

	var offer []poolCard

	switch {
	case len(eligible) > 0:
		size := min(len(g.players)+rules.ExtraOffer, len(eligible))
		offer = g.drawOffer(eligible, size, rules.AtLeastOneBoat)
	case !slices.ContainsFunc(g.players, func(p *player) bool { return len(g.personalOffer(p)) > 0 }):
		return false
	}

	ps := g.draftOrder()

	order := make([]string, 0, len(ps)+extra)
	for _, p := range ps {
		order = append(order, p.id)
	}

	for range extra {
		order = append(order, ps[0].id)
		g.addLog("ereignis", ps[0].id, msg("log.sponsor", "player", ps[0].name))
	}

	g.draft = &draftState{offer: offer, order: order, turn: -1}

	return g.nextDraftPick()
}

// drawOffer draws size cards from the eligible pool indices, starting with one random boat if
// atLeastOneBoat is set and a boat is eligible.
func (g *Game) drawOffer(eligible []int, size int, atLeastOneBoat bool) []poolCard {
	offer := make([]poolCard, 0, size)

	if atLeastOneBoat {
		var boats []int

		for _, i := range eligible {
			if g.pool[i].boat != nil {
				boats = append(boats, i)
			}
		}

		if len(boats) > 0 {
			pick := boats[g.rng.IntN(len(boats))]
			offer = append(offer, g.pool[pick])
			eligible = slices.DeleteFunc(eligible, func(i int) bool { return i == pick })
		}
	}

	for len(offer) < size {
		k := g.rng.IntN(len(eligible))
		offer = append(offer, g.pool[eligible[k]])
		eligible = slices.Delete(eligible, k, k+1)
	}

	return offer
}

// draftOrder sorts the players for the draft: fewest medal points first (catch-up), then fewer
// golds, then seat order.
func (g *Game) draftOrder() []*player {
	ps := slices.Clone(g.players)
	slices.SortStableFunc(ps, func(a, b *player) int {
		if pa, pb := g.points(a), g.points(b); pa != pb {
			return pa - pb
		}

		if a.gold != b.gold {
			return a.gold - b.gold
		}

		return a.seat - b.seat
	})

	return ps
}

// nextDraftPick opens the next pick of a player who has something to take (open offer or a
// personal card); it ends the draft and returns false when nobody is left.
func (g *Game) nextDraftPick() bool {
	for g.draft.turn++; g.draft.turn < len(g.draft.order); g.draft.turn++ {
		pl := g.playerByID(g.draft.order[g.draft.turn])

		g.draft.personal = g.personalOffer(pl)
		if len(g.draft.offer) > 0 || len(g.draft.personal) > 0 {
			g.newPhase(PhaseDraft, stageNone, []string{pl.id}, msg("title.draft"), msg("prompt.draft", "player", pl.name))
			return true
		}
	}

	g.draft = nil

	return false
}

// personalOffer returns the humanOffer boats offered to pl at this pick: only to human players who
// don't own the boat yet, and only in the last draft before the competition that rows its class.
func (g *Game) personalOffer(pl *player) []poolCard {
	if pl.isBot {
		return nil
	}

	var cards []poolCard

	for i := range g.d.Boats {
		b := &g.d.Boats[i]
		if !b.HumanOffer || !g.nextCompetitionRows(BoatClass(b.Class)) ||
			slices.ContainsFunc(pl.boats, func(own *boat) bool { return own.defID == b.ID }) {
			continue
		}

		cards = append(cards, poolCard{boat: ownBoat(b, pl)})
	}

	return cards
}

// nextCompetitionRows reports whether the next competition after the current station has a race of
// the boat class.
func (g *Game) nextCompetitionRows(cls BoatClass) bool {
	for i := g.station + 1; i < len(g.d.Calendar); i++ {
		def := g.d.Competitions[g.d.Calendar[i].Competition]
		if def == nil {
			continue
		}

		return slices.ContainsFunc(g.d.RacesFor(def, len(g.players)), func(r data.Race) bool {
			return BoatClass(r.Class) == cls
		})
	}

	return false
}

func (g *Game) applyDraftPick(pl *player, cardID string) {
	if k := g.draft.personalIndex(cardID); k >= 0 {
		c := g.draft.personal[k]
		g.draft.personal = nil
		pl.boats = append(pl.boats, c.boat)
		g.addLog("draft", pl.id, msg("log.draftPick", "player", pl.name, "card", c.nameMsg(), "kind", msg("cardKind."+c.kind())))

		return
	}

	i := g.draft.offerIndex(cardID)
	c := g.draft.offer[i]
	g.draft.offer = slices.Delete(g.draft.offer, i, i+1)
	g.pool = slices.DeleteFunc(g.pool, func(pc poolCard) bool { return pc.id() == cardID })

	switch {
	case c.rower != nil:
		pl.rowers = append(pl.rowers, c.rower)
	case c.boat != nil:
		pl.boats = append(pl.boats, c.boat)
	case c.trainer != nil:
		pl.trainers = append(pl.trainers, c.trainer)
	}

	g.addLog("draft", pl.id, msg("log.draftPick", "player", pl.name, "card", c.nameMsg(), "kind", msg("cardKind."+c.kind())))
}
