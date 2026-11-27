package engine

import (
	"fmt"
	"slices"
)

// draftState is the open offer after a competition. Picks are sequential.
type draftState struct {
	offer []poolCard
	order []string // player IDs in pick order (Sponsor may add an extra pick)
	turn  int
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

var cardKindNames = map[string]string{"rower": "Ruderer", "boat": "Boot", "trainer": "Trainer"}

// openDraft draws an open offer of (players + extraOffer) cards from the pool and opens the first
// pick. Pick order: fewest medal points first (catch-up), then fewer golds, then seat order.
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

	if len(eligible) == 0 {
		return false
	}

	size := min(len(g.players)+rules.ExtraOffer, len(eligible))
	offer := g.drawOffer(eligible, size, rules.AtLeastOneBoat)
	ps := g.draftOrder()

	order := make([]string, 0, len(ps)+extra)
	for _, p := range ps {
		order = append(order, p.id)
	}

	for range extra {
		order = append(order, ps[0].id)
		g.addLog("ereignis", ps[0].id, fmt.Sprintf("Sponsor: %s darf eine zusätzliche Karte wählen.", ps[0].name))
	}

	g.draft = &draftState{offer: offer, order: order}
	g.openDraftPick()

	return true
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

func (g *Game) openDraftPick() {
	pid := g.draft.order[g.draft.turn]
	pl := g.playerByID(pid)
	g.newPhase(PhaseDraft, stageNone, []string{pid}, "Kartenauswahl",
		pl.name+" wählt eine Karte aus dem offenen Angebot.")
}

func (g *Game) nextDraftPick() bool {
	g.draft.turn++
	if g.draft.turn < len(g.draft.order) && len(g.draft.offer) > 0 {
		g.openDraftPick()
		return true
	}

	g.draft = nil

	return false
}

func (g *Game) applyDraftPick(pl *player, cardID string) {
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

	g.addLog("draft", pl.id, fmt.Sprintf("%s wählt %s (%s).", pl.name, c.name(), cardKindNames[c.kind()]))
}
