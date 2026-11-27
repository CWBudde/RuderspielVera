# API contract (PoC)

Source of truth for all JSON shapes: `internal/engine/contract.go`. TS types are generated from it
into `web/src/api/types.gen.ts` (`just gen-types`). All UI-facing strings in views are German.

## Endpoints

| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/api/games` | `{"playerName": "Vera", "seed": 123 (optional), "bots": 2}` | `{"gameId", "playerId", "token", "view": GameView}` |
| GET | `/api/games/{id}` | header `Authorization: Bearer <token>` | `GameView` for the token's player |
| POST | `/api/games/{id}/decisions` | `Decision` (bearer token) | `GameView` after bots acted and stations resolved as far as possible |
| GET | `/api/games/{id}/log` | bearer token | `{"log": LogEntry[], "results": CompetitionResult[]}` |
| GET | `/api/catalog` | – | `{"rowers": RowerView[], "boats": BoatView[], "trainers": TrainerView[], "events": [{type,name,description}]}` |

Errors: HTTP 4xx with `{"error": "<German message for the UI>"}`. A decision with a stale
`phaseId` returns 409.

## Phase model

The game is a sequence of calendar stations. Each station opens one or more phases. A phase needs a
decision from every player (bots decide instantly server-side). When all are in, the engine
resolves and opens the next phase.

- Station types map to phases: `setup` → ack; `training`/`camp` → training; `selection` → selection;
  `event` → ack (reveal; plus `eventswap` for a player who got a Tausch card); `ereignis` → ack
  (reveal); `regatta`/`ergocup`/`jtfo` → regatta, then an ack phase showing `lastResult`, then a
  `draft` phase.
- **Draft is sequential**: the phase lists `draftOrder`; only the player on turn decides, others
  see `waiting: true`. Bots ahead of a human pick immediately, so the human sees the reduced offer.
- After the final RVB-Endspurt result: `gameover` with `winners`.
- Hidden info: other players' regatta decisions are never in a view before resolution.
