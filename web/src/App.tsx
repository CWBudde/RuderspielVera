import { lazy, Suspense, useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import {
  ApiError,
  createGame,
  createLobby,
  isMockRequested,
  joinCodeFromUrl,
  joinLobby,
  loadSession,
  resumeSessionFromUrl,
  saveSession,
  validateSession,
  type Session,
} from "@/api/client";
import { useGameEvents } from "@/api/useGameEvents";
import { GameProvider, useGameQuery } from "@/hooks/game";
import { StartScreen } from "@/screens/StartScreen";
import { LobbyScreen } from "@/screens/LobbyScreen";
// The game screens load after the start screen (route-level code split).
const AppShell = lazy(() => import("@/screens/AppShell").then((m) => ({ default: m.AppShell })));
import { Button } from "@/components/ui/button";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { t, useLang } from "@/i18n";

export function App() {
  // Subscribing here re-renders the whole tree when the language changes.
  useLang();
  const mockRequested = isMockRequested();
  // A resume link (?resume=…) moves a seat to this device; it wins over everything else.
  const [resume] = useState(() => resumeSessionFromUrl());
  // An invite link (?join=CODE) opens the start screen with the code filled in.
  const [joinCode] = useState(() => (resume === undefined ? joinCodeFromUrl() : ""));
  const [stored, setStored] = useState<Session | null>(() => loadSession());
  const [session, setSession] = useState<Session | null>(() => {
    const s = loadSession();
    return s && (!mockRequested || s.mock) && !joinCode && resume === undefined ? s : null;
  });
  const [creating, setCreating] = useState(false);
  const [resuming, setResuming] = useState(resume !== undefined);
  const qc = useQueryClient();

  // Validate the resume link before it replaces the stored session, so a bad link never costs
  // this device its current seat. The ref keeps StrictMode's double effect from running it twice.
  const resumeHandled = useRef(false);
  useEffect(() => {
    if (resume === undefined || resumeHandled.current) return;
    resumeHandled.current = true;
    window.history.replaceState(null, "", window.location.pathname);
    void (async () => {
      try {
        if (!resume) throw new ApiError(0, t("resume.invalid"));
        const view = await validateSession(resume);
        if (view) qc.setQueryData(["game", resume.gameId], view);
        saveSession(resume);
        setStored(resume);
        setSession(resume);
      } catch (e) {
        // Deferred a tick: the Toaster mounts after App and would miss a toast from this effect.
        setTimeout(() => toast.error(e instanceof ApiError ? e.message : t("resume.invalid")));
      } finally {
        setResuming(false);
      }
    })();
  }, [resume, qc]);

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

  // Multiplayer: the session is stored right away; GameRoot shows the lobby until the start.
  const enterLobby = async (open: () => Promise<{ session: Session }>) => {
    setCreating(true);
    try {
      const { session: s } = await open();
      if (joinCode) window.history.replaceState(null, "", window.location.pathname);
      setStored(s);
      setSession(s);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : t("app.startFailed"));
    } finally {
      setCreating(false);
    }
  };

  if (resuming) return <Loading />;
  if (!session) {
    return (
      <StartScreen
        stored={stored}
        mock={mockRequested}
        busy={creating}
        onCreate={create}
        onCreateLobby={(name, seats, seed) => void enterLobby(() => createLobby(name, seats, seed))}
        onJoin={(name, code) => void enterLobby(() => joinLobby(code, name))}
        onResume={() => setSession(stored)}
        initialCode={joinCode}
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
  // One event stream serves both the lobby and the game view; solo games (one human) need none.
  const [multiHuman, setMultiHuman] = useState(false);
  const live = useGameEvents(session, multiHuman);
  const q = useGameQuery(session, live);
  const lost = q.error instanceof ApiError && [401, 403, 404].includes(q.error.status);
  // 409 on the game view = a multiplayer lobby that has not started yet.
  const inLobby = q.error instanceof ApiError && q.error.status === 409;
  const humans = q.data ? q.data.players.filter((p) => !p.isBot).length : 0;
  const wantsStream = inLobby || humans > 1;
  // Sticky: a refetch error or the lobby-to-game switch must not drop the stream.
  if (wantsStream && !multiHuman) setMultiHuman(true);
  const refetch = q.refetch;
  const onStarted = useCallback(() => void refetch(), [refetch]);

  useEffect(() => {
    if (lost && q.error) {
      toast.error(q.error.message);
      onLost();
    }
  }, [lost, q.error, onLost]);

  if (inLobby && !q.data) {
    return <LobbyScreen session={session} live={live} onStarted={onStarted} onLeave={onLeave} />;
  }
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
