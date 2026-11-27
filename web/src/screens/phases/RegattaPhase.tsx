import { useMemo, useState, type Dispatch, type ReactNode, type SetStateAction } from "react";
import { AlertTriangle, ArrowLeft, ArrowRight, Check, Eraser, Plus, Send, Users } from "lucide-react";
import type { BoatView, CompetitionView, Entry, RaceSpec, RowerView, TrainerView } from "@/api/types.gen";
import { useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import { Art } from "@/components/Art";
import { AgeBadge, GenderMark, MiniStats, SumTag } from "@/components/RowerCard";
import { TrainerCard } from "@/components/GearCards";
import { WeatherIcon, weatherTint, ShellIcon } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { AGE_LABEL, BOAT_CLASS_LABEL, GENDER_PLURAL, STAT_LABEL, weatherLabel } from "@/lib/labels";
import {
  boatPart,
  crewProblems,
  crewScore,
  isEligible,
  needsBoat,
  rowerScore,
  trainerBonus,
  trainerPart,
} from "@/lib/score";
import { cn } from "@/lib/utils";

type Step = "weather" | "trainer" | "lineup";

interface Draft {
  seats: (string | undefined)[];
  boatId?: string;
}

export function RegattaPhase() {
  const { view, you, submit, submitting } = useGame();
  const comp = view.phase.competition as CompetitionView;
  const hasWeather = !!comp.weather;
  const [step, setStep] = useState<Step>(hasWeather ? "weather" : "trainer");
  const [trainerId, setTrainerId] = useState<string | undefined>(() => {
    const best = [...you.trainers].sort((a, b) => trainerBonus(b, comp) - trainerBonus(a, comp))[0];
    return best?.id;
  });
  const [lineup, setLineup] = useState<Record<string, Draft>>({});
  const trainer = you.trainers.find((t) => t.id === trainerId);

  const steps: { id: Step; label: string }[] = [
    ...(hasWeather ? [{ id: "weather" as const, label: "Wetter" }] : []),
    { id: "trainer", label: "Trainer" },
    { id: "lineup", label: "Aufstellung" },
  ];

  const stepper = (
    <ol className="mt-3 flex gap-1.5" aria-label="Ablauf">
      {steps.map((s, i) => {
        const idx = steps.findIndex((x) => x.id === step);
        const state = i < idx ? "done" : i === idx ? "now" : "next";
        return (
          <li key={s.id} className="min-w-0 flex-1">
            <button
              type="button"
              onClick={() => i <= idx && setStep(s.id)}
              disabled={i > idx}
              aria-current={state === "now" ? "step" : undefined}
              className={cn(
                "flex min-h-11 w-full items-center gap-1.5 rounded-xl border-2 border-ink px-2 text-left text-sm font-bold max-[380px]:gap-1 max-[380px]:px-1.5 max-[380px]:text-xs",
                state === "now" ? "bg-navy text-white" : state === "done" ? "bg-st-regatta" : "bg-white/60 text-ink/60",
              )}
            >
              <span className="num text-lg leading-none">{i + 1}</span>
              <span className="truncate">{s.label}</span>
            </button>
          </li>
        );
      })}
    </ol>
  );

  if (step === "weather") {
    return (
      <PhaseFrame
        title={comp.name}
        prompt="Das Wetter steht fest, bevor du Trainer und Boote wählst."
        headerExtra={stepper}
        footer={
          <Button size="lg" className="w-full" onClick={() => setStep("trainer")}>
            Trainer wählen <ArrowRight />
          </Button>
        }
      >
        <WeatherReveal comp={comp} />
      </PhaseFrame>
    );
  }

  if (step === "trainer") {
    return (
      <PhaseFrame
        title={comp.name}
        prompt={
          hasWeather
            ? `Wetter heute: ${weatherLabel(comp.weather)}. Welcher Trainer holt am meisten heraus?`
            : "Welcher Trainer begleitet dein Team?"
        }
        headerExtra={
          <>
            {stepper}
            {!hasWeather && <CompetitionFacts comp={comp} />}
          </>
        }
        footer={
          <div className="flex gap-2">
            {hasWeather && (
              <Button variant="outline" size="icon" aria-label="Zurück" onClick={() => setStep("weather")}>
                <ArrowLeft />
              </Button>
            )}
            <Button size="lg" className="flex-1" onClick={() => setStep("lineup")}>
              Aufstellung <ArrowRight />
            </Button>
          </div>
        }
      >
        <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3" role="radiogroup" aria-label="Trainer">
          {you.trainers.map((t) => {
            const on = t.id === trainerId;
            return (
              <li key={t.id}>
                <button
                  type="button"
                  role="radio"
                  aria-checked={on}
                  onClick={() => setTrainerId(on ? undefined : t.id)}
                  className="relative w-full text-left"
                >
                  <TrainerCard
                    trainer={t}
                    weather={comp.weather ?? "normal"}
                    todayBonus={trainerBonus(t, comp)}
                    className={cn(on && "ring-4 ring-lake")}
                  />
                  {on && (
                    <span className="absolute -top-2 -left-2 grid size-8 place-items-center rounded-full border-2 border-ink bg-lake text-white">
                      <Check className="size-5" strokeWidth={3} />
                    </span>
                  )}
                </button>
              </li>
            );
          })}
        </ul>
        <p className="mt-3 text-sm text-muted-foreground">
          Trainerwerte gelten pro Ruderplatz: im Einer einfach, im Zweier doppelt, im Vierer vierfach.
        </p>
      </PhaseFrame>
    );
  }

  return (
    <LineupBuilder
      comp={comp}
      trainer={trainer}
      lineup={lineup}
      setLineup={setLineup}
      stepper={stepper}
      onBack={() => setStep("trainer")}
      submitting={submitting}
      onSubmit={(entries) => submit({ trainerId, entries })}
    />
  );
}

function weightsText(comp: CompetitionView): string {
  return Object.entries(comp.weights)
    .filter(([, w]) => w > 0)
    .sort((a, b) => b[1] - a[1])
    .map(([s, w]) => `${STAT_LABEL[s] ?? s} ×${String(w).replace(".", ",")}`)
    .join(", ");
}

function CompetitionFacts({ comp }: { comp: CompetitionView }) {
  return (
    <div className="mt-3 rounded-xl border-2 border-ink bg-white/70 p-2.5 text-sm">
      <p>
        <span className="font-bold">Gewertet:</span> {weightsText(comp)}
      </p>
      {comp.notes?.map((n) => (
        <p key={n} className="mt-0.5 text-muted-foreground">
          {n}
        </p>
      ))}
    </div>
  );
}

function WeatherReveal({ comp }: { comp: CompetitionView }) {
  const w = comp.weather ?? "normal";
  return (
    <div className="grid gap-3 md:grid-cols-[minmax(0,20rem)_1fr] md:items-start">
      <div
        className={cn(
          "mx-auto flex aspect-[5/4] w-full max-w-[17rem] flex-col items-center justify-center rounded-[22px] border-2 border-ink shadow-[0_3px_0_var(--ink)] motion-safe:animate-flip-in",
          weatherTint(w),
        )}
      >
        <WeatherIcon weather={w} className="size-20" strokeWidth={1.6} />
        <p className="mt-2 num text-4xl leading-none">{weatherLabel(w)}</p>
      </div>
      <CompetitionFacts comp={comp} />
    </div>
  );
}

// ---------------- lineup ----------------

function raceRule(race: RaceSpec): string {
  const ages =
    race.ages?.length === 3 || !race.ages?.length
      ? "alle Altersklassen"
      : race.ages.map((a) => AGE_LABEL[a] ?? a).join(", ");
  const genders = race.genders?.length === 1 ? GENDER_PLURAL[race.genders[0]] : "Mädchen und Jungen";
  return `${genders}, ${ages}`;
}

function LineupBuilder({
  comp,
  trainer,
  lineup,
  setLineup,
  stepper,
  onBack,
  onSubmit,
  submitting,
}: {
  comp: CompetitionView;
  trainer?: TrainerView;
  lineup: Record<string, Draft>;
  setLineup: Dispatch<SetStateAction<Record<string, Draft>>>;
  stepper: ReactNode;
  onBack: () => void;
  onSubmit: (entries: Entry[]) => void;
  submitting: boolean;
}) {
  const { you } = useGame();
  const [slot, setSlot] = useState<{ raceId: string; seat: number } | null>(null);
  const tBonus = trainerBonus(trainer, comp);
  const rowerById = useMemo(() => new Map(you.rowers.map((r) => [r.id, r])), [you.rowers]);

  /** rowerId -> race name it is already entered in */
  const usedIn = useMemo(() => {
    const m = new Map<string, string>();
    for (const race of comp.races) for (const id of lineup[race.id]?.seats ?? []) if (id) m.set(id, race.id);
    return m;
  }, [lineup, comp.races]);

  const draftFor = (race: RaceSpec): Draft => lineup[race.id] ?? { seats: Array(race.seats).fill(undefined) };

  const setDraft = (raceId: string, d: Draft) => setLineup((l) => ({ ...l, [raceId]: d }));

  /** boatId -> raceId of the first race (with rowers) that uses it: a boat starts once per competition. */
  const boatUsedIn = useMemo(() => {
    const m = new Map<string, string>();
    for (const race of comp.races) {
      const d = lineup[race.id];
      if (d?.boatId && needsBoat(race) && d.seats.some(Boolean) && !m.has(d.boatId)) m.set(d.boatId, race.id);
    }
    return m;
  }, [lineup, comp.races]);
  const raceName = (id: string) => comp.races.find((r) => r.id === id)?.name ?? "";

  const status = comp.races.map((race) => {
    const d = draftFor(race);
    const crew = d.seats.map((id) => (id ? rowerById.get(id) : undefined));
    const filled = crew.filter(Boolean).length;
    const boat = needsBoat(race) ? you.boats.find((b) => b.id === d.boatId) : undefined;
    const problems = crewProblems(race, crew, !!boat);
    const owner = boat ? boatUsedIn.get(boat.id) : undefined;
    if (filled > 0 && owner && owner !== race.id) problems.push(`${boat?.name} startet schon: ${raceName(owner)}`);
    return { race, d, crew, filled, boat, problems, complete: filled === race.seats && problems.length === 0 };
  });
  const complete = status.filter((s) => s.complete);
  const blocking = status.filter((s) => s.filled > 0 && !s.complete);

  const send = () => {
    onSubmit(
      complete.map((s) => ({
        raceId: s.race.id,
        rowerIds: s.d.seats as string[],
        boatId: needsBoat(s.race) ? s.d.boatId : undefined,
      })),
    );
  };

  const activeRace = slot ? comp.races.find((r) => r.id === slot.raceId) : undefined;

  return (
    <PhaseFrame
      title={comp.name}
      prompt="Tippe auf einen Platz, um einen Ruderer zu setzen. Jeder startet höchstens einmal. Rennen dürfen leer bleiben."
      headerExtra={
        <>
          {stepper}
          <div className="mt-3 flex flex-wrap gap-1.5 text-sm">
            {comp.weather && (
              <span
                className={cn(
                  "inline-flex h-8 items-center gap-1 rounded-full border-2 border-ink px-2.5 font-bold",
                  weatherTint(comp.weather),
                )}
              >
                <WeatherIcon weather={comp.weather} className="size-4" /> {weatherLabel(comp.weather)}
              </span>
            )}
            <span className="inline-flex h-8 items-center gap-1 rounded-full border-2 border-ink bg-white px-2.5 font-bold">
              {trainer
                ? `${trainer.name}: ${tBonus > 0 ? `+${tBonus} pro Platz` : tBonus === 0 ? "kein Bonus" : `${tBonus} pro Platz`}`
                : "Kein Trainer"}
            </span>
            <span className="inline-flex h-8 items-center rounded-full border-2 border-ink bg-white px-2.5 font-semibold">
              {weightsText(comp)}
            </span>
          </div>
        </>
      }
      footer={
        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon" aria-label="Zurück zum Trainer" onClick={onBack}>
            <ArrowLeft />
          </Button>
          <div className="min-w-0 flex-1 text-sm leading-tight">
            <p className="font-bold">
              {complete.length} von {comp.races.length} Rennen gemeldet
            </p>
            {blocking.length > 0 ? (
              <p className="flex items-center gap-1 truncate text-destructive">
                <AlertTriangle className="size-3.5 shrink-0" /> {blocking[0].race.name}: {blocking[0].problems[0]}
              </p>
            ) : (
              complete.length === 0 && <p className="text-muted-foreground">Du meldest noch kein Rennen.</p>
            )}
          </div>
          <Button size="lg" onClick={send} disabled={blocking.length > 0 || submitting}>
            <Send /> Melden
          </Button>
        </div>
      }
    >
      <ul className="grid gap-3 lg:grid-cols-2">
        {status.map(({ race, d, crew, boat, problems, complete: ok, filled }) => {
          const est =
            crewScore(
              crew.filter((r): r is RowerView => !!r),
              comp.weights,
            ) +
            boatPart(race, boat, comp) +
            (filled ? trainerPart(race, trainer, comp) : 0);
          const boats = you.boats.filter((b) => b.class === race.class).sort((a, b) => b.value - a.value);
          return (
            <li key={race.id} className={cn("rounded-[16px] bg-cream p-3 cut", ok && "bg-[#eef7e6]")}>
              <header className="flex items-start gap-2">
                <div className="min-w-0 flex-1">
                  <h3 className="num text-xl leading-tight">{race.name}</h3>
                  <p className="mt-0.5 flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
                    <span className="inline-flex items-center gap-1 rounded-md bg-navy px-1.5 py-0.5 font-bold text-white">
                      <ShellIcon className="h-3 w-5" /> {BOAT_CLASS_LABEL[race.class] ?? race.class}
                    </span>
                    {race.mixed && (
                      <span className="rounded-md bg-st-event px-1.5 py-0.5 font-bold text-ink">Mixed</span>
                    )}
                    <span>{raceRule(race)}</span>
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-[11px] font-semibold text-muted-foreground">geschätzt</p>
                  <p className="num text-3xl leading-none">{filled ? est : "–"}</p>
                  {!!race.guestScore && (
                    <p
                      className="mt-0.5 text-[11px] font-semibold whitespace-nowrap text-muted-foreground"
                      title="Gastmannschaft"
                    >
                      Gast {race.guestScore}
                    </p>
                  )}
                </div>
              </header>

              <div
                className={cn(
                  "mt-2 grid gap-1.5",
                  race.seats >= 4
                    ? "grid-cols-2 sm:grid-cols-4 lg:grid-cols-2 xl:grid-cols-4"
                    : race.seats === 2
                      ? "grid-cols-2"
                      : "grid-cols-1",
                )}
              >
                {d.seats.map((id, seat) => {
                  const r = id ? rowerById.get(id) : undefined;
                  return (
                    <button
                      key={seat}
                      type="button"
                      onClick={() => setSlot({ raceId: race.id, seat })}
                      className={cn(
                        "flex min-h-12 items-center gap-2 rounded-xl border-2 px-1.5 text-left",
                        r ? "border-ink bg-white" : "border-dashed border-ink/50 bg-white/40 text-ink/60",
                      )}
                      aria-label={r ? `Platz ${seat + 1}: ${r.name} – ändern` : `Platz ${seat + 1} besetzen`}
                    >
                      {r ? (
                        <>
                          <span className="size-8 shrink-0 overflow-hidden rounded-md border-2 border-ink">
                            <Art artKey={r.artKey} cardId={r.id} label={r.name} thumb />
                          </span>
                          <span className="min-w-0 flex-1 truncate font-display text-base font-bold">{r.name}</span>
                          <span className="text-xs font-bold tabular-nums text-muted-foreground">
                            {Math.round(rowerScore(r, comp.weights))}
                          </span>
                        </>
                      ) : (
                        <>
                          <Plus className="size-4 shrink-0" />
                          <span className="text-sm font-semibold">Platz {seat + 1}</span>
                        </>
                      )}
                    </button>
                  );
                })}
              </div>

              {needsBoat(race) && (
                <BoatPicker
                  scale={race.boatScale}
                  loan={!!race.loanBoat}
                  loanValue={race.fixedBoatValue ?? 0}
                  boats={boats}
                  value={filled > 0 || !d.boatId || !boatUsedIn.has(d.boatId) ? d.boatId : undefined}
                  usedIn={(id) => {
                    const owner = boatUsedIn.get(id);
                    return owner && owner !== race.id ? raceName(owner) : undefined;
                  }}
                  onChange={(boatId) => setDraft(race.id, { ...d, boatId })}
                />
              )}
              {race.class === "8x" && (
                <div className="mt-2 flex items-center gap-3">
                  <div className="aspect-[3/2] w-24 shrink-0 overflow-hidden rounded-[8px] border-2 border-ink">
                    <Art artKey="boat-8x" label="Achter" />
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Gemeinsamer Achter für alle Teams, Bootswert {race.fixedBoatValue ?? 0}.
                  </p>
                </div>
              )}

              {(problems.length > 0 || filled > 0) && (
                <div className="mt-2 flex items-center gap-2">
                  {problems.length > 0 && (
                    <ul className="min-w-0 flex-1 text-xs font-semibold text-destructive">
                      {problems.map((p) => (
                        <li key={p} className="flex items-center gap-1">
                          <AlertTriangle className="size-3.5 shrink-0" /> {p}
                        </li>
                      ))}
                    </ul>
                  )}
                  {ok && <p className="flex-1 text-xs font-bold text-leaf">Bereit zum Melden</p>}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="ml-auto min-h-11"
                    onClick={() => setDraft(race.id, { seats: Array(race.seats).fill(undefined) })}
                  >
                    <Eraser /> Leeren
                  </Button>
                </div>
              )}
            </li>
          );
        })}
      </ul>

      <Sheet open={!!slot} onOpenChange={(o) => !o && setSlot(null)}>
        <SheetContent
          side="bottom"
          className="max-h-[85dvh] gap-2 rounded-t-[22px] border-t-2 border-ink bg-cream pb-[calc(var(--safe-bottom)+0.75rem)] md:mx-auto md:max-w-2xl"
        >
          {activeRace && slot && (
            <RowerPicker
              race={activeRace}
              comp={comp}
              seat={slot.seat}
              draft={draftFor(activeRace)}
              usedIn={usedIn}
              raceName={(id) => comp.races.find((r) => r.id === id)?.name ?? ""}
              onPick={(rowerId) => {
                const d = draftFor(activeRace);
                const seats = [...d.seats];
                const existing = seats.indexOf(rowerId);
                if (existing >= 0) seats[existing] = seats[slot.seat];
                seats[slot.seat] = rowerId;
                // Preselect the best own boat of the class on first placement.
                const best = you.boats
                  .filter(
                    (b) =>
                      b.class === activeRace.class && (!boatUsedIn.has(b.id) || boatUsedIn.get(b.id) === activeRace.id),
                  )
                  .sort((a, b) => b.value - a.value)[0];
                const beatsLoan = !activeRace.loanBoat || (best?.value ?? 0) > (activeRace.fixedBoatValue ?? 0);
                const first = !d.seats.some(Boolean);
                const boatId = d.boatId ?? (first && needsBoat(activeRace) && beatsLoan ? best?.id : undefined);
                setDraft(activeRace.id, { ...d, seats, boatId });
                // Advance to the next empty seat in this race, or close.
                const nextEmpty = seats.findIndex((s) => !s);
                setSlot(nextEmpty >= 0 ? { raceId: activeRace.id, seat: nextEmpty } : null);
              }}
              onClear={() => {
                const d = draftFor(activeRace);
                const seats = [...d.seats];
                seats[slot.seat] = undefined;
                setDraft(activeRace.id, { ...d, seats });
                setSlot(null);
              }}
            />
          )}
        </SheetContent>
      </Sheet>
    </PhaseFrame>
  );
}

function BoatPicker({
  boats,
  value,
  usedIn,
  loan,
  loanValue,
  scale,
  onChange,
}: {
  boats: BoatView[];
  value?: string;
  usedIn: (boatId: string) => string | undefined;
  loan: boolean;
  loanValue: number;
  scale: number;
  onChange: (id: string | undefined) => void;
}) {
  if (boats.length === 0 && !loan) {
    return <p className="mt-2 text-xs font-semibold text-destructive">Du hast kein Boot dieser Klasse.</p>;
  }
  return (
    <div className="mt-2">
      <p className="mb-1 text-xs font-semibold text-muted-foreground">Boot{scale > 1 && ` (Wert zählt ×${scale})`}</p>
      <div className="flex flex-wrap gap-1.5" role="radiogroup" aria-label="Boot">
        {loan && (
          <button
            type="button"
            role="radio"
            aria-checked={!value}
            onClick={() => onChange(undefined)}
            className={cn(
              "inline-flex min-h-11 items-center gap-1.5 rounded-xl border-2 border-ink px-2.5 text-sm font-bold",
              !value ? "bg-navy text-white" : "bg-white",
            )}
          >
            Leihboot
            <span
              className={cn("num rounded-md px-1 text-base leading-tight", !value ? "bg-white/20" : "bg-st-regatta")}
            >
              {loanValue}
            </span>
          </button>
        )}
        {boats.map((b) => {
          const on = b.id === value;
          const elsewhere = usedIn(b.id);
          return (
            <button
              key={b.id}
              type="button"
              role="radio"
              aria-checked={on}
              disabled={!!elsewhere && !on}
              title={elsewhere ? `startet schon: ${elsewhere}` : undefined}
              onClick={() => onChange(on && !loan ? undefined : b.id)}
              className={cn(
                "inline-flex min-h-11 items-center gap-1.5 rounded-xl border-2 border-ink px-2.5 text-left text-sm font-bold",
                on ? (elsewhere ? "bg-destructive text-white" : "bg-navy text-white") : "bg-white",
                elsewhere && !on && "border-dashed opacity-60",
              )}
            >
              <span className="flex flex-col leading-tight">
                {b.name}
                {elsewhere && <span className="text-[11px] font-semibold">startet: {elsewhere}</span>}
              </span>
              <span className={cn("num rounded-md px-1 text-base leading-tight", on ? "bg-white/20" : "bg-st-regatta")}>
                {b.value}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function RowerPicker({
  race,
  comp,
  seat,
  draft,
  usedIn,
  raceName,
  onPick,
  onClear,
}: {
  race: RaceSpec;
  comp: CompetitionView;
  seat: number;
  draft: Draft;
  usedIn: Map<string, string>;
  raceName: (id: string) => string;
  onPick: (id: string) => void;
  onClear: () => void;
}) {
  const { you } = useGame();
  const current = draft.seats[seat];
  const eligible = you.rowers
    .filter((r) => isEligible(r, race))
    .map((r) => ({ r, score: rowerScore(r, comp.weights) }))
    .sort((a, b) => b.score - a.score);
  const ineligible = you.rowers.length - eligible.length;
  const highlight = Object.keys(comp.weights);
  return (
    <>
      <SheetHeader className="pr-14 pb-0">
        <SheetTitle className="font-display text-2xl leading-tight">
          {race.name}, Platz {seat + 1}
        </SheetTitle>
        <SheetDescription>
          Sortiert nach Rennwert für diesen Wettkampf.{" "}
          {ineligible > 0 && `${ineligible} Ruderer sind hier nicht startberechtigt.`}
        </SheetDescription>
      </SheetHeader>
      <div className="overflow-y-auto px-4">
        {current && (
          <Button variant="outline" className="mb-2 w-full" onClick={onClear}>
            <Eraser /> Platz leeren
          </Button>
        )}
        {eligible.length === 0 && (
          <p className="flex items-center gap-2 rounded-xl border-2 border-ink bg-white p-3 text-sm">
            <Users className="size-4" /> Kein passender Ruderer im Team.
          </p>
        )}
        <ul className="flex flex-col gap-1.5 pb-2">
          {eligible.map(({ r, score }) => {
            const other = usedIn.get(r.id);
            const inThisRace = other === race.id;
            const blocked = !!other && !inThisRace;
            const isCurrent = r.id === current;
            return (
              <li key={r.id}>
                <button
                  type="button"
                  disabled={blocked || isCurrent}
                  onClick={() => onPick(r.id)}
                  className={cn(
                    "flex min-h-14 w-full items-center gap-2.5 rounded-xl border-2 border-ink bg-white p-1.5 pr-2.5 text-left disabled:cursor-not-allowed",
                    blocked && "opacity-45",
                    isCurrent && "bg-st-regatta",
                  )}
                >
                  <span className="size-11 shrink-0 overflow-hidden rounded-lg border-2 border-ink">
                    <Art artKey={r.artKey} cardId={r.id} label={r.name} thumb />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="flex items-center gap-1.5">
                      <span className="truncate num text-lg leading-none text-navy">{r.name}</span>
                      <AgeBadge age={r.age} short className="h-5 px-1.5 text-xs" />
                      <GenderMark gender={r.gender} className="size-5 text-[11px]" />
                      <SumTag rower={r} className="ml-auto" />
                    </span>
                    {blocked ? (
                      <span className="mt-1 block truncate text-xs font-semibold">
                        startet schon: {raceName(other)}
                      </span>
                    ) : inThisRace && !isCurrent ? (
                      <span className="mt-1 block text-xs font-semibold">schon in diesem Boot – wird getauscht</span>
                    ) : (
                      <MiniStats rower={r} highlight={highlight} />
                    )}
                  </span>
                  <span className="text-right">
                    <span className="block num text-2xl leading-none">{Math.round(score)}</span>
                    <span className="text-[10px] font-semibold text-muted-foreground">Rennwert</span>
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      </div>
    </>
  );
}
