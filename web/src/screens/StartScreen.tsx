import { useState } from "react";
import { ChevronDown, FlaskConical, LogIn, Play, RotateCcw, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { LakeScene } from "@/components/LakeScene";
import { readStorage, writeStorage } from "@/lib/storage";
import { multiplayerAvailable, type Session } from "@/api/client";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";
import { LanguageToggle } from "@/components/LanguageToggle";

interface Props {
  stored: Session | null;
  mock: boolean;
  busy: boolean;
  onCreate: (name: string, seed: number | undefined) => void;
  onCreateLobby: (name: string, seats: number, seed: number | undefined) => void;
  onJoin: (name: string, code: string) => void;
  onResume: () => void;
  /** Join code from an invite link (?join=…). */
  initialCode: string;
}

const NAME_KEY = "ruderspiel.name";

const SEAT_OPTIONS = [2, 3, 4];
const CODE_LENGTH = 6;

export function StartScreen({ stored, mock, busy, onCreate, onCreateLobby, onJoin, onResume, initialCode }: Props) {
  const [name, setName] = useState(() => readStorage(NAME_KEY) ?? "");
  const [seed, setSeed] = useState("");
  const [showSeed, setShowSeed] = useState(false);
  const seedNum = seed.trim() === "" ? undefined : Number(seed);
  const seedValid = seedNum === undefined || (Number.isInteger(seedNum) && seedNum >= 0);
  const canStart = name.trim().length > 0 && seedValid && !busy;
  const [seats, setSeats] = useState(3);
  const [code, setCode] = useState(initialCode);
  const cleanCode = code.replace(/[\s-]/g, "");
  const canJoin = name.trim().length > 0 && cleanCode.length === CODE_LENGTH && !busy;
  const showMulti = multiplayerAvailable && !mock;
  const remember = () => writeStorage(NAME_KEY, name.trim());

  return (
    <div className="flex min-h-dvh flex-col bg-navy">
      {/* The hero takes whatever height the dock leaves; anchoring the art low means a crop eats
          sky first and keeps the boat in view. On wide screens it stops at the art's 16:9 ratio. */}
      <LakeScene className="min-h-[300px] flex-1 md:max-h-[56.25vw]" imgClassName="object-[50%_88%]">
        <div className="pointer-events-none absolute inset-x-0 top-0 h-2/5 bg-linear-to-b from-ink/45 to-transparent" />
        <div className="absolute inset-x-0 top-0 px-4 pt-[calc(var(--safe-top)+3.75rem)] text-center sm:pt-[calc(var(--safe-top)+1.25rem)]">
          <div className="absolute top-[calc(var(--safe-top)+0.75rem)] right-3">
            <LanguageToggle />
          </div>
          <h1
            className="num text-[clamp(3.4rem,17vw,7.5rem)] leading-[0.85] text-white uppercase"
            style={{ WebkitTextStroke: "2px var(--ink)", textShadow: "0 4px 0 var(--ink)" }}
          >
            {t("app.title")}
          </h1>
          <p className="mt-3 inline-block -rotate-1 rounded-full bg-cream px-4 py-1 font-display text-lg font-bold text-navy italic cut-sm md:text-2xl">
            {t("start.tagline")}
          </p>
        </div>
      </LakeScene>

      {/* The dock: a solid band under the water that holds the form. */}
      <main className="relative z-10 border-t-2 border-ink bg-navy px-4 pt-0 pb-[calc(var(--safe-bottom)+1.5rem)] md:pt-5">
        <form
          className="mx-auto -mt-6 w-full max-w-md rounded-[20px] bg-cream p-4 cut md:mt-0 md:grid md:max-w-3xl md:grid-cols-[1fr_19rem] md:items-start md:gap-x-6 md:p-5"
          onSubmit={(e) => {
            e.preventDefault();
            if (!canStart) return;
            remember();
            onCreate(name.trim(), seedNum);
          }}
        >
          <div>
            <label htmlFor="name" className="block font-display text-lg font-bold">
              {t("start.nameLabel")}
            </label>
            <input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={24}
              autoComplete="nickname"
              placeholder={t("start.namePlaceholder")}
              className="mt-1 h-12 w-full rounded-xl border-2 border-ink bg-white px-3 text-base font-semibold outline-none focus-visible:ring-4 focus-visible:ring-lake/30"
            />

            <button
              type="button"
              className="mt-3 inline-flex min-h-11 items-center gap-1 text-sm font-semibold text-muted-foreground md:mt-1"
              aria-expanded={showSeed}
              onClick={() => setShowSeed((v) => !v)}
            >
              <ChevronDown className={`size-4 transition-transform ${showSeed ? "rotate-180" : ""}`} />
              {t("start.seedToggle")}
            </button>
            {showSeed && (
              <div className="mb-1">
                <input
                  id="seed"
                  inputMode="numeric"
                  value={seed}
                  onChange={(e) => setSeed(e.target.value.replace(/[^\d]/g, ""))}
                  placeholder={t("start.seedPlaceholder")}
                  aria-label={t("start.seedLabel")}
                  aria-invalid={!seedValid}
                  className="h-12 w-full rounded-xl border-2 border-ink bg-white px-3 text-base font-semibold tabular-nums outline-none focus-visible:ring-4 focus-visible:ring-lake/30"
                />
                <p className="mt-1 text-sm text-muted-foreground">{t("start.seedHint")}</p>
              </div>
            )}
          </div>

          <div className="md:pt-8">
            <Button type="submit" size="lg" className="mt-3 w-full md:mt-0" disabled={!canStart}>
              <Play className="fill-current" />
              {busy ? t("start.shuffling") : t("start.newGame", { count: 2 })}
            </Button>
            {stored && (
              <Button
                type="button"
                variant="outline"
                size="lg"
                className="mt-3 w-full"
                onClick={onResume}
                disabled={busy}
              >
                <RotateCcw />
                {t("start.resume")}
              </Button>
            )}
          </div>

          {showMulti && (
            <section
              aria-labelledby="multi-heading"
              className="mt-5 border-t-2 border-dashed border-ink/25 pt-4 md:col-span-2 md:grid md:grid-cols-2 md:gap-x-6"
            >
              <h2 id="multi-heading" className="flex items-center gap-2 font-display text-lg font-bold md:col-span-2">
                <Users className="size-5" /> {t("lobby.heading")}
              </h2>
              <p className="mb-3 text-sm text-muted-foreground md:col-span-2">{t("lobby.intro")}</p>

              <div>
                <p className="text-sm font-semibold" id="seats-label">
                  {t("lobby.seatsLabel")}
                </p>
                <div role="radiogroup" aria-labelledby="seats-label" className="mt-1 grid grid-cols-3 gap-2">
                  {SEAT_OPTIONS.map((n) => (
                    <button
                      key={n}
                      type="button"
                      role="radio"
                      aria-checked={seats === n}
                      onClick={() => setSeats(n)}
                      className={cn(
                        "min-h-11 rounded-xl border-2 border-ink font-bold tabular-nums",
                        seats === n ? "bg-navy text-cream" : "bg-white",
                      )}
                    >
                      {n}
                    </button>
                  ))}
                </div>
                <Button
                  type="button"
                  size="lg"
                  variant="outline"
                  className="mt-2 w-full"
                  disabled={!canStart}
                  onClick={() => {
                    remember();
                    onCreateLobby(name.trim(), seats, seedNum);
                  }}
                >
                  <Users />
                  {t("lobby.open")}
                </Button>
              </div>

              <div className="mt-4 md:mt-0">
                <label htmlFor="join-code" className="text-sm font-semibold">
                  {t("lobby.codeLabel")}
                </label>
                <input
                  id="join-code"
                  value={code}
                  onChange={(e) => setCode(e.target.value.toUpperCase())}
                  maxLength={8}
                  autoCapitalize="characters"
                  autoComplete="off"
                  spellCheck={false}
                  placeholder="K7QM2X"
                  className="mt-1 h-11 w-full rounded-xl border-2 border-ink bg-white px-3 text-center font-mono text-lg font-bold tracking-[0.3em] uppercase outline-none focus-visible:ring-4 focus-visible:ring-lake/30"
                />
                <Button
                  type="button"
                  size="lg"
                  variant={initialCode ? "default" : "outline"}
                  className="mt-2 w-full"
                  disabled={!canJoin}
                  onClick={() => {
                    remember();
                    onJoin(name.trim(), cleanCode);
                  }}
                >
                  <LogIn />
                  {t("lobby.join")}
                </Button>
              </div>
            </section>
          )}
        </form>
        {mock && (
          <p className="mt-4 flex items-center justify-center gap-1.5 text-sm font-semibold text-cream/70">
            <FlaskConical className="size-4" /> {t("start.mockMode")}
          </p>
        )}
      </main>
    </div>
  );
}
