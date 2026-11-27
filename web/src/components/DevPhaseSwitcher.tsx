import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Bug } from "lucide-react";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { mockJump, type JumpTarget } from "@/api/mock";
import { useGame } from "@/hooks/game";

const TARGETS: { id: JumpTarget; label: string }[] = [
  { id: "setup", label: "Spielvorbereitung (ack)" },
  { id: "training", label: "Training" },
  { id: "camp", label: "Trainingslager" },
  { id: "selection", label: "Auswahl (WR5 / Basketball / Wanderfahrt)" },
  { id: "event", label: "Event-Aufdeckung" },
  { id: "eventswap", label: "Tausch-Event" },
  { id: "ereignis", label: "Ereignis" },
  { id: "regatta", label: "Regatta (Wasser, Wetter)" },
  { id: "ergocup", label: "Ergo-Cup" },
  { id: "jtfo", label: "JTFO" },
  { id: "rvb", label: "RVB-Endspurt (Mixed, Achter)" },
  { id: "result", label: "Ergebnis (lastResult)" },
  { id: "draft", label: "Draft – du bist dran" },
  { id: "draft-waiting", label: "Draft – wartet auf Bot" },
  { id: "gameover", label: "Saisonende" },
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
        aria-label="Phase wechseln (Testmodus)"
        data-testid="dev-switcher"
      >
        <Bug className="size-5" />
      </button>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="left" className="w-[min(22rem,88vw)] gap-0 overflow-y-auto bg-cream">
          <SheetHeader>
            <SheetTitle className="font-display text-2xl">Phase wechseln</SheetTitle>
            <SheetDescription>Nur im Testmodus. Springt zur nächsten passenden Station.</SheetDescription>
          </SheetHeader>
          <ul className="flex flex-col gap-1.5 px-4 pb-6">
            {TARGETS.map((t) => (
              <li key={t.id}>
                <button
                  type="button"
                  className="min-h-11 w-full rounded-xl border-2 border-ink bg-white px-3 text-left text-sm font-semibold hover:bg-accent"
                  onClick={() => {
                    setView(mockJump(session.gameId, t.id));
                    void qc.invalidateQueries({ queryKey: ["log", session.gameId] });
                    setOpen(false);
                  }}
                >
                  {t.label}
                </button>
              </li>
            ))}
          </ul>
        </SheetContent>
      </Sheet>
    </>
  );
}
