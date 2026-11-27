import {
  Anchor,
  ClipboardList,
  CloudRain,
  CloudSun,
  Dice5,
  Dumbbell,
  Gauge,
  HeartPulse,
  Mountain,
  Settings,
  Snowflake,
  Sparkles,
  Sun,
  Tent,
  Trophy,
  Wind,
  Zap,
  type LucideIcon,
  type LucideProps,
} from "lucide-react";
import type { Stat, StationType, Weather } from "@/api/types.gen";
import { cn } from "@/lib/utils";

export const STAT_ICON: Record<string, LucideIcon> = {
  kraft: Dumbbell,
  ausdauer: HeartPulse,
  geschwindigkeit: Gauge,
  technik: Settings,
  motivation: Zap,
};
export const STAT_BG: Record<string, string> = {
  kraft: "bg-stat-kraft",
  ausdauer: "bg-stat-ausdauer",
  geschwindigkeit: "bg-stat-speed",
  technik: "bg-stat-technik",
  motivation: "bg-stat-motivation",
};

/** Round coloured stat badge as on the vera card. */
export function StatBadge({ stat, className, iconClassName }: { stat: Stat; className?: string; iconClassName?: string }) {
  const Icon = STAT_ICON[stat] ?? Zap;
  return (
    <span
      className={cn(
        "grid shrink-0 place-items-center rounded-full border-2 border-ink text-ink",
        STAT_BG[stat] ?? "bg-muted",
        className ?? "size-8",
      )}
      aria-hidden
    >
      <Icon className={cn("size-[55%]", iconClassName)} strokeWidth={2.4} />
    </span>
  );
}

const WEATHER_ICON: Record<string, LucideIcon> = {
  regen: CloudRain,
  hitze: Sun,
  wind: Wind,
  kaelte: Snowflake,
  normal: CloudSun,
};
const WEATHER_TINT: Record<string, string> = {
  regen: "bg-[#9cc3e6] text-[#123a63]",
  hitze: "bg-[#ffcf5a] text-[#7a3b00]",
  wind: "bg-[#cfe6ee] text-[#21495a]",
  kaelte: "bg-[#e3f1ff] text-[#24508a]",
  normal: "bg-[#fff1b8] text-[#5b4a10]",
};

export function WeatherIcon({ weather, className, ...rest }: { weather: Weather } & LucideProps) {
  const Icon = WEATHER_ICON[weather] ?? CloudSun;
  return <Icon className={className} {...rest} />;
}
export function weatherTint(w: Weather | undefined): string {
  return (w && WEATHER_TINT[w]) || WEATHER_TINT.normal;
}

/** Crossed sculls — training station glyph. */
export function OarsIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round">
      <path d="M5 19 17 5" />
      <path d="M19 19 7 5" />
      <path d="M15.5 4.5c1.2-1.2 3.2-1.4 4-.6.8.8.6 2.8-.6 4l-1.4-2z" fill="currentColor" />
      <path d="M8.5 4.5C7.3 3.3 5.3 3.1 4.5 3.9c-.8.8-.6 2.8.6 4l1.4-2z" fill="currentColor" />
    </svg>
  );
}

/** Small racing shell with rowers — regatta glyph. */
export function ShellIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 28 16" className={className} fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round">
      <path d="M1 11.5c6 1.6 20 1.6 26 0" />
      <circle cx="8" cy="5" r="1.6" fill="currentColor" stroke="none" />
      <circle cx="14" cy="5" r="1.6" fill="currentColor" stroke="none" />
      <circle cx="20" cy="5" r="1.6" fill="currentColor" stroke="none" />
      <path d="M8 7v3.5M14 7v3.5M20 7v3.5M5 14.5l5-5M11 14.5l5-5M17 14.5l5-5" />
    </svg>
  );
}

/** Crossed oars with laurel — card emblem. */
export function Emblem({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 48 48" className={className} aria-hidden>
      <g fill="none" stroke="#fff" strokeWidth={3.2} strokeLinecap="round">
        <path d="M12 38 34 10" />
        <path d="M36 38 14 10" />
      </g>
      <g fill="#fff">
        <path d="M31 12c1-3 4.5-5 6.5-3.6 1.8 1.3.6 4.8-2.2 6.4z" />
        <path d="M17 12c-1-3-4.5-5-6.5-3.6-1.8 1.3-.6 4.8 2.2 6.4z" />
      </g>
      <g fill="var(--sun)">
        {[0, 1, 2, 3].map((i) => (
          <ellipse key={`l${i}`} cx={8 + i * 1.2} cy={36 - i * 5} rx={1.7} ry={3.4} transform={`rotate(${-35 + i * 10} ${8 + i * 1.2} ${36 - i * 5})`} />
        ))}
        {[0, 1, 2, 3].map((i) => (
          <ellipse key={`r${i}`} cx={40 - i * 1.2} cy={36 - i * 5} rx={1.7} ry={3.4} transform={`rotate(${35 - i * 10} ${40 - i * 1.2} ${36 - i * 5})`} />
        ))}
      </g>
    </svg>
  );
}

export function StationIcon({ type, name, className }: { type: StationType; name?: string; className?: string }) {
  const cls = cn("size-5", className);
  if (type === "training") return <OarsIcon className={cls} />;
  if (type === "regatta") return <ShellIcon className={cls} />;
  if (type === "camp") return <Tent className={cls} strokeWidth={2} />;
  if (type === "event") return <Sparkles className={cls} strokeWidth={2} />;
  if (type === "ereignis") return <Dice5 className={cls} strokeWidth={2} />;
  if (type === "ergocup" || type === "jtfo") return <Trophy className={cls} strokeWidth={2} />;
  if (type === "setup") return <ClipboardList className={cls} strokeWidth={2} />;
  if (type === "selection") {
    if (name === "Wanderfahrt") return <Mountain className={cls} strokeWidth={2} />;
    if (name === "WR5") return <Snowflake className={cls} strokeWidth={2} />;
    return <BasketballIcon className={cls} />;
  }
  return <Anchor className={cls} />;
}

function BasketballIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth={2}>
      <circle cx="12" cy="12" r="9" />
      <path d="M3 12h18M12 3v18M5.6 5.6c3 3 3 9.8 0 12.8M18.4 5.6c-3 3-3 9.8 0 12.8" />
    </svg>
  );
}

export function MedalIcon({ medal, className }: { medal: "gold" | "silber"; className?: string }) {
  const fill = medal === "gold" ? "var(--gold)" : "var(--silver)";
  return (
    <svg viewBox="0 0 24 24" className={cn("size-5", className)} aria-label={medal === "gold" ? "Gold" : "Silber"}>
      <path d="M7 2h4l2 6H9z" fill="#c2412d" stroke="var(--ink)" strokeWidth={1.2} />
      <path d="M13 2h4l-2 6h-4z" fill="#2b78bb" stroke="var(--ink)" strokeWidth={1.2} />
      <circle cx="12" cy="15" r="6.5" fill={fill} stroke="var(--ink)" strokeWidth={1.6} />
      <text x="12" y="18" textAnchor="middle" fontSize="8" fontWeight={800} fill="var(--ink)" fontFamily="Barlow Condensed">
        {medal === "gold" ? "1" : "2"}
      </text>
    </svg>
  );
}
