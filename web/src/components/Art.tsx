import { useState } from "react";
import { cn } from "@/lib/utils";
import { PEOPLE_MANIFEST } from "@/components/peopleArt";

export { PEOPLE_MANIFEST };

/**
 * Art manifest: every `src/assets/art/<artKey>.{webp,png,jpg}` is picked up at build time.
 * Dropping a file with the right name replaces the placeholder — no code change needed.
 */
const files = import.meta.glob<string>("../assets/art/*.{webp,png,jpg}", {
  eager: true,
  query: "?url",
  import: "default",
});

export const ART_MANIFEST: Record<string, string> = Object.fromEntries(
  Object.entries(files).map(([path, url]) => [path.replace(/^.*\/(.+)\.\w+$/, "$1"), url]),
);

/**
 * Generic art keys without an own file, mapped to the numbered portraits that fit them
 * (trainer-1/-3 are men, trainer-2/-4 women). Unlisted keys fall back to their family.
 */
const VARIANTS: Record<string, string[]> = {
  "trainer-m": ["trainer-1", "trainer-3"],
  "trainer-w": ["trainer-2", "trainer-4"],
};

/**
 * Resolves an artKey to a file. Exact match first; otherwise one of the key's variants (or the
 * numbered files of its family, e.g. "boat-x" -> "boat-1..n") is picked deterministically by
 * `seed`, which should be the card id so a card always shows the same face.
 */
export function artUrl(key: string | undefined, seed = ""): string | undefined {
  if (!key) return undefined;
  if (ART_MANIFEST[key]) return ART_MANIFEST[key];
  const family = key.split("-")[0];
  let variants = (VARIANTS[key] ?? []).filter((k) => ART_MANIFEST[k]);
  if (variants.length === 0) {
    variants = Object.keys(ART_MANIFEST)
      .filter((k) => new RegExp(`^${family}-\\d+$`).test(k))
      .sort();
  }
  if (variants.length === 0) return undefined;
  return ART_MANIFEST[variants[hash(seed + key) % variants.length]];
}

interface ArtProps {
  artKey: string;
  /** Used for the placeholder's initials and colour variation. */
  label?: string;
  className?: string;
  alt?: string;
  /** Small square crop: zooms portrait art onto the face. */
  thumb?: boolean;
  /** Rower or trainer card id: `people/<cardId>` wins over the artKey (a "-zwilling" twin shares it). */
  cardId?: string;
}

/** Image for an artKey, falling back to a drawn placeholder (portrait, boat or trainer). */
export function Art({ artKey, label, className, alt, thumb, cardId }: ArtProps) {
  const url = (cardId && PEOPLE_MANIFEST[cardId.replace(/-zwilling$/, "")]) || artUrl(artKey, cardId ?? label);
  const [failed, setFailed] = useState(false);
  if (url && !failed) {
    return (
      <img
        src={url}
        alt={alt ?? label ?? ""}
        className={cn(
          "size-full object-cover",
          thumb && !artKey.startsWith("boat") && "origin-[38%_30%] scale-[1.9]",
          className,
        )}
        loading="lazy"
        onError={() => setFailed(true)}
      />
    );
  }
  if (artKey.startsWith("boat")) return <BoatPlaceholder artKey={artKey} className={className} />;
  return <PortraitPlaceholder artKey={artKey} label={label ?? ""} className={className} />;
}

function hash(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
  return h >>> 0;
}

const HAIR = ["#6b4423", "#b8873a", "#e2b55c", "#3b2a1e", "#8a4b2a", "#1f1a17", "#c9a46a"];
const SKIN = ["#f2c8a4", "#e8b48f", "#d29b74", "#b57a55", "#f6d3b5"];

