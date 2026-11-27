import { useMemo, useState } from "react";
import { ChevronDown, Minus, Plus, RotateCcw, Send } from "lucide-react";
import type { RowerView, Stat, TrainingAlloc } from "@/api/types.gen";
import { useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import { Art } from "@/components/Art";
import { AgeBadge, GenderMark, MiniStats } from "@/components/RowerCard";
import { StatBadge } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { ALL_STATS, STAT_LABEL } from "@/lib/labels";
import { cn } from "@/lib/utils";
import { sortRowers } from "@/screens/TeamScreen";

type Alloc = Record<string, Partial<Record<Stat, number>>>;

function rowerTotal(a: Alloc, id: string): number {
  return Object.values(a[id] ?? {}).reduce<number>((s, n) => s + (n ?? 0), 0);
}

export function TrainingPhase() {
  const { view, you, submit, submitting } = useGame();
  const points = view.phase.points ?? 10;
  const maxRowers = view.phase.maxRowers ?? 0;
  const [alloc, setAlloc] = useState<Alloc>({});
  const [open, setOpen] = useState<Set<string>>(new Set());
  const rowers = useMemo(() => sortRowers(you.rowers), [you.rowers]);

  const used = Object.keys(alloc).reduce((s, id) => s + rowerTotal(alloc, id), 0);
  const remaining = points - used;
  const trainedRowers = Object.keys(alloc).filter((id) => rowerTotal(alloc, id) > 0);
  const atRowerLimit = maxRowers > 0 && trainedRowers.length >= maxRowers;

  const change = (id: string, stat: Stat, delta: number) => {
    setAlloc((prev) => {
      const cur = prev[id]?.[stat] ?? 0;
      const next = Math.max(0, cur + delta);
      return { ...prev, [id]: { ...prev[id], [stat]: next } };
    });
  };

  const canAddTo = (id: string) => remaining > 0 && (!atRowerLimit || rowerTotal(alloc, id) > 0);
  const valid = remaining === 0 && (maxRowers === 0 || trainedRowers.length <= maxRowers);

  const send = () => {
    const training: TrainingAlloc[] = [];
    for (const [rowerId, stats] of Object.entries(alloc)) {
      for (const [stat, n] of Object.entries(stats)) if (n && n > 0) training.push({ rowerId, stat, points: n });
    }
    submit({ training });
  };

  return (
    <PhaseFrame
      title={view.phase.title}
      prompt={view.phase.prompt}
      headerExtra={
        <div className="mt-3 flex flex-wrap items-center gap-2">
          {maxRowers > 0 && (
            <span
              className={cn(
                "inline-flex h-8 items-center rounded-full border-2 border-ink px-3 text-sm font-bold",
                atRowerLimit ? "bg-st-training" : "bg-white",
              )}
            >
              max. {maxRowers} Ruderer, {trainedRowers.length} gewählt
            </span>
          )}
        </div>
      }
      footer={
        <div className="flex items-center gap-2">
          <div className="min-w-0 flex-1 pl-1" aria-live="polite">
            <p className="flex items-baseline gap-1.5">
              <span className={cn("num text-3xl leading-none", remaining === 0 ? "text-leaf" : "text-ink")}>{remaining}</span>
              <span className="text-sm font-semibold">von {points} Punkten übrig</span>
            </p>
            <div className="mt-1 h-2 overflow-hidden rounded-full border border-ink bg-white">
              <div className="h-full bg-st-training transition-[width]" style={{ width: `${(used / points) * 100}%` }} />
            </div>
          </div>
          <Button variant="outline" size="icon" aria-label="Zurücksetzen" onClick={() => setAlloc({})} disabled={used === 0}>
            <RotateCcw />
          </Button>
          <Button size="lg" onClick={send} disabled={!valid || submitting}>
            <Send /> Trainieren
          </Button>
        </div>
      }
    >
      <ul className="grid gap-2 lg:grid-cols-2 xl:grid-cols-3">
        {rowers.map((r) => (
          <TrainRow
            key={r.id}
            rower={r}
            alloc={alloc[r.id] ?? {}}
            open={open.has(r.id)}
            onToggle={() =>
              setOpen((s) => {
                const n = new Set(s);
                if (n.has(r.id)) n.delete(r.id);
                else n.add(r.id);
                return n;
              })
            }
            canAdd={canAddTo(r.id)}
            locked={atRowerLimit && rowerTotal(alloc, r.id) === 0}
            onChange={(stat, d) => change(r.id, stat, d)}
          />
        ))}
      </ul>
    </PhaseFrame>
  );
}

function TrainRow({
  rower,
  alloc,
  open,
  onToggle,
  canAdd,
  locked,
  onChange,
}: {
  rower: RowerView;
  alloc: Partial<Record<Stat, number>>;
  open: boolean;
  onToggle: () => void;
  canAdd: boolean;
  locked: boolean;
  onChange: (stat: Stat, delta: number) => void;
}) {
  const total = Object.values(alloc).reduce<number>((s, n) => s + (n ?? 0), 0);
  const preview: RowerView = {
    ...rower,
    stats: Object.fromEntries(ALL_STATS.map((s) => [s, (rower.stats[s] ?? 0) + (alloc[s] ?? 0)])),
  };
  return (
    <li className={cn("rounded-[14px] border-2 border-ink bg-cream", total > 0 && "ring-4 ring-st-training", locked && "opacity-60")}>
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        className="flex min-h-14 w-full items-center gap-2.5 p-1.5 pr-2 text-left lg:pointer-events-none"
      >
        <div className="size-11 shrink-0 overflow-hidden rounded-lg border-2 border-ink">
          <Art artKey={rower.artKey} cardId={rower.id} label={rower.name} thumb />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate num text-lg leading-none text-navy">{rower.name}</span>
            <AgeBadge age={rower.age} short className="h-5 px-1.5 text-xs" />
            <GenderMark gender={rower.gender} className="size-5 text-[11px]" />
          </div>
          <MiniStats rower={preview} className={cn(open && "max-lg:hidden", "lg:hidden")} />
        </div>
        {total > 0 && <span className="num rounded-full border-2 border-ink bg-st-training px-2 text-lg leading-6">+{total}</span>}
        <ChevronDown className={cn("size-5 shrink-0 transition-transform lg:hidden", open && "rotate-180")} />
      </button>
      <div className={cn("border-t-2 border-ink/10 px-2 pt-1 pb-2", open ? "block" : "hidden", "lg:block")}>
        {ALL_STATS.map((s) => {
          const n = alloc[s] ?? 0;
          return (
            <div key={s} className="flex items-center gap-2 py-0.5">
              <StatBadge stat={s} className="size-7" />
              <span className="min-w-0 flex-1 truncate text-sm font-semibold">{STAT_LABEL[s]}</span>
              <span className="w-14 text-right tabular-nums">
                <span className="num text-xl">{(rower.stats[s] ?? 0) + n}</span>
                {n > 0 && <span className="ml-0.5 text-xs font-bold text-leaf">+{n}</span>}
              </span>
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  className="grid size-11 place-items-center rounded-xl border-2 border-ink bg-white disabled:opacity-35"
                  onClick={() => onChange(s, -1)}
                  disabled={n === 0}
                  aria-label={`${STAT_LABEL[s]} von ${rower.name} verringern`}
                >
                  <Minus className="size-4" strokeWidth={3} />
                </button>
                <button
                  type="button"
                  className="grid size-11 place-items-center rounded-xl border-2 border-ink bg-st-training shadow-[0_2px_0_var(--ink)] active:translate-y-[2px] active:shadow-none disabled:opacity-35 disabled:shadow-none"
                  onClick={() => onChange(s, 1)}
                  disabled={!canAdd}
                  aria-label={`${STAT_LABEL[s]} von ${rower.name} erhöhen`}
                >
                  <Plus className="size-4" strokeWidth={3} />
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </li>
  );
}
