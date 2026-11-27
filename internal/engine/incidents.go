package engine

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"

	"ruderspiel/internal/data"
)

// krebsChance is one crew's Krebs probability (‰): the base (rules or Ereignis) scaled up for
// bigger boats and by the reference Technik (TechnikPerSeat × seats) over the crew's Technik sum,
// clamped to [Min, Max].
func krebsChance(base, seats, technikSum int, ks data.KrebsScaling) int {
	ref := float64(ks.TechnikPerSeat * seats)
	v := float64(base) * float64(ks.SeatPercent[seats]) / 100 * ref / float64(max(technikSum, 1))

	return min(max(int(math.Round(v)), ks.Min), ks.Max)
}

// rollIncident rolls one d1000 for a water entry. Probabilities are in per mille and the incidents
// are mutually exclusive: 1..kentern = Kentern, the next material values = Materialschaden, the
// next krebs values = Krebs, else none.
func rollIncident(rng *rand.Rand, kentern, material, krebs int) IncidentKind {
	roll := rollD(rng, 1000)
	switch {
	case roll <= kentern:
		return IncidentKentern
	case roll <= kentern+material:
		return IncidentMaterial
	case roll <= kentern+material+krebs:
		return IncidentKrebs
	}

	return IncidentNone
}

// perMilleText formats a per-mille probability as a German percentage without the sign: 300 → "30",
// 5 → "0,5".
func perMilleText(v int) string {
	if v%10 == 0 {
		return strconv.Itoa(v / 10)
	}

	return strings.Replace(fmt.Sprintf("%.1f", float64(v)/10), ".", ",", 1)
}
