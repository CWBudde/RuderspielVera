import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Bot, Check, Copy, Crown, Loader2, Play, Share2, User } from "lucide-react";
import { ApiError, getLobby, joinLink, startLobby, type Session } from "@/api/client";
import { Button } from "@/components/ui/button";
import { LanguageToggle } from "@/components/LanguageToggle";
import { t } from "@/i18n";

/**
 * Waiting room of a multiplayer game: shows the join code and invite link, the seats taken so far,
 * and (for the host) the start button. Polls the lobby; once the game has started, onStarted
 * switches to the game view for everyone.
 */
export function LobbyScreen({
  session,
  onStarted,
  onLeave,
}: {
  session: Session;
  onStarted: () => void;
  onLeave: () => void;
}) {
  const qc = useQueryClient();
  const q = useQuery({
    queryKey: ["lobby", session.gameId],
    queryFn: () => getLobby(session),
    refetchInterval: (query) => (query.state.data?.started ? false : 2000),
    retry: (count, err) => !(err instanceof ApiError && err.status >= 400 && err.status < 500) && count < 3,
  });
  const lobby = q.data;

  const start = useMutation({
    mutationFn: () => startLobby(session),
    onSuccess: (view) => {
      qc.setQueryData(["game", session.gameId], view);
      onStarted();
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : t("common.unknownError")),
  });

  useEffect(() => {
    if (lobby?.started) onStarted();
  }, [lobby?.started, onStarted]);

  const [copied, setCopied] = useState(false);
  const link = lobby ? joinLink(lobby.code) : "";
  const share = async () => {
    if (!lobby) return;
    const data = { title: t("app.title"), text: t("lobby.shareText", { code: lobby.code }), url: link };
    try {
      if (navigator.share) {
        await navigator.share(data);
        return;
      }
      await navigator.clipboard.writeText(link);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* share sheet dismissed or clipboard blocked: the code is on screen anyway */
    }
  };

  const free = lobby ? lobby.seats - lobby.players.length : 0;

  return (
    <div className="min-h-dvh bg-navy px-4 pt-[calc(var(--safe-top)+1rem)] pb-[calc(var(--safe-bottom)+1.5rem)]">
      <div className="mx-auto flex max-w-md justify-end">
        <LanguageToggle />
      </div>
      <main className="mx-auto mt-3 max-w-md rounded-[20px] bg-cream p-4 cut md:p-5">
        <h1 className="num text-[2rem] leading-none text-ink">{t("lobby.title")}</h1>

        {!lobby ? (
          q.isError ? (
            <p className="mt-3 text-sm">{q.error instanceof ApiError ? q.error.message : t("app.unreachable")}</p>
          ) : (
            <div className="grid place-items-center py-10" role="status">
              <Loader2 className="size-8 animate-spin text-navy" aria-label={t("app.loading")} />
            </div>
          )
        ) : (
          <>
            <p className="mt-2 text-sm text-ink/80">{t("lobby.codeHint")}</p>
            <div className="mt-2 flex items-stretch gap-2">
              <output
                aria-label={t("lobby.codeLabel")}
                className="flex flex-1 items-center justify-center rounded-xl border-2 border-ink bg-white font-mono text-3xl font-bold tracking-[0.25em] select-all"
              >
                {lobby.code}
              </output>
              <Button
                type="button"
                size="lg"
                variant="outline"
                onClick={() => void share()}
                aria-label={t("lobby.share")}
              >
                {copied ? <Check /> : "share" in navigator ? <Share2 /> : <Copy />}
              </Button>
            </div>

            <h2 className="mt-5 font-display text-lg font-bold">
              {t("lobby.seats", { taken: lobby.players.length, seats: lobby.seats })}
            </h2>
            <ul className="mt-2 flex flex-col gap-2">
              {lobby.players.map((p) => (
                <li
                  key={p.id}
                  className="flex min-h-12 items-center gap-3 rounded-xl border-2 border-ink bg-white px-3"
                >
                  {p.isBot ? <Bot className="size-5 text-lake" /> : <User className="size-5 text-navy" />}
                  <span className="font-bold">{p.name}</span>
                  {p.id === lobby.you && <span className="text-sm text-muted-foreground">({t("lobby.you")})</span>}
                  {p.id === "p1" && (
                    <Crown className="ml-auto size-5 text-amber-600" aria-label={t("lobby.hostBadge")} />
                  )}
                </li>
              ))}
              {Array.from({ length: free }, (_, i) => (
                <li
                  key={`free-${i}`}
                  className="flex min-h-12 items-center gap-3 rounded-xl border-2 border-dashed border-ink/40 px-3 text-muted-foreground"
                >
                  <Loader2 className="size-5 animate-spin" />
                  {t("lobby.freeSeat")}
                </li>
              ))}
            </ul>

            {lobby.host ? (
              <>
                <Button
                  type="button"
                  size="lg"
                  className="mt-5 w-full"
                  disabled={start.isPending}
                  onClick={() => start.mutate()}
                >
                  <Play className="fill-current" />
                  {start.isPending ? t("start.shuffling") : t("lobby.start")}
                </Button>
                {free > 0 && (
                  <p className="mt-2 text-center text-sm text-muted-foreground">
                    {t("lobby.botsFill", { count: free })}
                  </p>
                )}
              </>
            ) : (
              <p className="mt-5 flex items-center justify-center gap-2 text-center font-semibold text-ink/80">
                <Loader2 className="size-4 animate-spin" /> {t("lobby.waitingForHost")}
              </p>
            )}
          </>
        )}

        <Button type="button" variant="outline" className="mt-3 w-full" onClick={onLeave}>
          {t("app.toStart")}
        </Button>
      </main>
    </div>
  );
}
