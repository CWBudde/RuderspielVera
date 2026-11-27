/**
 * In-browser fake of the Ruderspiel API (enabled with ?mock=1 or VITE_MOCK=1).
 *
 * It produces contract-shaped GameViews for every phase kind and cycles through a plausible
 * season. It is NOT the rules engine: scoring, events and bots are deliberately simple.
 */
import type {
  AgeGroup,
  BoatView,
  CompetitionResult,
  CompetitionView,
  Decision,
  DraftCard,
  EntryResult,
  EventCardView,
  GameView,
  Gender,
  LogEntry,
  PhaseView,
  PlayerSummary,
  RaceResult,
  RowerView,
  Stat,
  StationView,
  TrainerView,
  Weather,
} from "../types.gen";
import {
  EREIGNISSE,
  EVENT_DECK,
  MISSIONS,
  ROWER_NAMES,
  Rng,
  SUM_RANGE,
  TRAINERS,
  buildCalendar,
  cityBoats,
  competitionFor,
  namedBoats,
  splitStats,
} from "./data";
import { boatPart, crewScore, isEligible, needsBoat, trainerBonus, trainerPart } from "@/lib/score";
import { ALL_STATS, CORE_STATS, STAT_LABEL } from "@/lib/labels";

export type Step = "main" | "swap" | "result" | "draft" | "gameover";

export interface MockState {
  version: 1;
  gameId: string;
  seed: number;
  rng: number;
  calendar: StationView[];
  current: number;
  step: Step;
  phaseSeq: number;
  players: PlayerSummary[];
  pool: { rowers: RowerView[]; boats: BoatView[]; trainers: TrainerView[] };
  log: LogEntry[];
  results: CompetitionResult[];
  lastResult?: CompetitionResult;
  activeEreignisse: string[];
  weather?: Weather;
  swapRowerId?: string;
  /** Your event cards dealt at the current event station. */
  newEvents?: EventCardView[];
  ereignis?: { name: string; description: string };
  draftOffer: DraftCard[];
  draftOrder: string[];
  draftTurn: number;
  /** epoch ms; while set and the turn is a bot's, the human sees waiting=true. */
  draftWaitSince?: number;
  winners?: string[];
}

export const YOU = "p1";
const BOT_NAMES = ["Kalle Krebs", "Frieda Flink"];
const WEATHERS: Weather[] = ["regen", "hitze", "wind", "kaelte", "normal", "normal"];

// ---------- creation ----------

function makeRower(rng: Rng, id: string, name: string, age: AgeGroup, gender: Gender): RowerView {
  const [lo, hi] = SUM_RANGE[age];
  const stats = splitStats(rng, rng.int(lo, hi));
  return { id, name, age, gender, stats: { ...stats }, base: { ...stats }, artKey: `rower-${age}-${gender}` };
}

export function createState(playerName: string, seed: number): MockState {
  const rng = new Rng(seed);
  const all: RowerView[] = [];
  for (const age of ["kinder", "b", "a"] as AgeGroup[]) {
    ROWER_NAMES[age].forEach(([name, gender], i) => {
      all.push(makeRower(rng, `r-${age}-${i + 1}`, name, age, gender));
    });
  }
  rng.shuffle(all);
  const trainers = rng.shuffle([...TRAINERS]);
  const players: PlayerSummary[] = [
    { id: YOU, name: playerName || "Du", isBot: false },
    { id: "p2", name: BOT_NAMES[0], isBot: true },
    { id: "p3", name: BOT_NAMES[1], isBot: true },
  ].map((p, idx) => {
    const count = [10, 9, 11][idx];
    const rowers: RowerView[] = [];
    for (const age of ["a", "b", "kinder"] as AgeGroup[]) {
      const want = age === "b" ? count - 6 : 3;
      for (let k = 0; k < want; k++) {
        const needGender: Gender | undefined = k === 0 ? "w" : k === 1 ? "m" : undefined;
        let i = all.findIndex((r) => r.age === age && (!needGender || r.gender === needGender));
        if (i < 0) i = all.findIndex((r) => r.age === age);
        if (i >= 0) rowers.push(...all.splice(i, 1));
      }
    }
    return {
      ...p,
      gold: 0,
      silver: 0,
      points: 0,
      decided: false,
      rowers,
      boats: cityBoats(p.id),
      trainers: trainers.splice(0, 2),
    };
  });
  // Give each player one named boat so the boat picker has a choice from the start.
  const boats = rng.shuffle(namedBoats());
  for (const p of players) {
    const i = boats.findIndex((b) => b.class === rng.pick(["1x", "2x", "4x"]));
    p.boats.push(...boats.splice(i >= 0 ? i : 0, 1));
  }
  const state: MockState = {
    version: 1,
    gameId: `mock-${seed.toString(36)}`,
    seed,
    rng: rng.state,
    calendar: buildCalendar(),
    current: 0,
    step: "main",
    phaseSeq: 1,
    players,
    pool: { rowers: all, boats, trainers },
    log: [{ stationIndex: 0, kind: "station", text: "Die Saison beginnt am Maschsee." }],
    results: [],
    activeEreignisse: [],
    draftOffer: [],
    draftOrder: [],
    draftTurn: 0,
  };
  return state;
}

