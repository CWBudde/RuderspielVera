import type { ReactNode } from "react";
import { Sparkles } from "lucide-react";
import type { RowerView, Stat } from "@/api/types.gen";
import { Art } from "@/components/Art";
import { Emblem, StatBadge } from "@/components/icons";
import { AGE_LABEL, AGE_SHORT, CORE_STATS, STAT_LABEL, STAT_MEDIUM, STAT_SHORT, ALL_STATS } from "@/lib/labels";
import { cn } from "@/lib/utils";

const AGE_TONE: Record<string, string> = {
  kinder: "bg-[#ffe7a3]",
  b: "bg-[#cfe8c0]",
  a: "bg-[#cfe0f5]",
};

export function AgeBadge({ age, short, className }: { age: string; short?: boolean; className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex h-6 items-center rounded-full border-2 border-ink px-2 font-display text-[13px] leading-none font-bold text-ink",
        AGE_TONE[age] ?? "bg-muted",
        className,
      )}
      title={AGE_LABEL[age]}
    >
      {short ? AGE_SHORT[age] : (AGE_LABEL[age] ?? age)}
    </span>
  );
}

export function GenderMark({ gender, className }: { gender: string; className?: string }) {
  const female = gender === "w";
  return (
    <span
      className={cn(
        "inline-grid size-6 place-items-center rounded-full border-2 border-ink text-[13px] leading-none font-bold",
        female ? "bg-[#f6b9cc] text-[#5a1f33]" : "bg-[#acd5f3] text-[#173a5c]",
        className,
      )}
      title={female ? "Mädchen" : "Junge"}
      aria-label={female ? "Mädchen" : "Junge"}
    >
      {female ? "♀" : "♂"}
    </span>
  );
}

/** Summe: all five stats incl. Motivation (the Regelwerk's rank), plus optional preview points. */
export function statSum(r: RowerView, preview?: Partial<Record<Stat, number>>): number {
  return ALL_STATS.reduce((t, s) => t + (r.stats[s] ?? 0) + (preview?.[s] ?? 0), 0);
}

/** Compact "Summe 87" tag for list rows; with add > 0 it shows the change, e.g. "Summe 87 → 92". */
export function SumTag({ rower, add = 0, className }: { rower: RowerView; add?: number; className?: string }) {
  const sum = statSum(rower);
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-baseline gap-1 text-[11px] font-bold whitespace-nowrap text-muted-foreground",
        className,
      )}
      aria-label={add > 0 ? `Summe ${sum}, nach dem Training ${sum + add}` : `Summe ${sum}`}
      title="Summe aller fünf Werte"
    >
      Summe
      <span className="num text-base leading-none text-ink">{sum}</span>
      {add > 0 && (
        <>
          <span aria-hidden>→</span>
          <span className="num text-base leading-none text-leaf">{sum + add}</span>
        </>
      )}
    </span>
  );
}

export function gain(r: RowerView, s: Stat): number {
  return (r.stats[s] ?? 0) - (r.base?.[s] ?? r.stats[s] ?? 0);
}

function Gain({ n, className }: { n: number; className?: string }) {
  if (n === 0) return null;
  return (
    <span className={cn("num text-[0.55em] leading-none", n > 0 ? "text-leaf" : "text-destructive", className)}>
      {n > 0 ? `+${n}` : n}
    </span>
  );
}

interface RowerCardProps {
  rower: RowerView;
  className?: string;
  /** Overrides shown values (e.g. training preview). */
  preview?: Partial<Record<Stat, number>>;
}

