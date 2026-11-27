import { Fragment, type ReactNode } from "react";
import { tk, type UiKey } from "./index";

/**
 * Like t(), but params may be React nodes (e.g. a bold name): "Wähle {n} Ruderer" with
 * { n: <b>3</b> } renders the element in place. Plain params are formatted as in t().
 */
export function tr(key: UiKey, args: Record<string, ReactNode>): ReactNode {
  const plain: Record<string, string | number> = {};
  const nodes: Record<string, ReactNode> = {};
  for (const [k, v] of Object.entries(args)) {
    if (typeof v === "string" || typeof v === "number") plain[k] = v;
    else nodes[k] = v;
  }
  // Translate with node params left as placeholders, then splice the nodes in.
  const text = tk(key, plain);
  const parts = text.split(/\{(\w+)\}/g);
  return parts.map((p, i) =>
    i % 2 === 1 && p in nodes ? (
      <Fragment key={i}>{nodes[p]}</Fragment>
    ) : (
      <Fragment key={i}>{i % 2 === 1 ? `{${p}}` : p}</Fragment>
    ),
  );
}
