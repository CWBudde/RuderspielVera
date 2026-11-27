import type {
  AgeGroup,
  BoatView,
  CompetitionView,
  Gender,
  RaceSpec,
  Stat,
  StationType,
  StationView,
  TrainerView,
} from "../types.gen";

/** Small seeded PRNG (mulberry32) so a mock game is reproducible from its seed. */
export class Rng {
  state: number;
  constructor(seed: number) {
    this.state = seed >>> 0 || 1;
  }
  next(): number {
    let t = (this.state = (this.state + 0x6d2b79f5) >>> 0);
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  }
  int(min: number, max: number): number {
    return min + Math.floor(this.next() * (max - min + 1));
  }
  pick<T>(arr: readonly T[]): T {
    return arr[Math.floor(this.next() * arr.length)];
  }
  shuffle<T>(arr: T[]): T[] {
    for (let i = arr.length - 1; i > 0; i--) {
      const j = Math.floor(this.next() * (i + 1));
      [arr[i], arr[j]] = [arr[j], arr[i]];
    }
    return arr;
  }
}

// ---------- calendar (Regelwerk §2.2, 52 stations) ----------

type StationDef = [StationType, string, string?];

const SECTIONS: StationDef[][] = [
  [["setup", "Spielvorbereitung"], ["training", "Training"], ["regatta", "Hildesheim"]],
  [["event", "Event"], ["training", "Training"], ["ereignis", "Ereignis"], ["ergocup", "Mindor-Cup"]],
  [["event", "Event"], ["selection", "WR5"], ["training", "Training"], ["ereignis", "Ereignis"], ["ergocup", "DRC-Ergo-Cup"]],
  [["event", "Event"], ["training", "Training"], ["ereignis", "Ereignis"], ["ergocup", "Osnabrücker Ergo-Cup"]],
  [
    ["event", "Event"],
    ["selection", "Basketballturnier"],
    ["training", "Training"],
    ["ereignis", "Ereignis"],
    ["camp", "Hankensbüttel"],
    ["regatta", "Regatta Bremen"],
  ],
  [["event", "Event"], ["training", "Training"], ["ereignis", "Ereignis"], ["camp", "Kassel"], ["regatta", "Kasselregatta"]],
  [["event", "Event"], ["training", "Training"], ["ereignis", "Ereignis"], ["regatta", "Verbandsregatta I"]],
  [
    ["event", "Event"],
    ["selection", "Wanderfahrt"],
    ["training", "Training"],
    ["ereignis", "Ereignis"],
    ["regatta", "Regatta Schillerschule"],
  ],
  [["event", "Event"], ["training", "Training"], ["ereignis", "Ereignis"], ["regatta", "Verbandsregatta II"]],
  [["event", "Event"], ["training", "Training"], ["ereignis", "Ereignis"], ["jtfo", "JTFO"]],
  [["event", "Event"], ["training", "Training"], ["ereignis", "Ereignis"], ["regatta", "RVH-Herbstsprint"]],
  [["event", "Event"], ["training", "Training"], ["ereignis", "Ereignis"], ["regatta", "RVB-Endspurt"]],
];

const INFO: Record<string, string> = {
  setup: "Startkarten werden verteilt",
  training: "10 Trainingspunkte verteilen",
  camp: "Trainingslager: 30 Punkte auf höchstens 5 Ruderer",
  event: "Persönliche Eventkarten werden gezogen",
  ereignis: "Ein Ereignis betrifft alle Teams",
  regatta: "Regatta auf dem Wasser",
  ergocup: "Einzelrennen auf dem Ergometer",
  jtfo: "Teamwettbewerb Jugend trainiert für Olympia",
  WR5: "Winterruder-Fünfkampf: 5 Ruderer +10 Motivation",
  Basketballturnier: "5 Ruderer +10 Motivation",
  Wanderfahrt: "5 Ruderer +4 Ausdauer und +3 Motivation",
};

export function buildCalendar(): StationView[] {
  const out: StationView[] = [];
  SECTIONS.forEach((sec, si) => {
    for (const [type, name] of sec) {
      out.push({ index: out.length, section: si + 1, type, name, info: INFO[name] ?? INFO[type] ?? "" });
    }
  });
  return out;
}

