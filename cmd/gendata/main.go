// Command gendata generates internal/data/rowers.json deterministically.
//
// Every rower gets a rank (the SUM of the five starting stats, Regelwerk §5.3) from the handwritten
// list in docs/ruderspiel-basiswerte.md: A-Junioren 84…100 (Michel = 100), B-Junioren 58…93,
// Kinder 46…59; every value from 46 to 100 occurs exactly once. The sum is split by one of five archetypes with
// seeded jitter; every stat is at least 5. The output is committed and may be hand-edited afterwards
// (the loader validates that the stats still add up to the rank).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"sort"
	"strings"
)

// stat order used for archetype weight vectors and output.
var statOrder = []string{"kraft", "ausdauer", "geschwindigkeit", "technik", "motivation"}

type archetype struct {
	Name    string
	Weights [5]float64 // kraft, ausdauer, geschwindigkeit, technik, motivation
}

var archetypes = []archetype{
	{"Kraftpaket", [5]float64{0.30, 0.20, 0.18, 0.16, 0.16}},
	{"Ausdauerläufer", [5]float64{0.18, 0.30, 0.16, 0.18, 0.18}},
	{"Sprinter", [5]float64{0.20, 0.15, 0.30, 0.19, 0.16}},
	{"Techniker", [5]float64{0.16, 0.18, 0.19, 0.30, 0.17}},
	{"Kämpferherz", [5]float64{0.19, 0.19, 0.17, 0.17, 0.28}},
}

// Names exactly as in Regelwerk §14.1–14.3, in the order listed there.
var (
	kinder = []string{
		"Pauline", "Mara", "Nella", "Johanna", "Hannah", "Paul", "Johann", "Justus", "Joel",
		"Ella", "Yasmin", "Malte", "Sophia",
	}
	bJunioren = []string{
		"Vera", "Lotti", "Ida", "Lea", "Anton", "Mailo", "Luka", "Jonas", "Nilo", "Julius",
		"Sophia", "Noah", "Elin", "Arun", "Leni", "Catha", "Hannah", "Tobias", "Jacob", "Christoph", "Justus",
		"Johann", "Jan", "Fabian", "Lenn", "Yukari", "Mia",
	}
	aJunioren = []string{
		"Yara", "Matilda", "Lea", "Annabelle", "Emilie", "Jonathan", "Maxi", "Fabi", "Niclas",
		"Michel", "Jonte", "Simon", "Tara", "Johanna", "Matilda",
	}
)

// Ranks (stat sums) per name, in the same order as the name lists above
// (docs/ruderspiel-basiswerte.md).
var (
	kinderRanks    = []int{54, 52, 48, 47, 46, 59, 57, 50, 53, 56, 49, 55, 51}
	bJuniorenRanks = []int{
		64, 68, 70, 62, 79, 74, 76, 72, 81, 71, 60, 78, 61, 66, 65, 69, 73, 75, 83, 87, 93,
		80, 82, 77, 67, 58, 63,
	}
	aJuniorenRanks = []int{91, 92, 86, 88, 84, 97, 96, 98, 99, 100, 95, 94, 85, 89, 90}
)

// genders maps every first name to "w" (Mädchen) or "m" (Jungen). Names marked guessed in
// docs/poc-rules.md must be confirmed by the family.
var genders = map[string]string{
	"Pauline": "w", "Mara": "w", "Nella": "w", "Johanna": "w", "Hannah": "w", "Paul": "m", "Johann": "m",
	"Justus": "m", "Joel": "m", "Ella": "w", "Yasmin": "w", "Malte": "m", "Sophia": "w",
	"Vera": "w", "Lotti": "w", "Ida": "w", "Lea": "w", "Anton": "m", "Mailo": "m", "Luka": "m", "Jonas": "m",
	"Nilo": "m", "Julius": "m", "Noah": "m", "Elin": "w", "Arun": "m", "Leni": "w", "Catha": "w",
	"Tobias": "m", "Jacob": "m", "Christoph": "m", "Jan": "m", "Fabian": "m", "Lenn": "m", "Yukari": "w",
	"Mia":  "w",
	"Yara": "w", "Matilda": "w", "Annabelle": "w", "Emilie": "w", "Jonathan": "m", "Maxi": "m", "Fabi": "m",
	"Niclas": "m", "Michel": "m", "Jonte": "m", "Simon": "m", "Tara": "w",
}

