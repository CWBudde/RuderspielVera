# API contract (PoC)

Source of truth for all JSON shapes: `internal/engine/contract.go`. TS types are generated from it
into `web/src/api/types.gen.ts` (`just gen-types`).

**Text is language-neutral** (see `docs/i18n.md`): the engine never sends sentences. Player-facing
text is a `Msg` (`{"key": "log.draftPick", "args": {...}}`) in `log[].msg`, `phase.title`,
`phase.prompt`, `competition.notes`, `activeEreignisse` and `ereignis.details`; the UI renders it per
language. Data names (`name`, `description`, `ability`, `info`, `reward`, `text`, …) are German
fallbacks only; the UI looks up the translated text by id (`station.id`, `competition.id`,
`boat.defId`, `race.nameKey`, `mission.id`, `phase.selectionId`, `ereignis.id`, event `type`,
trainer `id`).

## Endpoints

| Method | Path                        | Body                                                        | Response                                                                                              |
| ------ | --------------------------- | ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| POST   | `/api/games`                | `{"playerName": "Vera", "seed": 123 (optional), "bots": 2}` | `{"gameId", "playerId", "token", "view": GameView}`                                                   |
| GET    | `/api/games/{id}`           | header `Authorization: Bearer <token>`                      | `GameView` for the token's player                                                                     |
| POST   | `/api/games/{id}/decisions` | `Decision` (bearer token)                                   | `GameView` after bots acted and stations resolved as far as possible                                  |
| GET    | `/api/games/{id}/log`       | bearer token                                                | `{"log": LogEntry[], "results": CompetitionResult[]}`                                                 |
| GET    | `/api/catalog`              | –                                                           | `CatalogView` (`{"rowers", "boats", "trainers", "events": EventTypeView[]}`, from `engine.Catalog()`) |

### Multiplayer lobby (HTTP server only; the wasm build has no multiplayer)

| Method | Path                       | Body                                               | Response                                             |
| ------ | -------------------------- | -------------------------------------------------- | ---------------------------------------------------- |
| POST   | `/api/lobbies`             | `{"playerName", "seats": 2..4, "seed" (optional)}` | 201 `{"gameId", "playerId": "p1", "token", "lobby"}` |
| POST   | `/api/lobbies/{code}/join` | `{"playerName"}`                                   | 201 same shape, next seat (`p2`, `p3`, …)            |
| GET    | `/api/games/{id}/lobby`    | bearer token                                       | `LobbyView` (also after the start, then incl. bots)  |
| POST   | `/api/games/{id}/start`    | bearer token of the host (`p1`)                    | `GameView`; free seats are filled with bots          |

`LobbyView` = `{gameId, code, seats, players: [{id, name, isBot}], you, host, started}` (Go:
`gameservice.LobbyView`, TS: `web/src/api/client.ts`). Join codes are 6 characters without
look-alikes; case, spaces and dashes are ignored. Until the start, `GET /api/games/{id}` answers 409
`err.gameNotStarted` (the UI shows the lobby). Errors: `err.seatCount` 400, `err.lobbyNotFound` 404,
`err.lobbyFull` / `err.lobbyStarted` 409, `err.notHost` 403. Every human seat decides itself; the
phase resolves when all humans have decided (bots answer at once), so `phase.waiting` +
`players[].decided` drive the "waiting for …" screen, which polls every 2 s.

Errors: HTTP 4xx/5xx with `{"error": "<message key>", "msg": Msg}` (the wasm transport returns the
same in its envelope). A decision with a stale
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
- Draft: each pick is its own phase (new `phaseId`); no draft after the final competition.
  A `DraftCard` with `personal: true` is offered only to the player on turn (the eight Paintrain
  for human players in the last draft before the RVB-Endspurt); taking it uses up the pick and
  leaves the open offer for the others. That draft opens even when the pool is empty.
