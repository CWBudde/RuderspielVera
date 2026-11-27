package data

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	d, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(d.Rowers) != 55 {
		t.Errorf("rowers = %d, want 55", len(d.Rowers))
	}

	if len(d.Calendar) != 52 {
		t.Errorf("stations = %d, want 52", len(d.Calendar))
	}

	if len(d.Trainers) != 10 {
		t.Errorf("trainers = %d, want 10", len(d.Trainers))
	}

	if len(d.Events) != 12 {
		t.Errorf("event types = %d, want 12", len(d.Events))
	}
}

func TestRowers(t *testing.T) {
	t.Parallel()

	d := MustLoad()
	ids := map[string]bool{}
	perAge := map[string]int{}

	for _, rw := range d.Rowers {
		if ids[rw.ID] {
			t.Errorf("duplicate id %s", rw.ID)
		}

		ids[rw.ID] = true
		perAge[rw.Age]++
		sum := 0

		for _, s := range StatNames {
			if rw.Stats[s] < 5 {
				t.Errorf("%s: %s = %d < 5", rw.ID, s, rw.Stats[s])
			}

			sum += rw.Stats[s]
		}

		if sum != rw.Rank {
			t.Errorf("%s: sum %d != rank %d", rw.ID, sum, rw.Rank)
		}

		if want := "rower-" + rw.Age + "-" + rw.Gender; rw.ArtKey != want {
			t.Errorf("%s: artKey %q, want %q", rw.ID, rw.ArtKey, want)
		}
	}

	if perAge["kinder"] != 13 || perAge["b"] != 27 || perAge["a"] != 15 {
		t.Errorf("age split = %v, want kinder 13, b 27, a 15", perAge)
	}

	for _, id := range []string{"michel-a", "matilda-a", "matilda-a-2", "lea-b", "lea-a", "johanna-kinder", "johanna-a"} {
		if !ids[id] {
			t.Errorf("missing id %s", id)
		}
	}

	if d.Rowers[0].ID != "michel-a" || d.Rowers[0].Rank != 100 {
		t.Errorf("Michel must be rank 100, top is %s (%d)", d.Rowers[0].ID, d.Rowers[0].Rank)
	}
}

func TestCompetitionStations(t *testing.T) {
	t.Parallel()

	d := MustLoad()
	for _, s := range d.Calendar {
		switch s.Type {
		case "regatta", "ergocup", "jtfo":
			c := d.Competitions[s.Competition]
			if c == nil {
				t.Fatalf("station %s has no competition", s.ID)
			}

			for _, n := range []int{2, 3, 4} {
				if len(d.RacesFor(c, n)) == 0 {
					t.Errorf("%s: no races for %d players", s.ID, n)
				}
			}
		}
	}
	// Every race fillable is covered by Validate; check it runs on a broken copy too.
	c := *d.Competitions["bremen"]
	r := c.Races[0]
	r.Ages = []string{"kinder"}
	r.Genders = []string{"m"}
	r.Class, r.Seats = "8x", 8

	err := d.checkRace(&c, r)
	if err == nil || !strings.Contains(err.Error(), "not fillable") {
		t.Errorf("expected not fillable, got %v", err)
	}
}

func TestStationTypes(t *testing.T) {
	t.Parallel()

	d := MustLoad()
	want := map[string]string{
		"s5-hankensbuettel": "camp", "s6-kassel-camp": "camp", "s3-wr5": "selection",
		"s5-basketball": "selection", "s8-wanderfahrt": "selection", "s2-mindor": "ergocup",
		"s3-drc": "ergocup", "s4-osnabrueck": "ergocup", "s10-jtfo": "jtfo", "s1-hildesheim": "regatta",
		"s11-rvh": "regatta", "s12-rvb": "regatta",
	}
	got := map[string]string{}
	count := map[string]int{}

	for _, s := range d.Calendar {
		got[s.ID] = s.Type
		count[s.Type]++
	}

	for id, typ := range want {
		if got[id] != typ {
			t.Errorf("%s: type %q, want %q", id, got[id], typ)
		}
	}

	if count["event"] != 11 || count["ereignis"] != 11 || count["training"] != 12 {
		t.Errorf("counts = %v", count)
	}
}
