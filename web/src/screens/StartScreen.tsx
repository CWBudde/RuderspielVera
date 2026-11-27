import { useState } from "react";
import { ChevronDown, FlaskConical, Play, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { LakeScene } from "@/components/LakeScene";
import { readStorage, writeStorage } from "@/lib/storage";
import type { Session } from "@/api/client";
import { t } from "@/i18n";
import { LanguageToggle } from "@/components/LanguageToggle";

interface Props {
  stored: Session | null;
  mock: boolean;
  busy: boolean;
  onCreate: (name: string, seed: number | undefined) => void;
  onResume: () => void;
}

const NAME_KEY = "ruderspiel.name";

export function StartScreen({ stored, mock, busy, onCreate, onResume }: Props) {
  const [name, setName] = useState(() => readStorage(NAME_KEY) ?? "");
  const [seed, setSeed] = useState("");
  const [showSeed, setShowSeed] = useState(false);
  const seedNum = seed.trim() === "" ? undefined : Number(seed);
  const seedValid = seedNum === undefined || (Number.isInteger(seedNum) && seedNum >= 0);
  const canStart = name.trim().length > 0 && seedValid && !busy;

  return (
    <div className="flex min-h-dvh flex-col">
      <LakeScene className="h-[44dvh] min-h-[260px] border-b-2 border-ink md:h-[52dvh]">
        <div className="absolute inset-x-0 top-0 px-4 pt-[calc(var(--safe-top)+1.25rem)] text-center">
          <div className="absolute top-[calc(var(--safe-top)+0.75rem)] right-3">
            <LanguageToggle />
          </div>
          <h1
            className="num text-[clamp(3.4rem,17vw,7.5rem)] leading-[0.85] text-white uppercase"
            style={{ WebkitTextStroke: "2px var(--ink)", textShadow: "0 4px 0 var(--ink)" }}
          >
            {t("app.title")}
          </h1>
          <p className="mt-2 font-display text-lg font-bold text-navy italic md:text-xl">{t("start.tagline")}</p>
        </div>
      </LakeScene>

      <main className="relative z-10 mx-auto -mt-8 w-full max-w-md px-4 pb-[calc(var(--safe-bottom)+2rem)]">
        <form
          className="rounded-[20px] bg-cream p-4 cut"
          onSubmit={(e) => {
            e.preventDefault();
            if (!canStart) return;
            writeStorage(NAME_KEY, name.trim());
            onCreate(name.trim(), seedNum);
          }}
        >
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
            className="mt-3 inline-flex min-h-11 items-center gap-1 text-sm font-semibold text-muted-foreground"
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

          <Button type="submit" size="lg" className="mt-3 w-full" disabled={!canStart}>
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
        </form>
        {mock && (
          <p className="mt-4 flex items-center justify-center gap-1.5 text-sm font-semibold text-muted-foreground">
            <FlaskConical className="size-4" /> {t("start.mockMode")}
          </p>
        )}
      </main>
    </div>
  );
}