// ---------- names ----------

export const ROWER_NAMES: Record<string, [string, Gender][]> = {
  kinder: [
    ["Pauline", "w"], ["Mara", "w"], ["Nella", "w"], ["Johanna", "w"], ["Hannah", "w"], ["Paul", "m"],
    ["Johann", "m"], ["Justus", "m"], ["Joel", "m"], ["Ella", "w"], ["Yasmin", "w"], ["Malte", "m"], ["Sophia", "w"],
  ],
  b: [
    ["Vera", "w"], ["Lotti", "w"], ["Ida", "w"], ["Lea", "w"], ["Anton", "m"], ["Mailo", "m"], ["Luka", "m"],
    ["Jonas", "m"], ["Nilo", "m"], ["Julius", "m"], ["Sophia", "w"], ["Noah", "m"], ["Elin", "w"], ["Arun", "m"],
    ["Leni", "w"], ["Catha", "w"], ["Hannah", "w"], ["Tobias", "m"], ["Jacob", "m"], ["Christoph", "m"],
    ["Justus", "m"], ["Johann", "m"], ["Jan", "m"], ["Fabian", "m"], ["Lenn", "m"], ["Yukari", "w"], ["Mia", "w"],
  ],
  a: [
    ["Yara", "w"], ["Matilda", "w"], ["Lea", "w"], ["Annabelle", "w"], ["Emilie", "w"], ["Jonathan", "m"],
    ["Maxi", "m"], ["Fabi", "m"], ["Niclas", "m"], ["Michel", "m"], ["Jonte", "m"], ["Simon", "m"], ["Tara", "w"],
    ["Johanna", "w"], ["Matilda", "w"],
  ],
};

export const SUM_RANGE: Record<string, [number, number]> = { a: [86, 100], b: [59, 85], kinder: [46, 58] };

/** Archetypes split a stat sum; weights per stat in display order. */
export const ARCHETYPES: { name: string; w: Record<Stat, number> }[] = [
  { name: "Kraftpaket", w: { kraft: 3, ausdauer: 2, geschwindigkeit: 1.5, technik: 1, motivation: 1 } },
  { name: "Ausdauerläufer", w: { kraft: 1.5, ausdauer: 3, geschwindigkeit: 1, technik: 1.5, motivation: 1 } },
  { name: "Sprinter", w: { kraft: 2, ausdauer: 1, geschwindigkeit: 3, technik: 1.5, motivation: 1 } },
  { name: "Techniker", w: { kraft: 1, ausdauer: 1.5, geschwindigkeit: 1.5, technik: 3, motivation: 1 } },
  { name: "Kämpferherz", w: { kraft: 1.5, ausdauer: 1.5, geschwindigkeit: 1.5, technik: 1, motivation: 2.5 } },
];

export function splitStats(rng: Rng, sum: number): Record<Stat, number> {
  const arch = rng.pick(ARCHETYPES);
  const stats: Stat[] = ["kraft", "ausdauer", "geschwindigkeit", "technik", "motivation"];
  const raw = stats.map((s) => arch.w[s] * (0.8 + rng.next() * 0.4));
  const total = raw.reduce((a, b) => a + b, 0);
  const free = sum - 5 * stats.length;
  const vals = raw.map((r) => 5 + Math.floor((r / total) * free));
  let rest = sum - vals.reduce((a, b) => a + b, 0);
  while (rest > 0) {
    vals[rng.int(0, 4)]++;
    rest--;
  }
  return Object.fromEntries(stats.map((s, i) => [s, vals[i]])) as Record<Stat, number>;
}

// ---------- boats ----------

export function cityBoats(owner: string): BoatView[] {
  return (["1x", "2x", "4x"] as const).map((c) => ({
    id: `${owner}-city-${c}`,
    name: { "1x": "Stadt-Einer", "2x": "Stadt-Zweier", "4x": "Stadt-Vierer" }[c],
    class: c,
    value: 3,
    city: true,
    artKey: `boat-${c}`,
  }));
}

