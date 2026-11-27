import { ArrowRight } from "lucide-react";
import { useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import { ResultView } from "@/components/ResultView";
import { Button } from "@/components/ui/button";

export function ResultAck() {
  const { view, you, submit, submitting } = useGame();
  if (!view.lastResult) return null;
  const last = view.current >= view.calendar.length - 1;
  return (
    <PhaseFrame
      title={view.phase.title}
      prompt={view.phase.prompt}
      footer={
        <Button size="lg" className="w-full" onClick={() => submit({})} disabled={submitting}>
          {last ? "Zur Abschlusswertung" : "Weiter zur Kartenwahl"} <ArrowRight />
        </Button>
      }
    >
      <ResultView result={view.lastResult} players={view.players} you={you.id} />
    </PhaseFrame>
  );
}
