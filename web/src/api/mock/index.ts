import type { Backend, Session } from "../client";
import { ApiError } from "../errors";
import { readStorage, writeStorage } from "@/lib/storage";
import {
  MockError,
  YOU,
  buildView,
  catalog,
  createState,
  decide,
  jump,
  tick,
  type JumpTarget,
  type MockState,
} from "./server";

const KEY = "ruderspiel.mock.state";
let cache: MockState | null = null;

function load(gameId: string): MockState {
  if (cache?.gameId === gameId) return cache;
  const raw = readStorage(KEY);
  if (raw) {
    try {
      const s = JSON.parse(raw) as MockState;
      if (s.version === 1 && s.gameId === gameId) {
        cache = s;
        return s;
      }
    } catch {
      /* corrupt: fall through */
    }
  }
  throw new ApiError(404, "Dieses Spiel gibt es nicht mehr. Bitte ein neues starten.");
}

function save(s: MockState) {
  cache = s;
  writeStorage(KEY, JSON.stringify(s));
}

/** Simulated network latency so loading states are visible. */
function delay<T>(fn: () => T, ms = 160): Promise<T> {
  return new Promise((resolve, reject) => {
    window.setTimeout(() => {
      try {
        resolve(fn());
      } catch (e) {
        if (e instanceof MockError) reject(new ApiError(e.status, e.message));
        else reject(e);
      }
    }, ms);
  });
}

function check(s: Session): MockState {
  const st = load(s.gameId);
  if (s.playerId !== YOU) throw new ApiError(403, "Ungültiges Spieler-Token.");
  return st;
}

export const mockApi: Backend = {
  createGame: (req) =>
    delay(() => {
      const seed = req.seed ?? Math.floor(Math.random() * 1_000_000);
      const s = createState(req.playerName.trim(), seed);
      save(s);
      return { gameId: s.gameId, playerId: YOU, token: `mock-token-${seed}`, view: buildView(s) };
    }, 350),
  getGame: (sess) =>
    delay(() => {
      const s = check(sess);
      if (tick(s)) save(s);
      return buildView(s);
    }),
  submitDecision: (sess, d) =>
    delay(() => {
      const s = check(sess);
      decide(s, d);
      save(s);
      return buildView(s);
    }, 260),
  getLog: (sess) =>
    delay(() => {
      const s = check(sess);
      return { log: s.log, results: s.results };
    }),
  getCatalog: () => delay(() => catalog()),
};

/** Dev-only: jump the mock game to a phase of the given kind. */
export function mockJump(gameId: string, target: JumpTarget) {
  const s = load(gameId);
  jump(s, target);
  save(s);
  return buildView(s);
}

export type { JumpTarget };