export const NAMED_BOATS: [string, string, number][] = [
  ["Ice-Man", "1x", 5], ["Manpower", "1x", 6], ["Kalypso", "1x", 6], ["Reinfall", "1x", 5],
  ["Bj-Turbo", "2x", 6], ["Black Sword", "2x", 7], ["Goodspeed", "2x", 6],
  ["Simsalabim", "4x", 6], ["Oshra", "4x", 7], ["Need 4 Speed", "4x", 7],
  ["Zwergenkreuzer", "1x", 8], ["Traumstart", "1x", 9], ["Rennschlumpf", "1x", 8], ["Singlespeed", "1x", 10],
  ["Freiheit", "2x", 8], ["X-Wing", "2x", 10], ["Laufwunder", "2x", 9], ["H zwei O", "2x", 9],
  ["Q", "4x", 10], ["5 Freunde", "4x", 9], ["Wilhelm", "4x", 8],
];

export function namedBoats(): BoatView[] {
  return NAMED_BOATS.map(([name, cls, value], i) => ({
    id: `boat-${i + 1}`,
    name,
    class: cls,
    value,
    city: false,
    artKey: `boat-${cls}`,
  }));
}

// ---------- trainers ----------

export const TRAINERS: TrainerView[] = [
  {
    id: "tr-sack", name: "Herr Sack", ability: "Sonnenanbeter", artKey: "trainer-m",
    description: "Bei Hitze +5 pro Boot, bei Regen −5.",
    effects: [{ weather: "hitze", value: 5 }, { weather: "regen", value: -5 }],
  },
  {
    id: "tr-linke", name: "Frau Linke", ability: "Regenjacke", artKey: "trainer-w",
    description: "Bei Regen +4 pro Boot, bei Wind −3.",
    effects: [{ weather: "regen", value: 4 }, { weather: "wind", value: -3 }],
  },
  {
    id: "tr-caro", name: "Caro", ability: "Rückenwind", artKey: "trainer-w",
    description: "Bei Wind +5 pro Boot, bei Kälte −2.",
    effects: [{ weather: "wind", value: 5 }, { weather: "kaelte", value: -2 }],
  },
  {
    id: "tr-hannah", name: "Hannah", ability: "Ergo-Expertin", artKey: "trainer-w",
    description: "+3 pro Ergo-Start.",
    effects: [{ context: "ergo", value: 3 }],
  },
  {
    id: "tr-julian", name: "Julian", ability: "Sprintcoach", artKey: "trainer-m",
    description: "+2 pro Boot bei Sprintregatten.",
    effects: [{ context: "sprint", value: 2 }],
  },
  {
    id: "tr-aron", name: "Aron", ability: "Langer Atem", artKey: "trainer-m",
    description: "+3 pro Boot auf der Langstrecke.",
    effects: [{ context: "langstrecke", value: 3 }],
  },
  {
    id: "tr-salim", name: "Salim", ability: "Eisbader", artKey: "trainer-m",
    description: "Bei Kälte +4 pro Boot, bei Hitze −2.",
    effects: [{ weather: "kaelte", value: 4 }, { weather: "hitze", value: -2 }],
  },
  {
    id: "tr-friedjof", name: "Friedjof", ability: "Ruhige Hand", artKey: "trainer-m",
    description: "Bei normalem Wetter +3 pro Boot.",
    effects: [{ weather: "normal", value: 3 }],
  },
  {
    id: "tr-josef", name: "Josef", ability: "Allwetter", artKey: "trainer-m",
    description: "Bei Regen +3, bei Hitze +2 pro Boot.",
    effects: [{ weather: "regen", value: 3 }, { weather: "hitze", value: 2 }],
  },
  {
    id: "tr-clara", name: "Clara", ability: "Windflüsterin", artKey: "trainer-w",
    description: "Bei Wind +3, bei Regen +2 pro Boot.",
    effects: [{ weather: "wind", value: 3 }, { weather: "regen", value: 2 }],
  },
];

// ---------- events and Ereignisse ----------