// ---------- helpers ----------

function withRng<T>(s: MockState, fn: (rng: Rng) => T): T {
  const rng = new Rng(1);
  rng.state = s.rng;
  const out = fn(rng);
  s.rng = rng.state;
  return out;
}

function station(s: MockState): StationView {
  return s.calendar[s.current];
}

function player(s: MockState, id: string): PlayerSummary {
  const p = s.players.find((x) => x.id === id);
  if (!p) throw new MockError(404, "Spieler nicht gefunden.");
  return p;
}

export class MockError extends Error {
  readonly status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

function addStat(r: RowerView, stat: Stat, n: number) {
  r.stats[stat] = Math.max(0, (r.stats[stat] ?? 0) + n);
}

function bestStat(r: RowerView, only = ALL_STATS): Stat {
  return only.reduce((a, b) => ((r.stats[b] ?? 0) > (r.stats[a] ?? 0) ? b : a));
}

function weakestStat(r: RowerView): Stat {
  return ALL_STATS.reduce((a, b) => ((r.stats[b] ?? 0) < (r.stats[a] ?? 0) ? b : a));
}

function log(s: MockState, kind: string, text: string, playerId?: string) {
  s.log.push({ stationIndex: s.current, kind, text, playerId });
}

function consumeEreignis(s: MockState, prefix: string): boolean {
  const i = s.activeEreignisse.findIndex((t) => t.startsWith(prefix));
  if (i < 0) return false;
  s.activeEreignisse.splice(i, 1);
  return true;
}

function trainingPoints(s: MockState): number {
  const st = station(s);
  if (st.type === "camp") return 30;
  if (s.activeEreignisse.some((t) => t.startsWith("Trainerstreik"))) return 5;
  if (s.activeEreignisse.some((t) => t.startsWith("Doppeltraining"))) return 15;
  return 10;
}

// ---------- station entry ----------

function competitionRotation(s: MockState): number {
  return s.calendar.slice(0, s.current).filter((x) => x.type === "regatta").length;
}

function currentCompetition(s: MockState): CompetitionView {
  const st = station(s);
  const comp = competitionFor(st.name, st.type, competitionRotation(s));
  if (st.type === "regatta") comp.weather = s.weather;
  if (st.type === "ergocup" && s.activeEreignisse.some((t) => t.startsWith("Neue Ergometer"))) {
    comp.weights = { ...comp.weights, kraft: 3 };
    comp.notes.push("Neue Ergometer: Kraft zählt dreifach.");
  }
  if (st.type === "regatta" && s.activeEreignisse.some((t) => t.startsWith("Maschsee-Monster"))) {
    comp.notes.push("Maschsee-Monster: Krebsgefahr 30 %.");
  }
  if (st.type === "regatta" && s.activeEreignisse.some((t) => t.startsWith("Renovierung"))) {
    comp.notes.push("Renovierung: städtische Boote −1.");
  }
  return comp;
}

function drawEvent(s: MockState, rng: Rng, p: PlayerSummary): EventCardView | undefined {
  const free = p.rowers.filter((r) => !r.event && !r.isTwin);
  if (free.length === 0) return undefined;
  const rower = rng.pick(free);
  const total = EVENT_DECK.reduce((a, e) => a + e.weight, 0);
  let roll = rng.next() * total;
  let def = EVENT_DECK[0];
  for (const e of EVENT_DECK) {
    roll -= e.weight;
    if (roll <= 0) {
      def = e;
      break;
    }
  }
  const card: EventCardView = { type: def.type, name: def.name, description: def.description, rowerId: rower.id };
  rower.event = card;
  switch (def.type) {
    case "kraft":
    case "ausdauer":
    case "geschwindigkeit":
    case "technik":
    case "motivation":
      addStat(rower, def.type, 8);
      break;
    case "phoenix":
      addStat(rower, weakestStat(rower), 5);
      break;
    case "verdopplung": {
      const b = bestStat(rower);
      addStat(rower, b, rower.stats[b] ?? 0);
      break;
    }
    case "mission":
      card.mission = { ...rng.pick(MISSIONS), done: false, failed: false };
      break;
    case "zwilling": {
      const twinStats = Object.fromEntries(ALL_STATS.map((k) => [k, Math.round((rower.stats[k] ?? 0) * 0.8)]));
      p.rowers.push({
        id: `${rower.id}-twin`,
        name: `${rower.name} (Zwilling)`,
        age: rower.age,
        gender: rower.gender,
        stats: { ...twinStats },
        base: { ...twinStats },
        artKey: rower.artKey,
        isTwin: true,
      });
      break;
    }
  }
  log(s, "event", `${p.name}: ${rower.name} erhält „${def.name}“.`, p.id);
  if (p.id === YOU) s.newEvents = [...(s.newEvents ?? []), card];
  if (def.type === "tausch" && p.id === YOU) s.swapRowerId = rower.id;
  return card;
}

/** Called whenever `current` moves to a new station. */
function enterStation(s: MockState) {
  const st = station(s);
  s.step = "main";
  s.weather = undefined;
  s.newEvents = undefined;
  s.ereignis = undefined;
  s.swapRowerId = undefined;
  log(s, "station", `Station ${st.index + 1}: ${st.name}`);
  withRng(s, (rng) => {
    if (st.type === "event") {
      const cards = st.section >= 8 ? 2 : 1;
      for (const p of s.players) for (let i = 0; i < cards; i++) drawEvent(s, rng, p);
    } else if (st.type === "ereignis") {
      const e = rng.pick(EREIGNISSE);
      log(s, "ereignis", `${e.name}: ${e.text}`);
      s.ereignis = { name: e.name, description: e.text };
      if (e.active) s.activeEreignisse.push(e.active);
      if (e.name === "Vereinsfest") for (const p of s.players) for (const r of p.rowers) addStat(r, "motivation", 1);
      if (e.name === "Grippewelle") for (const p of s.players) addStat(rng.pick(p.rowers), "motivation", -5);
    } else if (st.type === "regatta") {
      s.weather = consumeEreignis(s, "Hitzewelle") ? "hitze" : rng.pick(WEATHERS);
    }
  });
}

function advance(s: MockState) {
  if (s.current >= s.calendar.length - 1) {
    finishGame(s);
    return;
  }
  s.current++;
  s.phaseSeq++;
  enterStation(s);
}

function finishGame(s: MockState) {
  s.step = "gameover";
  s.phaseSeq++;
  const best = Math.max(...s.players.map((p) => p.points));
  s.winners = s.players.filter((p) => p.points === best).map((p) => p.id);
  const names = s.players.filter((p) => s.winners?.includes(p.id)).map((p) => p.name);
  log(s, "info", `Saisonende! Sieg für ${names.join(" und ")}.`);
}

// ---------- decisions ----------

function applyTraining(s: MockState, p: PlayerSummary, allocs: { rowerId: string; stat: Stat; points: number }[]) {
  for (const a of allocs) {
    const r = p.rowers.find((x) => x.id === a.rowerId);
    if (r) addStat(r, a.stat, a.points);
  }
  for (const r of p.rowers) if (r.event?.type === "wachstum") addStat(r, bestStat(r, CORE_STATS), 2);
  void s;
}

function botTraining(rng: Rng, p: PlayerSummary, points: number, maxRowers: number) {
  const allocs: { rowerId: string; stat: Stat; points: number }[] = [];
  const chosen = rng.shuffle([...p.rowers]).slice(0, Math.min(maxRowers, p.rowers.length));
  for (let i = 0; i < points; i++) {
    const r = rng.pick(chosen);
    allocs.push({ rowerId: r.id, stat: rng.pick(CORE_STATS), points: 1 });
  }
  return allocs;
}

function validateTraining(s: MockState, d: Decision, p: PlayerSummary) {
  const points = trainingPoints(s);
  const allocs = d.training ?? [];
  const sum = allocs.reduce((a, x) => a + x.points, 0);
  if (allocs.some((a) => a.points < 0)) throw new MockError(400, "Negative Trainingspunkte sind nicht erlaubt.");
  if (allocs.some((a) => a.points > 0 && !CORE_STATS.includes(a.stat)))
    throw new MockError(400, "Motivation kann man nicht trainieren.");
  if (sum !== points) throw new MockError(400, `Bitte genau ${points} Punkte verteilen (verteilt: ${sum}).`);
  const rowers = new Set(allocs.filter((a) => a.points > 0).map((a) => a.rowerId));
  if (station(s).type === "camp" && rowers.size > 5) throw new MockError(400, "Im Trainingslager höchstens 5 Ruderer.");
  for (const id of rowers) {
    if (!p.rowers.some((r) => r.id === id)) throw new MockError(400, "Unbekannter Ruderer.");
  }
}

function selectionEffect(name: string): { text: string; apply: (r: RowerView) => void } {
  if (name === "Wanderfahrt") {
    return {
      text: "+4 Ausdauer, +3 Motivation",
      apply: (r) => {
        addStat(r, "ausdauer", 4);
        addStat(r, "motivation", 3);
      },
    };
  }
  return { text: "+10 Motivation", apply: (r) => addStat(r, "motivation", 10) };
}

// ----- regatta resolution -----

interface Lineup {
  trainerId?: string;
  entries: { raceId: string; rowerIds: string[]; boatId?: string }[];
}

function botLineup(p: PlayerSummary, comp: CompetitionView): Lineup {
  const trainer = [...p.trainers].sort((a, b) => trainerBonus(b, comp) - trainerBonus(a, comp))[0];
  const used = new Set<string>();
  const entries: Lineup["entries"] = [];
  for (const race of comp.races) {
    const pool = p.rowers
      .filter((r) => !used.has(r.id) && isEligible(r, race))
      .sort((a, b) => crewScore([b], comp.weights) - crewScore([a], comp.weights));
    if (pool.length < race.seats) continue;
    let crew = pool.slice(0, race.seats);
    if (race.mixed) {
      const hasW = crew.some((r) => r.gender === "w");
      const hasM = crew.some((r) => r.gender === "m");
      if (!hasW || !hasM) {
        const missing = hasW ? "m" : "w";
        const swap = pool.find((r) => r.gender === missing && !crew.includes(r));
        if (!swap) continue;
        crew = [...crew.slice(0, -1), swap];
      }
      if (new Set(crew.map((r) => r.age)).size < 2) continue;
    }
    let boatId: string | undefined;
    if (needsBoat(race)) {
      const boat = p.boats.filter((b) => b.class === race.class).sort((a, b) => b.value - a.value)[0];
      if (!boat && !race.loanBoat) continue;
      boatId = boat?.id;
    }
    crew.forEach((r) => used.add(r.id));
    entries.push({ raceId: race.id, rowerIds: crew.map((r) => r.id), boatId });
  }
  return { trainerId: trainer?.id, entries };
}

function validateLineup(p: PlayerSummary, comp: CompetitionView, d: Decision) {
  const used = new Set<string>();
  for (const e of d.entries ?? []) {
    const race = comp.races.find((r) => r.id === e.raceId);
    if (!race) throw new MockError(400, "Unbekanntes Rennen.");
    if (e.rowerIds.length !== race.seats)
      throw new MockError(400, `${race.name}: Bitte alle ${race.seats} Plätze besetzen.`);
    for (const id of e.rowerIds) {
      const r = p.rowers.find((x) => x.id === id);
      if (!r) throw new MockError(400, "Unbekannter Ruderer.");
      if (used.has(id)) throw new MockError(400, `${r.name} darf nur in einem Rennen pro Regatta starten.`);
      if (!isEligible(r, race)) throw new MockError(400, `${r.name} ist für ${race.name} nicht startberechtigt.`);
      used.add(id);
    }
    if (
      needsBoat(race) &&
      !(race.loanBoat && !e.boatId) &&
      !p.boats.some((b) => b.id === e.boatId && b.class === race.class)
    ) {
      throw new MockError(400, `${race.name}: Bitte ein passendes Boot wählen.`);
    }
  }
  if (comp.kind !== "ergocup" && d.trainerId && !p.trainers.some((t) => t.id === d.trainerId)) {
    throw new MockError(400, "Dieser Trainer gehört nicht zu deinem Team.");
  }
}

function rankEntries(entries: EntryResult[]): EntryResult[] {
  const finished = entries.filter((e) => e.incident !== "material" && e.incident !== "kentern");
  const capsized = entries.filter((e) => e.incident === "kentern");
  const dnf = entries.filter((e) => e.incident === "material");
  finished.sort((a, b) => b.total - a.total);
  finished.forEach((e, i) => {
    e.place = i > 0 && finished[i - 1].total === e.total ? finished[i - 1].place : i + 1;
  });
  capsized.forEach((e) => (e.place = finished.length + 1));
  dnf.forEach((e) => (e.place = 0));
  for (const e of finished) {
    if (e.place === 1) e.medal = "gold";
    else if (e.place === 2) e.medal = "silber";
  }
  return [...finished, ...capsized, ...dnf];
}

function resolveCompetition(s: MockState, yours: Lineup) {
  const comp = currentCompetition(s);
  const water = comp.kind !== "ergocup";
  // Per mille, mirrors rules.json incidents: Kentern by boat size, Material, Krebs (Maschsee-Monster 300 ‰).
  const krebs = consumeEreignis(s, "Maschsee-Monster") ? 300 : 150;
  const renovation = comp.kind === "regatta" && consumeEreignis(s, "Renovierung");
  if (comp.kind === "ergocup") consumeEreignis(s, "Neue Ergometer");
  const lineups = new Map<string, Lineup>();
  for (const p of s.players) lineups.set(p.id, p.id === YOU ? yours : botLineup(p, comp));

  const races: RaceResult[] = withRng(s, (rng) =>
    comp.races.map((race) => {
      const entries: EntryResult[] = [];
      for (const p of s.players) {
        const lu = lineups.get(p.id);
        const e = lu?.entries.find((x) => x.raceId === race.id);
        if (!lu || !e) continue;
        const crew = e.rowerIds.map((id) => p.rowers.find((r) => r.id === id)).filter((r): r is RowerView => !!r);
        const boat = p.boats.find((b) => b.id === e.boatId);
        const trainer = p.trainers.find((t) => t.id === lu.trainerId);
        const eventBonus =
          crew.some((r) => r.event?.type === "wetter") && comp.weather && comp.weather !== "normal" ? 4 : 0;
        let boatValue = boatPart(race, boat, comp);
        if (renovation && boat?.city) boatValue -= race.boatScale;
        const res: EntryResult = {
          playerId: p.id,
          rowerIds: e.rowerIds,
          boatId: boat?.id,
          trainerId: trainer?.id,
          crew: crewScore(crew, comp.weights),
          boatValue,
          trainer: trainerPart(race, trainer, comp),
          eventBonus,
          penalty: 0,
          total: 0,
          place: 0,
        };
        if (water) {
          const kentern = ({ 1: 20, 2: 10, 4: 5, 8: 2 } as Record<number, number>)[race.seats] ?? 0;
          const roll = rng.int(1, 1000);
          if (roll <= kentern) res.incident = "kentern";
          else if (roll <= kentern + 30) res.incident = "material";
          else if (roll <= kentern + 30 + krebs) {
            res.incident = "krebs";
            res.penalty = 10;
          }
        }
        res.total = res.crew + res.boatValue + res.trainer + res.eventBonus - res.penalty;
        entries.push(res);
      }
      if (entries.length > 0 && !comp.teamEvent) {
        // Gastmannschaft: fixed-strength crew that takes places but never scores (rules.json guests).
        const wsum = Object.values(comp.weights).reduce((a, w) => a + w, 0);
        const total =
          Math.round(race.seats * wsum * (19 + s.current * 0.3)) + (needsBoat(race) ? 5 * race.boatScale : 0);
        entries.push({
          playerId: "",
          guest: "Gastmannschaft",
          rowerIds: [],
          crew: total,
          boatValue: 0,
          trainer: 0,
          eventBonus: 0,
          penalty: 0,
          total,
          place: 0,
        });
      }
      return { raceId: race.id, name: race.name, entries: rankEntries(entries) };
    }),
  );

  let scored = races;
  if (comp.teamEvent) {
    // JTFO: per-race rows without medals plus one team ranking.
    const team: EntryResult[] = s.players
      .map((p) => {
        const mine = races.flatMap((r) => r.entries.filter((e) => e.playerId === p.id));
        if (mine.length === 0) return undefined;
        const sum = (k: "crew" | "boatValue" | "trainer" | "eventBonus" | "penalty") =>
          mine.reduce((a, e) => a + (e.incident === "material" ? 0 : e[k]), 0);
        const t: EntryResult = {
          playerId: p.id,
          rowerIds: mine.flatMap((e) => e.rowerIds),
          crew: sum("crew"),
          boatValue: sum("boatValue"),
          trainer: sum("trainer"),
          eventBonus: sum("eventBonus"),
          penalty: sum("penalty"),
          total: 0,
          place: 0,
        };
        t.total = t.crew + t.boatValue + t.trainer + t.eventBonus - t.penalty;
        return t;
      })
      .filter((e): e is EntryResult => !!e);
    for (const r of races) for (const e of r.entries) e.medal = undefined;
    scored = [...races, { raceId: "jtfo-team", name: "Teamwertung", entries: rankEntries(team) }];
  }

  for (const r of comp.teamEvent ? scored.slice(-1) : scored) {
    for (const e of r.entries) {
      const p = player(s, e.playerId);
      if (e.medal === "gold") {
        p.gold++;
        for (const id of e.rowerIds) {
          const rower = p.rowers.find((x) => x.id === id);
          if (rower) addStat(rower, "motivation", 1);
        }
      } else if (e.medal === "silber") p.silver++;
      p.points = 2 * p.gold + p.silver;
    }
  }

  const result: CompetitionResult = { stationIndex: s.current, name: comp.name, weather: comp.weather, races: scored };
  s.lastResult = result;
  s.results.push(result);
  const you = player(s, YOU);
  const mine = scored.flatMap((r) => r.entries).filter((e) => e.playerId === YOU);
  const golds = mine.filter((e) => e.medal === "gold").length;
  const silvers = mine.filter((e) => e.medal === "silber").length;
  log(s, "result", `${comp.name}: ${you.name} gewinnt ${golds} Gold und ${silvers} Silber.`, YOU);
}

// ----- draft -----

function makeOffer(s: MockState, rng: Rng): DraftCard[] {
  const n = s.players.length + 1;
  const offer: DraftCard[] = [];
  const section = station(s).section;
  for (let i = 0; i < n; i++) {
    const kinds = ["rower", "boat", ...(section >= 5 && s.pool.trainers.length ? ["trainer"] : [])];
    const kind = rng.pick(kinds);
    if (kind === "rower" && s.pool.rowers.length) {
      const r = s.pool.rowers.splice(rng.int(0, s.pool.rowers.length - 1), 1)[0];
      offer.push({ id: `dc-${r.id}`, kind, rower: r });
    } else if (kind === "trainer" && s.pool.trainers.length) {
      const t = s.pool.trainers.splice(0, 1)[0];
      offer.push({ id: `dc-${t.id}`, kind, trainer: t });
    } else if (s.pool.boats.length) {
      const b = s.pool.boats.splice(rng.int(0, s.pool.boats.length - 1), 1)[0];
      offer.push({ id: `dc-${b.id}`, kind: "boat", boat: b });
    }
  }
  return offer;
}

function takeCard(s: MockState, playerId: string, cardId: string) {
  const i = s.draftOffer.findIndex((c) => c.id === cardId);
  if (i < 0) throw new MockError(400, "Diese Karte liegt nicht mehr aus.");
  const [card] = s.draftOffer.splice(i, 1);
  const p = player(s, playerId);
  if (card.rower) p.rowers.push(card.rower);
  if (card.boat) p.boats.push(card.boat);
  if (card.trainer) p.trainers.push(card.trainer);
  const name = card.rower?.name ?? card.boat?.name ?? card.trainer?.name ?? "?";
  log(s, "draft", `${p.name} wählt ${name}.`, playerId);
}

function botPick(s: MockState, playerId: string) {
  const p = player(s, playerId);
  const score = (c: DraftCard) =>
    c.rower
      ? crewScore([c.rower], { kraft: 1, ausdauer: 1, geschwindigkeit: 1, technik: 1 }) / 8
      : c.boat
        ? c.boat.value
        : 6;
  const best = [...s.draftOffer].sort((a, b) => score(b) - score(a))[0];
  if (best) takeCard(s, p.id, best.id);
}

function startDraft(s: MockState, forceWait = false) {
  s.step = "draft";
  s.phaseSeq++;
  s.draftOffer = withRng(s, (rng) => makeOffer(s, rng));
  s.draftOrder = [...s.players].sort((a, b) => a.points - b.points || a.id.localeCompare(b.id)).map((p) => p.id);
  if (forceWait && s.draftOrder[0] === YOU) s.draftOrder.push(s.draftOrder.shift() as string);
  s.draftTurn = 0;
  s.draftWaitSince = undefined;
  runBotPicks(s, forceWait);
}

/** Bots pick until it's the human's turn. With `wait`, the first bot pick is delayed (polling demo). */
function runBotPicks(s: MockState, wait = false) {
  while (s.draftTurn < s.draftOrder.length) {
    const id = s.draftOrder[s.draftTurn];
    if (id === YOU) return;
    if (wait) {
      s.draftWaitSince = Date.now();
      return;
    }
    botPick(s, id);
    s.draftTurn++;
  }
  finishDraft(s);
}

function finishDraft(s: MockState) {
  // Unpicked cards go back to the pool.
  for (const c of s.draftOffer) {
    if (c.rower) s.pool.rowers.push(c.rower);
    if (c.boat) s.pool.boats.push(c.boat);
    if (c.trainer) s.pool.trainers.push(c.trainer);
  }
  s.draftOffer = [];
  s.draftWaitSince = undefined;
  advance(s);
}

/** Lets time-based mock progress happen on reads (sequential draft "wartet auf …"). */
export function tick(s: MockState): boolean {
  if (s.step === "draft" && s.draftWaitSince && Date.now() - s.draftWaitSince > 2500) {
    s.draftWaitSince = undefined;
    const id = s.draftOrder[s.draftTurn];
    if (id && id !== YOU) {
      botPick(s, id);
      s.draftTurn++;
    }
    runBotPicks(s);
    return true;
  }
  return false;
}

// ---------- decide ----------

export function decide(s: MockState, d: Decision) {
  const view = buildView(s);
  if (d.phaseId !== view.phase.id) {
    throw new MockError(409, "Diese Entscheidung ist veraltet. Der Spielstand wurde neu geladen.");
  }
  if (d.kind !== view.phase.kind) throw new MockError(400, "Falsche Art von Entscheidung für diese Phase.");
  if (view.phase.waiting) throw new MockError(409, "Du bist gerade nicht an der Reihe.");
  const st = station(s);
  const you = player(s, YOU);

  switch (view.phase.kind) {
    case "ack":
      if (s.step === "result") {
        if (s.current >= s.calendar.length - 1) finishGame(s);
        else startDraft(s);
      } else if (st.type === "event" && s.swapRowerId) {
        s.step = "swap";
        s.phaseSeq++;
      } else {
        advance(s);
      }
      return;
    case "eventswap": {
      const r = you.rowers.find((x) => x.id === s.swapRowerId);
      if (d.swap && r) {
        const [a, b] = d.swap;
        if (a === b) throw new MockError(400, "Bitte zwei verschiedene Werte wählen.");
        const va = r.stats[a] ?? 0;
        r.stats[a] = r.stats[b] ?? 0;
        r.stats[b] = va;
        log(s, "event", `${r.name}: ${STAT_LABEL[a]} und ${STAT_LABEL[b]} getauscht.`, YOU);
      }
      s.swapRowerId = undefined;
      advance(s);
      return;
    }
    case "training": {
      validateTraining(s, d, you);
      const points = trainingPoints(s);
      applyTraining(s, you, d.training ?? []);
      withRng(s, (rng) => {
        for (const p of s.players.filter((x) => x.isBot)) {
          applyTraining(s, p, botTraining(rng, p, points, st.type === "camp" ? 5 : 99));
        }
      });
      if (st.type === "training") {
        consumeEreignis(s, "Trainerstreik") || consumeEreignis(s, "Doppeltraining");
      }
      advance(s);
      return;
    }
    case "selection": {
      const ids = d.rowerIds ?? [];
      const count = view.phase.count ?? 5;
      if (new Set(ids).size !== Math.min(count, you.rowers.length)) {
        throw new MockError(400, `Bitte genau ${count} Ruderer auswählen.`);
      }
      const eff = selectionEffect(st.name);
      for (const id of ids) {
        const r = you.rowers.find((x) => x.id === id);
        if (r) eff.apply(r);
      }
      withRng(s, (rng) => {
        for (const p of s.players.filter((x) => x.isBot))
          rng
            .shuffle([...p.rowers])
            .slice(0, count)
            .forEach(eff.apply);
      });
      advance(s);
      return;
    }
    case "regatta": {
      const comp = view.phase.competition;
      if (!comp) throw new MockError(500, "Kein Wettkampf aktiv.");
      validateLineup(you, comp, d);
      resolveCompetition(s, { trainerId: d.trainerId, entries: d.entries ?? [] });
      s.step = "result";
      s.phaseSeq++;
      return;
    }
    case "draft":
      if (!d.draftCardId) throw new MockError(400, "Bitte eine Karte wählen.");
      takeCard(s, YOU, d.draftCardId);
      s.draftTurn++;
      runBotPicks(s);
      return;
    default:
      throw new MockError(400, "Das Spiel ist vorbei.");
  }
}

// ---------- view ----------

function phaseFor(s: MockState): PhaseView {
  const st = station(s);
  const id = `ph-${s.current}-${s.step}-${s.phaseSeq}`;
  const base = { id, waiting: false };
  if (s.step === "gameover") {
    return {
      ...base,
      kind: "gameover",
      title: "Saisonende",
      prompt: "Die Saison ist vorbei. Hier ist die Abschlusswertung.",
    };
  }
  if (s.step === "result") {
    return {
      ...base,
      kind: "ack",
      stage: "result",
      title: `Ergebnis: ${st.name}`,
      prompt: "Die Rennen sind gefahren.",
    };
  }
  if (s.step === "draft") {
    const onTurn = s.draftOrder[s.draftTurn];
    return {
      ...base,
      kind: "draft",
      title: "Kartenwahl",
      prompt: "Wähle eine Karte aus der offenen Auslage. Wer weniger Medaillenpunkte hat, wählt zuerst.",
      draftOffer: s.draftOffer,
      draftOrder: s.draftOrder,
      draftTurn: s.draftTurn,
      waiting: onTurn !== YOU,
    };
  }
  if (s.step === "swap") {
    return {
      ...base,
      kind: "eventswap",
      title: "Tausch-Event",
      prompt: "Du darfst zwei Werte dieses Ruderers tauschen – oder alles lassen, wie es ist.",
      swapRowerId: s.swapRowerId,
    };
  }
  switch (st.type) {
    case "setup":
      return {
        ...base,
        kind: "ack",
        stage: "setup",
        title: "Spielvorbereitung",
        prompt: "Deine Startkarten sind verteilt: Ruderer aus allen Altersklassen, städtische Boote und zwei Trainer.",
      };
    case "training":
      return {
        ...base,
        kind: "training",
        title: "Training",
        prompt: "Verteile deine Trainingspunkte auf beliebige Ruderer und Werte.",
        points: trainingPoints(s),
      };
    case "camp":
      return {
        ...base,
        kind: "training",
        title: `Trainingslager ${st.name}`,
        prompt: "Verteile 30 Punkte auf höchstens 5 Ruderer.",
        points: 30,
        maxRowers: 5,
      };
    case "selection": {
      const eff = selectionEffect(st.name);
      return {
        ...base,
        kind: "selection",
        title: st.name === "WR5" ? "Winterruder-Fünfkampf" : st.name,
        prompt: "Wähle genau 5 Ruderer aus, die mitmachen.",
        count: 5,
        selectionEffect: eff.text,
      };
    }
    case "event":
      return {
        ...base,
        kind: "ack",
        stage: "event",
        title: "Event",
        prompt: "Neue Eventkarten wurden gezogen.",
        newEvents: s.newEvents,
      };
    case "ereignis":
      return {
        ...base,
        kind: "ack",
        stage: "ereignis",
        title: "Ereignis",
        prompt: "Ein Ereignis betrifft alle Teams.",
        ereignis: s.ereignis,
      };
    default:
      return {
        ...base,
        kind: "regatta",
        title: st.name,
        prompt:
          st.type === "ergocup"
            ? "Stelle deine Starter für die Ergo-Rennen auf."
            : "Wähle deinen Trainer und stelle deine Boote auf. Jeder Ruderer startet höchstens einmal.",
        competition: currentCompetition(s),
      };
  }
}

export function buildView(s: MockState): GameView {
  const phase = phaseFor(s);
  const decidedAll = phase.kind !== "draft";
  return {
    gameId: s.gameId,
    you: YOU,
    calendar: s.calendar,
    current: s.current,
    phase,
    players: s.players.map((p) => ({
      ...p,
      decided: decidedAll ? p.isBot : s.draftOrder.indexOf(p.id) < s.draftTurn,
    })),
    lastResult: s.lastResult,
    activeEreignisse: [...s.activeEreignisse],
    log: s.log.slice(-80),
    winners: s.winners,
  };
}

// ---------- dev jump ----------

export type JumpTarget =
  | "setup"
  | "training"
  | "camp"
  | "selection"
  | "event"
  | "eventswap"
  | "ereignis"
  | "regatta"
  | "ergocup"
  | "jtfo"
  | "rvb"
  | "result"
  | "draft"
  | "draft-waiting"
  | "gameover";

/** Fast-forward (or rewind) to a station of the given kind, simulating progress on the way. */
export function jump(s: MockState, target: JumpTarget) {
  const typeFor: Record<string, string> = {
    eventswap: "event",
    result: "regatta",
    draft: "regatta",
    "draft-waiting": "regatta",
    rvb: "regatta",
    gameover: "regatta",
  };
  const wantType = typeFor[target] ?? target;
  let idx = -1;
  if (target === "rvb" || target === "gameover") idx = s.calendar.length - 1;
  else {
    for (let k = 1; k <= s.calendar.length; k++) {
      const i = (s.current + k) % s.calendar.length;
      if (s.calendar[i].type === wantType) {
        idx = i;
        break;
      }
    }
  }
  if (idx < 0) return;
  if (idx < s.current) {
    const fresh = createState(player(s, YOU).name, s.seed);
    Object.assign(s, fresh);
  }
  // Simulate skipped stations: some training and medals so the UI shows progress.
  withRng(s, (rng) => {
    for (let i = s.current + 1; i < idx; i++) {
      const t = s.calendar[i].type;
      for (const p of s.players) {
        if (t === "training" || t === "camp") {
          for (let k = 0; k < (t === "camp" ? 30 : 10); k++) addStat(rng.pick(p.rowers), rng.pick(CORE_STATS), 1);
        }
        if (t === "regatta" || t === "ergocup") {
          p.gold += rng.int(0, 2);
          p.silver += rng.int(0, 2);
          p.points = 2 * p.gold + p.silver;
        }
      }
      if (t === "event") {
        s.current = i;
        for (const p of s.players) drawEvent(s, rng, p);
        s.swapRowerId = undefined;
      }
    }
  });
  s.current = idx;
  s.phaseSeq++;
  enterStation(s);
  if (target === "eventswap") {
    const you = player(s, YOU);
    const r = you.rowers.find((x) => !x.event) ?? you.rowers[0];
    const def = EVENT_DECK.find((e) => e.type === "tausch");
    if (r && def) {
      r.event = { type: def.type, name: def.name, description: def.description, rowerId: r.id };
      s.swapRowerId = r.id;
      s.step = "swap";
    }
  }
  if (target === "result" || target === "draft" || target === "draft-waiting" || target === "gameover") {
    const you = player(s, YOU);
    resolveCompetition(s, botLineup(you, currentCompetition(s)));
    s.step = "result";
    if (target === "draft") startDraft(s);
    if (target === "draft-waiting") startDraft(s, true);
    if (target === "gameover") finishGame(s);
  }
}

export function catalog() {
  const s = createState("Katalog", 1);
  return {
    rowers: [...s.players.flatMap((p) => p.rowers), ...s.pool.rowers],
    boats: [...cityBoats("stadt"), ...namedBoats()],
    trainers: TRAINERS,
    events: EVENT_DECK.map(({ type, name, description }) => ({ type, name, description })),
  };
}
