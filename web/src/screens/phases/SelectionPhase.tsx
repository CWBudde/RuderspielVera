import { useMemo, useState } from "react";
import { Check, Send, Sparkles } from "lucide-react";
import { useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import { RowerRow } from "@/components/RowerCard";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { sortRowers } from "@/screens/TeamScreen";
import { t, tm } from "@/i18n";
import { selectionEffect } from "@/i18n/data";

export function SelectionPhase() {
  const { view, you, submit, submitting } = useGame();
  const count = Math.min(view.phase.count ?? 5, you.rowers.length);
  const [picked, setPicked] = useState<string[]>([]);
  const rowers = useMemo(() => sortRowers(you.rowers), [you.rowers]);
  const full = picked.length >= count;

  const toggle = (id: string) =>
    setPicked((p) => (p.includes(id) ? p.filter((x) => x !== id) : p.length < count ? [...p, id] : p));

  return (
    <PhaseFrame
      title={tm(view.phase.title)}
      prompt={tm(view.phase.prompt)}
      headerExtra={
        view.phase.selectionEffect && (
          <p className="mt-3 inline-flex items-center gap-2 rounded-xl border-2 border-ink bg-st-training px-3 py-2 font-display text-lg font-bold">
            <Sparkles className="size-5" /> {t("selection.effect", { effect: selectionEffect(view.phase) })}
          </p>
        )
      }
      footer={
        <div className="flex items-center gap-2">
          <p className="min-w-0 flex-1 pl-1" aria-live="polite">
            <span className={cn("num text-3xl leading-none", full ? "text-leaf" : "")}>{picked.length}</span>
            <span className="text-sm font-semibold"> {t("selection.ofPicked", { count })}</span>
          </p>
          <Button
            size="lg"
            onClick={() => submit({ rowerIds: picked })}
            disabled={picked.length !== count || submitting}
          >
            <Send /> {t("common.confirm")}
          </Button>
        </div>
      }
    >
      <ul className="grid gap-2 lg:grid-cols-2">
        {rowers.map((r) => {
          const on = picked.includes(r.id);
          return (
            <li key={r.id}>
              <button
                type="button"
                role="checkbox"
                aria-checked={on}
                onClick={() => toggle(r.id)}
                disabled={!on && full}
                className="w-full text-left disabled:opacity-50"
              >
                <RowerRow
                  rower={r}
                  className={cn(on && "bg-st-training ring-4 ring-ink/80")}
                  right={
                    <span
                      className={cn(
                        "grid size-8 shrink-0 place-items-center rounded-full border-2 border-ink",
                        on ? "bg-navy text-white" : "bg-white",
                      )}
                    >
                      {on && <Check className="size-5" strokeWidth={3} />}
                    </span>
                  }
                />
              </button>
            </li>
          );
        })}
      </ul>
    </PhaseFrame>
  );
}
