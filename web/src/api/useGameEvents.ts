import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { multiplayerAvailable, type Session } from "./client";

/** Polling interval while the event stream is connected (only a safety net). */
export const LIVE_POLL_MS = 15_000;
/** Polling interval without a stream (wasm/mock build, no EventSource, stream errored). */
export const FALLBACK_POLL_MS = 2_000;

/** URL of the server's event stream; EventSource cannot send headers, so the token goes in the query. */
function eventsUrl(gameId: string, token: string): string {
  return `/api/games/${encodeURIComponent(gameId)}/events?token=${encodeURIComponent(token)}`;
}

/**
 * Listens to GET /api/games/{id}/events (Server-Sent Events) and refetches the game and lobby
 * queries whenever the server reports a change. Returns whether the stream is connected, so
 * callers can poll slowly while it is and fall back to fast polling while it is not.
 * Does nothing for mock sessions, in the wasm build, or when enabled is false.
 */
export function useGameEvents(session: Session | null, enabled = true): boolean {
  const qc = useQueryClient();
  const [connected, setConnected] = useState(false);
  const gameId = session?.gameId;
  const token = session?.token;
  const active = enabled && !!session && !session.mock && multiplayerAvailable && typeof EventSource !== "undefined";

  useEffect(() => {
    if (!active || !gameId || !token) return;
    const refresh = () => {
      void qc.invalidateQueries({ queryKey: ["game", gameId] });
      void qc.invalidateQueries({ queryKey: ["lobby", gameId] });
    };
    const es = new EventSource(eventsUrl(gameId, token));
    es.onopen = () => {
      setConnected(true);
      // Changes between the last fetch and (re)connecting would otherwise go unnoticed.
      refresh();
    };
    // EventSource reconnects on its own after network drops; on a non-200 answer (e.g. a bad
    // token) it gives up (readyState CLOSED). Either way, poll fast until it is open again.
    es.onerror = () => setConnected(false);
    es.addEventListener("change", refresh);
    return () => {
      es.close();
      setConnected(false);
    };
  }, [active, gameId, token, qc]);

  return active && connected;
}
