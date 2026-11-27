package gameservice

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"ruderspiel/internal/engine"
)

// TestStuckResultNullEntries replays a real exported WASM game that got stuck on the result screen
// of its first regatta: races nobody entered were serialized as "entries": null and crashed the
// client. The imported game must show the result without any JSON null and keep playing.
func TestStuckResultNullEntries(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/stuck-result-null-entries.json")
	if err != nil {
		t.Fatal(err)
	}

	var snap Snapshot

	err = json.Unmarshal(raw, &snap)
	if err != nil {
		t.Fatal(err)
	}

	s := newService(t, nil)

	id, err := s.Import(snap)
	if err != nil {
		t.Fatal(err)
	}
	var token string

	for tok, pid := range snap.Tokens {
		if pid == "p1" {
			token = tok
		}
	}

	view, err := s.View(id, token)
	if err != nil {
		t.Fatal(err)
	}

	if view.Phase.ID != "2-ack-3" || view.Phase.Stage != engine.StageResult || view.LastResult == nil {
		t.Fatalf("phase %s stage %q, lastResult %v", view.Phase.ID, view.Phase.Stage, view.LastResult != nil)
	}

	empty := 0

	for _, r := range view.LastResult.Races {
		if len(r.Entries) == 0 {
			empty++
		}
	}

	if empty == 0 {
		t.Fatal("expected races without entries in this game")
	}

	js, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Contains(js, []byte("null")) {
		t.Fatalf("view JSON contains null: %s", js)
	}

	_, err = s.Submit(id, token, engine.Decision{PhaseID: view.Phase.ID, Kind: engine.PhaseAck})
	if err != nil {
		t.Fatalf("continue after result: %v", err)
	}
}
