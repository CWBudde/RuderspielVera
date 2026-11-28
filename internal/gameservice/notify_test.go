package gameservice

import "testing"

// TestNotifierCoalesces: notify never blocks; a reader that falls behind gets only the latest
// version, other games stay silent, and nothing arrives after unsubscribe.
func TestNotifierCoalesces(t *testing.T) {
	t.Parallel()

	events := newNotifier()
	ch, unsubscribe := events.subscribe("g1")
	other, unsubscribeOther := events.subscribe("g2")

	defer unsubscribeOther()

	for range 5 {
		events.notify("g1")
	}

	if v := <-ch; v != 5 {
		t.Fatalf("got version %d, want 5", v)
	}

	select {
	case v := <-ch:
		t.Fatalf("unexpected second version %d", v)
	case v := <-other:
		t.Fatalf("other game got version %d", v)
	default:
	}

	unsubscribe()
	unsubscribe() // idempotent
	events.notify("g1")

	select {
	case v := <-ch:
		t.Fatalf("version %d after unsubscribe", v)
	default:
	}

	if len(events.subs["g1"]) != 0 {
		t.Fatal("subscriber not removed")
	}
}
