# Ruderspiel – Implementation Plan

## Context

`/mnt/projekte/Code/RuderspielVera` holds only the rules draft `ruderspiel-regelwerk.md` (German, many
points still **Offen**) and two style examples: `Kartendesign fürs Ruderspiel.png` (navy/white card
frames, watercolour-ink portraits, stat rows with icons) and `Maschsee Hannover_ Das große
Regatta-Spielbrett.png` (illustrated board around the Maschsee, pastel round stations by type). It is
not yet a git repo.

Goal: a responsive (smartphone-first) multiplayer rowing-management game with a Go backend and a
React-TS frontend. The first PoC is playable as **one human vs. two computer opponents**. Decision taken:
the **rules engine lives in Go from day one** behind a small HTTP API, so multiplayer later adds lobby
and realtime on top instead of a rewrite. Missing rules are invented for the PoC, collected in one
document and kept data-driven so the family can change them later.

Language: UI text German (the game, cards and rules are German); code, comments, docs and commits
English (user's global rule).

---

## Architecture

```
RuderspielVera/
  go.mod                      module ruderspiel
  cmd/server/                 HTTP server; serves /api and the built web/dist (embed)
  cmd/gendata/                one-shot generator for rower base stats -> internal/data/rowers.json
  cmd/sim/                    headless season simulator (bots only) for balancing
  internal/data/              embedded JSON: rowers, boats, trainers, events, ereignisse,
                              calendar, competitions, rules (tunable numbers)
  internal/engine/            pure, deterministic rules engine (no I/O)
    state.go rng.go calendar.go phases.go training.go events.go ereignisse.go
    regatta.go scoring.go incidents.go draft.go view.go decisions.go
  internal/bot/               computer opponents (decision functions over a PlayerView)
  internal/api/               REST handlers, game store (in-memory + JSON snapshot)
  web/                        Vite + React + TS + Tailwind + shadcn/ui
  art/                        style.md (prompt preamble), generated images, prompts
  .claude/skills/gfx-imagegen adapted copy of the Historyline skill
  docs/PLAN.md                this plan (copied in at start)
  docs/poc-rules.md           every invented rule, marked "PoC-Annahme"
  docs/api.md                 API + view/decision contract
  justfile                    dev, build, test, sim, gen-types
```

Key design choices:

- **Simultaneous-decision state machine.** Every calendar station opens a _phase_ that needs one
  decision per player (training allocation, 5-rower selection, regatta entries, draft pick, or just
  "weiter"). When all decisions are in, the engine resolves the station and advances. Bots submit
  instantly. This is exactly the multiplayer model, so the PoC already exercises it.
- **Determinism / event sourcing.** State = `seed + ordered decision log`. A seeded PCG RNG
  (`math/rand/v2`) is drawn in a fixed order at resolution. Replays rebuild any game; this gives
  persistence, undo-free debugging and later server-authoritative multiplayer for free.
- **Hidden information.** `engine.ViewFor(state, playerID)` filters: opponents' lineups and trainer
  choice stay hidden until resolution; own cards fully visible; opponents' rosters visible (cards are
  public on the table).
- **Data-driven rules.** All numbers (weights, probabilities, event magnitudes, deal sizes) sit in
  `internal/data/*.json` so the open points in the Regelwerk become config edits.
- **Shared types.** Go structs for views/decisions are the source of truth; TS types generated with
  `tygo` (`just gen-types` → `web/src/api/types.gen.ts`).

### API (PoC, documented in docs/api.md)

| Method | Path                        | Purpose                                                             |
| ------ | --------------------------- | ------------------------------------------------------------------- |
| POST   | `/api/games`                | `{seed?, playerName, bots: 2}` → `{gameId, playerId, token}`        |
| GET    | `/api/games/{id}?player=`   | `GameView` for that player (bearer token)                           |
| POST   | `/api/games/{id}/decisions` | `{phaseId, decision}` → new `GameView` (bots act, station resolves) |
| GET    | `/api/games/{id}/log`       | full resolution log (races, incidents, events)                      |
| GET    | `/api/catalog`              | static card data (names, art keys)                                  |