/** Full trading card in the vera-karte style. Uses container queries to stay readable when narrow. */
export function RowerCard({ rower, className, preview }: RowerCardProps) {
  const val = (s: Stat) => (rower.stats[s] ?? 0) + (preview?.[s] ?? 0);
  const g = (s: Stat) => gain(rower, s) + (preview?.[s] ?? 0);
  const sum = statSum(rower, preview);
  const sumGain = ALL_STATS.reduce((t, s) => t + g(s), 0);
  return (
    <article
      className={cn("@container rounded-[18px] bg-cream p-1.5 cut", className)}
      aria-label={`${rower.name}, ${AGE_LABEL[rower.age] ?? rower.age}`}
    >
      <div className="relative aspect-[5/6] overflow-hidden rounded-[13px] border-2 border-ink">
        <div className="absolute inset-0 [&_img]:object-[20%_center]">
          <Art artKey={rower.artKey} cardId={rower.id} label={rower.name} />
        </div>

        <div className="absolute top-0 left-0 grid size-[22%] place-items-center rounded-br-[14px] border-r-2 border-b-2 border-ink bg-navy">
          <Emblem className="size-[80%]" />
        </div>

        <div className="absolute top-1.5 right-1.5 flex items-center gap-1">
          <AgeBadge age={rower.age} short className="@[15rem]:hidden" />
          <AgeBadge age={rower.age} className="hidden @[15rem]:inline-flex" />
          <GenderMark gender={rower.gender} />
        </div>

        {rower.event && (
          <div
            className="absolute top-[24%] left-0 flex max-w-[62%] items-center gap-1 rounded-r-full border-y-2 border-r-2 border-ink bg-[#f6b9cc] py-0.5 pr-2.5 pl-1.5 text-[11px] font-bold text-[#5a1f33] @[15rem]:text-xs"
            title={rower.event.description}
          >
            <Sparkles className="size-3.5 shrink-0" />
            <span className="truncate">{rower.event.name}</span>
          </div>
        )}

        <ul
          className="absolute top-[17%] right-1.5 flex w-[56%] flex-col gap-[5%] @[15rem]:top-[19%] @[15rem]:w-[60%]"
          style={{ height: "52%" }}
        >
          {CORE_STATS.map((s) => (
            <li key={s} className="relative flex flex-1 items-center">
              <StatBadge stat={s} className="absolute left-0 z-10 aspect-square h-[118%] w-auto" />
              <div className="ml-[12%] flex h-full w-full items-center justify-between gap-1 rounded-r-[10px] rounded-l-[4px] border-2 border-ink bg-cream pr-1.5 pl-[20%]">
                <span className="truncate font-display text-[11px] leading-none font-bold italic @[15rem]:text-sm">
                  <span className="@[15rem]:hidden">{STAT_SHORT[s]}</span>
                  <span className="hidden @[15rem]:inline @[20rem]:hidden">{STAT_MEDIUM[s]}</span>
                  <span className="hidden @[20rem]:inline">{STAT_LABEL[s]}</span>
                </span>
                <span className="flex items-baseline gap-0.5 num text-lg leading-none @[15rem]:text-2xl">
                  <Gain n={g(s)} />
                  {val(s)}
                </span>
              </div>
            </li>
          ))}
        </ul>

        <div className="absolute right-1.5 bottom-1.5 w-[48%] overflow-hidden rounded-[10px] border-2 border-ink bg-cream">
          <div className="bg-stat-motivation px-2 py-0.5 font-display text-[11px] leading-tight font-bold italic @[15rem]:text-sm">
            Motivation
          </div>
          <div className="flex items-baseline justify-end gap-1 px-2 num text-lg leading-tight @[15rem]:text-2xl">
            <Gain n={g("motivation")} />
            {val("motivation")}
          </div>
        </div>

        <div className="absolute bottom-2 left-0 max-w-[52%] -skew-x-6 rounded-r-[12px] border-y-2 border-r-2 border-ink bg-cream py-0.5 pr-2.5 pl-2">
          <span className="block skew-x-6 truncate num text-[15px] leading-tight text-navy uppercase @[11rem]:text-[17px] @[15rem]:text-[26px]">
            {rower.name}
          </span>
        </div>
      </div>
      <div
        className="flex items-baseline justify-between gap-1 px-1.5 pt-1 font-display text-xs font-bold italic @[15rem]:text-sm"
        title="Summe aller fünf Werte"
      >
        <span>Summe</span>
        <span className="flex items-baseline gap-0.5 num text-lg leading-none not-italic @[15rem]:text-xl">
          <Gain n={sumGain} />
          {sum}
        </span>
      </div>
    </article>
  );
}

/** One-line rower row for lists (phones, pickers). */
export function RowerRow({
  rower,
  className,
  right,
  highlight,
}: {
  rower: RowerView;
  className?: string;
  right?: ReactNode;
  highlight?: Stat[];
}) {
  return (
    <div
      className={cn("flex min-h-14 items-center gap-2.5 rounded-xl border-2 border-ink bg-cream p-1.5 pr-2", className)}
    >
      <div className="relative size-11 shrink-0 overflow-hidden rounded-lg border-2 border-ink">
        <Art artKey={rower.artKey} cardId={rower.id} label={rower.name} thumb />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <span className="truncate num text-lg leading-none text-navy">{rower.name}</span>
          <AgeBadge age={rower.age} short className="h-5 px-1.5 text-xs" />
          <GenderMark gender={rower.gender} className="size-5 text-[11px]" />
          {rower.event && <Sparkles className="size-4 shrink-0 text-[#c2507a]" aria-label={rower.event.name} />}
          <SumTag rower={rower} className="ml-auto" />
        </div>
        <MiniStats rower={rower} highlight={highlight} />
      </div>
      {right}
    </div>
  );
}

export function MiniStats({
  rower,
  highlight,
  className,
}: {
  rower: RowerView;
  highlight?: Stat[];
  className?: string;
}) {
  return (
    <div className={cn("mt-1 grid grid-cols-5 gap-1", className)}>
      {ALL_STATS.map((s) => {
        const n = gain(rower, s);
        const hot = highlight?.includes(s);
        return (
          <span
            key={s}
            className={cn(
              "flex items-center gap-0.5 rounded-md px-0.5 text-[13px] leading-5 font-bold tabular-nums",
              hot ? "bg-ink/10" : "",
              highlight && !hot ? "opacity-55" : "",
            )}
            title={STAT_LABEL[s]}
          >
            <span className={cn("size-2.5 shrink-0 rounded-full border border-ink", statDot(s))} />
            {rower.stats[s] ?? 0}
            {n > 0 && <span className="text-[10px] text-leaf">+{n}</span>}
          </span>
        );
      })}
    </div>
  );
}

export function statDot(s: Stat): string {
  return (
    {
      kraft: "bg-stat-kraft",
      ausdauer: "bg-stat-ausdauer",
      geschwindigkeit: "bg-stat-speed",
      technik: "bg-stat-technik",
      motivation: "bg-stat-motivation",
    }[s] ?? "bg-muted"
  );
}
