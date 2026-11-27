// Command sim plays all-bot seasons and prints balancing statistics.
//
//	go run ./cmd/sim -games 1000 -seed 1 [-players 3] [-mode default|taktiker|bauch|rotate]
package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"

	"ruderspiel/internal/bot"
	"ruderspiel/internal/engine"
)

type gameStats struct {
	err         error
	personality []bot.Personality
	points      []int
	gold        []int
	silver      []int
	winners     []int // seats
	results     []engine.CompetitionResult
	draftKinds  map[bot.Personality]map[string]int
}

func main() {
	games := flag.Int("games", 500, "number of seasons")
	seed := flag.Uint64("seed", 1, "first game seed")
	players := flag.Int("players", 3, "players per game (2-4)")
	mode := flag.String("mode", "default", "personalities: default (bot.PersonalityFor), taktiker, bauch, rotate")
	workers := flag.Int("workers", runtime.NumCPU(), "parallel games")
	flag.Parse()

	out := make([]gameStats, *games)
	var wg sync.WaitGroup
	jobs := make(chan int)
	for range max(*workers, 1) {
		wg.Go(func() {
			for i := range jobs {
				out[i] = play(*seed+uint64(i), *players, personalities(*mode, *players, i))
			}
		})
	}
	for i := range *games {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	for i, gs := range out {
		if gs.err != nil {
			fmt.Fprintf(os.Stderr, "game %d (seed %d): %v\n", i, *seed+uint64(i), gs.err)
			os.Exit(1)
		}
	}
	report(out, *players, *mode)
}

func personalities(mode string, n, game int) []bot.Personality {
	ps := make([]bot.Personality, n)
	for s := range ps {
		switch mode {
		case "taktiker":
			ps[s] = bot.Taktiker
		case "bauch":
			ps[s] = bot.Bauchgefuehl
		case "rotate":
			ps[s] = bot.PersonalityFor(s + game)
		default:
			ps[s] = bot.PersonalityFor(s)
		}
	}
	return ps
}

func play(seed uint64, n int, ps []bot.Personality) gameStats {
	cfg := engine.Config{Seed: seed}
	for s := range n {
		cfg.Players = append(cfg.Players, engine.PlayerConfig{ID: fmt.Sprintf("p%d", s), Name: fmt.Sprintf("Bot %d", s), IsBot: true})
	}
	gs := gameStats{personality: ps, draftKinds: map[bot.Personality]map[string]int{}}
	g, err := engine.New(cfg)
	if err != nil {
		gs.err = err
		return gs
	}
	rngs := make([]*rand.Rand, n)
	for s := range rngs {
		rngs[s] = rand.New(rand.NewPCG(seed, 1000+uint64(s)))
	}
	seatOf := func(pid string) int {
		var s int
		fmt.Sscanf(pid, "p%d", &s)
		return s
	}
	for steps := 0; !g.Over(); steps++ {
		if steps > 10000 {
			gs.err = fmt.Errorf("no game over")
			return gs
		}
		for _, pid := range g.Pending() {
			s := seatOf(pid)
			d, err := bot.Decide(g, pid, ps[s], rngs[s])
			if err != nil {
				gs.err = err
				return gs
			}
			if err := g.Submit(pid, d); err != nil {
				gs.err = fmt.Errorf("phase %s, %s (%s): rejected: %w", g.PhaseID(), pid, ps[s], err)
				return gs
			}
			if d.Kind == engine.PhaseDraft {
				k, _, _ := strings.Cut(d.DraftCardID, ":")
				if gs.draftKinds[ps[s]] == nil {
					gs.draftKinds[ps[s]] = map[string]int{}
				}
				gs.draftKinds[ps[s]][k]++
			}
		}
	}
	v := g.View("p0")
	for _, p := range v.Players {
		gs.points = append(gs.points, p.Points)
		gs.gold = append(gs.gold, p.Gold)
		gs.silver = append(gs.silver, p.Silver)
	}
	for _, w := range g.Winners() {
		gs.winners = append(gs.winners, seatOf(w))
	}
	gs.results = g.Results()
	return gs
}

type agg struct{ n, sum float64 }

func (a *agg) add(x float64) { a.n++; a.sum += x }
func (a agg) mean() float64 {
	if a.n == 0 {
		return 0
	}
	return a.sum / a.n
}

func pct(a, b float64) string {
	if b == 0 {
		return "   -  "
	}
	return fmt.Sprintf("%5.1f%%", 100*a/b)
}

func report(out []gameStats, n int, mode string) {
	G := float64(len(out))
	fmt.Printf("Ruderspiel sim: %d games, %d players, personalities %s\n\n", len(out), n, mode)

	// Wins.
	seatWins := make([]float64, n)
	persWins, persSeats := map[bot.Personality]float64{}, map[bot.Personality]float64{}
	ties := 0
	for _, gs := range out {
		if len(gs.winners) > 1 {
			ties++
		}
		for _, s := range gs.winners {
			seatWins[s] += 1 / float64(len(gs.winners))
			persWins[gs.personality[s]] += 1 / float64(len(gs.winners))
		}
		for _, p := range gs.personality {
			persSeats[p]++
		}
	}
	fmt.Println("Win rate (ties split):")
	for s := range n {
		fmt.Printf("  seat %d (%s): %s\n", s, out[0].personality[s], pct(seatWins[s], G))
	}
	for _, p := range []bot.Personality{bot.Taktiker, bot.Bauchgefuehl} {
		if persSeats[p] > 0 {
			fmt.Printf("  %-12s  %s per seat-game (fair share %s)\n", p, pct(persWins[p], persSeats[p]), pct(1, float64(n)))
		}
	}
	fmt.Printf("  shared wins: %s of games\n\n", pct(float64(ties), G))

	// Points.
	var all []int
	var gold, silver, spread agg
	persPts := map[bot.Personality]*agg{}
	for _, gs := range out {
		for s, p := range gs.points {
			all = append(all, p)
			gold.add(float64(gs.gold[s]))
			silver.add(float64(gs.silver[s]))
			if persPts[gs.personality[s]] == nil {
				persPts[gs.personality[s]] = &agg{}
			}
			persPts[gs.personality[s]].add(float64(p))
		}
		spread.add(float64(slices.Max(gs.points) - slices.Min(gs.points)))
	}
	sort.Ints(all)
	sum, sq := 0.0, 0.0
	in := 0
	for _, p := range all {
		sum += float64(p)
		if p >= 15 && p <= 40 {
			in++
		}
	}
	mean := sum / float64(len(all))
	for _, p := range all {
		sq += (float64(p) - mean) * (float64(p) - mean)
	}
	q := func(f float64) int { return all[int(f*float64(len(all)-1))] }
	fmt.Printf("Final points: mean %.1f (sd %.1f), min %d, p10 %d, median %d, p90 %d, max %d; in 15..40: %s\n",
		mean, math.Sqrt(sq/float64(len(all))), all[0], q(0.1), q(0.5), q(0.9), all[len(all)-1], pct(float64(in), float64(len(all))))
	fmt.Printf("  winner minus last: %.1f points on average\n", spread.mean())
	for _, p := range []bot.Personality{bot.Taktiker, bot.Bauchgefuehl} {
		if a := persPts[p]; a != nil {
			fmt.Printf("  %-12s mean %.1f points\n", p, a.mean())
		}
	}
	fmt.Printf("Medals per player and season: gold %.1f, silver %.1f\n\n", gold.mean(), silver.mean())

	// Races.
	var entries, races, guestGold, guestMedal agg
	byCount := map[int]float64{}
	var water, kentern, material, krebs float64
	var crew, boat, trainer, event, penalty float64
	var contested, byBoat, byTrainer float64
	type compAgg struct {
		name                string
		total, entries, gst agg
	}
	comps := map[int]*compAgg{}
	for _, gs := range out {
		for _, cr := range gs.results {
			ca := comps[cr.StationIndex]
			if ca == nil {
				ca = &compAgg{name: cr.Name}
				comps[cr.StationIndex] = ca
			}
			for _, rr := range cr.Races {
				if rr.RaceID == "team" {
					continue
				}
				var ps []engine.EntryResult
				for _, e := range rr.Entries {
					if e.Guest != "" {
						if e.Place == 1 {
							guestGold.add(1)
							ca.gst.add(1)
						} else {
							guestGold.add(0)
							ca.gst.add(0)
						}
						if e.Place <= 2 && e.Place > 0 {
							guestMedal.add(1)
						} else {
							guestMedal.add(0)
						}
						continue
					}
					ps = append(ps, e)
				}
				races.add(1)
				entries.add(float64(len(ps)))
				byCount[len(ps)]++
				ca.entries.add(float64(len(ps)))
				for _, e := range ps {
					ca.total.add(float64(e.Total))
					crew += float64(e.Crew)
					boat += float64(e.BoatValue)
					trainer += float64(e.Trainer)
					event += float64(e.EventBonus)
					penalty += float64(e.Penalty)
					if cr.Weather != "" {
						water++
						switch e.Incident {
						case engine.IncidentKentern:
							kentern++
						case engine.IncidentMaterial:
							material++
						case engine.IncidentKrebs:
							krebs++
						}
					}
				}
				// Decided-by analysis between the two best finishing player entries.
				var fin []engine.EntryResult
				for _, e := range ps {
					if e.Place > 0 && e.Incident != engine.IncidentKentern && e.Incident != engine.IncidentMaterial {
						fin = append(fin, e)
					}
				}
				if len(fin) >= 2 && fin[0].Total > fin[1].Total {
					contested++
					a, b := fin[0], fin[1]
					if a.Total-a.BoatValue < b.Total-b.BoatValue {
						byBoat++
					}
					if a.Total-a.Trainer < b.Total-b.Trainer {
						byTrainer++
					}
				}
			}
		}
	}
	fmt.Printf("Races: %.1f player entries per race; races with 0/1/2/3/4 entrants: %s / %s / %s / %s / %s\n",
		entries.mean(), pct(byCount[0], races.n), pct(byCount[1], races.n), pct(byCount[2], races.n), pct(byCount[3], races.n), pct(byCount[4], races.n))
	if guestGold.n > 0 {
		fmt.Printf("  guest crew placed 1st in %s and in the top 2 in %s of contested races\n", pct(guestGold.sum, guestGold.n), pct(guestMedal.sum, guestMedal.n))
	}
	fmt.Printf("  races between >=2 finishing players: %s; of those decided by the boat %s, by the trainer %s\n",
		pct(contested, races.n), pct(byBoat, contested), pct(byTrainer, contested))
	fmt.Printf("Incidents per water entry: Kentern %s, Material %s, Krebs %s\n", pct(kentern, water), pct(material, water), pct(krebs, water))
	tot := crew + boat + trainer + event
	fmt.Printf("Score sources (all player entries): crew %s, boat %s, trainer %s, events %s; Krebs penalties %s\n\n",
		pct(crew, tot), pct(boat, tot), pct(trainer, tot), pct(event, tot), pct(penalty, tot))

	fmt.Println("Per competition: mean entry total, entries per race, guest crew 1st")
	keys := make([]int, 0, len(comps))
	for k := range comps {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		ca := comps[k]
		fmt.Printf("  %-30s %6.1f  %.2f  %s\n", ca.name, ca.total.mean(), ca.entries.mean(), pct(ca.gst.sum, ca.gst.n))
	}

	fmt.Println("\nDraft picks:")
	for _, p := range []bot.Personality{bot.Taktiker, bot.Bauchgefuehl} {
		kinds := map[string]float64{}
		total := 0.0
		for _, gs := range out {
			for k, c := range gs.draftKinds[p] {
				kinds[k] += float64(c)
				total += float64(c)
			}
		}
		if total > 0 {
			fmt.Printf("  %-12s rower %s, boat %s, trainer %s\n", p, pct(kinds["rower"], total), pct(kinds["boat"], total), pct(kinds["trainer"], total))
		}
	}
}
