//go:build js && wasm

// Command wasm runs the Ruderspiel game service (rules engine + bots) inside the browser.
//
// It registers globalThis.ruderspielWasm with synchronous functions that take and return JSON
// strings. Every result is an envelope:
//
//	{"status": 200|201, "data": <payload>}   on success
//	{"status": 400|401|404|409|500, "error": "<German message>"}   on failure
//
// so the frontend can treat it exactly like an HTTP response. Persistence is the caller's job:
// after every change it stores exportGame(gameId) (seed + decision log; data is a JSON *string*,
// because the uint64 seed does not survive JS number parsing) and re-imports it with importGame
// on the next page load (engine.Replay).
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"syscall/js"

	"ruderspiel/internal/engine"
	"ruderspiel/internal/gameservice"
)

var svc *gameservice.Service

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	var err error
	svc, err = gameservice.New(gameservice.Options{Decider: botDecider, Logger: log})
	if err != nil {
		panic(err)
	}

	api := js.Global().Get("Object").New()
	api.Set("createGame", export(1, func(a []js.Value) (int, any, error) {
		var req gameservice.CreateRequest
		if err := decode(a[0], &req); err != nil {
			return 0, nil, err
		}
		resp, err := svc.Create(req)
		return 201, resp, err
	}))
	api.Set("getView", export(2, func(a []js.Value) (int, any, error) {
		v, err := svc.View(a[0].String(), a[1].String())
		return 200, v, err
	}))
	api.Set("submitDecision", export(3, func(a []js.Value) (int, any, error) {
		if err := svc.CheckToken(a[0].String(), a[1].String()); err != nil {
			return 0, nil, err
		}
		var d engine.Decision
		if err := decode(a[2], &d); err != nil {
			return 0, nil, err
		}
		v, err := svc.Submit(a[0].String(), a[1].String(), d)
		return 200, v, err
	}))
	api.Set("getLog", export(2, func(a []js.Value) (int, any, error) {
		lr, err := svc.Log(a[0].String(), a[1].String())
		return 200, lr, err
	}))
	api.Set("getCatalog", export(0, func([]js.Value) (int, any, error) {
		c, err := svc.Catalog()
		return 200, c, err
	}))
	api.Set("exportGame", export(1, func(a []js.Value) (int, any, error) {
		snap, err := svc.Export(a[0].String())
		if err != nil {
			return 0, nil, err
		}
		// data is the snapshot as an opaque JSON string: the seed is a uint64 and would lose
		// precision if JS parsed it as a number. Store it as-is and pass it back to importGame.
		raw, err := json.Marshal(snap)
		return 200, string(raw), err
	}))
	api.Set("importGame", export(1, func(a []js.Value) (int, any, error) {
		var snap gameservice.Snapshot
		if err := decode(a[0], &snap); err != nil {
			return 0, nil, err
		}
		id, err := svc.Import(snap)
		return 200, map[string]string{"gameId": id}, err
	}))
	js.Global().Set("ruderspielWasm", api)

	if ready := js.Global().Get("ruderspielWasmReady"); ready.Type() == js.TypeFunction {
		ready.Invoke()
	}
	select {} // keep the Go runtime alive for callbacks
}

// export wraps fn as a JS function taking string arguments and returning a JSON envelope.
func export(nargs int, fn func([]js.Value) (int, any, error)) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) (result any) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in wasm call", "panic", fmt.Sprint(r))
				result = envelope(0, nil, &gameservice.Error{Status: gameservice.StatusInternal, Msg: "Interner Fehler."})
			}
		}()
		if len(args) < nargs {
			return envelope(0, nil, &gameservice.Error{Status: gameservice.StatusBadRequest, Msg: "Ungültige Anfrage: Parameter fehlen."})
		}
		for _, a := range args[:nargs] {
			if a.Type() != js.TypeString {
				return envelope(0, nil, &gameservice.Error{Status: gameservice.StatusBadRequest, Msg: "Ungültige Anfrage: Parameter müssen Zeichenketten sein."})
			}
		}
		status, data, err := fn(args)
		return envelope(status, data, err)
	})
}

func decode(v js.Value, out any) error {
	if err := json.Unmarshal([]byte(v.String()), out); err != nil {
		return &gameservice.Error{Status: gameservice.StatusBadRequest, Msg: "Ungültige Anfrage: Die Daten konnten nicht gelesen werden."}
	}
	return nil
}

func envelope(status int, data any, err error) string {
	var v any
	if err != nil {
		se := gameservice.AsError(err)
		v = struct {
			Status int    `json:"status"`
			Error  string `json:"error"`
		}{se.Status, se.Msg}
	} else {
		v = struct {
			Status int `json:"status"`
			Data   any `json:"data"`
		}{status, data}
	}
	b, mErr := json.Marshal(v)
	if mErr != nil {
		return `{"status":500,"error":"Interner Fehler."}`
	}
	return string(b)
}
