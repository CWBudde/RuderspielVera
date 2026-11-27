import { LANGS, setLang, t, useLang, type Lang } from "@/i18n";
import { cn } from "@/lib/utils";

const LANG_NAME: Record<Lang, "lang.de" | "lang.en"> = { de: "lang.de", en: "lang.en" };

/** Compact DE / EN switch; the active language is highlighted. */
export function LanguageToggle({ className }: { className?: string }) {
  const lang = useLang();
  return (
    <div
      role="group"
      aria-label={t("lang.label")}
      className={cn(
        "flex shrink-0 overflow-hidden rounded-full border-2 border-ink bg-cream text-xs font-bold",
        className,
      )}
    >
      {LANGS.map((l) => (
        <button
          key={l}
          type="button"
          lang={l}
          aria-pressed={l === lang}
          aria-label={t(LANG_NAME[l])}
          onClick={() => setLang(l)}
          className={cn(
            "min-h-8 min-w-9 px-2 uppercase transition-colors",
            l === lang ? "bg-navy text-white" : "text-ink/70 hover:bg-ink/10",
          )}
        >
          {l}
        </button>
      ))}
    </div>
  );
}
