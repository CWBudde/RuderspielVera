import type { ReactNode } from "react";
import { Dice5 } from "lucide-react";
import { useGame } from "@/hooks/game";
import { stationName } from "@/i18n/data";
import { cn } from "@/lib/utils";
import { t, tm } from "@/i18n";

/** Common frame for every Aktion view: title, prompt, content and a sticky action bar. */
export function PhaseFrame({
  title,
  prompt,
  children,
  footer,
  headerExtra,
  className,
}: {
  title: string;
  prompt?: string;
  children: ReactNode;
  footer?: ReactNode;
  headerExtra?: ReactNode;
  className?: string;
}) {
  const { view } = useGame();
  const st = view.calendar[view.current];
  const ownTitle = title !== stationName(st);
  const ereignisse = view.activeEreignisse ?? [];
  return (
    <div className={cn("flex flex-col", className)}>
      <header className="mb-3">
        {/* The shell header already shows the station icon and name; repeat neither. Only a
            phase with a title of its own (draft, swap, result …) gets a visible heading. */}
        <h2 className={cn("num text-[2rem] leading-[0.95] text-ink sm:text-4xl", ownTitle ? "mb-1.5" : "sr-only")}>
          {title}
        </h2>
        {prompt && (
          <p
            className={cn(
              "max-w-prose leading-snug",
              ownTitle ? "text-[15px] text-ink/80" : "text-base font-semibold text-ink/85 sm:text-lg",
            )}
          >
            {prompt}
          </p>
        )}
        {headerExtra}
        {ereignisse.length > 0 && (
          <ul className="mt-2 flex flex-wrap gap-1.5" aria-label={t("phase.activeEreignisse")}>
            {ereignisse.map((m, i) => (
              <li
                key={i}
                className="inline-flex items-center gap-1 rounded-full border-2 border-ink bg-st-ereignis px-2 py-0.5 text-xs font-bold"
              >
                <Dice5 className="size-3.5" /> {tm(m)}
              </li>
            ))}
          </ul>
        )}
      </header>
      <div className="flex-1">{children}</div>
      {footer && (
        <div className="sticky bottom-[calc(var(--safe-bottom)+4.75rem)] z-20 mt-4 md:bottom-4">
          <div className="rounded-[18px] border-2 border-ink bg-cream/95 p-2 shadow-[0_3px_0_var(--ink)] backdrop-blur">
            {footer}
          </div>
        </div>
      )}
    </div>
  );
}
