import { useState } from "react";
import { ChevronDown, Waves, Wrench } from "lucide-react";
import type { CompetitionResult, EntryResult, PlayerSummary, RaceResult } from "@/api/types.gen";
import { MedalIcon, WeatherIcon, weatherTint } from "@/components/icons";
import { incidentLabel, playerColor, weatherLabel } from "@/lib/labels";
import { cn } from "@/lib/utils";

export function CrabIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round">
      <ellipse cx="12" cy="14" rx="6" ry="4" fill="currentColor" fillOpacity="0.2" />
      <path d="M6.5 12 3.5 8.5M17.5 12l3-3.5M3 8.5c-.8-2 1-3.8 2.7-2.6M21 8.5c.8-2-1-3.8-2.7-2.6M7 17l-2.5 2M17 17l2.5 2M9 18l-1 2.5M15 18l1 2.5" />
      <circle cx="10" cy="12.5" r=".8" fill="currentColor" />
      <circle cx="14" cy="12.5" r=".8" fill="currentColor" />
    </svg>
  );
}

export function IncidentIcon({ incident, className }: { incident?: string; className?: string }) {
  if (!incident) return null;
  const cls = cn("size-4", className);
  const icon =
    incident === "kentern" ? <Waves className={cls} /> : incident === "material" ? <Wrench className={cls} /> : <CrabIcon className={cls} />;
  return (
    <span
      className="inline-flex items-center gap-1 rounded-full border-2 border-ink bg-[#ffe1d6] px-1.5 text-xs font-bold text-[#7a2614]"
      title={incidentLabel(incident)}
    >
      {icon}
      <span>{incidentLabel(incident)}</span>
    </span>
  );
}

export function WeatherChip({ weather, className }: { weather?: string; className?: string }) {
  if (!weather) return null;
  return (
    <span className={cn("inline-flex h-8 items-center gap-1.5 rounded-full border-2 border-ink px-2.5 text-sm font-bold", weatherTint(weather), className)}>
      <WeatherIcon weather={weather} className="size-4" />
      {weatherLabel(weather)}
    </span>
  );
}

function rowerNames(players: PlayerSummary[], e: EntryResult): string {
  if (e.guest) return "Gastmannschaft";
  const p = players.find((x) => x.id === e.playerId);
  return (e.rowerIds ?? []).map((id) => p?.rowers.find((r) => r.id === id)?.name ?? "?").join(", ");
}

function boatName(players: PlayerSummary[], e: EntryResult): string | undefined {
  if (!e.boatId) return undefined;
  return players.find((x) => x.id === e.playerId)?.boats.find((b) => b.id === e.boatId)?.name;
}

function Breakdown({ e }: { e: EntryResult }) {
  const parts: [string, number][] = [
    ["Crew", e.crew],
    ["Boot", e.boatValue],
    ["Trainer", e.trainer],
    ["Bonus", e.eventBonus],
    ["Abzug", -e.penalty],
  ];
  return (
    <dl className="mt-1.5 flex flex-wrap gap-1">
      {parts
        .filter(([label, v]) => v !== 0 || label === "Crew")
        .map(([label, v]) => (
          <div key={label} className="flex items-baseline gap-1 rounded-md bg-white/70 px-1.5 py-0.5 text-xs">
            <dt className="font-semibold text-muted-foreground">{label}</dt>
            <dd className={cn("font-bold tabular-nums", v < 0 && "text-destructive")}>{v > 0 && label !== "Crew" ? `+${v}` : v}</dd>
          </div>
        ))}
    </dl>
  );
}

