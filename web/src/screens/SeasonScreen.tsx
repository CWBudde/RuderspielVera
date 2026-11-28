import { useEffect, useMemo, useRef, useState } from "react";
import type { StationView } from "@/api/types.gen";
import { useGame, useGameLog } from "@/hooks/game";
import { StationIcon } from "@/components/icons";
import { artUrl } from "@/components/Art";
import { ResultView } from "@/components/ResultView";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { stationLabel, isCompetition, stationColor, venueLine } from "@/lib/labels";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";
import { stationInfo, stationName } from "@/i18n/data";

/** Tiny rowing boat used as the shared game piece on the calendar. */
function BoatFigure({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 48 24" className={cn("motion-safe:animate-bob", className)} aria-hidden>
      <path d="M2 15c10 4 34 4 44 0l-3 4c-10 3-28 3-38 0z" fill="#f3c431" stroke="var(--ink)" strokeWidth="1.6" />
      <circle cx="24" cy="9" r="3.6" fill="#2a4f8f" stroke="var(--ink)" strokeWidth="1.4" />
      <path d="M10 22 38 8M10 8l28 14" stroke="var(--ink)" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  );
}

type Status = "done" | "current" | "future";

function StationDot({
  st,
  status,
  size = "md",
  onClick,
}: {
  st: StationView;
  status: Status;
  size?: "md" | "lg" | "board";
  onClick: () => void;
}) {
  const c = stationColor(st.type);
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={t(
        status === "current" ? "season.dotAriaCurrent" : status === "done" ? "season.dotAriaDone" : "season.dotAria",
        { n: st.index + 1, name: stationName(st) },
      )}
      aria-current={status === "current" ? "step" : undefined}
      className={cn(
        "relative grid shrink-0 place-items-center rounded-full border-2 border-ink transition-transform active:scale-95",
        size === "md" && "size-14",
        size === "lg" && "size-[4.5rem]",
        size === "board" && "size-[5.6cqw]",
        status === "done" && "opacity-60 saturate-50",
        status === "current" && "ring-4 ring-white shadow-[0_0_0_6px_var(--ink)]",
      )}
      style={{ background: c.bg, color: c.text }}
    >
      <span className={cn("flex flex-col items-center leading-none", size === "board" && "scale-[0.9]")}>
        <span className={cn("num", size === "board" ? "text-[1.5cqw]" : "text-base")}>{st.index + 1}</span>
        <StationIcon type={st.type} name={st.name} className={size === "board" ? "size-[1.8cqw]" : "size-5"} />
      </span>
    </button>
  );
}

