import { useState } from "react";
import { ArrowLeftRight } from "lucide-react";
import type { Stat } from "@/api/types.gen";
import { useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import { RowerCard } from "@/components/RowerCard";
import { StatBadge } from "@/components/icons";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { CORE_STATS, statLabel } from "@/lib/labels";
import { cn } from "@/lib/utils";
import { t, tm } from "@/i18n";
import { rowerName } from "@/i18n/data";

export function SwapPhase() {
  const { view, you, submit, submitting } = useGame();
  const rower = you.rowers.find((r) => r.id === view.phase.swapRowerId);
  const [open, setOpen] = useState(true);
  const [pick, setPick] = useState<Stat[]>([]);

  const toggle = (s: Stat) =>
    setPick((p) => (p.includes(s) ? p.filter((x) => x !== s) : p.length < 2 ? [...p, s] : [p[1], s]));
  const preview: Partial<Record<Stat, number>> = {};
  if (rower && pick.length === 2) {
    const [a, b] = pick;
    preview[a] = (rower.stats[b] ?? 0) - (rower.stats[a] ?? 0);
    preview[b] = (rower.stats[a] ?? 0) - (rower.stats[b] ?? 0);
  }

  return (
    <PhaseFrame
      title={tm(view.phase.title)}
      prompt={tm(view.phase.prompt)}
      footer={
        <div className="flex gap-2">
          <Button variant="outline" size="lg" className="flex-1" onClick={() => submit({})} disabled={submitting}>
            {t("swap.skip")}
          </Button>
          <Button size="lg" className="flex-1" onClick={() => setOpen(true)}>
            <ArrowLeftRight /> {t("swap.chooseStats")}
          </Button>
        </div>
      }
    >
      {rower ? (
        <div className="mx-auto max-w-xs">
          <RowerCard rower={rower} preview={preview} />
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">{t("swap.rowerNotFound")}</p>
      )}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="bg-cream">
          <DialogHeader>
            <DialogTitle className="font-display text-2xl">{t("swap.title")}</DialogTitle>
            <DialogDescription>
              {rower ? t("swap.descriptionRower", { name: rowerName(rower) }) : t("swap.description")}
            </DialogDescription>
          </DialogHeader>
          {rower && (
            <ul className="flex flex-col gap-1.5" role="group" aria-label={t("swap.stats")}>
              {CORE_STATS.map((s) => {
                const on = pick.includes(s);
                const other = pick.length === 2 && on ? pick.find((x) => x !== s) : undefined;
                return (
                  <li key={s}>
                    <button
                      type="button"
                      role="checkbox"
                      aria-checked={on}
                      onClick={() => toggle(s)}
                      className={cn(
                        "flex min-h-12 w-full items-center gap-2.5 rounded-xl border-2 border-ink px-2.5 text-left",
                        on ? "bg-st-event" : "bg-white",
                      )}
                    >
                      <StatBadge stat={s} className="size-8" />
                      <span className="flex-1 font-semibold">{statLabel(s)}</span>
                      <span className="num text-2xl">{rower.stats[s] ?? 0}</span>
                      {other && <span className="num text-2xl text-[#c2507a]">→ {rower.stats[other] ?? 0}</span>}
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button disabled={pick.length !== 2 || submitting} onClick={() => submit({ swap: [pick[0], pick[1]] })}>
              <ArrowLeftRight /> {t("swap.swap")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PhaseFrame>
  );
}
