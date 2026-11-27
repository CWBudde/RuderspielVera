import type { CompetitionView, BoatView, RaceSpec, RowerView, Stat, TrainerView } from "@/api/types.gen";

/**
 * Client-side estimates of the engine's race scoring (docs/PLAN.md "Race scoring"):
 * rowerScore = Σ weight_s · stat_s; race = crew + boat + trainer (+ events − penalties).
 * The engine is authoritative; these numbers only guide lineup choices.
 */
export function rowerScore(rower: RowerView, weights: CompetitionView["weights"]): number {
  let sum = 0;
  for (const [stat, w] of Object.entries(weights)) {
    sum += w * (rower.stats[stat as Stat] ?? 0);
  }
  return sum;
}

export function crewScore(rowers: RowerView[], weights: CompetitionView["weights"]): number {
  return Math.round(rowers.reduce((acc, r) => acc + rowerScore(r, weights), 0));
}

/**
 * Trainer effect per seat under the competition's weather and context (TrainerEffect.context),
 * plus the trainer's current mood (Laune), which the engine adds per seat as well.
 */
export function trainerBonus(trainer: TrainerView | undefined, comp: CompetitionView): number {
  if (!trainer) return 0;
  let total = trainer.mood ?? 0;
  for (const e of trainer.effects ?? []) {
    const weatherOk = !e.weather || e.weather === comp.weather;
    const contextOk = !e.context || e.context === (comp.context ?? "");
    if (weatherOk && contextOk) total += e.value;
  }
  return total;
}

/** Trainer part of an entry: per-seat effect × race.trainerScale (engine scoreEntry). */
export function trainerPart(race: RaceSpec, trainer: TrainerView | undefined, comp: CompetitionView): number {
  return trainerBonus(trainer, comp) * race.trainerScale;
}

/** Base value of a boat in this competition (städtische Boote get the Ereignis modifier). */
export function boatBaseValue(boat: BoatView, comp: CompetitionView): number {
  return Math.max(0, boat.value + (boat.city ? (comp.cityBoatMod ?? 0) : 0));
}

/** Boat part of an entry: own boat, or the loan boat / shared eight (race.fixedBoatValue), × race.boatScale. */
export function boatPart(race: RaceSpec, boat: BoatView | undefined, comp: CompetitionView): number {
  if (race.class === "ergo") return 0;
  const base = boat && race.class !== "8x" ? boatBaseValue(boat, comp) : (race.fixedBoatValue ?? 0);
  return base * race.boatScale;
}

export function needsBoat(race: RaceSpec): boolean {
  return race.class !== "ergo" && race.class !== "8x";
}

/** Eligibility of one rower for a seat, ignoring the mixed rule (checked per crew). */
export function isEligible(rower: RowerView, race: RaceSpec): boolean {
  const ageOk = !race.ages?.length || race.ages.includes(rower.age);
  const genderOk = !race.genders?.length || race.genders.includes(rower.gender);
  return ageOk && genderOk;
}

/** German validation messages for a (partial) crew; empty array = valid. */
export function crewProblems(race: RaceSpec, crew: (RowerView | undefined)[], hasBoat: boolean): string[] {
  const filled = crew.filter((r): r is RowerView => !!r);
  if (filled.length === 0) return [];
  const problems: string[] = [];
  if (filled.length < race.seats) {
    problems.push(`Noch ${race.seats - filled.length} ${race.seats - filled.length === 1 ? "Platz" : "Plätze"} frei`);
  }
  if (race.mixed && filled.length === race.seats) {
    const genders = new Set(filled.map((r) => r.gender));
    const ages = new Set(filled.map((r) => r.age));
    if (genders.size < 2) problems.push("Mixed: mindestens ein Mädchen und ein Junge");
    if (ages.size < 2) problems.push("Mixed: mindestens zwei Altersklassen");
  }
  if (needsBoat(race) && !hasBoat && !race.loanBoat) problems.push("Boot fehlt");
  return problems;
}
