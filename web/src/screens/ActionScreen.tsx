import { Check, Loader2 } from "lucide-react";
import { useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import type { Tab } from "@/screens/AppShell";
import { AckPhase } from "@/screens/phases/AckPhase";
import { EventReveal, EreignisReveal } from "@/screens/phases/RevealPhases";
import { ResultAck } from "@/screens/phases/ResultAck";
import { TrainingPhase } from "@/screens/phases/TrainingPhase";
import { SelectionPhase } from "@/screens/phases/SelectionPhase";
import { SwapPhase } from "@/screens/phases/SwapPhase";
import { RegattaPhase } from "@/screens/phases/RegattaPhase";
import { DraftPhase } from "@/screens/phases/DraftPhase";
import { GameOver } from "@/screens/phases/GameOver";

export function ActionScreen({ onShowTab }: { onShowTab: (t: Tab) => void }) {
  const { view } = useGame();
  const { phase } = view;
  if (phase.waiting && phase.kind !== "draft") return <Waiting />;

  switch (phase.kind) {
    case "ack":
      if (phase.stage === "result" && view.lastResult) return <ResultAck key={phase.id} />;
      if (phase.stage === "event") return <EventReveal key={phase.id} />;
      if (phase.stage === "ereignis") return <EreignisReveal key={phase.id} />;
      return <AckPhase key={phase.id} onShowTab={onShowTab} />;
    case "training":
      return <TrainingPhase key={phase.id} />;
    case "selection":
      return <SelectionPhase key={phase.id} />;
    case "eventswap":
      return <SwapPhase key={phase.id} />;
    case "regatta":
      return <RegattaPhase key={phase.id} />;
    case "draft":
      return <DraftPhase key={phase.id} />;
    case "gameover":
      return <GameOver />;
    default:
      return <AckPhase key={phase.id} onShowTab={onShowTab} />;
  }
}

function Waiting() {
  const { view } = useGame();
  return (
    <PhaseFrame title={view.phase.title} prompt="Deine Entscheidung ist abgegeben. Die anderen sind noch dran.">
      <ul className="flex flex-col gap-2">
        {view.players.map((p) => (
          <li key={p.id} className="flex min-h-12 items-center gap-3 rounded-xl border-2 border-ink bg-cream px-3">
            {p.decided ? <Check className="size-5 text-leaf" /> : <Loader2 className="size-5 animate-spin text-lake" />}
            <span className="font-bold">{p.name}</span>
            <span className="ml-auto text-sm text-muted-foreground">{p.decided ? "fertig" : "überlegt noch …"}</span>
          </li>
        ))}
      </ul>
    </PhaseFrame>
  );
}
