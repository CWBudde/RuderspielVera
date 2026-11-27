# Ruderspiel prompt entries

One entry per asset. The ID is the art key (`web/src/assets/art/<artKey>.png`). `assemble.sh <kind>
<ID> "<attachment line>"` takes the first ```text block after the line starting `\*\*<ID> `. The kind
(section of `art/style.md`) is given in brackets. All people are generic, fictional characters, except
the `person`template, which is only used with a photo from`reference-rowers/`.

Default attachments: `examples/vera-karte.png` as "style and backdrop reference only; invent a
completely different, fictional person, do not copy the face, hair or the card layout, text or stat
boxes"; `examples/kartendesign.png` for boat and trainer looks ("style only, ignore all text and
layout"); `examples/spielbrett.png` for the board ("style and map layout only; leave out every circle,
sign, number and the lake title").

## Rowers (Portraits)

**rower-kinder-w** [portraits] girl, age group Kinder

```text
A fictional girl of about 11 years, a young club rower. Short dark-brown bob with a hair clip,
round friendly face, light-brown skin, cheerful gap-toothed grin. Small child proportions, slim
shoulders. Navy rowing tank top as described.
```

**rower-kinder-m** [portraits] boy, age group Kinder

```text
A fictional boy of about 11 years, a young club rower. Messy straw-blond hair, fair skin with a
light sunburn on the nose, big curious eyes, open happy smile. Small child proportions, slim arms.
Navy rowing tank top as described.
```

**rower-b-w** [portraits] girl, B-Juniorinnen

```text
A fictional girl of about 14 years, a club rower in the B-junior age group. Long black hair in a
high ponytail, olive skin, dark eyes, confident determined half-smile. Lean athletic early-teen
build. Navy rowing tank top as described.
```

**rower-b-m** [portraits] boy, B-Junioren

```text
A fictional boy of about 14 years, a club rower in the B-junior age group. Short curly dark hair,
dark-brown skin, warm brown eyes, relaxed friendly smile. Lanky early-teen build with long arms.
Navy rowing tank top as described.
```

**rower-a-w** [portraits] young woman, A-Juniorinnen

```text
A fictional girl of about 16 years, a club rower in the A-junior age group. Copper-red hair in a
thick braid over one shoulder, pale skin with many freckles, green eyes, calm proud smile. Strong,
athletic shoulders of a trained rower. Navy rowing tank top as described.
```

**rower-a-m** [portraits] young man, A-Junioren

```text
A fictional boy of about 17 years, a club rower in the A-junior age group. Short light-brown hair
with an undercut, light skin, slight stubble-free jaw, sporty grin. Broad, muscular shoulders and
arms of a trained rower. Navy rowing tank top as described.
```

## Trainers (Portraits)

**trainer-1** [portraits] older male coach

```text
A fictional rowing coach, a man of about 60. Grey short beard, weathered tanned face with laugh
lines, navy baseball cap, navy zip-up club training jacket over a white polo collar, a whistle on a
lanyard. Kind but demanding look. Not a rower: no tank top.
```

**trainer-2** [portraits] female coach, 40s

```text
A fictional rowing coach, a woman of about 45. Short silver-blonde pixie haircut, sunglasses pushed
up on her head, light skin with sun freckles, light-blue windbreaker over a navy shirt, holding a
stopwatch near her chest. Warm, energetic smile. Not a rower: no tank top.
```

**trainer-3** [portraits] young male coach, 20s

```text
A fictional rowing coach, a young man of about 26. Dark-brown skin, short black twists, friendly
wide smile, navy hoodie with the hood down and a white t-shirt underneath, a megaphone held loosely
in one hand at chest height. Sporty and approachable. Not a rower: no tank top.
```

**trainer-4** [portraits] female coach, 30s

```text
A fictional rowing coach, a woman of about 35. Dark hair tied in a low bun under a red sun visor,
light olive skin, focused thoughtful expression with a slight smile, navy softshell vest over a
grey long-sleeve shirt, a clipboard tucked under her arm. Not a rower: no tank top.
```

## Named people (Portraits)

Template only: never run it with the placeholders. Copy it to `art/gen/<date>/people/<id>-entry.txt`,
fill every `<…>` and pick one age-group line; installed as `people/<card-id>`. Only for people with a
photo in `reference-rowers/` (private; see the skill, "Named-person portraits"). Attach the photo as
image 1 and `examples/vera-karte.png` as image 2.

**person** [portraits] stylized likeness of a named club member from a reference photo

```text
A stylized illustrated card portrait of <NAME>, a real <girl|boy|woman|man> from the rowing club,
drawn from the attached likeness reference photo (image 1). Age group: <one of: Kinder, about 10 to
12 years | B-Junioren, about 13 to 14 years | A-Junioren, about 15 to 17 years | Trainer, adult>.
Keep the person recognisable: the same face shape, eye shape, nose and smile, the same hair colour,
length and style<, the same glasses>, the same skin tone and the same apparent age as in the photo.
Seen in the photo: <short description: e.g. light-brown straight hair with a side fringe, blue eyes,
fair skin, slim face>. Translate the person into the game's ink and watercolour style; do not paint a
photorealistic copy. Do not copy the photo's background, clothing, lighting or framing: <rowers: the
navy rowing tank top as described | trainers: a navy zip-up club training jacket>, the Maschsee
backdrop as described. PORTRAIT orientation, aspect 3:4. No text anywhere.
```

## Boats (Objects)

Side views kept failing on direction and oar counts; these high-angle entries produced the picks
(1x, 2x, 4x). The 8x entry has not produced an image yet (.20 disk full).

**boat-1x** [objects] single scull

```text
A single scull on the Maschsee seen from a HIGH ANGLE, looking down at about 50 degrees from above and
slightly to the side, so that both sides of the boat and all oars are clearly visible, spread out
symmetrically like wings. Exactly 1 rower with two identical oars, one per hand and one per
side: exactly 1 oar on the left-hand side of the hull and 1 on the right-hand side, 2 in total, never
3. Each oar starts at the oarlock on a rigger beside the rower; the two oars form a symmetric V
and never cross each other or the hull; the left oar mirrors the right oar
exactly. Both oars identical (same length, dark shaft, same blue blades), all blades at the
water surface in the same stroke phase (the catch: arms straight, knees bent). The boat travels
diagonally toward the upper LEFT of the canvas: the pointed bow leads at the upper left, a white wake
trails behind the stern toward the lower right. White shell with a red stripe, the rower in a plain navy tank top
facing the stern (toward the lower right), back to the bow. Blue water with sparkles all around, a
strip of tree-lined shore along the top edge. LANDSCAPE, aspect 3:2 (for example 1536 x 1024 pixels).
The boat with its oars spans at most 60 percent of the canvas width, with at least 15 percent empty
water left and right.
```

**boat-2x** [objects] double scull

```text
A double scull on the Maschsee seen from a HIGH ANGLE, looking down at about 50 degrees from above and
slightly to the side, so that both sides of the boat and all oars are clearly visible, spread out
symmetrically like wings. Exactly 2 rowers, each with two identical oars, one per hand and one per
side: exactly 2 oars on the left-hand side of the hull and 2 on the right-hand side, 4 in total, never
3 or 5. Each oar starts at the oarlock on a rigger beside its own rower; the two oars on one side are
parallel and never cross each other or the hull; the oars on one side mirror those on the other side
exactly. All four oars identical (same length, dark shaft, same blue blades), all blades at the
water surface in the same stroke phase (the catch: arms straight, knees bent). The boat travels
diagonally toward the upper LEFT of the canvas: the pointed bow leads at the upper left, a white wake
trails behind the stern toward the lower right. Yellow shell, both rowers in plain navy tank tops
facing the stern (toward the lower right), backs to the bow. Blue water with sparkles all around, a
strip of tree-lined shore along the top edge. LANDSCAPE, aspect 3:2 (for example 1536 x 1024 pixels).
The boat with its oars spans at most 65 percent of the canvas width, with at least 15 percent empty
water left and right.
```

**boat-4x** [objects] quad scull

```text
A quad scull (no coxswain) on the Maschsee seen from a HIGH ANGLE, looking down at about 50 degrees
from above and slightly to the side, so that both sides of the boat and all oars are clearly visible,
spread out symmetrically like wings. Exactly 4 rowers, each with two identical oars, one per hand and
one per side: exactly 4 oars on the left-hand side of the hull and 4 on the right-hand side, 8 in
total. Each oar starts at the oarlock on a rigger beside its own rower; the four oars on one side are
parallel, evenly spaced and never cross each other or the hull; the oars on one side mirror those on
the other side exactly. All eight oars identical (same length, dark shaft, same yellow blades), all
blades at the water surface in the same stroke phase (the catch: arms straight, knees bent). The boat
travels diagonally toward the upper LEFT of the canvas: the pointed bow leads at the upper left, a
white wake trails behind the stern toward the lower right. Light-blue shell, four rowers in plain navy
tank tops, all facing the stern (toward the lower right), backs to the bow. Blue water with sparkles
all around, a strip of tree-lined shore along the top edge. LANDSCAPE, aspect 3:2 (for example 1536 x
1024 pixels). The boat with its oars spans at most 70 percent of the canvas width, with at least 15
percent empty water left and right.
```

**boat-8x** [objects] sweep eight with coxswain

```text
A sweep-rowing eight with coxswain on the Maschsee seen from a HIGH ANGLE, looking down at about 50
degrees from above and slightly to the side, so that both sides of the long boat and all oars are
clearly visible, spread out like the legs of a water strider. Exactly 8 rowers in one long shell, each
holding ONE oar with both hands, plus a coxswain. The oars alternate sides seat by seat: rower 1 left,
rower 2 right, rower 3 left, rower 4 right, and so on, so exactly 4 oars stick out on the left-hand
side and exactly 4 on the right-hand side, 8 in total, never more than 4 on one side; the riggers
alternate the same way. Each oar starts at the oarlock beside its own rower, the oars on one side are
parallel and evenly spaced and never cross each other or the hull. All eight oars identical (same
length, dark shaft, same white blades), all blades at the water surface in the same stroke phase. The
boat travels diagonally toward the upper LEFT of the canvas: the pointed bow leads at the upper left,
a white wake trails behind the stern toward the lower right. The coxswain (cap, navy jacket) sits at
the stern end at the lower right, facing the crew and the bow; all eight rowers face the coxswain,
backs to the bow. Long yellow-green shell, rowers in plain navy tank tops with no marks. Blue water
with sparkles all around, a strip of tree-lined shore along the top edge. LANDSCAPE, aspect 3:2 (for
example 1536 x 1024 pixels). The whole boat with its oars fits inside the canvas and spans at most 80
percent of its width.
```

## Board (Board)

**board-bg** [board] Maschsee board background without stations

```text
A top-down illustrated map of the Maschsee in Hannover and its park surroundings. The lake is a long
rounded shape running horizontally through the centre, about 60 percent of the canvas width and 35
percent of its height, with a promenade along the north shore, small jetties and a boathouse on the
south shore, a few tiny white sailboats and one small rowing boat on the water. Around it a broad
ring of dense park trees with winding light paths; at the canvas corners and edges small city blocks
with red-orange tiled roofs. No circles, spaces, numbers, signs, title or text anywhere.
LANDSCAPE, aspect 3:2 (for example 1536 x 1024 pixels).
```

## Title (Objects)

**title** [objects] title splash

```text
Title splash illustration: a coxless quad scull (four rowers in navy tank tops, each with two oars)
seen from a three-quarter side view in the lower half of the canvas, travelling toward the LEFT across
the Maschsee at golden hour. The pointed bow leads on the left; all four rowers face RIGHT toward the
stern (backs to the direction of travel), exactly 8 oars, 4 on each side, each rower
holding two identical oars, all parallel and in the same stroke phase, faces small and simplified in profile, varied short dark
and brown hair (no long curly blonde hair). The scene is lit by the low sun,
warming the scene with orange and gold light, long sparkling reflections on the water. In the
background, across the lake, the Neues Rathaus of Hannover with its large green copper dome and
towers, glowing in the evening light above the trees. A couple of small white sailboats, soft pink
and gold clouds. Keep the upper third of the canvas calm, open sky with only soft clouds: no branches, no
buildings or towers reaching into it (a title will be overlaid later by the game; paint no text
yourself). The Neues Rathaus sits in the middle band, below the upper third. LANDSCAPE, wide aspect 16:9 (for example 1792 x 1008
pixels).
```

## Icons (Icons)

**icons-weather** [icons] weather icon sheet

```text
Five weather icons in one row of five equal cells: 1 rain (a grey-blue cloud with slanted
raindrops), 2 heat (a big orange-yellow sun with a red thermometer), 3 wind (three curved white-blue
wind swirls with a leaf), 4 cold (a pale-blue snowflake with a small icicle), 5 normal fair weather
(a yellow sun half behind a small white cloud). LANDSCAPE, aspect 5:1 rows are not possible, so use
aspect 3:2 (for example 1536 x 1024 pixels) with the five icons in a single horizontal row across
the middle third of the canvas, each about 1/6 of the canvas width.
```

**icons-medals** [icons] medal sheet

```text
Two sports medals side by side: 1 a gold medal, 2 a silver medal. Each is a round metal disc with
a simple embossed crossed-oars motif (no text, no numbers), hanging from a short navy-and-white
ribbon. Metallic shine painted in watercolour. LANDSCAPE, aspect 3:2 (for example 1536 x 1024
pixels), each medal about 1/3 of the canvas width.
```
