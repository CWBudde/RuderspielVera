import { lazy, Suspense, useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import { ApiError, createGame, isMockRequested, loadSession, saveSession, type Session } from "@/api/client";
import { GameProvider, useGameQuery } from "@/hooks/game";
import { StartScreen } from "@/screens/StartScreen";
// The game screens load after the start screen (route-level code split).
const AppShell = lazy(() => import("@/screens/AppShell").then((m) => ({ default: m.AppShell })));
import { Button } from "@/components/ui/button";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { t, useLang } from "@/i18n";

export function App() {
  // Subscribing here re-renders the whole tree when the language changes.
  useLang();
  const mockRequested = isMockRequested();
  const [stored, setStored] = useState<Session | null>(() => loadSession());
  const [session, setSession] = useState<Session | null>(() => {
    const s = loadSession();
    return s && (!mockRequested || s.mock) ? s : null;
  });
  const [creating, setCreating] = useState(false);
  const qc = useQueryClient();

  const create = async (name: string, seed: number | undefined) => {
    setCreating(true);
    try {
      const { session: s, view } = await createGame(name, seed, mockRequested);
      qc.setQueryData(["game", s.gameId], view);
      setStored(s);
      setSession(s);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : t("app.startFailed"));
    } finally {
      setCreating(false);
    }
  };

  if (!session) {
    return (
      <StartScreen
        stored={stored}
        mock={mockRequested}
        busy={creating}
        onCreate={create}
        onResume={() => setSession(stored)}
      />
    );
  }
  return (
    <GameRoot
      session={session}
      onLeave={() => setSession(null)}
      onLost={() => {
        saveSession(null);
        setStored(null);
        setSession(null);
      }}
    />
  );
}

function GameRoot({ session, onLeave, onLost }: { session: Session; onLeave: () => void; onLost: () => void }) {
  const q = useGameQuery(session);
  const lost = q.error instanceof ApiError && [401, 403, 404].includes(q.error.status);

  useEffect(() => {
    if (lost && q.error) {
      toast.error(q.error.message);
      onLost();
    }
  }, [lost, q.error, onLost]);

  if (q.data) {
    return (
      <ErrorBoundary onHome={onLeave}>
        <GameProvider session={session} view={q.data} onLeave={onLeave}>
          <Suspense fallback={<Loading />}>
            <AppShell />
          </Suspense>
        </GameProvider>
      </ErrorBoundary>
    );
  }
  if (q.isError && !lost) {
    return (
      <div className="grid min-h-dvh place-items-center p-6 text-center">
        <div className="max-w-sm rounded-[20px] bg-cream p-5 cut">
          <p className="font-display text-xl font-bold">{t("app.unreachable")}</p>
          <p className="mt-1 text-sm text-muted-foreground">
            {q.error instanceof ApiError ? q.error.message : t("common.unknownError")}
          </p>
          <div className="mt-4 flex flex-col gap-2">
            <Button onClick={() => void q.refetch()}>{t("app.retry")}</Button>
            <Button variant="outline" onClick={onLeave}>
              {t("app.toStart")}
            </Button>
          </div>
        </div>
      </div>
    );
  }
  return <Loading />;
}

function Loading() {
  return (
    <div className="grid min-h-dvh place-items-center" role="status">
      <Loader2 className="size-8 animate-spin text-navy" aria-label={t("app.loading")} />
    </div>
  );
}
