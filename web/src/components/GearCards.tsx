import { Building2, Star } from "lucide-react";
import type { BoatView, TrainerEffect, TrainerView, Weather } from "@/api/types.gen";
import { Art } from "@/components/Art";
import { ShellIcon, WeatherIcon } from "@/components/icons";
import { BOAT_CLASS_LABEL, CONTEXT_LABEL, WEATHER_LABEL } from "@/lib/labels";
import { cn } from "@/lib/utils";

/** Boat card (kartendesign.png): name header with navy class tab, illustration, value. */
export function BoatCard({ boat, className }: { boat: BoatView; className?: string }) {
  return (
    <article className={cn("@container overflow-hidden rounded-[16px] bg-cream cut", className)}>
      <header className="flex items-stretch justify-between gap-2">
        <h3 className="min-w-0 px-3 pt-2 pb-1 num text-lg leading-tight text-ink @[14rem]:truncate @[14rem]:text-[22px]">{boat.name}</h3>
        <span className="flex shrink-0 items-center gap-1.5 self-start rounded-bl-[14px] bg-navy px-2 py-1.5 text-xs font-bold text-white @[14rem]:self-stretch @[14rem]:px-3 @[14rem]:text-sm">
          <ShellIcon className="hidden h-3.5 w-6 @[14rem]:block" />
          {BOAT_CLASS_LABEL[boat.class] ?? boat.class}
        </span>
      </header>
      <div className="mx-2 aspect-[2/1] overflow-hidden rounded-[10px] border-2 border-ink">
        <Art artKey={boat.artKey} label={boat.name} />
      </div>
      <div className="flex items-center justify-between gap-2 px-3 py-2">
        <span className="inline-flex min-w-0 items-center gap-1 text-xs font-semibold text-muted-foreground @[14rem]:text-sm">
          {boat.city && <Building2 className="size-4 shrink-0" />}
          <span className="truncate">{boat.city ? "Städtisch" : "Vereinsboot"}</span>
        </span>
        <span className="flex shrink-0 items-baseline gap-1">
          <span className="text-xs font-semibold @[14rem]:text-sm">Wert</span>
          <span className="num text-3xl leading-none">{boat.value}</span>
        </span>
      </div>
    </article>
  );
}

export function effectText(e: TrainerEffect): string {
  const sign = e.value > 0 ? `+${e.value}` : `${e.value}`;
  const parts: string[] = [];
  if (e.weather) parts.push(`bei ${WEATHER_LABEL[e.weather] ?? e.weather}`);
  if (e.context) parts.push(CONTEXT_LABEL[e.context] ?? e.context);
  return `${sign} ${parts.join(" ") || "immer"}`;
}

/** Trainer card: portrait, effects as chips, special-ability box. */
export function TrainerCard({
  trainer,
  className,
  weather,
  todayBonus,
}: {
  trainer: TrainerView;
  className?: string;
  weather?: Weather;
  /** If set, highlights the bonus that applies today. */
  todayBonus?: number;
}) {
  return (
    <article className={cn("overflow-hidden rounded-[16px] bg-cream cut", className)}>
      <header className="flex items-stretch justify-between gap-2">
        <h3 className="min-w-0 truncate px-3 pt-2 pb-1 num text-[22px] leading-tight">{trainer.name}</h3>
        <span className="flex shrink-0 items-center rounded-bl-[14px] bg-navy px-3 text-sm font-bold text-white">Trainer</span>
      </header>
      <div className="flex gap-2.5 px-2 pb-2">
        <div className="size-20 shrink-0 overflow-hidden rounded-[10px] border-2 border-ink">
          <Art artKey={trainer.artKey} label={trainer.name} thumb />
        </div>
        <ul className="flex min-w-0 flex-1 flex-col justify-center gap-1">
          {(trainer.effects ?? []).map((e, i) => {
            const active = weather !== undefined && (!e.weather || e.weather === weather);
            return (
              <li
                key={i}
                className={cn(
                  "flex items-center gap-1.5 rounded-lg border-2 px-2 py-0.5 text-sm font-semibold",
                  active ? "border-ink bg-white" : "border-transparent bg-cream-2",
                )}
              >
                {e.weather ? <WeatherIcon weather={e.weather} className="size-4 shrink-0" /> : <Star className="size-4 shrink-0" />}
                <span className="truncate">{effectText(e)}</span>
              </li>
            );
          })}
        </ul>
      </div>
      <div className="mx-2 mb-2 rounded-[10px] bg-[#dbe8f3] px-3 py-2">
        <div className="flex items-center justify-between gap-2">
          <span className="inline-flex items-center gap-1 rounded-md bg-navy px-2 py-0.5 text-xs font-bold text-white">
            <Star className="size-3 fill-sun text-sun" /> {trainer.ability}
          </span>
          {todayBonus !== undefined && (
            <span
              className={cn(
                "num text-2xl leading-none",
                todayBonus > 0 ? "text-leaf" : todayBonus < 0 ? "text-destructive" : "text-muted-foreground",
              )}
            >
              {todayBonus > 0 ? `+${todayBonus}` : todayBonus}
            </span>
          )}
        </div>
        <p className="mt-1 text-sm leading-snug">{trainer.description}</p>
      </div>
    </article>
  );
}