type rower struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Age       string         `json:"age"`
	Gender    string         `json:"gender"`
	Archetype string         `json:"archetype"`
	Rank      int            `json:"rank"`
	Stats     map[string]int `json:"stats"`
	ArtKey    string         `json:"artKey"`
}

func main() {
	out := flag.String("out", "internal/data/rowers.json", "output file")
	seed := flag.Uint64("seed", 2026, "generator seed")

	flag.Parse()

	rowers, err := generate(*seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var buf strings.Builder
	buf.WriteString("[\n")

	for i, r := range rowers {
		line, err := json.Marshal(r)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		buf.WriteString("  ")
		buf.Write(line)

		if i < len(rowers)-1 {
			buf.WriteString(",")
		}

		buf.WriteString("\n")
	}

	buf.WriteString("]\n")

	err = os.WriteFile(*out, []byte(buf.String()), 0o644) //nolint:gosec // checked-in data file
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("wrote %d rowers to %s\n", len(rowers), *out)
}

type group struct {
	age   string
	names []string
	ranks []int
}

func generate(seed uint64) ([]rower, error) {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	groups := []group{
		{"a", aJunioren, aJuniorenRanks},
		{"b", bJunioren, bJuniorenRanks},
		{"kinder", kinder, kinderRanks},
	}
	var all []rower

	for _, grp := range groups {
		if len(grp.ranks) != len(grp.names) {
			return nil, fmt.Errorf("group %s: %d names but %d ranks", grp.age, len(grp.names), len(grp.ranks))
		}

		// The ranks used to be shuffled; the draw is kept so every rower keeps its archetype and
		// jitter from before the fixed list.
		skipShuffle(rng, len(grp.names))

		rankOf := grp.ranks
		seen := map[string]int{}

		for idx, name := range grp.names {
			gender, ok := genders[name]
			if !ok {
				return nil, fmt.Errorf("no gender for %q", name)
			}

			slug := slugify(name)
			seen[slug]++
			id := slug + "-" + grp.age
			display := name

			if seen[slug] > 1 {
				id = fmt.Sprintf("%s-%d", id, seen[slug])
				display = fmt.Sprintf("%s (%d)", name, seen[slug])
			}

			arch := archetypes[rng.IntN(len(archetypes))]
			stats := split(rng, rankOf[idx], arch)
			all = append(all, rower{
				ID: id, Name: display, Age: grp.age, Gender: gender, Archetype: arch.Name,
				Rank: rankOf[idx], Stats: stats, ArtKey: "rower-" + grp.age + "-" + gender,
			})
		}
	}

	sort.SliceStable(all, func(i, j int) bool { return all[i].Rank > all[j].Rank })

	return all, nil
}

// skipShuffle consumes the random draws of the former rank shuffle of n names.
func skipShuffle(rng *rand.Rand, n int) {
	rng.Shuffle(n, func(int, int) {})
}

// split distributes total over the five stats by the archetype weights with ±15 % jitter per stat,
// every stat >= minStat, exact sum.
func split(rng *rand.Rand, total int, a archetype) map[string]int {
	const minStat = 5
	var w [5]float64
	sum := 0.0

	for i := range w {
		w[i] = a.Weights[i] * (1 + (rng.Float64()*0.3 - 0.15))
		sum += w[i]
	}

	vals := make([]int, 5)
	fracs := make([]float64, 5)
	assigned := 0

	for i := range w {
		exact := float64(total) * w[i] / sum
		vals[i] = int(math.Floor(exact))

		fracs[i] = exact - float64(vals[i])
		if vals[i] < minStat {
			vals[i] = minStat
			fracs[i] = 0
		}

		assigned += vals[i]
	}

	for assigned < total { // hand out the remainder by largest fraction
		best := 0
		for i := 1; i < 5; i++ {
			if fracs[i] > fracs[best] {
				best = i
			}
		}

		vals[best]++
		fracs[best] = -1
		assigned++
	}

	for assigned > total { // raising to the minimum overshot: take from the largest stat
		best := 0
		for i := 1; i < 5; i++ {
			if vals[i] > vals[best] {
				best = i
			}
		}

		vals[best]--
		assigned--
	}

	out := map[string]int{}
	for i, s := range statOrder {
		out[s] = vals[i]
	}

	return out
}

func slugify(s string) string {
	r := strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", " ", "-")
	return r.Replace(strings.ToLower(s))
}