Store: in-memory map with JSON snapshot per game under `data/games/` (seed + log), reloaded on start.
Go stdlib `net/http` ServeMux with method patterns; no framework.

---

## PoC rule decisions (to write into docs/poc-rules.md)

All marked as PoC assumptions; numbers live in `internal/data/rules.json`.

**Calendar.** Exactly the 12 sections of Regelwerk §2.2 (52 stations). Hankensbüttel and Kassel =
_Trainingslager_ (30 points, spread over at most 5 rowers, free split). Wanderfahrt = choose 5 rowers:
+4 Ausdauer, +3 Motivation each. RVH-Herbstsprint and RVB-Endspurt are separate regattas; game ends
after RVB.

**Rowers.** 55 cards from §14. Ranking: A-Junioren 100…86 (Michel 100), B-Junioren 85…59, Kinder 58…46
(sum of the five stats). `cmd/gendata` splits each sum by one of five archetypes (Kraftpaket,
Ausdauerläufer, Sprinter, Techniker, Kämpferherz) with seeded jitter, every stat ≥ 5; output committed
as JSON and hand-editable. Gender per name from a table in the generator (unknown/ambiguous names
flagged in docs for the family to confirm). Duplicates (Matilda ×2, Johann, Justus, Sophia, Hannah, Lea,
Johanna) get unique IDs and an age suffix in the UI.

**Deal (3 players).** 18 rowers each: 4 Kinder, 9 B-, 5 A-Junioren (2 players: 6/13/7, 4 players:
3/4/3; `rules.json` `deal.rowersByPlayers`), at least 1 girl and 1 boy per age group, random. The
remaining rower(s) form the pool. Each player: city 1x/2x/4x (value 3) and 2
random trainers. Remaining trainers (4) and the 22 named boats are distributed via **draft**: after
every scored competition each player picks 1 card from an open offer of (players+1) cards (mix of
rowers, boats, and trainers from section 5 on); pick order = fewest medal points first (catch-up). Boat
values: first name group 5–7, outlined second group 8–10.

**Stats scale.** Rower stats ~5–35 each; regular training +10 points per player per training; WR5 and
Basketball +10 Motivation to 5 chosen rowers; boats 3–10; trainer effects ±3…±6; Krebs −10. So one
training ≈ a 10–20 % boost to one rower in one discipline.

**Race scoring.** `rowerScore = Σ weight_s · stat_s`; `crew = Σ rowerScore`; `race = crew + boat +
trainerEffect + eventEffects − crabPenalty`. Rounded to integers → ties are real and kept. Medals:
competition ranking (1,1,3): two golds → next is 3rd (no silver). One medal **per race entry** (not per
crew member). Gold or silver: +1 temporary Motivation (Siegesmotivation) to **every crew member** for
the next 3 competitions, no stacking (docs/poc-rules.md 4.1). Season tie: shared win.

**Competition weights.**

