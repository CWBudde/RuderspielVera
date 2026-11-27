/** Message emitted by the engine (internal/engine/msg.go): a catalog key plus params. */
export interface Msg {
  key: string;
  args?: Record<string, MsgArg>;
}

/** A param: a literal (proper names, numbers), a nested translatable message, or a list of them. */
export type MsgArg = string | number | Msg | MsgArg[];

export type Lang = "de" | "en";
export const LANGS: Lang[] = ["de", "en"];
