package bot

import (
	"maps"
	"slices"

	"ruderspiel/internal/engine"
)

// ---------- regatta / ergo cup / JTFO ----------

func (b *brain) regatta() (string, []engine.Entry) {
	c := b.currentComp()
	me := teamOf(b.me)

	jitter := 0.0
	if b.p != Taktiker {
		jitter = 2
	}

	trainer := b.bestTrainer(me, c, jitter)
	pr, pl := b.lineup(me, c, b.estimateField(c), trainer, b.p.passes())

	return trainer, pr.entries(pl)
}

// ---------- training, camp, selection ----------

// start is one planned start of a rower in an upcoming competition.
type start struct {
	k    int // index into the upcoming competitions
	race int // race index, -1 for the team score of a team competition
}

// outlook is the planned lineup for the next few competitions, used to value stat gains.
type outlook struct {
	comps  []*compModel
	fields []*field
	scores [][]float64 // per comp: per race score (team comps: index 0 holds the team total)
	starts map[string][]start
}

func (b *brain) outlook(n int) *outlook {
	ol := &outlook{comps: b.upcoming(n), starts: map[string][]start{}}

	me := teamOf(b.me)
	for ci, c := range ol.comps {
		f := b.estimateField(c)
		pr, pl := b.lineup(me, c, f, "", b.p.passes())
		ol.fields = append(ol.fields, f)
		sc := make([]float64, len(c.races))
		total := 0.0

		for r, crew := range pl.crew {
			if crew == nil {
				continue
			}

			sc[r] = pl.score[r]

			total += pl.score[r]
			for _, x := range crew {
				st := start{k: ci, race: r}
				if c.team() {
					st.race = -1
				}

				ol.starts[pr.rowers[x].ID] = append(ol.starts[pr.rowers[x].ID], st)
			}
		}

		if c.team() {
			sc = []float64{total}
		}

		ol.scores = append(ol.scores, sc)
	}

	return ol
}

func (o *outlook) cur(st start) float64 {
	if st.race < 0 {
		return o.scores[st.k][0]
	}

	return o.scores[st.k][st.race]
}

func (o *outlook) add(st start, delta float64) {
	if st.race < 0 {
		o.scores[st.k][0] += delta
	} else {
		o.scores[st.k][st.race] += delta
	}
}

// gain values a stat change of one rower: expected medal points over the outlook plus a small
// raw term so that non-starters still develop sensibly.
func (b *brain) gain(ol *outlook, rowerID string, delta engine.Stats) float64 {
	val := 0.0

	for k, c := range ol.comps {
		w := 0.0
		for s, x := range delta {
			w += c.weights[s] * float64(x)
		}

		val += rawEps * lookahead[k] * w
	}

	for _, st := range ol.starts[rowerID] {
		c := ol.comps[st.k]

		w := 0.0
		for s, x := range delta {
			w += c.weights[s] * float64(x)
		}

		if w == 0 {
			continue
		}

		opps := ol.fields[st.k].team
		if st.race >= 0 {
			opps = ol.fields[st.k].race[st.race]
		}

		cur := ol.cur(st)
		val += lookahead[st.k] * (b.expMedals(cur+w, opps) - b.expMedals(cur, opps))
	}

	return val * (1 + b.noise[rowerID])
}

// twinPartners maps both rowers of every unspent Seelenzwilling card to each other: a point
// trained on one of them also lands on the other (engine applyTwinTraining).
func twinPartners(rowers []engine.RowerView) map[string]string {
	out := map[string]string{}

	for _, r := range rowers {
		if e := r.Event; e != nil && e.Type == "seelenzwilling" && !e.Spent && e.PartnerID != "" {
			out[r.ID], out[e.PartnerID] = e.PartnerID, r.ID
		}
	}

	return out
}

