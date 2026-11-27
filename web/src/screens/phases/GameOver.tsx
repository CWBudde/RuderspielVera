import { RotateCcw } from "lucide-react";
import { useGame } from "@/hooks/game";
import { LakeScene } from "@/components/LakeScene";
import { MedalIcon } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { MedalTable } from "@/screens/StandingsScreen";

function Laurel({ flip }: { flip?: boolean }) {
  return (
    <svg viewBox="0 0 40 90" className={`h-20 w-9 shrink-0 ${flip ? "-scale-x-100" : ""}`} aria-hidden>
      <path d="M32 86C12 70 8 40 20 6" fill="none" stroke="var(--leaf)" strokeWidth="3" strokeLinecap="round" />
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <ellipse
          key={i}
          cx={22 - i * 1.6 + (i > 3 ? i - 3 : 0)}
          cy={78 - i * 13}
          rx="4.5"
          ry="9"
          fill="var(--sun)"
          stroke="var(--ink)"
          strokeWidth="1.4"
          transform={`rotate(${-40 + i * 6} ${22 - i * 1.6} ${78 - i * 13})`}
        />
      ))}
    </svg>
  );
}

export function GameOver() {
  const { view, you, leave } = useGame();
  const winners = view.players.filter((p) => view.winners?.includes(p.id));
  const youWon = winners.some((w) => w.id === you.id);
  const shared = winners.length > 1;
  return (
    <div className="flex flex-col gap-5">
      <LakeScene className="rounded-[22px] border-2 border-ink shadow-[0_3px_0_var(--ink)]">
        <div className="relative flex min-h-[19rem] flex-col items-center justify-center px-4 py-6 text-center">
          <p className="rounded-full border-2 border-ink bg-cream px-3 py-0.5 font-display text-base font-bold">
            Saisonende am Maschsee
          </p>
          <div className="mt-3 flex items-center gap-1 motion-safe:animate-rise">
            <Laurel />
            <div className="min-w-0">
              <MedalIcon medal="gold" className="mx-auto size-12" />
              <h2
                className="num text-[clamp(2.4rem,11vw,4.5rem)] leading-[0.9] text-white uppercase"
                style={{ WebkitTextStroke: "1.5px var(--ink)", textShadow: "0 3px 0 var(--ink)" }}
              >
                {winners.map((w) => w.name).join(" & ") || "—"}
              </h2>
            </div>
            <Laurel flip />
          </div>
          <p className="mt-3 max-w-sm rounded-xl bg-cream/90 px-3 py-1.5 text-[15px] font-semibold">
            {youWon
              ? shared
                ? "Geteilter Sieg! Ihr habt gleich viele Medaillenpunkte."
                : "Glückwunsch, dein Team gewinnt die Saison!"
              : `${shared ? "Geteilter Sieg" : "Saisonsieg"} mit ${winners[0]?.points ?? 0} Punkten. Du hast ${you.points}.`}
          </p>
        </div>
      </LakeScene>
      <section>
        <h3 className="mb-2 num text-3xl leading-none">Abschlusswertung</h3>
        <MedalTable view={view} />
      </section>
      <Button size="lg" variant="lake" onClick={leave} className="self-center">
        <RotateCcw /> Neue Saison
      </Button>
    </div>
  );
}
