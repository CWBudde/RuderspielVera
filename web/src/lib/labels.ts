import { t, tData } from "@/i18n";
import type { AgeGroup, BoatClass, Gender, IncidentKind, Stat, StationType, Weather } from "@/api/types.gen";

/** Canonical display order (mirrors engine.AllStats). */
export const ALL_STATS: Stat[] = ["kraft", "ausdauer", "geschwindigkeit", "technik", "motivation"];
/** The four "card" stats; Motivation is shown separately as the yellow bar. */
export const CORE_STATS: Stat[] = ["kraft", "ausdauer", "geschwindigkeit", "technik"];

/** Full stat label ("Geschwindigkeit"). */
export function statLabel(s: Stat | string): string {
  return tData(`stat.${s}`, s);
}
/** Narrow stat label ("Ausd."). */
export function statShort(s: Stat | string): string {
  return tData(`statShort.${s}`, s);
}
/** Mid-width card label: only Geschwindigkeit is too long for a narrow pill. */
export function statMedium(s: Stat | string): string {
  return tData(`statMedium.${s}`, statLabel(s));
}
export function ageLabel(a: AgeGroup | string): string {
  return tData(`age.${a}`, a);
}
export function ageShort(a: AgeGroup | string): string {
  return tData(`ageShort.${a}`, a);
}
export const AGE_ORDER: Record<string, number> = { a: 0, b: 1, kinder: 2 };

export function genderLabel(g: Gender | string): string {
  return tData(`gender.${g}`, g);
}
export function genderPlural(g: Gender | string): string {
  return tData(`genderPlural.${g}`, g);
}

export function boatClassLabel(c: BoatClass | string): string {
  return tData(`boatClass.${c}`, c);
}

export function weatherLabel(w: Weather | string | undefined): string {
  if (!w) return t("weather.none");
  return tData(`weather.${w}`, w);
}

export function stationLabel(s: StationType | string): string {
  return tData(`station.${s}`, s);
}

/** Board colours (examples/spielbrett.png). `bg` is the pastel fill, `ring` a darker edge. */
export const STATION_COLOR: Record<string, { bg: string; ring: string; text: string }> = {
  setup: { bg: "var(--st-setup)", ring: "#9a9a92", text: "#2d2d2a" },
  training: { bg: "var(--st-training)", ring: "#d1a826", text: "#4a3a05" },
  camp: { bg: "var(--st-training)", ring: "#d1a826", text: "#4a3a05" },
  selection: { bg: "var(--st-training)", ring: "#d1a826", text: "#4a3a05" },
  event: { bg: "var(--st-event)", ring: "#d77b97", text: "#5a1f33" },
  ereignis: { bg: "var(--st-ereignis)", ring: "#78b25a", text: "#24451a" },
  regatta: { bg: "var(--st-regatta)", ring: "#5b9bd0", text: "#173a5c" },
  ergocup: { bg: "var(--st-regatta)", ring: "#5b9bd0", text: "#173a5c" },
  jtfo: { bg: "var(--st-regatta)", ring: "#5b9bd0", text: "#173a5c" },
};
export function stationColor(t: StationType) {
  return STATION_COLOR[t] ?? STATION_COLOR.setup;
}

export function isCompetition(t: StationType): boolean {
  return t === "regatta" || t === "ergocup" || t === "jtfo";
}

export function incidentLabel(i: IncidentKind | undefined): string {
  return i ? tData(`incident.${i}`, i) : "";
}

/** Context phrase of a trainer effect ("beim Ergo"). */
export function contextLabel(c: string): string {
  return tData(`context.${c}`, c);
}

/** Team colours for result bars and standings, by seat index. */
export const PLAYER_COLORS = ["#2b78bb", "#d9576b", "#4f8f32", "#a57ddf"];
export function playerColor(players: { id: string }[], id: string): string {
  const i = players.findIndex((p) => p.id === id);
  return PLAYER_COLORS[i >= 0 ? i % PLAYER_COLORS.length : 0];
}
