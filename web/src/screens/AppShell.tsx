import { useEffect, useRef, useState } from "react";
import { Flag, LogOut, Map as MapIcon, Medal, Menu, Users } from "lucide-react";
import { useGame } from "@/hooks/game";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Emblem, MedalIcon, StationIcon } from "@/components/icons";
import { DevPhaseSwitcher } from "@/components/DevPhaseSwitcher";
import { stationColor } from "@/lib/labels";
import { cn } from "@/lib/utils";
import { SeasonScreen } from "@/screens/SeasonScreen";
import { TeamScreen } from "@/screens/TeamScreen";
import { ActionScreen } from "@/screens/ActionScreen";
import { StandingsScreen } from "@/screens/StandingsScreen";

export type Tab = "saison" | "team" | "aktion" | "wertung";

const TABS: { id: Tab; label: string; icon: typeof MapIcon }[] = [
  { id: "saison", label: "Saison", icon: MapIcon },
  { id: "team", label: "Team", icon: Users },
  { id: "aktion", label: "Aktion", icon: Flag },
  { id: "wertung", label: "Wertung", icon: Medal },
];

export function AppShell() {
  const { view, you, session, leave } = useGame();
  const pending = view.phase.kind !== "gameover" && !view.phase.waiting;
  const [tab, setTab] = useState<Tab>(pending || view.phase.kind === "gameover" ? "aktion" : "saison");
  const [menu, setMenu] = useState(false);
  const mainRef = useRef<HTMLElement>(null);
  const station = view.calendar[view.current];
  const col = stationColor(station?.type ?? "setup");

  // A newly opened phase that needs a decision brings the Aktion view forward.
  useEffect(() => {
    if (pending) setTab("aktion");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view.phase.id]);

  // Scroll to top when switching tabs or when a new phase opens.
  useEffect(() => {
    window.scrollTo({ top: 0 });
  }, [tab, view.phase.id]);

  return (
    <div className="min-h-dvh md:grid md:grid-cols-[15rem_minmax(0,1fr)] xl:grid-cols-[16rem_minmax(0,1fr)]">
      {/* desktop sidebar */}
      <aside className="sticky top-0 hidden h-dvh flex-col gap-2 border-r-2 border-ink bg-navy p-4 text-white md:flex">
        <div className="mb-4 flex items-center gap-2">
          <Emblem className="size-10" />
          <span className="num text-3xl leading-none uppercase">Ruderspiel</span>
        </div>
        <nav className="flex flex-col gap-1.5" aria-label="Bereiche">
          {TABS.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => setTab(t.id)}
              aria-current={tab === t.id ? "page" : undefined}
              className={cn(
                "relative flex min-h-12 items-center gap-3 rounded-xl px-3 text-left font-display text-xl font-bold transition-colors",
                tab === t.id ? "bg-cream text-ink cut-sm" : "text-white/85 hover:bg-white/10",
              )}
            >
              <t.icon className="size-5" />
              {t.label}
              {t.id === "aktion" && pending && <PendingDot className="ml-auto" />}
            </button>
          ))}
        </nav>
        <div className="mt-auto rounded-xl bg-white/10 p-3">
          <p className="truncate text-sm font-semibold text-white/80">{you.name}</p>
          <MedalTally gold={you.gold} silver={you.silver} points={you.points} light />
        </div>
        <Button variant="ghost" className="justify-start text-white/80 hover:bg-white/10 hover:text-white" onClick={() => setMenu(true)}>
          <Menu /> Menü
        </Button>
      </aside>

      <div className="flex min-h-dvh min-w-0 flex-col">
        <header className="sticky top-0 z-30 border-b-2 border-ink bg-navy text-white">
          <div className="mx-auto flex max-w-6xl items-center gap-3 px-3 pt-[calc(var(--safe-top)+0.5rem)] pb-2 md:px-6">
            <span
              className="grid size-11 shrink-0 place-items-center rounded-full border-2 border-ink"
              style={{ background: col.bg, color: col.text }}
            >
              <StationIcon type={station?.type ?? "setup"} name={station?.name} className="size-5" />
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-xs font-semibold text-white/75">
                <span className="min-[400px]:hidden">
                  Station {view.current + 1}/{view.calendar.length}
                </span>
                <span className="hidden min-[400px]:inline">
                  Station {view.current + 1} von {view.calendar.length}
                </span>
              </p>
              <h1 className="truncate num text-2xl leading-tight">{station?.name}</h1>
            </div>
            <MedalTally gold={you.gold} silver={you.silver} points={you.points} light compact />
            {session.mock && <DevPhaseSwitcher />}
            <Button
              variant="ghost"
              size="icon"
              className="text-white hover:bg-white/10 hover:text-white md:hidden"
              aria-label="Menü"
              onClick={() => setMenu(true)}
            >
              <Menu />
            </Button>
          </div>
        </header>

        <main
          ref={mainRef}
          className="mx-auto w-full max-w-6xl flex-1 px-3 pt-4 pb-[calc(var(--safe-bottom)+6rem)] md:px-6 md:pb-10"
        >
          {tab === "saison" && <SeasonScreen />}
          {tab === "team" && <TeamScreen />}
          {tab === "aktion" && <ActionScreen onShowTab={setTab} />}
          {tab === "wertung" && <StandingsScreen />}
        </main>
      </div>

      {/* phone tab bar */}
      <nav
        className="fixed inset-x-0 bottom-0 z-40 border-t-2 border-ink bg-cream pb-[var(--safe-bottom)] md:hidden"
        aria-label="Bereiche"
      >
        <div className="mx-auto grid max-w-md grid-cols-4">
          {TABS.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => setTab(t.id)}
              aria-current={tab === t.id ? "page" : undefined}
              className={cn(
                "relative flex min-h-16 flex-col items-center justify-center gap-0.5 font-display text-[15px] font-bold",
                tab === t.id ? "text-navy" : "text-ink/55",
              )}
            >
              <span
                className={cn(
                  "grid h-8 w-14 place-items-center rounded-full transition-colors",
                  tab === t.id && "bg-st-regatta ring-2 ring-ink",
                )}
              >
                <t.icon className="size-5" strokeWidth={2.4} />
              </span>
              {t.label}
              {t.id === "aktion" && pending && <PendingDot className="absolute top-2 right-[calc(50%-1.9rem)]" />}
            </button>
          ))}
        </div>
      </nav>

      <Dialog open={menu} onOpenChange={setMenu}>
        <DialogContent className="bg-cream">
          <DialogHeader>
            <DialogTitle className="font-display text-2xl">Menü</DialogTitle>
            <DialogDescription>
              Dein Spielstand bleibt gespeichert. Du kannst ihn auf dem Startbildschirm fortsetzen.
            </DialogDescription>
          </DialogHeader>
          <Button variant="outline" onClick={leave}>
            <LogOut /> Zum Startbildschirm
          </Button>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function PendingDot({ className }: { className?: string }) {
  return (
    <span className={cn("relative flex size-3", className)} aria-label="Entscheidung offen">
      <span className="absolute inline-flex size-full rounded-full bg-[#e0475f] opacity-70 motion-safe:animate-ping" />
      <span className="relative inline-flex size-3 rounded-full border-2 border-ink bg-[#e0475f]" />
    </span>
  );
}

export function MedalTally({
  gold,
  silver,
  points,
  light,
  compact,
}: {
  gold: number;
  silver: number;
  points: number;
  light?: boolean;
  compact?: boolean;
}) {
  return (
    <div className={cn("flex items-center gap-2", light ? "text-white" : "text-ink")} aria-label={`${gold} Gold, ${silver} Silber, ${points} Punkte`}>
      <span className="flex items-center gap-0.5">
        <MedalIcon medal="gold" className="size-6" />
        <span className="num text-xl leading-none">{gold}</span>
      </span>
      <span className="flex items-center gap-0.5">
        <MedalIcon medal="silber" className="size-6" />
        <span className="num text-xl leading-none">{silver}</span>
      </span>
      <span className={cn("rounded-lg px-1.5 py-0.5 text-center", light ? "bg-white/15" : "bg-ink/10", compact && "hidden min-[400px]:block")}>
        <span className="num text-xl leading-none">{points}</span>
        <span className="ml-0.5 text-[11px] font-bold">Pkt</span>
      </span>
    </div>
  );
}
