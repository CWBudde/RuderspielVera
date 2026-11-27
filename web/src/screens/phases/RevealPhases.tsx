import { useMemo, useState } from "react";
import { ArrowRight, Dice5, Sparkles, Target } from "lucide-react";
import type { RowerView } from "@/api/types.gen";
import { useGame } from "@/hooks/game";
import { PhaseFrame } from "@/components/PhaseFrame";
import { FlipCard } from "@/components/FlipCard";
import { Art } from "@/components/Art";
import { AgeBadge } from "@/components/RowerCard";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { t, tm } from "@/i18n";
import { ereignisName, ereignisText } from "@/i18n/data";
import { eventDescription, eventName, missionReward, missionText, rowerName } from "@/i18n/data";

function EventFace({ rower }: { rower: RowerView }) {
  const ev = rower.event;
  return (
    <div className="@container flex size-full flex-col overflow-hidden rounded-[18px] border-2 border-ink bg-[#fde4ec] shadow-[0_3px_0_var(--ink)]">
      <div className="flex items-center justify-between gap-2 bg-[#e889a8] px-3 py-2 text-[#3d0f20]">
        <span className="inline-flex items-center gap-1 font-display text-sm font-bold">
          <Sparkles className="size-4" /> {t("reveal.event")}
        </span>
        <AgeBadge age={rower.age} short />
      </div>
      <div className="flex flex-1 flex-col p-2.5 @[13rem]:p-3">
        <h3 className="num text-xl leading-none @[13rem]:text-[1.9rem]">{ev ? eventName(ev) : ""}</h3>
        <p className="mt-1.5 text-xs leading-snug @[13rem]:mt-2 @[13rem]:text-[15px]">
          {ev ? eventDescription(ev) : ""}
        </p>
        {ev?.mission && (
          <p className="mt-2 rounded-lg border-2 border-ink bg-white p-1.5 text-xs @[13rem]:p-2 @[13rem]:text-sm">
            <Target className="mr-1 inline size-4" />
            <span className="font-bold">{missionText(ev.mission)}</span>
            <br />
            {t("reveal.reward", { reward: missionReward(ev.mission) })}
          </p>
        )}
        <div className="mt-auto flex items-center gap-2 pt-3">
          <div className="size-10 shrink-0 overflow-hidden rounded-lg border-2 border-ink @[13rem]:size-12">
            <Art artKey={rower.artKey} cardId={rower.id} label={rower.name} thumb />
          </div>
          <div>
            <p className="text-xs font-semibold text-muted-foreground">{t("reveal.for")}</p>
            <p className="num text-lg leading-none text-navy @[13rem]:text-xl">{rowerName(rower)}</p>
          </div>
        </div>
      </div>
    </div>
  );
}

export function EventReveal() {
  const { view, you, submit, submitting } = useGame();
  const here = view.log.filter((e) => e.stationIndex === view.current && e.kind === "event");
  // phase.newEvents lists exactly the cards dealt to you at this station.
  const mine = useMemo(() => {
    const ids = new Set((view.phase.newEvents ?? []).map((ev) => ev.rowerId));
    return you.rowers.filter((r) => ids.has(r.id) && r.event);
  }, [you, view.phase.newEvents]);
  const [revealed, setRevealed] = useState<Set<string>>(new Set());
  const allRevealed = mine.every((r) => revealed.has(r.id));
  const others = here.filter((e) => e.playerId !== you.id);

  return (
    <PhaseFrame
      title={tm(view.phase.title)}
      prompt={mine.length && !allRevealed ? t("reveal.newEvents") : tm(view.phase.prompt)}
      footer={
        allRevealed ? (
          <Button size="lg" className="w-full" onClick={() => submit({})} disabled={submitting}>
            {t("common.continue")} <ArrowRight />
          </Button>
        ) : (
          <Button
            size="lg"
            variant="lake"
            className="w-full"
            onClick={() => setRevealed(new Set(mine.map((r) => r.id)))}
          >
            {t("reveal.revealAll")}
          </Button>
        )
      }
    >
      {mine.length === 0 ? (
        <p className="rounded-xl border-2 border-ink bg-cream p-3 text-sm">{t("reveal.noneForYou")}</p>
      ) : (
        <div
          className={
            mine.length > 1
              ? "grid grid-cols-2 gap-3 sm:flex sm:flex-wrap sm:justify-center sm:gap-4"
              : "flex justify-center"
          }
        >
          {mine.map((r) => (
            <FlipCard
              key={r.id}
              className={cn("aspect-[3/4] sm:w-[17rem]", mine.length > 1 ? "w-full" : "w-[min(100%,17rem)]")}
              revealed={revealed.has(r.id)}
              onReveal={() => setRevealed((s) => new Set(s).add(r.id))}
              backTone="bg-[#c2507a]"
              backLabel={t("reveal.eventCard")}
              front={<EventFace rower={r} />}
            />
          ))}
        </div>
      )}
      {others.length > 0 && (
        <section className="mt-5">
          <h3 className="font-display text-lg font-bold">{t("reveal.others")}</h3>
          <ul className="mt-1 flex flex-col gap-1.5">
            {others.map((e, i) => (
              <li key={i} className="rounded-xl border-2 border-ink/15 bg-cream/80 p-2.5 text-sm">
                {tm(e.msg)}
              </li>
            ))}
          </ul>
        </section>
      )}
    </PhaseFrame>
  );
}

export function EreignisReveal() {
  const { view, submit, submitting } = useGame();
  const name = ereignisName(view.phase.ereignis);
  const text = view.phase.ereignis ? ereignisText(view.phase.ereignis) : tm(view.phase.prompt);
  const [revealed, setRevealed] = useState(false);

  return (
    <PhaseFrame
      title={tm(view.phase.title)}
      prompt={revealed ? t("reveal.appliesToAll") : t("reveal.newEreignis")}
      footer={
        revealed ? (
          <Button size="lg" className="w-full" onClick={() => submit({})} disabled={submitting}>
            {t("common.continue")} <ArrowRight />
          </Button>
        ) : (
          <Button size="lg" variant="lake" className="w-full" onClick={() => setRevealed(true)}>
            {t("reveal.reveal")}
          </Button>
        )
      }
    >
      <div className="flex justify-center">
        <FlipCard
          className="aspect-[3/4] w-[min(100%,18rem)]"
          revealed={revealed}
          onReveal={() => setRevealed(true)}
          backTone="bg-leaf"
          backLabel={t("reveal.ereignisCard")}
          front={
            <div className="flex size-full flex-col overflow-hidden rounded-[18px] border-2 border-ink bg-[#e8f5dc] shadow-[0_3px_0_var(--ink)]">
              <div className="flex items-center gap-1.5 bg-st-ereignis px-3 py-2 font-display text-sm font-bold">
                <Dice5 className="size-4" /> {t("reveal.ereignisForAll")}
              </div>
              <div className="flex flex-1 flex-col items-center justify-center p-4 text-center">
                <Dice5 className="size-14 text-leaf" strokeWidth={1.6} />
                <h3 className="mt-2 num text-[2rem] leading-none">{name}</h3>
                <p className="mt-3 text-[15px] leading-snug">{text}</p>
              </div>
            </div>
          }
        />
      </div>
    </PhaseFrame>
  );
}
