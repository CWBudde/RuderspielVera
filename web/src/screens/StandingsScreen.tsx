import { Bot, Dice5, Flag, Info, Sparkles, Trophy, UserRound, Shuffle } from "lucide-react";
import type { GameView, LogEntry } from "@/api/types.gen";
import { useGame, useGameLog } from "@/hooks/game";
import { MedalIcon } from "@/components/icons";
import { playerColor } from "@/lib/labels";
import { cn } from "@/lib/utils";

export function rankedPlayers(view: GameView) {
  const sorted = [...view.players].sort((a, b) => b.points - a.points || b.gold - a.gold);
  // Competition ranking (1, 1, 3) on points; equal points share a place.
  return sorted.map((p) => ({ p, rank: 1 + sorted.filter((o) => o.points > p.points).length }));
}

export function MedalTable({ view }: { view: GameView }) {
  const rows = rankedPlayers(view);
  return (
    <div className="overflow-hidden rounded-[16px] bg-cream cut">
      <table className="w-full text-left">
        <thead className="bg-navy text-white">
          <tr className="font-display text-sm font-bold">
            <th className="py-2 pl-3">Platz</th>
            <th className="py-2">Team</th>
            <th className="py-2 text-center">
              <MedalIcon medal="gold" className="mx-auto size-5" />
              <span className="sr-only">Gold</span>
            </th>
            <th className="py-2 text-center">
              <MedalIcon medal="silber" className="mx-auto size-5" />
              <span className="sr-only">Silber</span>
            </th>
            <th className="py-2 pr-3 text-right">Punkte</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(({ p, rank }) => (
            <tr key={p.id} className={cn("border-t-2 border-ink/10", p.id === view.you && "bg-white/70")}>
              <td className="py-2.5 pl-3 num text-2xl leading-none">{rank}.</td>
              <td className="py-2.5">
                <span className="flex min-w-0 items-center gap-2">
                  <span className="size-3 shrink-0 rounded-full border border-ink" style={{ background: playerColor(view.players, p.id) }} />
                  <span className="truncate font-bold">{p.name}</span>
                  {p.isBot ? <Bot className="size-4 shrink-0 text-muted-foreground" aria-label="Computer" /> : <UserRound className="size-4 shrink-0 text-muted-foreground" aria-label="Mensch" />}
                </span>
              </td>
              <td className="py-2.5 text-center num text-xl">{p.gold}</td>
              <td className="py-2.5 text-center num text-xl">{p.silver}</td>
              <td className="py-2.5 pr-3 text-right num text-2xl">{p.points}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="border-t-2 border-ink/10 px-3 py-2 text-xs text-muted-foreground">Gold zählt 2 Punkte, Silber 1 Punkt.</p>
    </div>
  );
}

const KIND_ICON: Record<string, typeof Info> = {
  station: Flag,
  event: Sparkles,
  ereignis: Dice5,
  result: Trophy,
  draft: Shuffle,
  mission: Sparkles,
  info: Info,
};

export function StandingsScreen() {
  const { view } = useGame();
  const log = useGameLog();
  const entries: LogEntry[] = [...(log.data?.log ?? view.log ?? [])].reverse();
  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
      <section>
        <h2 className="mb-2 num text-3xl leading-none">Medaillenspiegel</h2>
        <MedalTable view={view} />
        {(view.activeEreignisse ?? []).length > 0 && (
          <div className="mt-4 rounded-[16px] border-2 border-ink bg-st-ereignis p-3">
            <h3 className="flex items-center gap-1.5 font-display text-lg font-bold">
              <Dice5 className="size-5" /> Aktive Ereignisse
            </h3>
            <ul className="mt-1 list-disc pl-5 text-sm">
              {view.activeEreignisse.map((t) => (
                <li key={t}>{t}</li>
              ))}
            </ul>
          </div>
        )}
      </section>
      <section>
        <h2 className="mb-2 num text-3xl leading-none">Saisonverlauf</h2>
        <ol className="flex flex-col gap-1.5">
          {entries.map((e, i) => {
            const Icon = KIND_ICON[e.kind] ?? Info;
            const station = view.calendar[e.stationIndex];
            return (
              <li
                key={i}
                className={cn(
                  "flex gap-2.5 rounded-xl border-2 border-ink/15 bg-cream/80 p-2.5 text-sm",
                  e.kind === "station" && "border-transparent bg-transparent py-1 font-display text-base font-bold",
                )}
              >
                <Icon className="mt-0.5 size-4 shrink-0" />
                <span className="min-w-0 flex-1">
                  {e.text}
                  {e.kind !== "station" && station && <span className="ml-1 text-xs text-muted-foreground">({station.name})</span>}
                </span>
              </li>
            );
          })}
        </ol>
      </section>
    </div>
  );
}
