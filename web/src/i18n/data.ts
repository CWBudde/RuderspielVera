/**
 * Display text for game objects. The engine sends German names only as fallbacks; the UI looks up
 * the translated data text by id (keys from internal/data, see data.gen.ts).
 */
import type {
  BoatView,
  EreignisView,
  EventCardView,
  EventTypeView,
  MissionView,
  PhaseView,
  RowerView,
  StationView,
  TrainerView,
} from "@/api/types.gen";
import { t, tData, tk, tm } from "./index";

export function stationName(st: Pick<StationView, "id" | "name"> | undefined): string {
  if (!st) return "";
  return tData(`data.station.${st.id}.name`, st.name);
}

export function stationInfo(st: Pick<StationView, "id" | "info">): string {
  return tData(`data.station.${st.id}.info`, st.info);
}

/** Race name (RaceSpec or RaceResult). */
export function raceName(r: { nameKey?: string; name: string }): string {
  return r.nameKey ? tData(r.nameKey, r.name) : r.name;
}

export function eventName(e: Pick<EventCardView | EventTypeView, "type" | "name">): string {
  return tData(`data.event.${e.type}.name`, e.name);
}

export function eventDescription(e: Pick<EventCardView | EventTypeView, "type" | "description">): string {
  return tData(`data.event.${e.type}.description`, e.description);
}

export function missionText(m: MissionView): string {
  if (!m.id) return m.text;
  return tk(`data.mission.${m.id}.text`, { name: m.rowerName });
}

export function missionReward(m: MissionView): string {
  return m.id ? tData(`data.mission.${m.id}.reward`, m.reward) : m.reward;
}

export function trainerName(tr: Pick<TrainerView, "id" | "name">): string {
  return tData(`data.trainer.${tr.id}.name`, tr.name);
}

export function trainerAbility(tr: Pick<TrainerView, "id" | "ability">): string {
  return tData(`data.trainer.${tr.id}.ability`, tr.ability);
}

export function trainerDescription(tr: Pick<TrainerView, "id" | "description">): string {
  return tData(`data.trainer.${tr.id}.description`, tr.description);
}

export function boatName(b: Pick<BoatView, "defId" | "name">): string {
  return b.defId ? tData(`data.boat.${b.defId}.name`, b.name) : b.name;
}

/** Rower name. */
export function rowerName(r: Pick<RowerView, "name">): string {
  return r.name;
}

export function guestName(fallback: string): string {
  return tData("data.guest.name", fallback);
}

export function selectionEffect(p: Pick<PhaseView, "selectionId" | "selectionEffect">): string {
  return p.selectionId
    ? tData(`data.selection.${p.selectionId}.label`, p.selectionEffect ?? "")
    : (p.selectionEffect ?? "");
}

export function ereignisName(e: EreignisView | undefined): string {
  if (!e) return t("station.ereignis");
  return e.id ? tData(`data.ereignis.${e.id}.name`, e.name) : e.name;
}

/** Ereignis card text plus the per-player details (affected rowers). */
export function ereignisText(e: EreignisView): string {
  const desc = e.id ? tData(`data.ereignis.${e.id}.description`, e.description) : e.description;
  return [desc, ...(e.details ?? []).map(tm)].join(" ");
}