- Entries: `boatId` is empty for ergo races and eights (`class: "8x"`: the player's best own
  eight rows, Kraftwerk or a bought Paintrain; the result's `boatId` names it), and may be empty
  in races with `loanBoat: true` (JTFO and Hildesheim: a loan boat is used; it is not an own boat,
  so any number of entries may use one).
- Engine errors: `engine.ErrStalePhase` → 409; `*engine.ValidationError` (a `Msg`) → 400.
- Hidden info: other players' regatta decisions are never in a view before resolution.
- Guest crews (Gastmannschaften, `rules.json` `guests`): every contested race has one extra
  `EntryResult` with `playerId: ""` and `guest: "<name>"`. It is ranked like a player entry (it
  can take gold or silver away) but never scores points; it has no incident. At JTFO the guest
  appears only in the `team` row.
- **Phase hints** (no client-side guessing): `phase.stage` refines ack phases (`setup`, `event`,
  `ereignis`, `result`; on `result` the view's `lastResult` is this station's result).
  `phase.newEvents` lists the viewer's event cards dealt at this event station, `phase.ereignis`
  (`{name, description}`) the drawn Ereignis. On a selection, `phase.selectionAges` (if set) lists
  the only age groups that may be picked, and `phase.count` is already capped by the viewer's
  eligible rowers. `phase.draftTurn` is the index into `draftOrder` of the
  open pick (omitted = 0; `draftOrder` can name a player twice after a Sponsor Ereignis).
- **Scoring hints** for lineup estimates: `competition.context` is matched against
  `TrainerEffect.context`, `competition.cityBoatMod` is added to städtische Boote. Per race:
  `boatScale` and `trainerScale` multiply the boat value and the per-seat trainer effect
  (`rules.json` `boatValuePerSeat`/`boatValueFactor`, `trainerPerSeat`), `fixedBoatValue` is the base
  value without a chosen boat (the viewer's best eight, loan boat) and `guestScore` the guest crew's total. Entry
  total = round(Σ weight·stat) + boat·boatScale + (trainer effect + trainer.mood)·trainerScale +
  event bonus − penalty.
- **Trainer mood**: every `TrainerView` (team cards, draft offer) carries `mood` (Laune, public,
  rules.json `trainerMood`, −2…+2) and `moodTrend` (change at the last section start, for a trend
  arrow). The mood is added to the per-seat trainer effect of every entry with that trainer, also
  when no effect matches; it changes only when a new section begins. Catalog trainers have 0/0.
- **Siegesmotivation**: every `RowerView` (own and opponents' cards) carries `winMotivation` (the
  temporary Motivation bonus after a gold or silver placement, rules.json `winMotivation`, 0/absent
  when inactive) and `winMotivationLeft` (scored competitions it still counts in, the next one
  included). `stats.motivation` already includes it, so lineup estimates need no extra term; `base`
  does not. It is neither trainable nor swappable and never stacks (a new win resets the duration).
- `activeEreignisse` lists pending Ereignisse that a later station will still apply. The engine
  no longer draws water / ergo cards whose next competition is of the other kind, nor modifiers
  without a consuming station; the filter here stays as a fallback. Competition modifiers
  (weights, Krebs / Materialschaden chance, city boats, guest strength) appear in
  `competition.notes`; guest modifiers are already part of `race.guestScore`.
- **No nulls**: every non-`omitempty` slice/map in a view is serialized as `[]`/`{}` (e.g. a race
  without entries has `"entries": []`); fields marked optional in the TS types may be absent.
- Saved games (`gameservice.Snapshot`: server files, WASM localStorage) carry a `version`. It is
  bumped when a rule change makes old decision logs replay differently (2: per-mille incidents, no
  Motivation training, Hildesheim loan boats; 3: season-aware Ereignisse, larger event and Ereignis
  decks, trainer mood; 4: temporary Siegesmotivation, weaker guest crews, shorter regular regattas;
  5: Krebs chance scaled by boat size and crew Technik, Seelenzwilling links a teammate, Wanderfahrt
  for Junioren only; 6: named city boats dealt at random with values 1/2/4, loan boats worth 0,
  own eights Kraftwerk / Paintrain, boat values count once instead of twice per seat).
  Importing an older version fails with `err.snapshotOld`.
- Bots: `internal/bot.Decide(g, playerID, personality, rng)` (`Taktiker` / `Bauchgefuehl`,
  `bot.PersonalityFor(seat)`); the bot RNG is separate from the game RNG.
