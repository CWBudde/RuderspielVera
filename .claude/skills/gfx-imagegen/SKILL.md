---
name: gfx-imagegen
description: Use when generating, regenerating or trying images for Ruderspiel (rower or trainer portraits, named portraits of club members from reference-rowers/ photos, boat art, the board background, the title splash, icon sheets, card art), or when asked to run "imagegen on .20" or "on the gfx machine". Claude cannot make images itself; this runs Codex's built-in image tool over ssh on 192.168.213.20 and covers prompt assembly from art/style.md and art/prompts.md, attaching style references from examples/, the try loop, review records and installing a pick into web/src/assets/art/.
---

# Image generation through Codex on the gfx machine

Run from the repo root with `K=.claude/skills/gfx-imagegen`; repo paths (`art/`, `examples/`,
`web/`) are relative to the root. The scripts work from any cwd.

## Hard rules

- **NEVER run a local `codex`** on this machine: it is a different Codex account. Images come only from
  `ssh 192.168.213.20 ~/.local/bin/codex exec …`, and `$K/scripts/gen.sh` hard-codes that path. Do not
  write any other runner.
- On .20, write only inside the dedicated work dir `/mnt/Projekte/Code/games/RuderspielVera` (created
  with `mkdir -p`; `gen.sh` writes to `art/gen/<GEN_DIR>/<group>/` there). Never touch other repos
  there (Historyline lives next door), never touch `/mnt/Large`, never delete files there.
