import { useState } from "react";
import { Check, Hand, Loader2 } from "lucide-react";
import type { DraftCard } from "@/api/types.gen";
import { playerName, useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import { RowerCard } from "@/components/RowerCard";
import { BoatCard, TrainerCard } from "@/components/GearCards";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { t, tData, tm } from "@/i18n";
import { tr } from "@/i18n/rich";
import { boatName, rowerName, trainerName } from "@/i18n/data";

function cardName(c: DraftCard): string {
  if (c.rower) return rowerName(c.rower);
  if (c.boat) return boatName(c.boat);
  if (c.trainer) return trainerName(c.trainer);
  return t("draft.card");
}

export function DraftPhase() {
  const { view, submit, submitting } = useGame();
  const { phase } = view;
  const order = phase.draftOrder ?? [];
  const offer = phase.draftOffer ?? [];
  const [pick, setPick] = useState<string | undefined>();
  const turn = phase.draftTurn ?? 0;
  const onTurn = order[turn];
  const waiting = phase.waiting;
  const picked = offer.find((c) => c.id === pick);

  return (
    <PhaseFrame
      title={tm(phase.title)}
      prompt={tm(phase.prompt)}
      headerExtra={
        <ol className="mt-3 flex flex-wrap items-center gap-1.5" aria-label={t("draft.order")}>
          {order.map((id, i) => {
            const p = view.players.find((x) => x.id === id);
            const done = i < turn;
            const now = i === turn;
            return (
              <li
                key={`${id}-${i}`}
                className={cn(
                  "inline-flex h-9 items-center gap-1.5 rounded-full border-2 border-ink px-2.5 text-sm font-bold",
                  now ? "bg-navy text-white" : done ? "bg-white/50 text-ink/55 line-through" : "bg-white",
                )}
                aria-current={now ? "true" : undefined}
              >
                <span className="num text-base leading-none">{i + 1}.</span>
                {id === view.you ? t("draft.you") : (p?.name ?? id)}
                {done && <Check className="size-4" />}
              </li>
            );
          })}
        </ol>
      }
      footer={
        waiting ? (
          <p className="flex min-h-12 items-center justify-center gap-2 font-bold" role="status">
            <Loader2 className="size-5 animate-spin text-lake" />
            {onTurn ? t("draft.waitingFor", { name: playerName(view, onTurn) }) : t("draft.waitingOthers")}
          </p>
        ) : (
          <div className="flex items-center gap-2">
            <p className="min-w-0 flex-1 truncate pl-1 text-sm font-semibold">
              {picked ? (
                <>{tr("draft.picked", { name: <span className="font-bold">{cardName(picked)}</span> })}</>
              ) : (
                t("draft.tapCard")
              )}
            </p>
            <Button size="lg" disabled={!pick || submitting} onClick={() => submit({ draftCardId: pick })}>
              <Hand /> {t("draft.take")}
            </Button>
          </div>
        )
      }
    >
      {offer.length === 0 ? (
        <p className="rounded-xl border-2 border-ink bg-cream p-3 text-sm">{t("draft.empty")}</p>
      ) : (
        <ul
          className="grid grid-cols-1 gap-4 min-[380px]:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
          role="radiogroup"
          aria-label={t("draft.offer")}
        >
          {offer.map((c) => {
            const on = c.id === pick;
            return (
              <li key={c.id}>
                <button
                  type="button"
                  role="radio"
                  aria-checked={on}
                  disabled={waiting}
                  onClick={() => setPick(on ? undefined : c.id)}
                  className={cn(
                    "relative block w-full text-left transition-transform",
                    on && "-translate-y-1",
                    waiting && "opacity-70",
                  )}
                >
                  <span
                    className={cn(
                      "mb-3 inline-block rounded-full border-2 border-ink px-2 text-xs font-bold",
                      c.personal ? "bg-navy text-white" : "bg-white",
                    )}
                  >
                    {c.personal ? t("draft.personal") : tData(`draft.kind.${c.kind}`, c.kind)}
                  </span>
                  <div
                    className={cn(on && "[&>article]:outline-4 [&>article]:outline-offset-4 [&>article]:outline-lake")}
                  >
                    {c.rower && <RowerCard rower={c.rower} />}
                    {c.boat && <BoatCard boat={c.boat} />}
                    {c.trainer && <TrainerCard trainer={c.trainer} />}
                  </div>
                  {on && (
                    <span className="absolute top-0 right-0 grid size-7 place-items-center rounded-full border-2 border-ink bg-lake text-white">
                      <Check className="size-5" strokeWidth={3} />
                    </span>
                  )}
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </PhaseFrame>
  );
}
