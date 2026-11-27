package engine

import "math/rand/v2"

// rollIncident rolls one d100 for a water entry. Incidents are mutually exclusive:
// 1..kentern = Kentern, next material values = Materialschaden, next krebs values = Krebs, else none.
func rollIncident(rng *rand.Rand, kentern, material, krebs int) IncidentKind {
	roll := rollD(rng, 100)
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
