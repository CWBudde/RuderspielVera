// Command gendata generates internal/data/rowers.json deterministically.
//
// Every rower gets a rank (the SUM of the five starting stats, Regelwerk §5.3): A-Junioren 100…86
// (Michel = 100), B-Junioren 85…59, Kinder 58…46. The sum is split by one of five archetypes with
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
	age      string
	names    []string
	top, low int
}

func generate(seed uint64) ([]rower, error) {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	groups := []group{
		{"a", aJunioren, 100, 86},
		{"b", bJunioren, 85, 59},
		{"kinder", kinder, 58, 46},
	}
	var all []rower

	for _, grp := range groups {
		if grp.top-grp.low+1 != len(grp.names) {
			return nil, fmt.Errorf("group %s: %d names for ranks %d..%d", grp.age, len(grp.names), grp.top, grp.low)
		}

		rankOf := shuffledRanks(rng, grp)
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

// shuffledRanks assigns the ranks top..low to the group's names in random order. Michel is fixed
// at the top rank; everyone else gets a shuffled rank.
func shuffledRanks(rng *rand.Rand, grp group) []int {
	ranks := make([]int, 0, len(grp.names))
	for r := grp.top; r >= grp.low; r-- {
		ranks = append(ranks, r)
	}

	order := make([]int, len(grp.names))
	for i := range order {
		order[i] = i
	}

	rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

	if grp.age == "a" {
		mi := indexOf(grp.names, "Michel")
		for k, idx := range order {
			if idx == mi {
				order[0], order[k] = order[k], order[0]
				break
			}
		}
	}

	rankOf := make([]int, len(grp.names))
	for k, idx := range order {
		rankOf[idx] = ranks[k]
	}

	return rankOf
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

func indexOf(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}

	return -1
}
