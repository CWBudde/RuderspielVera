# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Ruderspiel

Multiplayer rowing-management game (board game + computer game). Rules draft:
`docs/ruderspiel-regelwerk.md` (German). Invented PoC rules: `docs/poc-rules.md`. Plan: `docs/PLAN.md`.

- Go backend (`cmd/`, `internal/`), React-TS frontend in `web/` (Vite, Tailwind, shadcn/ui).
- `internal/engine` is pure and deterministic: no I/O, no time, no global RNG. All randomness from the
  game's seeded RNG, drawn in a fixed order. State = seed + decision log.
- `internal/engine/contract.go` is the JSON contract with the frontend; keep `docs/api.md` in sync and
  regenerate TS types with `just gen-types`.
- Rule numbers live in `internal/data/*.json`, not in code.
- Player text is German (default) and English: UI text in `web/src/i18n/ui.*.ts`, engine messages in
  `web/src/i18n/messages.*.ts`, data texts in `internal/data/*.json` + `internal/data/i18n/en.json`
  (`just gen-i18n`). The engine emits message keys (`engine.Msg`), never sentences. See `docs/i18n.md`.
- Code, comments, docs and commit messages are English.
- Style references: `examples/` (vera-karte.png is the current card design). Generated art via the
  `gfx-imagegen` skill only; never run a local `codex`.

## Commands

- `just dev` — Go server on :8080 + Vite dev server (Vite proxies `/api` to :8080).
- `just test` (Go, CI runs `go test -race -count=1 ./...`); single test:
  `go test ./internal/engine -run TestName`.
- `just vet`, `just lint` (golangci-lint with `.golangci.toml`, nearly all linters on), `just fmt`
  (treefmt: gofumpt, gci, prettier, shfmt). `just setup-deps` installs the pinned tool versions.
- `cd web && pnpm build` — the only frontend check (tsc + bundle); there are no frontend tests.
- `just sim games=500 seed=1` — headless bot-vs-bot balance simulation (`cmd/sim`).
- `just wasm-smoke` — builds the engine to WebAssembly and plays a full game in Node.
- `just pages` — static GitHub Pages build (wasm transport). `just deploy` — k3s on pcjv.de.
- Generators: `just gen-types` (Go contract → `web/src/api/types.gen.ts` via tygo),
  `just gen-i18n` (→ `web/src/i18n/data.gen.ts`), `just gendata` (→ `internal/data/rowers.json`).
  Never edit `*.gen.ts` by hand.

## Architecture

Layers, each importing only the ones below:

1. `internal/data` — embedded JSON rule data + validation (`data_test.go` checks consistency).
2. `internal/engine` — the rules. `Game.Submit(playerID, Decision)` validates and records a decision;
   when all required players of a phase have submitted, the phase resolves and the game advances
   (`phases.go`: stations → draft, events, training, regatta, …). `Replay(cfg, log)` rebuilds a game
   from config + decision log. Views for clients are built in `view.go` per player (hidden info).
3. `internal/bot` — bot deciders; each bot has its own RNG seeded from game seed + seat, never the
   game's RNG.
4. `internal/gameservice` — transport-independent core: game registry, bearer tokens, lobbies, bot
   loop, turn timeout/janitor, change notification, snapshots, and mapping errors to
   `{status, msg key}`. Must not import `net/http` or touch the file system, because it is shared by:
   - `internal/api` + `cmd/server` — HTTP JSON API, SSE (`/api/games/{id}/events`), rate limiting,
     snapshots persisted to `-data` dir, serves `web/dist`.
   - `cmd/wasm` — the same service compiled to WebAssembly for the static build; snapshots go to
     localStorage.

Persistence is replay-based: a `Snapshot` stores config, tokens and the decision log. Any rule change
that makes old logs replay differently requires bumping `SnapshotVersion` in
`internal/gameservice/snapshot.go` (and noting why in its comment).

Frontend (`web/src`): `api/client.ts` picks the backend — HTTP (default), wasm
(`VITE_TRANSPORT=wasm`, `api/wasmTransport.ts`) or the in-browser mock (`?mock=1`, `api/mock/`).
The mock is a contract-shaped fake, not the rules engine; keep it compiling when the contract changes.
Screens per phase kind live in `screens/phases/`; game state flows through `hooks/game.tsx`
(TanStack Query + SSE via `api/useGameEvents.ts`).
