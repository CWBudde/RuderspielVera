import { ArrowRight } from "lucide-react";
import { useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import { ResultView } from "@/components/ResultView";
import { Button } from "@/components/ui/button";
import { t, tm } from "@/i18n";

export function ResultAck() {
  const { view, you, submit, submitting } = useGame();
  if (!view.lastResult) return null;
  const last = view.current >= view.calendar.length - 1;
  return (
    <PhaseFrame
      title={tm(view.phase.title)}
      prompt={tm(view.phase.prompt)}
      footer={
        <Button size="lg" className="w-full" onClick={() => submit({})} disabled={submitting}>
          {last ? t("result.toFinal") : t("result.toDraft")} <ArrowRight />
        </Button>
      }
    >
      <ResultView result={view.lastResult} players={view.players} you={you.id} />
    </PhaseFrame>
  );
}