function RaceCard({ race, players, you, index }: { race: RaceResult; players: PlayerSummary[]; you: string; index: number }) {
  const [open, setOpen] = useState(false);
  const entries = race.entries ?? []; // tolerate null from older engines
  const max = Math.max(1, ...entries.map((e) => e.total));
  const mine = entries.some((e) => e.playerId === you);
  return (
    <section className={cn("rounded-[16px] bg-cream p-3 cut", mine && "ring-4 ring-lake/25")}>
      <header className="flex items-center justify-between gap-2">
        <h3 className="min-w-0 truncate num text-xl leading-tight">{race.name}</h3>
        {entries.length > 0 && (
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            className="-my-2 -mr-2 inline-flex min-h-11 shrink-0 items-center gap-1 px-2 text-sm font-semibold text-muted-foreground"
            aria-expanded={open}
          >
            Details <ChevronDown className={cn("size-4 transition-transform", open && "rotate-180")} />
          </button>
        )}
      </header>
      {entries.length === 0 ? (
        <p className="mt-1 text-sm text-muted-foreground">Keine Meldungen.</p>
      ) : (
        <ol className="mt-2 flex flex-col gap-2">
          {entries.map((e, i) => {
            const p = players.find((x) => x.id === e.playerId);
            const color = e.guest ? "#9aa3ad" : playerColor(players, e.playerId);
            const dnf = e.place === 0;
            return (
              <li key={`${e.playerId || e.guest}-${i}`} className={cn("rounded-xl p-1.5", e.playerId === you && "bg-white/60")}>
                <div className="flex items-center gap-2">
                  <span
                    className={cn(
                      "grid size-8 shrink-0 place-items-center rounded-full border-2 border-ink num text-lg leading-none",
                      e.medal === "gold" ? "bg-gold" : e.medal === "silber" ? "bg-silver" : "bg-white",
                    )}
                    aria-label={dnf ? "Nicht im Ziel" : `Platz ${e.place}`}
                  >
                    {dnf ? "–" : e.place}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-1.5">
                      <span className="size-2.5 shrink-0 rounded-full border border-ink" style={{ background: color }} />
                      <span className="truncate text-sm font-bold">{e.guest ?? p?.name ?? e.playerId}</span>
                      {e.medal && <MedalIcon medal={e.medal === "gold" ? "gold" : "silber"} className="size-5 shrink-0" />}
                      <IncidentIcon incident={e.incident} />
                    </div>
                    <div className="mt-1 flex items-center gap-2">
                      <div className="h-3.5 flex-1 overflow-hidden rounded-full border-2 border-ink bg-white">
                        <div
                          className="h-full origin-left rounded-full motion-safe:animate-bar"
                          style={{
                            width: `${dnf ? 0 : Math.max(4, (e.total / max) * 100)}%`,
                            background: color,
                            animationDelay: `${index * 80 + i * 120}ms`,
                          }}
                        />
                      </div>
                      <span className="w-10 text-right num text-xl leading-none">{dnf ? "DNF" : e.total}</span>
                    </div>
                  </div>
                </div>
                {open && (
                  <div className="mt-1 pl-10 text-xs">
                    <p className="text-muted-foreground">
                      {rowerNames(players, e)}
                      {boatName(players, e) && <>, Boot {boatName(players, e)}</>}
                    </p>
                    <Breakdown e={e} />
                  </div>
                )}
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
}

export function ResultView({ result, players, you }: { result: CompetitionResult; players: PlayerSummary[]; you: string }) {
  const races = result.races ?? [];
  const mine = races.flatMap((r) => (r.entries ?? []).filter((e) => e.playerId === you));
  const golds = mine.filter((e) => e.medal === "gold").length;
  const silvers = mine.filter((e) => e.medal === "silber").length;
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <WeatherChip weather={result.weather} />
        <span className="inline-flex h-8 items-center gap-1.5 rounded-full border-2 border-ink bg-white px-2.5 text-sm font-bold">
          Deine Ausbeute:
          <MedalIcon medal="gold" className="size-5" /> {golds}
          <MedalIcon medal="silber" className="size-5" /> {silvers}
        </span>
      </div>
      <div className="grid gap-3 lg:grid-cols-2">
        {races.map((r, i) => (
          <RaceCard key={r.raceId} race={r} players={players} you={you} index={i} />
        ))}
      </div>
    </div>
  );
}