| Competition            | Weights                                        | Races (each player max 1 entry per race)                                                                                                                           |
| ---------------------- | ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Hildesheim (long)      | Ausdauer 2, Motivation 1                       | 1x and 2x for A and B, girls/boys → 8 races                                                                                                                        |
| Ergo-Cups ×3           | Kraft 2, Motivation 1                          | single, 3 age groups × 2 genders → 6 races; no boat, no incidents, no weather                                                                                      |
| Bremen, Schillerschule | Geschwindigkeit 1, Technik 1, Motivation 0.5   | 4 races per regatta, chosen from the 18-class matrix (1x/2x/4x × age × gender), fixed list in competitions.json, rotating so every class appears during the season |
| Kasselregatta          | Geschwindigkeit 1, Kraft 1, Motivation 0.5     | 4 races                                                                                                                                                            |
| Verbandsregatta I/II   | Technik 1.5, Geschwindigkeit 1, Motivation 0.5 | 4 races each                                                                                                                                                       |
| JTFO                   | Geschwindigkeit 1, Technik 1                   | team competition: 2 players → var. C, 3 → var. B (2x+4x+2x), 4 → var. A; team score = sum of the 3 boats, one medal for the team                                   |
| RVH-Herbstsprint       | Geschwindigkeit 2, Kraft 1                     | 1x/2x/4x per gender, ages mixed → 6 races                                                                                                                          |
| RVB-Endspurt           | Geschwindigkeit 2, Technik 1, Motivation 0.5   | Mixed-4x (≥1 girl, ≥1 boy, ≥2 age groups) and Mixed-8x (same rule, shared identical eight, boat value 0)                                                           |

A rower may start in **at most one race per regatta** (forces trade-offs with 9+ rowers). Empty races
award nothing; a race with a single starter gives Gold only if it finishes (capsize/breakage = no medal).

**Weather and trainer order.** Weather (d6 per §8) is revealed **before** trainer choice and lineup, so
trainer choice is the strategic reaction §7.1 describes. Each trainer has 1–2 weather effects per boat
(e.g. Herr Sack: Hitze +5, Regen −5; Frau Linke: Regen +4, Wind −3; Caro: Wind +5, Kälte −2; …) and
some have a non-weather ability (Ergo +3 per entry; Sprint +2 per boat; Langstrecke +3). All 10 defined
in trainers.json. Trainers apply once per boat / ergo entry.

**Incidents (water only).** One d100 per boat entry, mutually exclusive: 1–5 Kentern (last place, no
medal), 6–15 Materialschaden (DNF), 16–30 Krebs (−10), else clean (70 % clean). Multiple capsized
crews share last place.

