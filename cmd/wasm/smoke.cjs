// Node smoke test for the WebAssembly build: creates a game, plays it to the end with a trivial
// human strategy, and checks export/import (localStorage persistence path) and error envelopes.
//
// Usage: node cmd/wasm/smoke.cjs <dir with ruderspiel.wasm + wasm_exec.js>
"use strict";

const fs = require("fs");
const path = require("path");

const dir = path.resolve(process.argv[2] || "web/public");
globalThis.crypto ??= require("crypto").webcrypto;
require(path.join(dir, "wasm_exec.js"));

function call(fn, ...args) {
  const res = JSON.parse(globalThis.ruderspielWasm[fn](...args));
  return res;
}

function ok(fn, ...args) {
  const res = call(fn, ...args);
  if (res.error) throw new Error(`${fn}: ${res.status} ${res.error}`);
  return res.data;
}

function decide(view) {
  const p = view.phase;
  const me = view.players.find((x) => x.id === view.you);
  const d = { phaseId: p.id, kind: p.kind };
  if (p.kind === "training" && p.points > 0) {
    d.training = [{ rowerId: me.rowers[0].id, stat: "technik", points: p.points }];
  } else if (p.kind === "selection") {
    d.rowerIds = me.rowers.slice(0, p.count).map((r) => r.id);
  } else if (p.kind === "draft" && p.draftOffer?.length) {
    d.draftCardId = p.draftOffer[0].id;
  }
  return d;
}

async function main() {
  const go = new Go();
  const { instance } = await WebAssembly.instantiate(fs.readFileSync(path.join(dir, "ruderspiel.wasm")), go.importObject);
  go.run(instance); // never resolves; main registers the API synchronously, then blocks
  if (!globalThis.ruderspielWasm) throw new Error("ruderspielWasm not registered");

  const t0 = performance.now();
  const cr = ok("createGame", JSON.stringify({ playerName: "Vera", bots: 2 }));
  // Random seed on purpose: uint64 seeds above 2^53 must survive export/import.
  console.log(`created ${cr.gameId} in ${(performance.now() - t0).toFixed(0)} ms, phase ${cr.view.phase.kind}`);

  let view = cr.view;
  let steps = 0;
  const t1 = performance.now();
  while (view.phase.kind !== "gameover") {
    if (++steps > 2000) throw new Error("no game over");
    view = ok("submitDecision", cr.gameId, cr.token, JSON.stringify(decide(view)));
    if (steps === 10) {
      // Persistence path: export, import into a fresh registry entry, compare.
      const json = ok("exportGame", cr.gameId); // opaque JSON string
      const snap = JSON.parse(json);
      const imp = ok("importGame", json);
      const again = ok("getView", imp.gameId, cr.token);
      if (again.phase.id !== view.phase.id) throw new Error("import changed the phase");
      console.log(`export/import ok after ${steps} decisions (${json.length} bytes, ${snap.decisions.length} logged decisions)`);
    }
  }
  const ms = performance.now() - t1;
  console.log(`game over after ${steps} human decisions in ${ms.toFixed(0)} ms; winners ${view.winners}`);

  const log = ok("getLog", cr.gameId, cr.token);
  const cat = ok("getCatalog");
  console.log(`log ${log.log.length} entries, ${log.results.length} results; catalog ${cat.rowers.length} rowers`);

  const expect = (res, status) => {
    if (res.status !== status || !res.error) throw new Error(`want ${status}, got ${JSON.stringify(res)}`);
  };
  expect(call("getView", cr.gameId, "nope"), 401);
  expect(call("getView", "deadbeef", cr.token), 404);
  expect(call("submitDecision", cr.gameId, cr.token, "{"), 400);
  expect(call("createGame", JSON.stringify({ bots: 9 })), 400);
  expect(call("getView", cr.gameId), 400);
  console.log("error envelopes ok");
  process.exit(0);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
