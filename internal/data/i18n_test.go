package data

import (
	"bytes"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var (
	numberRe      = regexp.MustCompile(`\d+(?:[.,]\d+)?`)
	placeholderRe = regexp.MustCompile(`\{\w+\}`)
)

// TestDataTranslations checks i18n/en.json against the German data texts: every text is
// translated, no translation is stale, and numbers and {placeholders} match (a rule value changed
// in German only shows up here).
func TestDataTranslations(t *testing.T) {
	t.Parallel()

	d := MustLoad()

	en, err := Translations("en")
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}

	for _, e := range d.TextEntries() {
		if seen[e.Key] {
			t.Errorf("duplicate text key %s", e.Key)
		}

		seen[e.Key] = true

		tr, ok := en[e.Key]
		if !ok {
			t.Errorf("en.json: missing %q (German: %q)", e.Key, e.Text)
			continue
		}

		if got, want := numbers(tr), numbers(e.Text); !slices.Equal(got, want) {
			t.Errorf("en.json %s: numbers %v, German has %v (%q)", e.Key, got, want, e.Text)
		}

		if got, want := sorted(placeholderRe.FindAllString(tr, -1)), sorted(placeholderRe.FindAllString(e.Text, -1)); !slices.Equal(got, want) {
			t.Errorf("en.json %s: placeholders %v, German has %v", e.Key, got, want)
		}
	}

	for k := range en {
		if !seen[k] {
			t.Errorf("en.json: stale key %q (no German text)", k)
		}
	}
}

// TestDataGenUpToDate fails when web/src/i18n/data.gen.ts was not regenerated (`just gen-i18n`).
func TestDataGenUpToDate(t *testing.T) {
	t.Parallel()

	want, err := MustLoad().GenerateTS()
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile("../../web/src/i18n/data.gen.ts")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got, want) {
		t.Error("web/src/i18n/data.gen.ts is stale: run `just gen-i18n`")
	}
}

func numbers(s string) []string {
	ns := numberRe.FindAllString(s, -1)
	for i, n := range ns {
		ns[i] = strings.ReplaceAll(n, ",", ".")
	}

	return sorted(ns)
}

func sorted(s []string) []string {
	slices.Sort(s)
	return s
}
