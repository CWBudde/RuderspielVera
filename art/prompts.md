# Ruderspiel prompt entries

One entry per asset. The ID is the art key (`web/src/assets/art/<artKey>.png`). `assemble.sh <kind>
<ID> "<attachment line>"` takes the first ```text block after the line starting `**<ID> `. The kind
(section of `art/style.md`) is given in brackets. All people are generic, fictional characters.

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

## Boats (Objects)

**boat-1x** [objects] single scull

```text
A single scull on the Maschsee: exactly 1 rower with two identical oars, one per hand and one per
side, so exactly 1 shaft and blade on the near side and 1 on the far side, 2 in total. The boat
travels toward the LEFT, seen from a low three-quarter side view at water level: the pointed bow is at
the left end, the wake trails off to the right. Slim white shell with a red stripe. The rower in a
plain navy tank top sits facing RIGHT toward the stern, back to the bow, at the catch (arms straight,
knees bent, both blades just entering the water, mirror-symmetric). Tree-lined shore and the distant
domes of the Neues Rathaus in the background, one small white sailboat. LANDSCAPE, aspect 3:2 (for
example 1536 x 1024 pixels). The boat spans about 60 percent of the canvas width, centred slightly
below the middle, with at least 15 percent empty water left and right.
```

**boat-2x** [objects] double scull

```text
A double scull on the Maschsee: exactly 2 rowers, each with two identical oars, one per hand and one
per side. Exactly 2 shafts and blades on the near side and 2 on the far side, 4 in total, never 3 or
5; the far-side blades are visible beyond the hull and mirror the near-side ones; the two oars of a
side are parallel and never cross; each oar sits in the oarlock beside its own rower; the two handles
of each rower overlap slightly in front of the chest. All four oars identical (same length, shaft
colour, blade shape and blade colour) and in the same stroke phase. The boat travels toward the LEFT,
low three-quarter side view at water level: the pointed bow is at the left end, the wake trails off to
the right. Yellow shell, both rowers in plain navy tank tops facing RIGHT toward the stern, backs to
the bow. Tree-lined shore and small white sailboats behind. LANDSCAPE, aspect 3:2 (for example 1536 x
1024 pixels). The boat spans at most 65 percent of the canvas width, with at least 15 percent empty
water left and right.
```

**boat-4x** [objects] quad scull

```text
A quad scull (no coxswain) racing on the Maschsee: exactly 4 rowers, each with two identical oars,
one per hand and one per side. Exactly 4 shafts and blades on the near side and 4 on the far side, 8
in total; the far-side oars mirror the near-side oars and their blades are visible beyond the hull;
riggers symmetric at every seat; each oar sits in the oarlock beside its own rower and never crosses
the hull. All eight oars identical (same length, shaft colour, blade shape and colour), parallel and in
the same stroke phase: the catch, all blades just entering the water, every rower with arms straight
and knees bent. The boat travels toward the LEFT, three-quarter side view slightly from above: the
pointed bow with a white bow wave is at the left end, the wake trails off to the right. Light-blue
shell, four rowers in plain navy tank tops, all facing RIGHT toward the stern, backs to the bow.
Tree-lined shore, the distant domes of the Neues Rathaus above the trees. LANDSCAPE, aspect 3:2 (for
example 1536 x 1024 pixels). The boat spans at most 70 percent of the canvas width, fully inside the
canvas, with at least 15 percent empty water left and right.
```

**boat-8x** [objects] sweep eight with coxswain

```text
A sweep-rowing eight with coxswain racing on the Maschsee: exactly 8 rowers in one long shell, each
holding ONE oar with both hands, plus a coxswain. The sides alternate seat by seat (first rower's oar
on the far side, second on the near side, and so on), so exactly 4 oars on the near side and 4 on the
far side, never all 8 on one side; the riggers alternate the same way; the far-side blades are
visible beyond the hull. All eight oars identical (same length, shaft colour, blade shape and colour),
parallel on each side and in the same stroke phase. The boat travels toward the LEFT, three-quarter
side view slightly from above: the pointed bow with a strong white bow wave is at the left end. The
coxswain sits at the far RIGHT end (the stern), facing LEFT toward the bow, face to face with the
crew. All eight rowers face RIGHT toward the coxswain, backs to the bow. Long yellow-green shell,
rowers in plain navy tank tops with no marks. Tree-lined shore and small white sailboats behind.
LANDSCAPE, aspect 3:2 (for example 1536 x 1024 pixels). The whole boat including bow and stern fits
inside the canvas and spans at most 75 percent of its width, with at least 12 percent empty water
left and right.
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
