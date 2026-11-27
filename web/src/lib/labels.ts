import type {
  AgeGroup,
  BoatClass,
  Gender,
  IncidentKind,
  Stat,
  StationType,
  Weather,
} from "@/api/types.gen";

/** Canonical display order (mirrors engine.AllStats). */
export const ALL_STATS: Stat[] = ["kraft", "ausdauer", "geschwindigkeit", "technik", "motivation"];
/** The four "card" stats; Motivation is shown separately as the yellow bar. */
export const CORE_STATS: Stat[] = ["kraft", "ausdauer", "geschwindigkeit", "technik"];

export const STAT_LABEL: Record<string, string> = {
  kraft: "Kraft",
  ausdauer: "Ausdauer",
  geschwindigkeit: "Geschwindigkeit",
  technik: "Technik",
  motivation: "Motivation",
};
export const STAT_SHORT: Record<string, string> = {
  kraft: "Kraft",
  ausdauer: "Ausd.",
  geschwindigkeit: "Tempo",
  technik: "Tech.",
  motivation: "Motiv.",
};
/** Mid-width card labels: only Geschwindigkeit is too long for a narrow pill. */
export const STAT_MEDIUM: Record<string, string> = { ...STAT_LABEL, geschwindigkeit: "Tempo" };
export function statLabel(s: Stat): string {
  return STAT_LABEL[s] ?? s;
}

export const AGE_LABEL: Record<string, string> = {
  kinder: "Kinder",
  b: "B-Junioren",
  a: "A-Junioren",
};
export const AGE_SHORT: Record<string, string> = { kinder: "K", b: "B", a: "A" };
export function ageLabel(a: AgeGroup): string {
  return AGE_LABEL[a] ?? a;
}
export const AGE_ORDER: Record<string, number> = { a: 0, b: 1, kinder: 2 };

export const GENDER_LABEL: Record<string, string> = { w: "Mädchen", m: "Junge" };
export const GENDER_PLURAL: Record<string, string> = { w: "Mädchen", m: "Jungen" };
export function genderLabel(g: Gender): string {
  return GENDER_LABEL[g] ?? g;
}

export const BOAT_CLASS_LABEL: Record<string, string> = {
  "1x": "Einer",
  "2x": "Zweier",
  "4x": "Vierer",
  "8x": "Achter",
  ergo: "Ergometer",
};
export function boatClassLabel(c: BoatClass): string {
  return BOAT_CLASS_LABEL[c] ?? c;
}

export const WEATHER_LABEL: Record<string, string> = {
  regen: "Regen",
  hitze: "Hitze",
  wind: "Wind",
  kaelte: "Kälte",
  normal: "Normales Wetter",
};
export function weatherLabel(w: Weather | undefined): string {
  if (!w) return "Kein Wetter";
  return WEATHER_LABEL[w] ?? w;
}

export const STATION_LABEL: Record<string, string> = {
  setup: "Spielvorbereitung",
  training: "Training",
  event: "Event",
  ereignis: "Ereignis",
  regatta: "Regatta",
  ergocup: "Ergo-Cup",
  selection: "Sonderveranstaltung",
  camp: "Trainingslager",
  jtfo: "JTFO",
};

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

export const INCIDENT_LABEL: Record<string, string> = {
  kentern: "Gekentert",
  material: "Materialschaden",
  krebs: "Krebs gefangen",
};
export function incidentLabel(i: IncidentKind | undefined): string {
  return (i && INCIDENT_LABEL[i]) || "";
}

export const CONTEXT_LABEL: Record<string, string> = {
  ergo: "beim Ergo",
  sprint: "im Sprint",
  langstrecke: "auf Langstrecke",
};

/** Team colours for result bars and standings, by seat index. */
export const PLAYER_COLORS = ["#2b78bb", "#d9576b", "#4f8f32", "#a57ddf"];
export function playerColor(players: { id: string }[], id: string): string {
  const i = players.findIndex((p) => p.id === id);
  return PLAYER_COLORS[i >= 0 ? i % PLAYER_COLORS.length : 0];
}
