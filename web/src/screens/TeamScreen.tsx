import { useState } from "react";
import { ArrowDownWideNarrow, LayoutGrid, List } from "lucide-react";
import type { RowerView } from "@/api/types.gen";
import { useGame } from "@/hooks/game";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { RowerCard, RowerRow, statSum } from "@/components/RowerCard";
import { BoatCard, TrainerCard } from "@/components/GearCards";
import { AGE_ORDER, playerColor } from "@/lib/labels";
import { readStorage, writeStorage } from "@/lib/storage";
import { cn } from "@/lib/utils";

export function sortRowers(rowers: RowerView[]): RowerView[] {
  return [...rowers].sort(
    (a, b) =>
      (AGE_ORDER[a.age] ?? 9) - (AGE_ORDER[b.age] ?? 9) ||
      a.gender.localeCompare(b.gender) ||
      a.name.localeCompare(b.name),
  );
}

/** Strongest first by Summe (all five stats), then the usual age/gender/name order. */
function sortBySum(rowers: RowerView[]): RowerView[] {
  const base = sortRowers(rowers);
  return base.sort((a, b) => statSum(b) - statSum(a));
}

const MODE_KEY = "ruderspiel.teamMode";
const SORT_KEY = "ruderspiel.teamSort";

export const tabListClass =
  "h-auto w-full rounded-xl border-2 border-ink bg-cream-2 p-1 group-data-[orientation=horizontal]/tabs:h-auto";
export const tabTriggerClass =
  "min-h-10 rounded-lg font-display text-base font-bold data-[state=active]:bg-navy data-[state=active]:text-white data-[state=active]:shadow-none";

export function TeamScreen() {
  const { view } = useGame();
  const [playerId, setPlayerId] = useState(view.you);
  const [mode, setMode] = useState<"cards" | "list">(() => (readStorage(MODE_KEY) === "list" ? "list" : "cards"));
  const [sort, setSort] = useState<"age" | "sum">(() => (readStorage(SORT_KEY) === "sum" ? "sum" : "age"));
  const p = view.players.find((x) => x.id === playerId) ?? view.players[0];
  const rowers = sort === "sum" ? sortBySum(p.rowers ?? []) : sortRowers(p.rowers ?? []);

  const switchSort = (s: "age" | "sum") => {
    setSort(s);
    writeStorage(SORT_KEY, s);
  };

  const switchMode = (m: "cards" | "list") => {
    setMode(m);
    writeStorage(MODE_KEY, m);
  };

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center gap-2" role="radiogroup" aria-label="Team auswählen">
        {view.players.map((pl) => (
          <button
            key={pl.id}
            type="button"
            role="radio"
            aria-checked={pl.id === playerId}
            onClick={() => setPlayerId(pl.id)}
            className={cn(
              "inline-flex min-h-11 items-center gap-2 rounded-full border-2 border-ink px-3 font-display text-base font-bold",
              pl.id === playerId ? "bg-cream shadow-[0_2px_0_var(--ink)]" : "bg-white/50 text-ink/70",
            )}
          >
            <span
              className="size-3 rounded-full border border-ink"
              style={{ background: playerColor(view.players, pl.id) }}
            />
            {pl.id === view.you ? "Dein Team" : pl.name}
          </button>
        ))}
      </div>

      <Tabs defaultValue="rowers">
        <TabsList className={tabListClass}>
          <TabsTrigger value="rowers" className={tabTriggerClass}>
            Ruderer <span className="text-xs opacity-75">{rowers.length}</span>
          </TabsTrigger>
          <TabsTrigger value="boats" className={tabTriggerClass}>
            Boote <span className="text-xs opacity-75">{p.boats?.length ?? 0}</span>
          </TabsTrigger>
          <TabsTrigger value="trainers" className={tabTriggerClass}>
            Trainer <span className="text-xs opacity-75">{p.trainers?.length ?? 0}</span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value="rowers" className="mt-2">
          <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
            <div
              className="inline-flex items-center rounded-xl border-2 border-ink bg-cream p-0.5"
              role="radiogroup"
              aria-label="Sortierung"
            >
              <ArrowDownWideNarrow className="mx-1.5 size-4 text-ink/60" aria-hidden />
              {(
                [
                  ["age", "Alter"],
                  ["sum", "Summe"],
                ] as const
              ).map(([s, label]) => (
                <button
                  key={s}
                  type="button"
                  role="radio"
                  aria-checked={sort === s}
                  onClick={() => switchSort(s)}
                  className={cn(
                    "inline-flex min-h-10 items-center rounded-lg px-3 text-sm font-bold",
                    sort === s ? "bg-navy text-white" : "text-ink/70",
                  )}
                >
                  {label}
                </button>
              ))}
            </div>
            <div
              className="inline-flex rounded-xl border-2 border-ink bg-cream p-0.5"
              role="radiogroup"
              aria-label="Darstellung"
            >
              {(
                [
                  ["cards", LayoutGrid, "Karten"],
                  ["list", List, "Liste"],
                ] as const
              ).map(([m, Icon, label]) => (
                <button
                  key={m}
                  type="button"
                  role="radio"
                  aria-checked={mode === m}
                  onClick={() => switchMode(m)}
                  className={cn(
                    "inline-flex min-h-10 items-center gap-1.5 rounded-lg px-3 text-sm font-bold",
                    mode === m ? "bg-navy text-white" : "text-ink/70",
                  )}
                >
                  <Icon className="size-4" /> {label}
                </button>
              ))}
            </div>
          </div>
          {mode === "cards" ? (
            <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-3 sm:gap-3 xl:grid-cols-4">
              {rowers.map((r) => (
                <RowerCard key={r.id} rower={r} />
              ))}
            </div>
          ) : (
            <div className="grid gap-2 lg:grid-cols-2">
              {rowers.map((r) => (
                <RowerRow key={r.id} rower={r} />
              ))}
            </div>
          )}
          <EventList rowers={rowers} />
        </TabsContent>

        <TabsContent value="boats" className="mt-2">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {(p.boats ?? []).map((b) => (
              <BoatCard key={b.id} boat={b} />
            ))}
          </div>
        </TabsContent>

        <TabsContent value="trainers" className="mt-2">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {(p.trainers ?? []).map((t) => (
              <TrainerCard key={t.id} trainer={t} />
            ))}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}

/** Event cards of this team, with mission status. */
function EventList({ rowers }: { rowers: RowerView[] }) {
  const withEvents = rowers.filter((r) => r.event);
  if (withEvents.length === 0) return null;
  return (
    <section className="mt-5">
      <h3 className="num text-2xl">Eventkarten</h3>
      <ul className="mt-2 grid gap-2 sm:grid-cols-2">
        {withEvents.map((r) => (
          <li key={r.id} className="rounded-xl border-2 border-ink bg-[#fde4ec] p-3">
            <p className="font-display text-lg leading-tight font-bold">
              {r.event?.name} <span className="text-sm font-semibold text-muted-foreground">für {r.name}</span>
            </p>
            <p className="text-sm">{r.event?.description}</p>
            {r.event?.mission && (
              <p className="mt-1 text-sm">
                <span className="font-bold">Mission:</span> {r.event.mission.text} ({r.event.mission.reward})
                {r.event.mission.done && <span className="ml-1 font-bold text-leaf">Erfüllt</span>}
                {r.event.mission.failed && <span className="ml-1 font-bold text-destructive">Gescheitert</span>}
              </p>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}
