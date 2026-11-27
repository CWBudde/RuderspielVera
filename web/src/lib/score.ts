import type {
  CompetitionView,
  RaceSpec,
  RowerView,
  Stat,
  TrainerView,
  Weather,
} from "@/api/types.gen";

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

/** Trainer effect for one boat/ergo entry under the given weather and competition context. */
export function trainerBonus(
  trainer: TrainerView | undefined,
  weather: Weather | undefined,
  context: string,
): number {
  if (!trainer) return 0;
  let total = 0;
  for (const e of trainer.effects ?? []) {
    const weatherOk = !e.weather || e.weather === weather;
    const contextOk = !e.context || e.context === context;
    if (weatherOk && contextOk) total += e.value;
  }
  return total;
}

/** Context string the engine uses for trainer abilities, derived from the competition. */
export function competitionContext(comp: CompetitionView): string {
  if (comp.kind === "ergocup") return "ergo";
  const w = comp.weights;
  if ((w["ausdauer"] ?? 0) >= 2) return "langstrecke";
  if ((w["geschwindigkeit"] ?? 0) >= 2) return "sprint";
  return "";
}

/** rules.json loanBoatValue: boat value used when a loanBoat race is entered without own boat. */
export const LOAN_BOAT_VALUE = 2;

/** Value the boat contributes for an entry (own boat, loan boat or the shared eight). */
export function boatContribution(race: RaceSpec, boatValue: number | undefined): number {
  if (race.class === "8x" || race.class === "ergo") return 0;
  if (boatValue !== undefined) return boatValue;
  return race.loanBoat ? LOAN_BOAT_VALUE : 0;
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