export const EVENT_DECK: { type: string; name: string; description: string; weight: number }[] = [
  { type: "wetter", name: "Wetter-Event", description: "Boot dieses Ruderers +4, wenn das Wetter nicht normal ist.", weight: 2 },
  { type: "wachstum", name: "Wachstum", description: "Bester Wert +2 bei jedem Training, auch im Trainingslager.", weight: 2 },
  { type: "phoenix", name: "Phönix", description: "Schwächster Wert sofort +5.", weight: 2 },
  { type: "tausch", name: "Tausch", description: "Tausche zwei Werte dieses Ruderers.", weight: 2 },
  { type: "verdopplung", name: "Verdopplung", description: "Der aktuell beste Wert wird einmalig verdoppelt.", weight: 1 },
  { type: "kraft", name: "Kraft-Event", description: "Kraft +8.", weight: 2 },
  { type: "ausdauer", name: "Ausdauer-Event", description: "Ausdauer +8.", weight: 2 },
  { type: "geschwindigkeit", name: "Geschwindigkeits-Event", description: "Geschwindigkeit +8.", weight: 2 },
  { type: "technik", name: "Technik-Event", description: "Technik +8.", weight: 2 },
  { type: "motivation", name: "Motivations-Event", description: "Motivation +8.", weight: 2 },
  { type: "zwilling", name: "Seelenzwilling", description: "Ein Zwilling mit 80 % der Werte kommt ins Team.", weight: 1 },
  { type: "mission", name: "Geheime Mission", description: "Erfülle die Mission bis Saisonende.", weight: 1 },
];

export const MISSIONS: { text: string; reward: string }[] = [
  { text: "Gold im Einer vor JTFO", reward: "+10 auf den besten Wert" },
  { text: "Bei einem Ergo-Cup gewinnen", reward: "+1 Medaillenpunkt" },
  { text: "Zwei Goldmedaillen in einer Regatta", reward: "+1 Medaillenpunkt" },
  { text: "Bei Regen eine Medaille holen", reward: "+10 auf den besten Wert" },
];

export const EREIGNISSE: { name: string; text: string; active?: string }[] = [
  { name: "Trainerstreik", text: "Die Trainer streiken: Das nächste Training bringt nur 5 Punkte.", active: "Trainerstreik: nächstes Training nur 5 Punkte" },
  { name: "Maschsee-Monster", text: "Etwas Großes schwimmt im Maschsee. Bei der nächsten Wasserregatta steigt die Krebsgefahr auf 30 %.", active: "Maschsee-Monster: nächste Wasserregatta Krebs 30 %" },
  { name: "Grippewelle", text: "Je Team verliert ein zufälliger Ruderer 5 Motivation." },
  { name: "Vereinsfest", text: "Alle Ruderer bekommen +1 Motivation." },
  { name: "Neue Ergometer", text: "Beim nächsten Ergo-Cup zählt Kraft dreifach.", active: "Neue Ergometer: nächster Ergo-Cup Kraft ×3" },
  { name: "Sponsor", text: "Das Team auf dem letzten Platz bekommt beim nächsten Draft eine Extrakarte.", active: "Sponsor: Extrakarte für den Letzten im nächsten Draft" },
  { name: "Hitzewelle", text: "Das Wetter der nächsten Regatta ist sicher Hitze.", active: "Hitzewelle: nächste Regatta Hitze" },
  { name: "Ruhige Woche", text: "Nichts passiert. Durchatmen am Steg." },
  { name: "Bootshaus-Renovierung", text: "Städtische Boote −1 bei der nächsten Regatta.", active: "Renovierung: städtische Boote −1 bei der nächsten Regatta" },
  { name: "Doppeltraining", text: "Das nächste Training bringt 15 Punkte.", active: "Doppeltraining: nächstes Training 15 Punkte" },
];

// ---------- competitions ----------

function groupName(age: AgeGroup, gender: Gender): string {
  if (age === "kinder") return gender === "w" ? "Mädchen" : "Jungen";
  return `${gender === "w" ? "Juniorinnen" : "Junioren"} ${age.toUpperCase()}`;
}
const CLASS_NAME: Record<string, string> = { "1x": "Einer", "2x": "Zweier", "4x": "Vierer", "8x": "Achter", ergo: "Ergo" };
const SEATS: Record<string, number> = { "1x": 1, "2x": 2, "4x": 4, "8x": 8, ergo: 1 };

function race(id: string, cls: string, ages: AgeGroup[], genders: Gender[], mixed = false, name?: string): RaceSpec {
  const label =
    name ??
    (ages.length === 1 && genders.length === 1
      ? `${groupName(ages[0], genders[0])} ${CLASS_NAME[cls]}`
      : `${genders.length === 1 ? (genders[0] === "w" ? "Frauen" : "Männer") : "Offen"} ${CLASS_NAME[cls]}`);
  return { id, name: label, class: cls, ages, genders, mixed, seats: SEATS[cls] };
}

