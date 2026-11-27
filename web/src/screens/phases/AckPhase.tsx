import { ArrowRight, Users } from "lucide-react";
import { useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import { Button } from "@/components/ui/button";
import { AGE_LABEL } from "@/lib/labels";
import type { Tab } from "@/screens/AppShell";

/** Generic acknowledge phase; for the setup station it summarises the dealt cards. */
export function AckPhase({ onShowTab }: { onShowTab: (t: Tab) => void }) {
  const { view, you, submit, submitting } = useGame();
  const station = view.calendar[view.current];
  const isSetup = station?.type === "setup";
  const byAge = (["a", "b", "kinder"] as const).map((a) => ({
    age: a,
    w: you.rowers.filter((r) => r.age === a && r.gender === "w").length,
    m: you.rowers.filter((r) => r.age === a && r.gender === "m").length,
  }));

  return (
    <PhaseFrame
      title={view.phase.title}
      prompt={view.phase.prompt}
      footer={
        <Button size="lg" className="w-full" onClick={() => submit({})} disabled={submitting}>
          {isSetup ? "Saison starten" : "Weiter"} <ArrowRight />
        </Button>
      }
    >
      {isSetup && (
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="rounded-[16px] bg-cream p-3 cut sm:col-span-2">
            <h3 className="num text-2xl">Deine Ruderer</h3>
            <ul className="mt-2 grid grid-cols-3 gap-2">
              {byAge.map((g) => (
                <li key={g.age} className="rounded-xl border-2 border-ink bg-white p-2 text-center">
                  <p className="font-display text-sm font-bold">{AGE_LABEL[g.age]}</p>
                  <p className="num text-3xl leading-none">{g.w + g.m}</p>
                  <p className="text-xs text-muted-foreground">
                    {g.w} ♀ · {g.m} ♂
                  </p>
                </li>
              ))}
            </ul>
            <Button variant="outline" className="mt-3 w-full" onClick={() => onShowTab("team")}>
              <Users /> Team ansehen
            </Button>
          </div>
          <div className="rounded-[16px] bg-cream p-3 cut">
            <h3 className="num text-2xl">Material</h3>
            <p className="mt-1 text-sm">
              <span className="num text-2xl">{you.boats.length}</span> Boote
            </p>
            <p className="text-sm">
              <span className="num text-2xl">{you.trainers.length}</span> Trainer
            </p>
            <p className="mt-2 text-sm text-muted-foreground">
              Gegner: {view.players.filter((p) => p.id !== you.id).map((p) => p.name).join(" und ")}
            </p>
          </div>
        </div>
      )}
    </PhaseFrame>
  );
}
