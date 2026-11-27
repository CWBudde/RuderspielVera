package engine

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// autoDecision is a trivial but legal decision helper for tests. It uses its own RNG so it never
// touches the game's RNG.
func autoDecision(g *Game, pid string, rng *rand.Rand) Decision {
	pv := g.CurrentPhase(pid)
	dec := Decision{PhaseID: pv.ID, Kind: pv.Kind}
	pl := g.playerByID(pid)

	switch pv.Kind {
	case PhaseTraining:
		n := pv.MaxRowers
		if n == 0 {
			n = 3
		}

		ids := shuffledRowerIDs(pl, rng)[:min(n, len(pl.rowers))]
		alloc := map[[2]string]int{}
		var keys [][2]string

		for range pv.Points {
			k := [2]string{ids[rng.IntN(len(ids))], string(AllStats[rng.IntN(len(AllStats))])}
			if alloc[k] == 0 {
				keys = append(keys, k)
			}

			alloc[k]++
		}

		for _, k := range keys {
			dec.Training = append(dec.Training, TrainingAlloc{RowerID: k[0], Stat: Stat(k[1]), Points: alloc[k]})
		}
	case PhaseSelection:
		ids := shuffledRowerIDs(pl, rng)
		dec.RowerIDs = ids[:min(pv.Count, len(ids))]
	case PhaseEventSwap:
		if rng.IntN(2) == 0 {
			dec.Swap = &[2]Stat{StatKraft, StatTechnik}
		}
	case PhaseRegatta:
		dec.TrainerID, dec.Entries = autoEntries(g, pl, pv.Competition, rng)
	case PhaseDraft:
		dec.DraftCardID = pv.DraftOffer[rng.IntN(len(pv.DraftOffer))].ID
	}

	return dec
}

func shuffledRowerIDs(pl *player, rng *rand.Rand) []string {
	ids := make([]string, len(pl.rowers))
	for i, r := range pl.rowers {
		ids[i] = r.id
	}

	rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	return ids
}

// autoEntries greedily fills races with the best eligible unused rowers and the best free boat.
func autoEntries(g *Game, pl *player, cv *CompetitionView, rng *rand.Rand) (string, []Entry) {
	cs := g.phase.comp

	var best *trainer
	for _, t := range pl.trainers {
		if best == nil || trainerEffect(t, cs) > trainerEffect(best, cs) {
			best = t
		}
	}

	trainerID := ""
	if best != nil {
		trainerID = best.def.ID
	}

	usedR := map[string]bool{}
	usedB := map[string]bool{}
	var entries []Entry
	score := func(r *rower) float64 {
		s := 0.0
		for _, st := range AllStats {
			s += cv.Weights[st] * float64(r.stats[st])
		}

		return s
	}

	for _, race := range cv.Races {
		if rng.IntN(10) == 0 {
			continue // sometimes skip a race
		}
		var cands []*rower

		for _, r := range pl.rowers {
			if !usedR[r.id] && slices.Contains(race.Ages, r.age) && slices.Contains(race.Genders, r.gender) {
				cands = append(cands, r)
			}
		}

		if len(cands) < race.Seats {
			continue
		}

		slices.SortStableFunc(cands, func(a, b *rower) int {
			switch {
			case score(a) > score(b):
				return -1
			case score(a) < score(b):
				return 1
			}

			return 0
		})

		crew := cands[:race.Seats]
		if race.Mixed {
			crew = mixedCrew(cands, race.Seats)
			if crew == nil {
				continue
			}
		}

		boatID := ""

		if race.Class != BoatErgo && race.Class != Boat8x {
			var bb *boat
			for _, b := range pl.boats {
				if b.class == race.Class && !usedB[b.id] && (bb == nil || b.value > bb.value) {
					bb = b
				}
			}

			if bb == nil && !race.LoanBoat {
				continue
			}

			if bb != nil {
				boatID = bb.id
				usedB[bb.id] = true
			}
		}

		e := Entry{RaceID: race.ID, BoatID: boatID}

		for _, r := range crew {
			usedR[r.id] = true
			e.RowerIDs = append(e.RowerIDs, r.id)
		}

		entries = append(entries, e)
	}

	return trainerID, entries
}

// mixedCrew picks a crew satisfying the mixed rule from score-sorted candidates, or nil.
func mixedCrew(cands []*rower, seats int) []*rower {
	for i, a := range cands {
		for _, b := range cands[i+1:] {
			if a.gender == b.gender {
				continue
			}

			crew := []*rower{a, b}
			for _, c := range cands {
				if len(crew) < seats && c != a && c != b {
					crew = append(crew, c)
				}
			}

			if len(crew) == seats && !mixedOK(crew) && seats > 2 {
				for _, c := range cands {
					if c.age != a.age && !slices.Contains(crew, c) {
						crew[seats-1] = c
						break
					}
				}
			}

			if len(crew) == seats && mixedOK(crew) {
				return crew
			}
		}
	}

	return nil
}

func threePlayers(seed uint64) Config {
	return Config{Seed: seed, Players: []PlayerConfig{
		{ID: "p1", Name: "Vera"}, {ID: "p2", Name: "Taktiker", IsBot: true}, {ID: "p3", Name: "Bauchgefühl", IsBot: true},
	}}
}

// playToEnd lets the auto player decide for every seat until game over.
func playToEnd(tb testing.TB, g *Game, seed uint64) {
	tb.Helper()

	rng := rand.New(rand.NewPCG(seed, 77))

	for steps := 0; !g.Over(); steps++ {
		if steps > 5000 {
			tb.Fatalf("seed %d: no game over after %d phases (phase %s)", seed, steps, g.PhaseID())
		}

		for _, pid := range g.Pending() {
			err := g.Submit(pid, autoDecision(g, pid, rng))
			if err != nil {
				tb.Fatalf("seed %d: phase %s, player %s: %v", seed, g.PhaseID(), pid, err)
			}
		}
	}
}
