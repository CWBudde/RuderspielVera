package engine

import "math/rand/v2"

// newRNG returns the game's only source of randomness: a seeded PCG. All draws happen during
// resolution in a fixed order, so seed + decision log reproduce a game exactly.
func newRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0xda3e39cb94b95bdb))
}

func shuffle[T any](rng *rand.Rand, xs []T) {
	rng.Shuffle(len(xs), func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })
}

// rollD returns a value in 1..n.
func rollD(rng *rand.Rand, n int) int { return rng.IntN(n) + 1 }

// pickWeighted returns an index into weights chosen proportionally; -1 if all weights are <= 0.
func pickWeighted(rng *rand.Rand, weights []int) int {
	total := 0

	for _, w := range weights {
		if w > 0 {
			total += w
		}
	}

	if total == 0 {
		return -1
	}

	x := rng.IntN(total)

	for i, w := range weights {
		if w <= 0 {
			continue
		}

		if x < w {
			return i
		}

		x -= w
	}

	return -1
}
