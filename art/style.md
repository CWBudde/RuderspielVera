# Ruderspiel art style

Prompt text for the `gfx-imagegen` skill. `preamble.sh <kind>` prints the **Preamble** block, a blank
line and the block of the named section; `assemble.sh` appends an entry from `art/prompts.md` and the
attachment line. Edit the text blocks here, not in the scripts.

References in `examples/`: `vera-karte.png` (the current rower card), `kartendesign.png` (card layout
with boat and trainer art), `spielbrett.png` (the illustrated Maschsee board). The rowers on the real
cards are real children of a rowing club: generated art shows only generic, fictional people.

Palette anchors: navy kit and frames `#1f3a5f`, lake blue `#2f6fb0`–`#5aa0d8`, leaf green
`#5c9a2e`–`#a8c64a`, warm skin and hair tones, cream highlights `#f6efdc`, red-orange rooftops
`#c8643c`. Pastel station colours on the board (yellow, pink, mint, sky blue) belong to the UI, not
to the art.

## Preamble

```text
Illustration for a friendly family board and computer game about youth rowing in Hannover, Germany.
Style: hand-drawn illustration with fine, confident dark-brown-to-black ink outlines and loose
watercolour washes inside them, like a modern illustrated children's book or a premium card game.
Visible paper-like watercolour texture, soft colour bleeding at the edges of washes, crisp ink
contours on people, boats and buildings, lighter sketchy linework on foliage and water.
Palette: saturated but friendly and warm. Deep lake blues with white sparkle highlights, lush
yellow-green and mid-green tree canopies, clear summer sky blue with soft white cumulus clouds,
warm sunlit skin tones, navy blue rowing kit, cream and warm white highlights. Warm daylight from
the upper left, gentle shadows, no harsh contrast, no gloomy or desaturated areas.
Setting: the Maschsee lake in Hannover in summer, tree-lined shores, small white sailboats.
Absolutely no text, letters, numbers, words, signatures, watermarks, logos, brand marks or
lettering anywhere in the image, also not on clothing, boats, flags or signs. No borders, frames,
card layouts, UI panels or stat boxes: the image is pure illustration that fills the whole canvas.
All people are generic, fictional characters invented for this game; never a likeness of a real
person.
```

## Portraits

```text
Portrait art for a character card. Half-body portrait (head to mid-chest or waist), the person
turned slightly toward the viewer, relaxed confident smile, eyes clearly drawn with ink outlines and
watercolour shading, natural freckles or skin detail welcome. Expressive but not caricatured;
realistic proportions for the stated age.
Rowers wear a plain navy blue sleeveless rowing tank top (racing singlet) with a white side stripe
and a small white crossed-oars emblem on the chest, no other emblem, no text. Coaches wear outdoor
club clothing as described in the entry.
Backdrop: the Maschsee in Hannover on a sunny day: blue lake, tree-lined far shore, small white
sailboats, the green copper domes of the Neues Rathaus (New Town Hall) small and distant above the
trees, the slender Telemax telecommunications tower far away on the right, overhanging leafy branches
in the upper left corner. Background painted slightly softer and lighter than the person.
Composition: PORTRAIT orientation, aspect 3:4 (for example 1152 x 1536 pixels). The person occupies
the LEFT two thirds of the canvas. The RIGHT half of the canvas from about 30 percent to 95 percent
of the height must stay calm and uncluttered: only lake water and sky, no person, no hands, no
boats, no branches there, because game statistics are overlaid on it later.
```

## Objects

```text
Scene or object art for a game card or splash screen. Rowing boats are slim racing shells (sculling or
sweep boats as the entry says) with outriggers and long oars with identically coloured blades, painted
with accurate rowing details.
ROWERS ALWAYS SIT WITH THEIR BACKS TO THE DIRECTION OF TRAVEL: they face the stern and look back
over it, never toward the bow; the pointed bow leads, the wake trails behind the stern. In sculling
boats each rower holds one oar in each hand, one blade on each side of the boat, so a
sculling boat shows exactly as many oars on each side as it has rowers; in a sweep boat each rower holds
one oar and the sides alternate. All oars are identical, parallel and in the same stroke phase, evenly
spaced and synchronised, all oars identical in length, shaft colour and blade shape and colour; riggers
symmetric at every seat; each oar sits in the oarlock beside its own rower and never crosses the
hull; blades at the water surface; the far-side oars mirror the near-side ones and their blades are
visible beyond the hull; body posture matches the blade phase (catch: arms straight, knees bent;
finish: hands at the chest, legs flat); a crisp white bow wave and small puddles where the blades leave the
water. Crews wear the plain navy rowing tank tops with no text. Rowers are small figures, faces
simplified, no recognisable individuals, no marks or emblems on clothing in boat scenes.
Backdrop: the Maschsee in Hannover in summer, tree-lined shores, small white sailboats, distant
copper-green domes of the Neues Rathaus above the trees where the entry asks for it.
Composition: LANDSCAPE orientation with the aspect stated in the entry. Keep the main subject fully
inside the canvas with a margin on every side; nothing important cut off at the edges. A boat
leaves at least 15 percent empty water on the left and on the right (it spans at most 70 percent of
the canvas width; earlier tries overshot to 88-95 percent).
```

## Board

```text
Game board background, seen from directly above (top-down, slight oblique softness is fine, like an
illustrated city map). Dense, detailed ink-and-watercolour linework: individual round tree crowns in
many greens, winding light-beige park paths, small red-orange tiled rooftops and white facades of
city blocks around the park, a promenade and jetties along the shore.
The lake is a large, elongated, rounded shape with deep blue water, fine white ripple strokes,
a few tiny white sailboats and one small rowing boat.
The board must be empty of game elements: no circles, no spaces, no tokens, no numbers, no arrows,
no start or finish signs, no title, no text of any kind. Even, readable detail density across the
whole canvas so that UI elements can be placed anywhere on top.
```

## Icons

```text
Icon sheet for a game interface. Each icon is a single bold, simple motif drawn with thick dark ink
outlines and flat watercolour fill, readable at 48 pixels. Icons sit in a regular grid, one per cell,
centred, all at the same visual size, on a plain warm cream background (#f6efdc) with generous empty
space between them. No labels, no text, no numbers, no frames around the cells, no drop shadows.
```