/** Maschsee backdrop + bust silhouette in the club jersey, with initials. */
function PortraitPlaceholder({ artKey, label, className }: { artKey: string; label: string; className?: string }) {
  const h = hash(label + artKey);
  const hair = HAIR[h % HAIR.length];
  const skin = SKIN[(h >> 3) % SKIN.length];
  const isTrainer = artKey.startsWith("trainer");
  const female = artKey.endsWith("-w");
  const young = artKey.includes("kinder");
  const initials = label
    .replace(/\(.*\)/, "")
    .trim()
    .split(/\s+/)
    .map((p) => p[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
  const headY = young ? 47 : 43;
  const headR = young ? 13 : 12;
  return (
    <svg
      viewBox="0 0 100 100"
      preserveAspectRatio="xMidYMid slice"
      className={cn("size-full", className)}
      role="img"
      aria-label={label}
    >
      <defs>
        <linearGradient id={`sky${h}`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#bfe0f5" />
          <stop offset="1" stopColor="#eaf5fb" />
        </linearGradient>
      </defs>
      <rect width="100" height="100" fill={`url(#sky${h})`} />
      {/* far shore with town hall dome */}
      <path d="M0 52c10-4 18-2 26-5s14 1 22-2 18 2 26-1 18 1 26-2v12H0z" fill="#6fa257" />
      <path d="M58 46v-6h2v-3c0-2 2-4 4-4s4 2 4 4v3h2v6z" fill="#c9b79a" opacity="0.8" />
      <rect y="56" width="100" height="44" fill="#3d8bc9" />
      <path
        d="M0 66h100M0 76h100M0 88h100"
        stroke="#fff"
        strokeOpacity="0.18"
        strokeWidth="1.5"
        strokeDasharray="10 7"
      />
      {/* bust */}
      <path
        d={`M${isTrainer ? 14 : 16} 100c0-20 15-30 34-30s34 10 34 30z`}
        fill={isTrainer ? "#24476e" : "#2a4f8f"}
        stroke="var(--ink)"
        strokeWidth="1.6"
      />
      {!isTrainer && <path d="M36 74l14 14 14-14" fill="none" stroke="#fff" strokeWidth="3" strokeOpacity="0.85" />}
      <rect x={45} y={headY + headR - 4} width={10} height={10} fill={skin} />
      {female && (
        <path d={`M${50 - headR - 3} ${headY}c0 18 4 26 ${headR + 3} 26s${headR + 3}-8 ${headR + 3}-26z`} fill={hair} />
      )}
      <circle cx={50} cy={headY} r={headR} fill={skin} stroke="var(--ink)" strokeWidth="1.4" />
      {isTrainer ? (
        <path
          d={`M${50 - headR - 1} ${headY - 3}c0-10 6-13 ${headR + 1}-13s${headR + 1} 3 ${headR + 1} 13zM${50 + headR - 2} ${headY - 4}h12`}
          fill="#e7e2d6"
          stroke="var(--ink)"
          strokeWidth="1.4"
        />
      ) : (
        <path
          d={`M${50 - headR} ${headY - 1}c0-9 5-${headR} ${headR}-${headR}s${headR} 3 ${headR} ${headR}c-6-4-14-3-${headR * 2} 0z`}
          fill={hair}
        />
      )}
      {initials && (
        <text
          x="94"
          y="94"
          textAnchor="end"
          fontFamily="Barlow Condensed"
          fontStyle="italic"
          fontWeight={800}
          fontSize="15"
          fill="#fff"
          fillOpacity="0.55"
        >
          {initials}
        </text>
      )}
    </svg>
  );
}

function BoatPlaceholder({ artKey, className }: { artKey: string; className?: string }) {
  const seats = artKey.includes("8x") ? 8 : artKey.includes("4x") ? 4 : artKey.includes("2x") ? 2 : 1;
  const span = 70;
  return (
    <svg viewBox="0 0 100 50" preserveAspectRatio="xMidYMid slice" className={cn("size-full", className)} aria-hidden>
      <rect width="100" height="50" fill="#bfe0f5" />
      <path d="M0 18c12-3 20 0 30-2s16 1 26-1 22 1 44-1v6H0z" fill="#6fa257" />
      <rect y="22" width="100" height="28" fill="#3d8bc9" />
      <path d="M8 34c20 5 64 5 84 0l-3 3c-20 4-58 4-78 0z" fill="#f3c431" stroke="var(--ink)" strokeWidth="1.2" />
      {Array.from({ length: seats }, (_, i) => {
        const x = 50 - span / 2 + ((i + 0.5) * span) / seats;
        return (
          <g key={i}>
            <path d={`M${x} 34l-9 9M${x} 34l9 9`} stroke="var(--ink)" strokeWidth="1" opacity="0.6" />
            <circle cx={x} cy={29} r={2} fill="#2a4f8f" stroke="var(--ink)" strokeWidth="0.8" />
          </g>
        );
      })}
    </svg>
  );
}
