import type { CatalogView, CompetitionResult, Decision, GameView, LogEntry } from "./types.gen";
import { readStorage, writeStorage } from "@/lib/storage";
import { mockApi } from "./mock";
import { ApiError } from "./errors";
import { wasmBackend } from "./wasmTransport";

export { ApiError };

export interface Session {
  gameId: string;
  playerId: string;
  token: string;
  mock: boolean;
}

export interface CreateGameRequest {
  playerName: string;
  seed?: number;
  bots: number;
}

export interface CreateGameResponse {
  gameId: string;
  playerId: string;
  token: string;
  view: GameView;
}

export interface LogResponse {
  log: LogEntry[];
  results: CompetitionResult[];
}

export type Catalog = CatalogView;

/** The transport behind the client: real HTTP or the in-browser mock. */
export interface Backend {
  createGame(req: CreateGameRequest): Promise<CreateGameResponse>;
  getGame(s: Session): Promise<GameView>;
  submitDecision(s: Session, d: Decision): Promise<GameView>;
  getLog(s: Session): Promise<LogResponse>;
  getCatalog(): Promise<Catalog>;
}

const SESSION_KEY = "ruderspiel.session";

export function isMockRequested(): boolean {
  try {
    const p = new URLSearchParams(window.location.search);
    if (p.get("mock") === "1") return true;
    if (p.get("mock") === "0") return false;
  } catch {
    /* ignore */
  }
  return import.meta.env.VITE_MOCK === "1";
}

export function loadSession(): Session | null {
  const raw = readStorage(SESSION_KEY);
  if (!raw) return null;
  try {
    const s = JSON.parse(raw) as Partial<Session>;
    if (typeof s.gameId === "string" && typeof s.token === "string" && typeof s.playerId === "string") {
      return { gameId: s.gameId, playerId: s.playerId, token: s.token, mock: s.mock === true };
    }
  } catch {
    /* corrupt entry */
  }
  return null;
}

export function saveSession(s: Session | null): void {
  writeStorage(SESSION_KEY, s ? JSON.stringify(s) : null);
}

async function request<T>(path: string, init: RequestInit = {}, token?: string): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);
  let res: Response;
  try {
    res = await fetch(path, { ...init, headers });
  } catch {
    throw new ApiError(0, "Keine Verbindung zum Server. Bitte gleich nochmal versuchen.");
  }
  const text = await res.text();
  let body: unknown = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = null;
    }
  }
  if (!res.ok) {
    const msg =
      body && typeof body === "object" && "error" in body && typeof body.error === "string"
        ? body.error
        : `Serverfehler (${res.status})`;
    throw new ApiError(res.status, msg);
  }
  return body as T;
}

const httpBackend: Backend = {
  createGame: (req) => request<CreateGameResponse>("/api/games", { method: "POST", body: JSON.stringify(req) }),
  getGame: (s) => request<GameView>(`/api/games/${encodeURIComponent(s.gameId)}`, {}, s.token),
  submitDecision: (s, d) =>
    request<GameView>(
      `/api/games/${encodeURIComponent(s.gameId)}/decisions`,
      { method: "POST", body: JSON.stringify(d) },
      s.token,
    ),
  getLog: (s) => request<LogResponse>(`/api/games/${encodeURIComponent(s.gameId)}/log`, {}, s.token),
  getCatalog: () => request<Catalog>("/api/catalog"),
};

// VITE_TRANSPORT=wasm (static GitHub Pages build): the engine runs in the browser as WebAssembly.
const realBackend: Backend = import.meta.env.VITE_TRANSPORT === "wasm" ? wasmBackend : httpBackend;

function backendFor(mock: boolean): Backend {
  return mock ? mockApi : realBackend;
}

export async function createGame(playerName: string, seed: number | undefined, mock: boolean) {
  const res = await backendFor(mock).createGame({ playerName, seed, bots: 2 });
  const session: Session = { gameId: res.gameId, playerId: res.playerId, token: res.token, mock };
  saveSession(session);
  return { session, view: res.view };
}

export const getGame = (s: Session) => backendFor(s.mock).getGame(s);
export const submitDecision = (s: Session, d: Decision) => backendFor(s.mock).submitDecision(s, d);
export const getLog = (s: Session) => backendFor(s.mock).getLog(s);
export const getCatalog = (mock: boolean) => backendFor(mock).getCatalog();