export function SeasonScreen() {
  const { view, you } = useGame();
  const log = useGameLog();
  const [selected, setSelected] = useState<StationView | null>(null);
  const currentRef = useRef<HTMLLIElement>(null);
  const status = (i: number): Status => (i < view.current ? "done" : i === view.current ? "current" : "future");
  const results = useMemo(() => {
    const all = [...(log.data?.results ?? [])];
    if (view.lastResult && !all.some((r) => r.stationIndex === view.lastResult?.stationIndex))
      all.push(view.lastResult);
    return all;
  }, [log.data, view.lastResult]);
  const selectedResult = selected ? results.find((r) => r.stationIndex === selected.index) : undefined;

  useEffect(() => {
    currentRef.current?.scrollIntoView({ block: "center" });
  }, []);

  const sections = useMemo(() => {
    const out: StationView[][] = [];
    for (const st of view.calendar) (out[st.section - 1] ??= []).push(st);
    return out.filter(Boolean);
  }, [view.calendar]);

  return (
    <div>
      <div className="mb-3 flex items-end justify-between gap-2">
        <div>
          <h2 className="num text-3xl leading-none">{t("season.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("season.subtitle")}</p>
        </div>
      </div>
      <Legend />

      {/* phones/tablets: winding vertical path */}
      <ol className="relative mt-4 lg:hidden" aria-label={t("season.stations")}>
        {sections.map((sec, si) => (
          <li key={si} className="mb-2">
            <p className="sticky top-[calc(var(--safe-top)+4.25rem)] z-10 mb-1 inline-block rounded-full border-2 border-ink bg-cream px-3 py-0.5 font-display text-sm font-bold">
              {t("season.section", { n: si + 1 })}
            </p>
            <ol>
              {sec.map((st) => {
                const s = status(st.index);
                const offset = Math.sin(st.index * 0.9) * 28;
                const result = results.some((r) => r.stationIndex === st.index);
                return (
                  <li
                    key={st.index}
                    ref={s === "current" ? currentRef : undefined}
                    className="relative flex items-center gap-3 py-1.5"
                    style={{ paddingLeft: `calc(50% - 7.5rem + ${offset}px)` }}
                  >
                    <span
                      aria-hidden
                      className="absolute top-0 bottom-0 left-[calc(50%-7.5rem+1.75rem)] border-l-[3px] border-dotted border-ink/30"
                      style={{ transform: `translateX(${offset}px)` }}
                    />
                    <StationDot
                      st={st}
                      status={s}
                      size={s === "current" ? "lg" : "md"}
                      onClick={() => setSelected(st)}
                    />
                    <button
                      type="button"
                      onClick={() => setSelected(st)}
                      className={cn("min-h-11 min-w-0 text-left", s === "done" && "opacity-55")}
                    >
                      <span className="block truncate font-display text-lg leading-tight font-bold">
                        {stationName(st)}
                      </span>
                      <span className="block truncate text-xs text-muted-foreground">
                        {s === "current"
                          ? t("season.youAreHere")
                          : result
                            ? t("season.viewResult")
                            : stationInfo(st) || stationLabel(st.type)}
                      </span>
                    </button>
                    {s === "current" && <BoatFigure className="h-7 w-14 shrink-0" />}
                  </li>
                );
              })}
            </ol>
          </li>
        ))}
      </ol>

      {/* desktop: the board around the Maschsee */}
      <Board status={status} onSelect={setSelected} />

      <Sheet open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        <SheetContent
          side="bottom"
          className="max-h-[88dvh] overflow-y-auto rounded-t-[22px] border-t-2 border-ink bg-water pb-[calc(var(--safe-bottom)+1rem)] md:mx-auto md:max-w-3xl"
        >
          {selected && (
            <>
              <SheetHeader className="pr-14">
                <SheetTitle className="flex items-center gap-2 font-display text-2xl">
                  <span
                    className="grid size-9 place-items-center rounded-full border-2 border-ink"
                    style={{ background: stationColor(selected.type).bg }}
                  >
                    <StationIcon type={selected.type} name={selected.name} className="size-4" />
                  </span>
                  {selected.index + 1}. {stationName(selected)}
                </SheetTitle>
                <SheetDescription>
                  {stationInfo(selected) || stationLabel(selected.type)}
                  {venueLine(selected) && <span className="block">{venueLine(selected)}</span>}
                </SheetDescription>
              </SheetHeader>
              <div className="px-4">
                {selectedResult ? (
                  <ResultView result={selectedResult} players={view.players} you={you.id} />
                ) : isCompetition(selected.type) && selected.index < view.current ? (
                  <p className="text-sm text-muted-foreground">
                    {log.isLoading ? t("season.resultLoading") : t("season.noResult")}
                  </p>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    {selected.index > view.current
                      ? t("season.stationsAhead", { count: selected.index - view.current })
                      : selected.index === view.current
                        ? t("season.pieceHere")
                        : t("season.behind")}
                  </p>
                )}
              </div>
            </>
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
}

function Legend() {
  const items: [string, string][] = [
    ["training", t("season.legend.training")],
    ["event", t("season.legend.event")],
    ["ereignis", t("season.legend.ereignis")],
    ["regatta", t("season.legend.competition")],
    ["setup", t("season.legend.setup")],
  ];
  return (
    <ul className="flex flex-wrap gap-x-3 gap-y-1">
      {items.map(([type, label]) => (
        <li key={type} className="flex items-center gap-1.5 text-sm font-semibold">
          <span className="size-3.5 rounded-full border-2 border-ink" style={{ background: stationColor(type).bg }} />
          {label}
        </li>
      ))}
    </ul>
  );
}

/** Stations on a rounded loop around the lake, clockwise from top-left, like spielbrett.png. */
function Board({ status, onSelect }: { status: (i: number) => Status; onSelect: (s: StationView) => void }) {
  const { view } = useGame();
  const n = view.calendar.length;
  const current = view.calendar[view.current];
  // Rectangle perimeter in percent of the board box, with inset so dots stay inside.
  const W = 100,
    H = 66.7,
    ix = 4.2,
    iy = 4.2 / 0.667;
  const w = W - 2 * ix,
    h = 100 - 2 * iy;
  const per = 2 * (w + h * (H / W));
  const pos = (i: number) => {
    let d = (i / n) * per;
    const hw = h * (H / W);
    if (d < w) return { x: ix + d, y: iy };
    d -= w;
    if (d < hw) return { x: ix + w, y: iy + (d / hw) * h };
    d -= hw;
    if (d < w) return { x: ix + w - d, y: iy + h };
    d -= w;
    return { x: ix, y: iy + h - (d / hw) * h };
  };
  const cur = pos(view.current);
  const boardBg = artUrl("board-bg");
  // Inward normal of the board edge the current station sits on: the boat floats on the lake side.
  const toCenter =
    Math.abs(cur.y - iy) < 0.01
      ? { x: 0, y: 1 }
      : Math.abs(cur.y - (iy + h)) < 0.01
        ? { x: 0, y: -1 }
        : cur.x > 50
          ? { x: -1, y: 0 }
          : { x: 1, y: 0 };
  return (
    <div className="@container mt-5 hidden lg:block">
      <div className="relative aspect-[3/2] overflow-hidden rounded-[28px] border-2 border-ink bg-[#7fb35f] shadow-[0_4px_0_var(--ink)]">
        {boardBg ? (
          <img src={boardBg} alt="" className="absolute inset-0 size-full object-cover" />
        ) : (
          <div className="absolute inset-[17%_15%] rounded-[45%] border-2 border-ink bg-[linear-gradient(180deg,#3f8fcf,#1f5f99)]" />
        )}
        <div className="absolute inset-x-0 top-[44%] text-center text-white">
          <p
            className="font-display text-[4cqw] leading-none font-bold italic"
            style={{ textShadow: "0 2px 0 var(--ink)", WebkitTextStroke: "1px var(--ink)" }}
          >
            Maschsee Hannover
          </p>
          {current && (
            <p className="mx-auto mt-[0.8cqw] w-fit rounded-full border-2 border-ink bg-cream px-[1.2cqw] py-[0.2cqw] text-[1.4cqw] font-bold text-ink">
              {t("season.currentStation", { n: view.current + 1, name: stationName(current) })}
            </p>
          )}
        </div>
        <span className="absolute top-[0.6cqw] left-[0.6cqw] z-10 -rotate-12 rounded-md border-2 border-ink bg-[#f39c4a] px-[0.8cqw] font-display text-[1.6cqw] font-bold">
          {t("season.start")}
        </span>
        {view.calendar.map((st) => {
          const p = pos(st.index);
          return (
            <div
              key={st.index}
              className="absolute -translate-x-1/2 -translate-y-1/2"
              style={{ left: `${p.x}%`, top: `${p.y}%` }}
              title={stationName(st)}
            >
              <StationDot st={st} status={status(st.index)} size="board" onClick={() => onSelect(st)} />
            </div>
          );
        })}
        <div
          className="pointer-events-none absolute z-10 -translate-x-1/2"
          style={{
            left: `calc(${cur.x}% + ${toCenter.x * 5.5}cqw)`,
            top: `calc(${cur.y}% + ${toCenter.y * 4.5}cqw - 1.5cqw)`,
          }}
        >
          <BoatFigure className="h-[3cqw] w-[6cqw]" />
        </div>
      </div>
    </div>
  );
}
