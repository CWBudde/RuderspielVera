import { tm, type Msg } from "@/i18n";

/**
 * Error carrying the HTTP status and a message: either the server's message key
 * (`{"error": key, "msg": {key, args}}`) or a client-side text. `message` is rendered on access,
 * so a server error follows a later language switch.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly msg: Msg;

  constructor(status: number, message: string | Msg) {
    super();
    this.status = status;
    this.msg = typeof message === "string" ? { key: "raw", args: { text: message } } : message;
    this.name = "ApiError";
  }

  override get message(): string {
    return tm(this.msg);
  }
}

/** Reads the Msg of a failed call's body ({"error": key, "msg": {...}}), if there is one. */
export function errorMsg(body: unknown): Msg | undefined {
  if (!body || typeof body !== "object") return undefined;
  const b = body as { error?: unknown; msg?: unknown };
  const m = b.msg as Msg | undefined;
  if (m && typeof m === "object" && typeof m.key === "string") return m;
  if (typeof b.error === "string") return { key: b.error };
  return undefined;
}
