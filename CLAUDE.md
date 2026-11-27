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
- Tests: `just test` (Go), `cd web && pnpm build` (types + bundle).
