import { createContext, useCallback, useContext, useMemo, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import type { Decision, GameView, PlayerSummary } from "@/api/types.gen";
import { ApiError, getGame, getLog, submitDecision, type LogResponse, type Session } from "@/api/client";
import { FALLBACK_POLL_MS, LIVE_POLL_MS } from "@/api/useGameEvents";
import { t } from "@/i18n";

interface GameContextValue {
  session: Session;
  view: GameView;
  you: PlayerSummary;
  submit: (d: Omit<Decision, "phaseId" | "kind">) => void;
  submitting: boolean;
  setView: (v: GameView) => void;
  leave: () => void;
}

const GameContext = createContext<GameContextValue | null>(null);

/** live: the server's event stream is connected, so polling is only a slow safety net. */
export function useGameQuery(session: Session | null, live = false) {
  return useQuery({
    queryKey: ["game", session?.gameId],
    queryFn: () => getGame(session as Session),
    enabled: !!session,
    retry: (count, err) => !(err instanceof ApiError && err.status >= 400 && err.status < 500) && count < 3,
    refetchInterval: (q) => (q.state.data?.phase.waiting ? (live ? LIVE_POLL_MS : FALLBACK_POLL_MS) : false),
  });
}

export function GameProvider({
  session,
  view,
  onLeave,
  children,
}: {
  session: Session;
  view: GameView;
  onLeave: () => void;
  children: ReactNode;
}) {
  const qc = useQueryClient();
  const key = useMemo(() => ["game", session.gameId], [session.gameId]);
  const setView = useCallback((v: GameView) => qc.setQueryData(key, v), [qc, key]);

  const mutation = useMutation({
    mutationFn: (d: Decision) => submitDecision(session, d),
    onSuccess: (v) => {
      setView(v);
      void qc.invalidateQueries({ queryKey: ["log", session.gameId] });
    },
    onError: (err) => {
      const msg = err instanceof ApiError ? err.message : t("common.unknownError");
      toast.error(msg);
      if (err instanceof ApiError && err.status === 409) void qc.invalidateQueries({ queryKey: key });
    },
  });

  const you = view.players.find((p) => p.id === view.you) ?? view.players[0];
  const value: GameContextValue = {
    session,
    view,
    you,
    submit: (d) => mutation.mutate({ ...d, phaseId: view.phase.id, kind: view.phase.kind }),
    submitting: mutation.isPending,
    setView,
    leave: onLeave,
  };
  return <GameContext.Provider value={value}>{children}</GameContext.Provider>;
}

export function useGame(): GameContextValue {
  const ctx = useContext(GameContext);
  if (!ctx) throw new Error("useGame outside GameProvider");
  return ctx;
}

export function useGameLog() {
  const { session, view } = useGame();
  return useQuery<LogResponse>({
    queryKey: ["log", session.gameId, view.phase.id],
    queryFn: () => getLog(session),
    staleTime: 10_000,
  });
}

export function playerName(view: GameView, id: string): string {
  return view.players.find((p) => p.id === id)?.name ?? id;
}
