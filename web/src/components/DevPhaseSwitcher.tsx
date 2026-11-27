import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Bug } from "lucide-react";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { mockJump, type JumpTarget } from "@/api/mock";
import { useGame } from "@/hooks/game";
import { t, type UiKey } from "@/i18n";

const TARGETS: { id: JumpTarget; label: UiKey }[] = [
  { id: "setup", label: "dev.target.setup" },
  { id: "training", label: "dev.target.training" },
  { id: "camp", label: "dev.target.camp" },
  { id: "selection", label: "dev.target.selection" },
  { id: "event", label: "dev.target.event" },
  { id: "eventswap", label: "dev.target.eventswap" },
  { id: "ereignis", label: "dev.target.ereignis" },
  { id: "regatta", label: "dev.target.regatta" },
  { id: "ergocup", label: "dev.target.ergocup" },
  { id: "jtfo", label: "dev.target.jtfo" },
  { id: "rvb", label: "dev.target.rvb" },
  { id: "result", label: "dev.target.result" },
  { id: "draft", label: "dev.target.draft" },
  { id: "draft-waiting", label: "dev.target.draftWaiting" },
  { id: "gameover", label: "dev.target.gameover" },
];

/** Mock-mode only: jump to any phase kind to review every screen. */
export function DevPhaseSwitcher() {
  const { session, setView } = useGame();
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="grid size-11 shrink-0 place-items-center rounded-full text-[#ffe7a3] hover:bg-white/10"
        aria-label={t("dev.switchAria")}
        data-testid="dev-switcher"
      >
        <Bug className="size-5" />
      </button>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="left" className="w-[min(22rem,88vw)] gap-0 overflow-y-auto bg-cream">
          <SheetHeader>
            <SheetTitle className="font-display text-2xl">{t("dev.switch")}</SheetTitle>
            <SheetDescription>{t("dev.hint")}</SheetDescription>
          </SheetHeader>
          <ul className="flex flex-col gap-1.5 px-4 pb-6">
            {TARGETS.map((tg) => (
              <li key={tg.id}>
                <button
                  type="button"
                  className="min-h-11 w-full rounded-xl border-2 border-ink bg-white px-3 text-left text-sm font-semibold hover:bg-accent"
                  onClick={() => {
                    setView(mockJump(session.gameId, tg.id));
                    void qc.invalidateQueries({ queryKey: ["log", session.gameId] });
                    setOpen(false);
                  }}
                >
                  {t(tg.label)}
                </button>
              </li>
            ))}
          </ul>
        </SheetContent>
      </Sheet>
    </>
  );
}
