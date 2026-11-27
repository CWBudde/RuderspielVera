/**
 * WebAssembly transport: the Go rules engine + bots (cmd/wasm) run in the browser, so the app can
 * be hosted fully static (GitHub Pages). Selected with VITE_TRANSPORT=wasm at build time.
 *
 * The Go side registers globalThis.ruderspielWasm; every call takes/returns JSON strings shaped
 * {status, data} or {status, error} and is mapped to the same results/ApiErrors as the HTTP client.
 * Persistence: after every change the game snapshot (seed + decision log) is written to
 * localStorage and re-imported (engine replay) the first time the game is used after a reload.
 */
import type { Backend, Catalog, CreateGameRequest, CreateGameResponse, LogResponse, Session } from "./client";
import type { Decision, GameView } from "./types.gen";
import { ApiError, errorMsg } from "./errors";
import type { Msg } from "@/i18n";
import { readStorage, writeStorage } from "@/lib/storage";
import { t } from "@/i18n";

interface WasmApi {
  createGame(req: string): string;
  getView(gameId: string, token: string): string;
  submitDecision(gameId: string, token: string, decision: string): string;
  getLog(gameId: string, token: string): string;
  getCatalog(): string;
  exportGame(gameId: string): string;
  importGame(snapshot: string): string;
}

type Envelope<T> = { status: number; data: T; error?: undefined } | { status: number; error: string; msg?: Msg };

declare global {
  // wasm_exec.js defines Go; cmd/wasm defines ruderspielWasm.
  var Go: new () => { importObject: WebAssembly.Imports; run(instance: WebAssembly.Instance): Promise<void> };
  var ruderspielWasm: WasmApi | undefined;
}

const GAME_KEY_PREFIX = "ruderspiel.wasm.game.";

let loading: Promise<WasmApi> | null = null;
const imported = new Set<string>();
/** Why a stored game could not be restored (e.g. a save from an older rules version), by game ID. */
const importErrors = new Map<string, string>();

/** URL of a file in public/. VITE_WASM_VERSION (content hash, set by `just pages`) busts caches:
 * ruderspiel.wasm and wasm_exec.js have fixed names, and a stale engine breaks saved games. */
function assetUrl(file: string): string {
  const v = import.meta.env.VITE_WASM_VERSION as string | undefined;
  return `${import.meta.env.BASE_URL.replace(/\/?$/, "/")}${file}${v ? `?v=${encodeURIComponent(v)}` : ""}`;
}

function loadScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const el = document.createElement("script");
    el.src = src;
    el.async = true;
    el.onload = () => resolve();
    el.onerror = () => reject(new Error(`failed to load ${src}`));
    document.head.appendChild(el);
  });
}

async function instantiate(go: InstanceType<typeof Go>): Promise<WebAssembly.Instance> {
  const url = assetUrl("ruderspiel.wasm");
  try {
    const { instance } = await WebAssembly.instantiateStreaming(fetch(url), go.importObject);
    return instance;
  } catch {
    // Wrong MIME type or no streaming support: fall back to a plain download.
    const res = await fetch(url);
    if (!res.ok) throw new Error(`${url}: ${res.status}`);
    const { instance } = await WebAssembly.instantiate(await res.arrayBuffer(), go.importObject);
    return instance;
  }
}

function load(): Promise<WasmApi> {
  loading ??= (async () => {
    if (typeof globalThis.Go !== "function") await loadScript(assetUrl("wasm_exec.js"));
    const go = new globalThis.Go();
    const instance = await instantiate(go);
    void go.run(instance); // resolves only when Go exits; main registers the API synchronously
    const api = globalThis.ruderspielWasm;
    if (!api) throw new Error("ruderspielWasm not registered");
    return api;
  })().catch((err: unknown) => {
    loading = null; // allow a retry
    console.error("wasm load failed", err);
    throw new ApiError(0, t("api.engineLoadFailed"));
  });
  return loading;
}

function unwrap<T>(raw: string): T {
  let env: Envelope<T>;
  try {
    env = JSON.parse(raw) as Envelope<T>;
  } catch {
    throw new ApiError(500, t("api.internalError"));
  }
  if (env.error !== undefined) throw new ApiError(env.status, errorMsg(env) ?? env.error);
  return env.data;
}

/** Writes the game's snapshot to localStorage (no-op if storage is unavailable). */
function persist(api: WasmApi, gameId: string): void {
  try {
    // An opaque JSON string: never parse it, the uint64 seed would lose precision.
    const snap = unwrap<unknown>(api.exportGame(gameId));
    if (typeof snap !== "string") throw new Error("exportGame: expected a JSON string (stale engine?)");
    writeStorage(GAME_KEY_PREFIX + gameId, snap);
  } catch (err) {
    console.warn("could not persist game", err);
  }
}

/** Drops stored snapshots of other games so localStorage does not fill up. */
function forgetOtherGames(keep: string): void {
  try {
    const keys: string[] = [];
    for (let i = 0; i < window.localStorage.length; i++) {
      const k = window.localStorage.key(i);
      if (k?.startsWith(GAME_KEY_PREFIX) && k !== GAME_KEY_PREFIX + keep) keys.push(k);
    }
    keys.forEach((k) => writeStorage(k, null));
  } catch {
    /* storage unavailable */
  }
}

/** Loads the engine and, once per page load, re-imports the session's game from localStorage. */
async function ready(gameId?: string): Promise<WasmApi> {
  const api = await load();
  if (gameId && !imported.has(gameId)) {
    imported.add(gameId);
    const raw = readStorage(GAME_KEY_PREFIX + gameId);
    if (raw) {
      try {
        unwrap(api.importGame(raw));
      } catch (err) {
        console.warn("stored game could not be restored", err); // getView then reports 404
        if (err instanceof ApiError) importErrors.set(gameId, err.message);
      }
    }
  }
  return api;
}

export const wasmBackend: Backend = {
  async createGame(req: CreateGameRequest) {
    const api = await ready();
    const res = unwrap<CreateGameResponse>(api.createGame(JSON.stringify(req)));
    imported.add(res.gameId);
    persist(api, res.gameId);
    forgetOtherGames(res.gameId);
    return res;
  },
  async getGame(s: Session) {
    const api = await ready(s.gameId);
    try {
      return unwrap<GameView>(api.getView(s.gameId, s.token));
    } catch (err) {
      // The game is unknown because its save could not be restored: show the reason instead.
      const reason = importErrors.get(s.gameId);
      if (reason && err instanceof ApiError && err.status === 404) throw new ApiError(404, reason);
      throw err;
    }
  },
  async submitDecision(s: Session, d: Decision) {
    const api = await ready(s.gameId);
    try {
      return unwrap<GameView>(api.submitDecision(s.gameId, s.token, JSON.stringify(d)));
    } finally {
      persist(api, s.gameId); // also after a bot error: the human's decision may already be in the log
    }
  },
  async getLog(s: Session) {
    const api = await ready(s.gameId);
    return unwrap<LogResponse>(api.getLog(s.gameId, s.token));
  },
  async getCatalog() {
    const api = await ready();
    return unwrap<Catalog>(api.getCatalog());
  },
};