**Events (11 Event stations).** Cards per player: events 1–6 → 1 card, 7–11 → 2 cards (≤ 16), capped by
eligible rowers. Rower chosen uniformly at random among own rowers without an event; event type drawn
from a weighted deck. Effects: Wetter-Event (that rower's boat +4 when weather ≠ Normal); Wachstum
(rower's best stat +2 at every training incl. camps); Phönix (weakest +5, tie → random); Tausch
(player swaps two stat values of that rower, one decision); Verdopplung (current best stat doubled
once, permanent); Kraft/Ausdauer/Geschwindigkeit/Technik/Motivation-Event (+8 to that stat);
Seelenzwilling (new rower card "<Name> (Zwilling)", same age/gender, stats = 80 % of current, can't get
an event); Geheime Mission (random mission from a catalog of ~8, e.g. "Gold im Einer vor JTFO",
"Bei einem Ergo-Cup gewinnen"; reward +10 on best stat or +1 medal point; expires at season end).

**Ereignisse (11 stations).** Deck of ~10 global effects, one drawn per station, shown to all:
Trainerstreik (next training only 5 points), Maschsee-Monster (next water regatta Krebs 30 %), Grippewelle
(one random rower per team −5 Motivation), Vereinsfest (all rowers +1 Motivation), Neue Ergometer (next
Ergo-Cup Kraft weight 3), Sponsor (extra draft pick for last place), Hitzewelle (next weather forced
Hitze), Ruhige Woche (nothing), Bootshaus-Renovierung (city boats −1 next regatta), Doppeltraining (next
training 15 points).

**Bots (internal/bot).** Greedy with light noise: training on stats weighted by the next 2
competitions for the rowers most likely to start there; selections = top 5 by upcoming relevance;
regatta = pick trainer with best net weather effect, then assign rowers to races by descending marginal
score (Hungarian-light greedy), best boat per class; draft = highest value for its weakest race class.
Two personalities: "Taktiker" and "Bauchgefühl" (more noise).

---

## Phases

### Phase 0 – Foundation

`git init`, `.gitignore`, `go.mod`, `justfile`, `docs/PLAN.md` (this plan), CLAUDE.md with repo rules
(German UI / English code, rules-data-driven, never run local codex). Copy and adapt the gfx skill.

### Phase 1 – Rules data + PoC rule spec

`docs/poc-rules.md`; `internal/data/*.json`; `cmd/gendata` + generated `rowers.json`; loaders with
validation tests (55 rowers, unique IDs, every stat ≥ 5, sums match ranks, calendar has 52 stations).

### Phase 2 – Engine

State model, RNG, calendar/phase machine, decisions + validation (clear German error strings for the
UI), training, selections, camps, events, Ereignisse, regatta resolution (weather, trainer, scoring,
incidents, ranking with ties, medals, gold motivation), draft, final standings, `ViewFor`. Table-driven
tests per rule; golden replay test (seed + decisions → identical final state).

### Phase 3 – Bots + simulator

`internal/bot`; `cmd/sim -games 1000` prints win rate per seat/personality, medal distribution, average
scores per competition, incident rates. Used to tune rules.json (no seat should exceed ~40 % wins in
3-bot games; final scores 15–40 points).

### Phase 4 – API server

Handlers, token per player, in-memory store + JSON snapshot, CORS for vite dev, `embed` of web/dist,
`tygo` type generation. httptest coverage for the happy path and invalid decisions.

### Phase 5 – Frontend (mobile-first)

Vite/React/TS, Tailwind, shadcn/ui, TanStack Query for API state. Bottom tab bar on phones (Saison,
Team, Wettkampf, Wertung), side layout on ≥ md.

- **Start**: name, seed (optional), "Neues Spiel gegen 2 Computer".
- **Saison**: illustrated Maschsee board; stations as pastel circles in a scrollable path (phone:
  vertical timeline; desktop: board image with station overlay), current station highlighted, figure
  animates forward.
- **Team**: card grid in the example card style (portrait, name, age badge, five stats with icons,
  Motivation highlighted, event card ribbon); boats and trainers tabs.
- **Training / Trainingslager**: per-rower stat steppers with remaining-points counter.
- **Auswahl** (WR5, Basketball, Wanderfahrt): tap to pick 5.
- **Event / Ereignis**: card reveal animation; Tausch dialog.
- **Regatta flow**: weather reveal → trainer pick → lineup builder (race list; tap a slot, choose
  eligible rower from a sheet with computed race score; boat picker; live estimated score; validation
  badges) → submit → result screen per race (ranked bars, incident icons, medals) → standings.
- **Draft**: open offer, pick one.
- **Wertung**: medal table, season log. Final screen with winner.
  Touch targets ≥ 44 px, no horizontal scroll at 360 px width, works offline-tolerant (retry).

### Phase 6 – Art (gfx-imagegen, background)

Adapt the skill (see below), write `art/style.md` preamble from the two examples (watercolour + fine ink,
warm daylight, Maschsee Hannover, navy #1f3a5f frames, pastel station colours). PoC asset list (~20
images): board background without stations (landscape + portrait crop), 6 generic rower portraits
(Kinder/B/A × Mädchen/Junge; generic, not likenesses of the real kids), 4 trainer portraits, boat
illustrations 1x/2x/4x/8x, station icon sheet (training, event, Ereignis, regatta, ergo, basketball,
Wanderfahrt, camp), weather icons (5), gold/silver medals, title splash. Converted to WebP and placed in
`web/src/assets/`. The UI must work with placeholder art until images land.

### Phase 7 – PoC integration and playtest

Full season playable in the browser against 2 bots; Playwright run at 390×844 and 1440×900; fix
layout/flow issues; README with `just dev`.

### Later phases (multiplayer and beyond)

8. **Persistence**: PostgreSQL (games, players, decision log; snapshots every N decisions), migrations
   with goose, sqlc queries; replay on load.
9. **Accounts + lobby**: lightweight accounts (name + magic link or simple password), create/join game
   by code, seat bots in empty seats, 2–4 players (deal tables per §4 for 2/3/4 players).
10. **Realtime multiplayer**: WebSocket per game (`coder/websocket`), server pushes view diffs; phase
    status ("wartet auf Vera…"), optional decision timers with bot fallback, reconnect/resume,
    take-over of a disconnected seat by a bot; anti-cheat via server authority + hidden-info views.
11. **Deployment**: single Go binary embedding the SPA, Dockerfile, PWA manifest + install on phone,
    HTTPS behind Caddy; CI (go test, golangci-lint, tsc, vitest, Playwright smoke).
12. **Rules finalization + balancing**: work through Regelwerk §16 with the family; sim reports per
    rule change; rules editor/view in the app showing the active rules.json.
13. **Board-game export**: print-and-play PDF of cards and board from the same data and art.
14. **Polish**: animations (race progress), sound, tutorial/first-season hints, accessibility, more
    events/Ereignisse/missions, per-rower portraits if the family wants them.

---

## gfx-imagegen adaptation

Copy `/mnt/projekte/Code/Historyline-1914-1918-remake/.claude/skills/gfx-imagegen` to
`.claude/skills/gfx-imagegen` and change:

- `gen.sh`: `remote=` → a dedicated working dir on 192.168.213.20 (`/mnt/Projekte/Code/games/RuderspielVera`,
  created with mkdir; only `art-gen/` written there), local repo detection via `go.mod`/`CLAUDE.md`
  instead of `AGENTS.md`, output under `art/gen/<date>/<group>/` (git-ignored except picks). Keep:
  never run local codex, max 3 parallel runs, disk check, exec-image fetching.
- `preamble.sh`/`assemble.sh`: read `art/style.md` sections `Preamble`, `Portraits`, `Objects`,
  `Board`, `Icons`; entries from `art/prompts.md`.
- SKILL.md rewritten for this project (attach the two example PNGs as style references; drop RefDesign2,
  terrain/seasonal/install sections; keep try-loop, measuring with `wstat`/`bbox`/`comp`, lessons).
- Keep only the Go helpers that matter (wstat, bbox, comp, rectcrop); WebP conversion via a tiny Go
  tool (`golang.org/x/image` can't encode WebP → use `cwebp` if installed, else PNG).

---

## Orchestration (subagents; main agent only coordinates and reviews)

1. **Me**: Phase 0 skeleton (git init, go.mod, justfile, docs/PLAN.md, docs/api.md contract with Go
   view/decision struct definitions) so all agents share one contract.
2. **Wave 1 (parallel)**
   - Agent A (general-purpose): Phase 1 data + gendata + docs/poc-rules.md, then Phase 2 engine with
     tests (TDD).
   - Agent B (general-purpose): web scaffold + design system + all screens against a mock of the
     contract (MSW-style fixtures from docs/api.md).
   - Agent C (general-purpose, background): skill adaptation + art generation runs.
3. **Wave 2 (parallel)**: Agent D bots + sim (after engine), Agent E API server + tygo; B continues on
   the regatta flow.
4. **Wave 3**: integration agent wires frontend to the real API, Playwright mobile/desktop playthrough;
   `cmd/sim` balancing pass.
5. **Review**: go-reviewer + code-reviewer agents on the result; fixes by agents; I verify.

## Verification

- `go test ./...` (rules, golden replay, API httptest), `go vet`, golangci-lint.
- `go run ./cmd/sim -games 1000 -seed 1` → no panics, sane win/score distribution.
- `cd web && npm run build && npx tsc --noEmit`.
- `just dev` (server :8080 + vite), then Playwright: start game, play the full season at 390×844
  (iPhone) and 1440×900 — every station resolves, results and medals show, final ranking appears, no
  horizontal scroll, no console errors. Screenshots shared with the user.
- Art: each picked image reviewed visually and stored with its prompt.
