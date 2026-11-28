# Ruderspiel

A multiplayer rowing-management game, designed as both a board game and a computer game. Each player
runs a rowing club through a season: draft rowers and boats, train crews, handle events and race at
regattas. 2–4 players per game; open seats can be filled by bots.

- Live server: <https://ruderspiel.pcjv.de>
- Static build (engine runs in the browser as WebAssembly): GitHub Pages, deployed from `main`

The game text is German by default, with an English translation.

## Documentation

- [docs/ruderspiel-regelwerk.md](docs/ruderspiel-regelwerk.md) — rules draft (German)
- [docs/poc-rules.md](docs/poc-rules.md) — the rules the proof of concept implements
- [docs/api.md](docs/api.md) — HTTP/JSON API and the engine contract
- [docs/i18n.md](docs/i18n.md) — how translations work
- [docs/PLAN.md](docs/PLAN.md) — roadmap

## Stack

- **Backend:** Go (`cmd/`, `internal/`). The rules engine (`internal/engine`) is pure and
  deterministic: a game is fully defined by its seed and its decision log. Rule numbers live in
  `internal/data/*.json`.
- **Frontend:** React + TypeScript in `web/` (Vite, Tailwind, shadcn/ui).
- **Transports:** HTTP server (`cmd/server`, with Server-Sent Events for live updates) or the same
  game service compiled to WebAssembly (`cmd/wasm`) for a serverless static build.

## Development

Requirements: Go (see `go.mod`), Node.js with pnpm, and [just](https://github.com/casey/just).

```sh
cd web && pnpm install && cd ..
just dev            # Go server on :8080 + Vite dev server (proxies /api)
```

Append `?mock=1` to the dev URL to use an in-browser fake backend (UI work without the server).

| Command                  | Purpose                                                        |
| ------------------------ | -------------------------------------------------------------- |
| `just test`              | Go tests                                                       |
| `just lint` / `just fmt` | golangci-lint / treefmt (install tools with `just setup-deps`) |
| `cd web && pnpm build`   | Type-check and bundle the frontend                             |
| `just sim`               | Simulate bot-only games for balancing                          |
| `just gen-types`         | Regenerate TS types from the Go contract                       |
| `just gen-i18n`          | Regenerate translated data texts for the UI                    |
| `just build`             | Build the SPA and the server binary (`bin/ruderspiel`)         |
| `just pages`             | Static WebAssembly build into `web/dist`                       |
| `just deploy`            | Build the Docker image and roll it out to k3s                  |

Run the server binary with `-h` for its flags (listen address, data directory, turn timeout, rate
limits).