const MATRIX: [string, AgeGroup, Gender][] = [];
for (const cls of ["1x", "2x", "4x"]) {
  for (const age of ["kinder", "b", "a"] as AgeGroup[]) {
    for (const g of ["w", "m"] as Gender[]) MATRIX.push([cls, age, g]);
  }
}

export function competitionFor(name: string, kind: StationType, rotation: number): CompetitionView {
  const base = { name, kind, teamEvent: false, notes: [] as string[] };
  if (kind === "ergocup") {
    const races: RaceSpec[] = [];
    for (const age of ["kinder", "b", "a"] as AgeGroup[]) {
      for (const g of ["w", "m"] as Gender[]) races.push(race(`ergo-${age}-${g}`, "ergo", [age], [g]));
    }
    return { ...base, weights: { kraft: 2, motivation: 1 }, races, notes: ["Kein Boot, kein Wetter, keine Zwischenfälle."] };
  }
  if (kind === "jtfo") {
    return {
      ...base,
      teamEvent: true,
      weights: { geschwindigkeit: 1, technik: 1 },
      races: [
        { ...race("jtfo-2x-1", "2x", ["b", "a"], ["w", "m"], false, "Schulteam Zweier I"), loanBoat: true },
        { ...race("jtfo-4x", "4x", ["kinder", "b", "a"], ["w", "m"], false, "Schulteam Vierer"), loanBoat: true },
        { ...race("jtfo-2x-2", "2x", ["kinder", "b"], ["w", "m"], false, "Schulteam Zweier II"), loanBoat: true },
      ],
      notes: ["Teamwertung: Die drei Boote zählen zusammen, eine Medaille für das ganze Team."],
    };
  }
  if (name === "Hildesheim") {
    const races: RaceSpec[] = [];
    for (const cls of ["1x", "2x"]) {
      for (const age of ["b", "a"] as AgeGroup[]) {
        for (const g of ["w", "m"] as Gender[]) races.push(race(`hi-${cls}-${age}-${g}`, cls, [age], [g]));
      }
    }
    return { ...base, weights: { ausdauer: 2, motivation: 1 }, races, notes: ["Langstrecke: Ausdauer zählt doppelt."] };
  }
  if (name === "RVH-Herbstsprint") {
    const races: RaceSpec[] = [];
    for (const cls of ["1x", "2x", "4x"]) {
      for (const g of ["w", "m"] as Gender[]) races.push(race(`rvh-${cls}-${g}`, cls, ["kinder", "b", "a"], [g]));
    }
    return { ...base, weights: { geschwindigkeit: 2, kraft: 1 }, races, notes: ["Sprint: Altersklassen gemischt."] };
  }
  if (name === "RVB-Endspurt") {
    return {
      ...base,
      weights: { geschwindigkeit: 2, technik: 1, motivation: 0.5 },
      races: [
        race("rvb-4x", "4x", ["kinder", "b", "a"], ["w", "m"], true, "Mixed-Vierer"),
        race("rvb-8x", "8x", ["kinder", "b", "a"], ["w", "m"], true, "Mixed-Achter"),
      ],
      notes: [
        "Mixed: mindestens ein Mädchen, ein Junge und zwei Altersklassen.",
        "Der Achter ist für alle gleich (Bootswert 0).",
      ],
    };
  }
  const weights: Record<string, number> = name.startsWith("Verbandsregatta")
    ? { technik: 1.5, geschwindigkeit: 1, motivation: 0.5 }
    : name === "Kasselregatta"
      ? { geschwindigkeit: 1, kraft: 1, motivation: 0.5 }
      : { geschwindigkeit: 1, technik: 1, motivation: 0.5 };
  const races: RaceSpec[] = [];
  for (let i = 0; i < 6; i++) {
    const [cls, age, g] = MATRIX[(rotation + i * 3) % MATRIX.length];
    races.push(race(`r${rotation}-${cls}-${age}-${g}`, cls, [age], [g]));
  }
  return { ...base, weights, races };
}