- At most **three** Codex runs at once on .20 (other agents share it); `gen.sh` waits for a free slot,
  but the check is not atomic: launch at most three `gen.sh` at once yourself. Both remote disks are
  nearly full: `gen.sh` refuses under 2 GB free on `/mnt/Projekte` and under 1 GB free for `~/.codex`
  (the root disk, which holds Codex's transcripts and images). When the root disk is full, Codex says
  the image tool "lost" its result; ask the user to free space there, do not delete anything yourself.
- **Real children**: the rowers on the real cards (`examples/vera-karte.png`, `kartendesign.png`) are
  real children of a rowing club. Generic art keys (`rower-*`, `trainer-*`) show only generic,
  fictional characters. Attach `vera-karte.png` only as a style/backdrop reference and say so: "style
  only, do not copy the face, hair or features" (plus "invent a completely different, fictional
  person" for generic keys).
- **Named-person portraits** (user decision 2026-11-27): a portrait of a real club member is allowed
  **only** when the user put a reference photo of that person into `reference-rowers/`. See
  "Named-person portraits" below. Without a photo, no likeness of a named person.
- **`reference-rowers/` is private**: git-ignored, never committed, never copied anywhere except the
  run's gen dir on .20 (`gen.sh` copies attachments there). Do not attach these photos to anything
  else, do not upload or publish them.
- No text in images unless the entry asks for it; no logos or brands. The UI overlays names and stats.
- No Python, no ImageMagick: the helpers here are Go.

## Runner

```sh
$K/scripts/gen.sh <group>-<NN> <prompt.txt> [ref.png ...]     # e.g. rowers-kw01; GEN_DIR=YYYY-MM-DD (default today)
```

It copies the prompt and refs to `art/gen/$GEN_DIR/<group>/` on .20 (same relative path as locally;
`art/gen/` is git-ignored), runs `codex exec --skip-git-repo-check -C <workdir> -s workspace-write
--json -o <id>-last.txt -i <ref>… "<wrapper>" < prompt`, and copies back `<id>.jsonl` and
`<id>-last.txt`. The local `<id>.png` is the newest non-empty
`~/.codex/generated_images/<thread>/exec-<uuid>.png` on .20, where `<thread>` is the `thread_id` in
this run's log (sha-checked; a `WARN` line if the remote `<id>.png` differs). Without one it falls back
to the `<id>.png` Codex wrote, with a `WARN`, and the OK line names `thread-<id>` instead of an exec id.
An id whose `<id>.png` exists on .20 or here is refused with a suggested free id (`<id>b`);
`GEN_FORCE=1` runs anyway. The last line is `OK <png> <exec-….png>` or `FAILED <id>…` with the log
tail. The wrapper tells Codex to use the imagegen skill and image_gen tool, generate exactly one image,
copy it unchanged, and touch nothing else. 1–3 min per image; run it with `run_in_background` and
launch at most three at once (for a batch: three background shell loops, each running its share of
the ids one after another). Attachments are numbered in the order given.

## Prompt assembly

```sh
$K/scripts/assemble.sh portraits rower-kinder-w "Image 1 (vera-karte.png) is a STYLE AND BACKDROP REFERENCE ONLY: …" > p.txt
$K/scripts/assemble.sh objects my-edited-entry.txt "" > p.txt
```

- A full prompt is the `art/style.md` **Preamble** text block, a blank line, the section block
  (`$K/scripts/preamble.sh portraits|objects|board|icons`), a blank line, the entry from
  `art/prompts.md` **verbatim** (the ```text block after `**<ID> …**`; the ID is the art key and the
  bracket names the kind), a blank line, then one line naming each attachment by number: what to take
  from it and what not to copy.
- To change an entry for a try, write the edited entry to a file and pass the file instead of the ID.
- Style references live in `examples/`: `vera-karte.png` (portrait style, Maschsee backdrop with Neues
  Rathaus and Telemax), `kartendesign.png` (boat card and coach portrait looks; ignore its text),
  `spielbrett.png` (top-down board map; tell Codex to leave out circles, numbers, signs and the title).
  Once a pick exists, attach it to lock the palette for the rest of a group.
- Write the prompt to `art/gen/$GEN_DIR/<group>/<id>-prompt.txt` before running.

## Measuring

Build once: `$K/tools/build.sh` (each helper has its own `go.mod`; binaries in `bin/`, git-ignored).

| Helper                                                       | Use                                                                                                                                                                         |
| ------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `bin/wstat in.png [gx gy]`                                   | mean RGB, L\*, σL\*, a\*, b\*, chroma, hue per grid cell (e.g. `2 3`: is the right half of a portrait calm?)                                                                |
| `bin/bbox in.png cols rows`                                  | alpha>128 box, cover and edge contact per sheet cell (icon sheets with alpha)                                                                                               |
| `bin/comp in.png out.png r g b`                              | composite RGBA over a flat colour for viewing                                                                                                                               |
| `bin/rectcrop in.png out.png x y w h`                        | cut a rectangle (one icon from a sheet)                                                                                                                                     |
| `bin/fit [-bits N] [-q Q] in.png out.(png\|jpg) w h [cx cy]` | scale to cover w×h, crop the rest; cx/cy 0..1 place the window (default centre); `-bits` reduces channel precision for smaller PNGs; `.jpg` output writes JPEG at quality Q |

Check the canvas size (`file x.png`) of every try, then **look at it** (Read the PNG) before judging.

## Try loop

1. Assemble the prompt, write it next to the output, run `gen.sh`.
2. Look at the result; check size, text, framing, subject, the calm right half for portraits.
3. Change the prompt **only** for a visible or measured failure, and name it concretely ("the previous
   try put a sailboat in the right half; keep the right half empty water").
4. Each try gets a **new id** (`rowers-kw01`, `rowers-kw01b`, …). At most 4 tries per image; then pick
   the best or report that the entry needs a different approach.

## Records

Per group, `art/gen/<date>/<group>/` holds `<id>-prompt.txt`, `<id>.png`, `<id>.jsonl`,
`<id>-last.txt` and a `review.md`:

```markdown
# Rowers (portraits)

Attachments (all tries): image 1 examples/vera-karte.png (style/backdrop only). Target: PORTRAIT 3:4.

## rowers-kw01 (exec-0f3b4d41-…)

Entry: rower-kinder-w. Change: first try.
Size: 1086x1448. Verdict: best | second | reject. <faults>.

Pick: rowers-kw01 -> rower-kinder-w.
```

## Installing a pick

Web sizes: portraits 600×800, boats 900×600, board 1600×1067, title 1600×900.

```sh
$K/bin/fit -q 78 art/gen/<date>/<group>/<id>.png web/src/assets/art/<artKey>.jpg 600 800 [cx cy]
```

Opaque art is installed as JPEG quality 78 (a 600×800 portrait is ~140 KB, the whole set stays under
3 MB); keep PNG (`fit -bits 6 … .png`) only for images with transparency. `go run ./tools/artconv`
converts any opaque PNG left in `web/src/assets/art/` (not `people/`). `Art.tsx` globs
`*.{webp,png,jpg}`. Use WebP (`cwebp -q 82`) only if `cwebp` is installed (it currently is not). Then add one line to `art/manifest.md`:

```markdown
| <artKey> | web/src/assets/art/<artKey>.jpg | 600x800 | <id> | exec-<uuid> | <date> | <note> |
```

Committing is a separate step (only when asked); `art/gen/` itself stays uncommitted.

## Named-person portraits

For a club member with a photo in `reference-rowers/<Name>.jpg`:

1. **Card id**: look the first name up in `internal/data/rowers.json` / `trainers.json` (`id`, `name`,
   `age`, `gender`). Spellings differ ("Annabel" → `annabelle-a`, "Frau Lincke" → `frau-linke`). A
   name with several cards (e.g. Johann in Kinder and B) is assigned by the apparent age in the photos;
   when it stays ambiguous or the name is not in the data, skip it and tell the user. Empty or
   unreadable photo files: skip and report.
2. **Prompt**: copy the `person` template from `art/prompts.md` into
   `art/gen/$GEN_DIR/people/<id>-entry.txt`, fill the placeholders (name, age group, gender, what to
   keep from the photo after looking at it yourself: face shape, hair colour and style, glasses, skin
   tone, age) and assemble with kind `portraits`. Group `people`, try ids `people-<card-id>01`, `…01b`.
3. **Attachments**: image 1 the photo ("LIKENESS REFERENCE: keep this person's recognisable features;
   do not copy the photo's background, clothing, lighting or framing"), image 2 `examples/vera-karte.png`
   ("STYLE AND BACKDROP REFERENCE ONLY: do not copy the face, hair or features of the person in it, nor
   the card layout, text or stat boxes").
4. **Review**: Read the photo and the result side by side. Reject (new id, at most 3 tries per person)
   for wrong apparent age, lost likeness, text in the image, the photo's background or clothing copied,
   or a broken composition (person in the right half, cropped head).
5. **Install** with `fit -bits 6 … web/src/assets/art/people/<card-id>.png 600 800` and add a manifest
   line with artKey `people/<card-id>`. `Art.tsx` uses `people/<card id>` for a rower or trainer card
   when it exists and falls back to the generic artKey portrait otherwise.

**These files are likenesses of real minors.** `web/src/assets/art/people/` is git-ignored: never
commit it, never publish it (GitHub Pages, a public repo, a shared artifact) and never move it
elsewhere without the user's explicit OK. A local `pnpm build` bundles them into `web/dist/`; do not
deploy such a build.

## Lessons

- **Canvases drift** (3:4 asked → 1086×1448; 3:2 → 1536×1024 or 1672×941). Say LANDSCAPE/PORTRAIT and
  the aspect explicitly in every entry; record the real size in the review; `fit` crops to the target.
- **Sizes come out too large.** State subject sizes as canvas fractions ("spans about 60 percent of the
  canvas width") and feed the measured size back in the next try.
- **Text creeps in** (signs, boat names, lettering on shirts). The preamble forbids it; if a try still
  has text, name exactly where it appeared.
- **Reference faces leak.** Always say "style only, invent a different, fictional person" when a
  reference contains a person, and reject a try that looks like the reference.
- Name what to avoid concretely once a try shows it.
- **The remote `<id>.png` is not proof of the run's image**: Codex sometimes copies with `cp -n`, so a
  retry under an old id keeps the old picture. The exec image is the source of truth; give each try a
  new id rather than `GEN_FORCE=1`.
- **Codex sometimes streams the image straight into `<id>.png`** (base64 through a one-off script) and
  leaves a 0-byte `exec-*.png` or no thread folder behind. `gen.sh` skips empty exec files and then
  takes the remote `<id>.png` (with a `WARN`); it fails rather than install an empty file.
- **Never trust exec names grepped from the log**: while hunting for a missing image, Codex lists all of
  `generated_images/`, so the log names other runs' images. With three concurrent runs, an early
  `gen.sh` fetched one rower image for two boat ids. Only the run's own `<thread>` folder counts.
- **Codex may copy another run's image into `<id>.png`**: when its own image was not saved, it once
  copied the newest PNG in `generated_images/` (a concurrent boat run) as the title. The wrapper now
  forbids that, and `gen.sh` fails when the fallback `<id>.png` equals another thread's exec image.
  Still look at every result: a picture of the wrong subject is the tell.
- **Boats are hard**: rowers facing the bow, 3 oars on a side, oars pointing in different directions,
  mixed blade colours, boats filling 90 % of the width. State the direction (bow left, rowers face
  right), the oar count per side, "identical, parallel, same stroke phase", and a maximum width; a high
  camera angle (about 50 degrees down) makes the oars on both sides visible and countable.
- **`ssh` eats stdin**: in a `while read` loop over ids, run `gen.sh … </dev/null`, or the loop ends
  after the first image.
