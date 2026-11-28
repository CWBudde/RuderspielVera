import type { CatalogView, CompetitionResult, Decision, GameView, LogEntry } from "./types.gen";
import { readStorage, writeStorage } from "@/lib/storage";
import { ApiError, errorMsg } from "./errors";
import { wasmBackend } from "./wasmTransport";
import { t } from "@/i18n";

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
    throw new ApiError(0, t("api.noConnection"));
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
    throw new ApiError(res.status, errorMsg(body) ?? t("api.serverError", { status: res.status }));
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

// The in-browser mock (?mock=1) is loaded on demand, so it stays out of the main bundle.
const loadMock = () => import("./mock").then((m) => m.mockApi);
const lazyMock: Backend = {
  createGame: async (req) => (await loadMock()).createGame(req),
  getGame: async (s) => (await loadMock()).getGame(s),
  submitDecision: async (s, d) => (await loadMock()).submitDecision(s, d),
  getLog: async (s) => (await loadMock()).getLog(s),
  getCatalog: async () => (await loadMock()).getCatalog(),
};

function backendFor(mock: boolean): Backend {
  return mock ? lazyMock : realBackend;
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

// ---------- Multiplayer lobby (HTTP server only; not in the wasm or mock build) ----------

/** Multiplayer needs the Go server; the static wasm build and the mock run everything locally. */
export const multiplayerAvailable = import.meta.env.VITE_TRANSPORT !== "wasm";

export interface LobbyPlayer {
  id: string;
  name: string;
  isBot: boolean;
}

/** Mirrors gameservice.LobbyView. */
export interface LobbyView {
  gameId: string;
  code: string;
  seats: number;
  players: LobbyPlayer[];
  you: string;
  host: boolean;
  started: boolean;
}

interface LobbyResponse {
  gameId: string;
  playerId: string;
  token: string;
  lobby: LobbyView;
}

function lobbySession(res: LobbyResponse): Session {
  const session: Session = { gameId: res.gameId, playerId: res.playerId, token: res.token, mock: false };
  saveSession(session);
  return session;
}

export async function createLobby(playerName: string, seats: number, seed: number | undefined) {
  const res = await request<LobbyResponse>("/api/lobbies", {
    method: "POST",
    body: JSON.stringify({ playerName, seats, seed }),
  });
  return { session: lobbySession(res), lobby: res.lobby };
}

export async function joinLobby(code: string, playerName: string) {
  const res = await request<LobbyResponse>(`/api/lobbies/${encodeURIComponent(code.trim())}/join`, {
    method: "POST",
    body: JSON.stringify({ playerName }),
  });
  return { session: lobbySession(res), lobby: res.lobby };
}

export const getLobby = (s: Session) =>
  request<LobbyView>(`/api/games/${encodeURIComponent(s.gameId)}/lobby`, {}, s.token);

export const startLobby = (s: Session) =>
  request<GameView>(`/api/games/${encodeURIComponent(s.gameId)}/start`, { method: "POST" }, s.token);

/** The invite link for a lobby code (opens the start screen with the code filled in). */
export function joinLink(code: string): string {
  const url = new URL(import.meta.env.BASE_URL, window.location.origin);
  url.searchParams.set("join", code);
  return url.toString();
}

/** Reads ?join=CODE from the address bar. */
export function joinCodeFromUrl(): string {
  try {
    return new URLSearchParams(window.location.search).get("join")?.trim().toUpperCase() ?? "";
  } catch {
    return "";
  }
}

// ---------- Resume link: continue a seat on another device ----------

/** A seat can move to another device only if it lives on the Go server (not wasm, not the mock). */
export const resumeAvailable = (s: Session) => multiplayerAvailable && !s.mock;

// Game ids and tokens are hex, player ids "p1".."p4": "gameId.playerId.token" is compact and URL-safe.
const RESUME_PART = /^[A-Za-z0-9_-]{1,128}$/;

/**
 * The personal resume link for a seat (?resume=gameId.playerId.token). It carries the seat token,
 * so whoever opens it controls the seat.
 */
export function resumeLink(s: Session): string {
  const url = new URL(import.meta.env.BASE_URL, window.location.origin);
  url.searchParams.set("resume", [s.gameId, s.playerId, s.token].join("."));
  return url.toString();
}

/**
 * Reads ?resume=… from the address bar: undefined if absent, null if present but malformed.
 * Ignored in builds without the server and in mock mode.
 */
export function resumeSessionFromUrl(): Session | null | undefined {
  if (!multiplayerAvailable || isMockRequested()) return undefined;
  let raw: string | null = null;
  try {
    raw = new URLSearchParams(window.location.search).get("resume");
  } catch {
    return undefined;
  }
  if (raw === null) return undefined;
  const parts = raw.trim().split(".");
  if (parts.length !== 3 || !parts.every((p) => RESUME_PART.test(p))) return null;
  const [gameId, playerId, token] = parts;
  return { gameId, playerId, token, mock: false };
}

/**
 * Checks a seat against the server. Resolves with the game view, or null when the game is a lobby
 * that has not started yet (409); rejects with the ApiError otherwise (e.g. 401/404 for a bad link).
 */
export async function validateSession(s: Session): Promise<GameView | null> {
  try {
    return await getGame(s);
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      await getLobby(s); // the lobby endpoint checks the token
      return null;
    }
    throw e;
  }
}
