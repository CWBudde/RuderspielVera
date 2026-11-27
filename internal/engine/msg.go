package engine

import (
	"fmt"
	"sort"
	"strings"
)

// NewMsg builds a Msg from a key and name/value pairs: NewMsg("log.x", "name", "Mia", "n", 3).
func NewMsg(key string, kv ...any) Msg {
	m := Msg{Key: key}
	if len(kv) == 0 {
		return m
	}

	m.Args = make(map[string]any, len(kv)/2)

	for i := 0; i+1 < len(kv); i += 2 {
		name, ok := kv[i].(string)
		if !ok {
			panic(fmt.Sprintf("engine.NewMsg(%q): arg name %v is not a string", key, kv[i]))
		}

		m.Args[name] = kv[i+1]
	}

	return m
}

func msg(key string, kv ...any) Msg { return NewMsg(key, kv...) }

// String renders the key with its args in a stable order (server logs, tests, cmd/sim).
func (m Msg) String() string {
	if len(m.Args) == 0 {
		return m.Key
	}

	names := make([]string, 0, len(m.Args))
	for k := range m.Args {
		names = append(names, k)
	}

	sort.Strings(names)

	parts := make([]string, len(names))
	for i, k := range names {
		parts[i] = fmt.Sprintf("%s=%v", k, m.Args[k])
	}

	return m.Key + "{" + strings.Join(parts, " ") + "}"
}

// dataMsg references a text of internal/data: dataMsg("event", "phoenix", "name") is the key
// "data.event.phoenix.name" (generated into web/src/i18n/data.gen.ts by `just gen-i18n`).
func dataMsg(parts ...string) Msg { return Msg{Key: "data." + strings.Join(parts, ".")} }

func statMsg(s Stat) Msg       { return Msg{Key: "stat." + string(s)} }
func weatherMsg(w Weather) Msg { return Msg{Key: "weather." + string(w)} }

// rowerName is a rower's name as a Msg arg.
func rowerName(r *rower) any { return r.name }

// signed formats a number with an explicit sign ("+2", "-3").
func signed(v int) string { return fmt.Sprintf("%+d", v) }

// names turns strings into a list arg.
func list[T any](items []T) []any {
	out := make([]any, len(items))
	for i, it := range items {
		out[i] = it
	}

	return out
}

// text is the mission text with the card holder's name.
func (ms *missionState) text() Msg {
	m := dataMsg("mission", ms.def.ID, "text")
	m.Args = map[string]any{"name": ms.rower}

	return m
}

func (ms *missionState) reward() Msg { return dataMsg("mission", ms.def.ID, "reward") }

func raceName(r *RaceSpec) Msg { return Msg{Key: r.NameKey} }
