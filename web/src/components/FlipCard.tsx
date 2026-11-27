import type { ReactNode } from "react";
import { Emblem } from "@/components/icons";
import { cn } from "@/lib/utils";

/**
 * A card that flips from its back to its face. The flip is a CSS 3D transition; with
 * prefers-reduced-motion the global rule shortens it to an instant swap.
 */
export function FlipCard({
  revealed,
  onReveal,
  front,
  backTone = "bg-navy",
  backLabel,
  className,
}: {
  revealed: boolean;
  onReveal: () => void;
  front: ReactNode;
  backTone?: string;
  backLabel: string;
  className?: string;
}) {
  return (
    <div className={cn("perspective-distant", className)}>
      <div
        className={cn(
          "relative size-full transition-transform duration-700 ease-[cubic-bezier(0.2,0.9,0.3,1.15)] transform-3d",
          revealed && "rotate-y-180",
        )}
      >
        <button
          type="button"
          onClick={onReveal}
          disabled={revealed}
          aria-label={`${backLabel} aufdecken`}
          className={cn(
            "absolute inset-0 grid place-items-center rounded-[18px] border-2 border-ink shadow-[0_3px_0_var(--ink)] backface-hidden",
            backTone,
          )}
        >
          <span className="flex flex-col items-center gap-2 text-white">
            <Emblem className="size-12 sm:size-16" />
            <span className="font-display text-lg font-bold">{backLabel}</span>
            <span className="px-2 text-center text-xs font-semibold text-white/80">Tippen zum Aufdecken</span>
          </span>
        </button>
        <div className="absolute inset-0 rotate-y-180 backface-hidden" aria-hidden={!revealed}>
          {front}
        </div>
      </div>
    </div>
  );
}