func (b *brain) training(points, maxRowers int) []engine.TrainingAlloc {
	ol := b.outlook(3)
	twins := twinPartners(b.me.Rowers)
	alloc := map[[2]string]int{}
	var order [][2]string
	chosen := map[string]bool{}

	for range points {
		bestV := -1e18
		var bestR string
		var bestS engine.Stat

		for _, r := range b.me.Rowers {
			if maxRowers > 0 && len(chosen) >= maxRowers && !chosen[r.ID] {
				continue
			}

			for _, s := range engine.TrainableStats {
				v := b.gain(ol, r.ID, engine.Stats{s: 1})
				if p, ok := twins[r.ID]; ok {
					v += b.gain(ol, p, engine.Stats{s: 1})
				}

				if v > bestV {
					bestV, bestR, bestS = v, r.ID, s
				}
			}
		}

		key := [2]string{bestR, string(bestS)}
		if alloc[key] == 0 {
			order = append(order, key)
		}

		alloc[key]++

		chosen[bestR] = true
		for _, st := range ol.starts[bestR] {
			ol.add(st, ol.comps[st.k].weights[bestS])
		}

		if p, ok := twins[bestR]; ok {
			for _, st := range ol.starts[p] {
				ol.add(st, ol.comps[st.k].weights[bestS])
			}
		}
	}

	out := make([]engine.TrainingAlloc, 0, len(order))
	for _, k := range order {
		out = append(out, engine.TrainingAlloc{RowerID: k[0], Stat: engine.Stat(k[1]), Points: alloc[k]})
	}

	return out
}

func (b *brain) selection(count int) []string {
	sel := b.d.Rules.Selections[b.d.Calendar[b.view.Current].Selection]

	delta := engine.Stats{}
	for s, v := range sel.Effect {
		delta[engine.Stat(s)] = v
	}

	o := b.outlook(3)
	type cand struct {
		id string
		v  float64
	}

	cs := make([]cand, 0, len(b.me.Rowers))
	for _, r := range b.me.Rowers {
		if len(sel.Ages) > 0 && !slices.Contains(sel.Ages, string(r.Age)) {
			continue
		}

		cs = append(cs, cand{r.ID, b.gain(o, r.ID, delta)})
	}

	slices.SortStableFunc(cs, func(a, c cand) int {
		switch {
		case a.v > c.v:
			return -1
		case a.v < c.v:
			return 1
		}

		return 0
	})
	n := min(count, len(cs))

	out := make([]string, n)
	for i := range n {
		out[i] = cs[i].id
	}

	return out
}

// ---------- Tausch ----------

// swap exchanges two stats of the rower if that raises its weighted value over the remaining
// season (nearer competitions count more); otherwise it skips.
func (b *brain) swap(rowerID string) *[2]engine.Stat {
	var rw *engine.RowerView

	for i := range b.me.Rowers {
		if b.me.Rowers[i].ID == rowerID {
			rw = &b.me.Rowers[i]
		}
	}

	if rw == nil {
		return nil
	}

	comps := b.upcoming(len(b.d.Calendar))
	value := func(st engine.Stats) float64 {
		v := 0.0
		for k, c := range comps {
			v += weighted(st, c.weights) / (1 + 0.3*float64(k))
		}

		return v
	}
	base := value(rw.Stats)
	best, bestV := (*[2]engine.Stat)(nil), base+0.5

	for i, s1 := range engine.TrainableStats {
		for _, s2 := range engine.TrainableStats[i+1:] {
			st := engine.Stats{}
			maps.Copy(st, rw.Stats)

			st[s1], st[s2] = st[s2], st[s1]

			v := value(st) * (1 + b.rng.NormFloat64()*b.p.noise()*0.2)
			if v > bestV {
				best, bestV = &[2]engine.Stat{s1, s2}, v
			}
		}
	}

	return best
}

// ---------- draft ----------

// draft picks the card that adds the most expected medal points over the next competitions.
func (b *brain) draft(offer []engine.DraftCard) string {
	comps := b.upcoming(4)

	fields := make([]*field, len(comps))
	for k, c := range comps {
		fields[k] = b.estimateField(c)
	}

	me := teamOf(b.me)
	eval := func(t *team) float64 {
		v := 0.0

		for k, c := range comps {
			pr, pl := b.lineup(t, c, fields[k], "", 0)
			v += lookahead[k] * pr.value(pl)
		}

		return v
	}
	base := eval(me)

	bestID, bestV := offer[0].ID, -1e18
	for _, card := range offer {
		t := &team{id: me.id, rowers: me.rowers, boats: me.boats, trainers: me.trainers}

		switch {
		case card.Rower != nil:
			t.rowers = append(slices.Clone(me.rowers), *card.Rower)
		case card.Boat != nil:
			t.boats = append(slices.Clone(me.boats), *card.Boat)
		case card.Trainer != nil:
			t.trainers = append(slices.Clone(me.trainers), *card.Trainer)
		}

		v := eval(t) - base

		v += b.rng.NormFloat64() * b.p.noise() * 2
		if v > bestV {
			bestID, bestV = card.ID, v
		}
	}

	return bestID
}
