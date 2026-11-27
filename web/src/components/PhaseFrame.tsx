import type { ReactNode } from "react";
import { Dice5 } from "lucide-react";
import { useGame } from "@/hooks/game";
import { StationIcon } from "@/components/icons";
import { stationColor } from "@/lib/labels";
import { cn } from "@/lib/utils";

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
  const c = stationColor(st?.type ?? "setup");
  const ereignisse = view.activeEreignisse ?? [];
  return (
    <div className={cn("flex flex-col", className)}>
      <header className="mb-3">
        <div className="flex items-start gap-3">
          <span
            className="mt-0.5 hidden size-12 shrink-0 place-items-center rounded-full border-2 border-ink sm:grid"
            style={{ background: c.bg, color: c.text }}
          >
            <StationIcon type={st?.type ?? "setup"} name={st?.name} className="size-6" />
          </span>
          <div className="min-w-0 flex-1">
            <h2 className="num text-[2rem] leading-[0.95] text-ink sm:text-4xl">{title}</h2>
            {prompt && <p className="mt-1.5 max-w-prose text-[15px] leading-snug text-ink/80">{prompt}</p>}
          </div>
        </div>
        {headerExtra}
        {ereignisse.length > 0 && (
          <ul className="mt-2 flex flex-wrap gap-1.5" aria-label="Aktive Ereignisse">
            {ereignisse.map((t) => (
              <li key={t} className="inline-flex items-center gap-1 rounded-full border-2 border-ink bg-st-ereignis px-2 py-0.5 text-xs font-bold">
                <Dice5 className="size-3.5" /> {t}
              </li>
            ))}
          </ul>
        )}
      </header>
      <div className="flex-1">{children}</div>
      {footer && (
        <div className="sticky bottom-[calc(var(--safe-bottom)+4.75rem)] z-20 mt-4 md:bottom-4">
          <div className="rounded-[18px] border-2 border-ink bg-cream/95 p-2 shadow-[0_3px_0_var(--ink)] backdrop-blur">{footer}</div>
        </div>
      )}
    </div>
  );
}
