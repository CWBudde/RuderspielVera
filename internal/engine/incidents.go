package engine

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

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
