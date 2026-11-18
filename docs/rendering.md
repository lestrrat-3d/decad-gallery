# Model renderer

This page documents `tools/render`.

This separate Go module builds gallery models with Decad and renders their
tessellated meshes with SolidLens. Each image preset names a model and a shot.
Add a scene builder and a preset entry in `main.go` for another model or shot.

The spinning wheel also has an explicit work-in-progress mode:

```sh
go -C tools/render run . -model spinningwheel-preview -shot readme -out ../../models/spinningwheel/images
```

It draws completed native parts using the same constructors and pose as the strict model.
It records failed construction or meshing in `preview-status.json`, leaves assembly-wide joint cuts pending,
and labels every static image. Failed tools are excluded. It writes the hero, index, and five paired views.
The `spinningwheel/exploded` preset pulls the full wheel apart by sub-assembly (frame, wheel, treadle, footman and
gimbal, head, flyer, brake and drive band). The `readme` shot writes it as `exploded.png`, and writes one image per
distinct body (identical bodies share one) into `parts/` for the README's parts table.
The ordinary `spinningwheel` preset still requires the complete assembly. Preview mode does not regenerate
the animation or establish retention, clearance or interference checks.

The `ballbearing/hero` preset renders the 608-2RS nominal mounting envelope from its Decad body.
The `readme` shot writes a 1600 × 900 hero and 480 × 270 index:

```sh
go -C tools/render run . -model ballbearing -shot readme -out ../../models/ballbearing/images
```

`ballbearing/605zz` renders the 5 × 14 × 5 mm 605ZZ envelope through the same builder.
The readme shot also writes `605zz.png` at 1600 × 900, plus `top.png` (looking down the bore)
and `side.png` (looking at the outer surface) of the 608-2RS at the same size.

`m2bolt/hero` renders the default M2x10 bolt lying on its side with its head toward the camera.
The `head`, `side`, `underside` and `lengths` presets show other angles and all six standard lengths.
The `readme` shot writes all of them.

`m2nut/hero` renders the M2 nut. The `top`, `side`, `low` and `withbolt` presets show other angles and the nut on an M2x10 bolt.
The `readme` shot writes all of them.

`m5bolt/hero` renders the default M5x25 bolt lying on its side with its head toward the camera.
The `head`, `side` and `lengths` presets show other angles and all eleven standard lengths.
The `readme` shot writes all of them and the index image.

`m5nut/hero` renders the M5 nut. The `top`, `side` and `withbolt` presets show other angles and the nut on an M5x25 bolt.
The `readme` shot writes all of them and the index image.

`spiralspring/hero` renders the default spring lying flat with its axis up. The `top`, `low`, `root` and `sizes`
presets show the spiral from above, the underside, a close-up of the hub and spoke, and three springs of
different sizes. The `readme` shot writes all of them and the index image:
`go -C tools/render run . -model spiralspring -shot readme -out ../../models/spiralspring/images`.

`crankhandle/hero` renders the default 605ZZ crank lying on the arm's back face with the handle standing up, in the
hand-crank siren's crank colour with a dark handle. The `side` and `back` presets show it from the side and from below.
`cutaway` clips every body by the plane through the shaft axis and the pin axis, removes the half toward +X, and looks
from that side. `exploded` pulls the parts along the pin axis: the bolt 18 mm down, the nut 16 mm, the handle 32 mm and
the pin 75 mm up; each of the two clamp bolts goes 30 mm out of the arm's +X side and each clamp nut 45 mm out of the
-X side, far enough to clear the arm's outline in the view. `sizes` puts the 608-2RS crank 15 mm beside the default one. The `readme` shot builds each crank once
and writes all of them, the index image and one 480 × 270 image of each part in `parts/`. The `clamp` and `clamp-nut` views show both clamp bolts and nuts, aimed at z = 8 between them and framed from z = -1
to the hub's top. `clamp-section` cuts the arm
across the clamp nuts' pockets at x = -7, and `cross-section` cuts the pin across the first cross nut's pocket at
y = Throw - 3; both look into the cut:
`go -C tools/render run . -model crankhandle -shot readme -out ../../models/crankhandle/images`.

`plaquestand/hero` renders the default stand, blue, with the default plaque, grey, leaning in its slot, seen from the
front left and above. The `stand` preset shows the stand alone, `side` shows both from +X, and `back` shows both from
behind and above. `exploded` lifts the plaque out of the slot by the back rest's height plus 40 mm along the slot axis.
`print` turns the stand onto its x = 0 face, the way it prints. `sizes` stands the smallest (150 × 120 × 2 mm), the default
and the largest (400 × 350 × 15 mm) plaque with their stands left to right, 40 mm between the plaques. `part-stand` and
`part-plaque` show each part alone. The `readme` shot builds the default stand once and the smallest and largest once each,
and writes every image, the index image and the two 480 × 270 part images in `parts/`:
`go -C tools/render run . -model plaquestand -shot readme -out ../../models/plaquestand/images`.

`onewayclutch/hero` renders the default clutch with every body cut by the plane z = 9.25, the hub's mid-height, and the
+Z side removed, seen from the front and above, so the pawl heads in the ring's teeth, the arms and the back plate show, with the ring's
three ears outside them. The `front`, `back` and `side` presets show the whole clutch. `section` looks straight down the shaft axis onto
the same cut, and `cutaway` cuts every body by the plane x = 0 and looks from +X. `exploded` pulls the parts apart along
the shaft axis, with the axis running left to right on the screen: the nuts 30 mm below the back plate's hex pockets, then the hub 30 mm,
the ring 60 mm, the cover 90 mm and the bolts 120 mm above it. `adapter` draws the default clutch with the standard adapter
ring (`BuildAdapter`) bolted to its front, from the front, right and above; `adapter-exploded` pulls that assembly apart the
same way and adds the adapter 120 mm and the clutch's bolts and the adapter's bolts 150 mm above the back plate, with the
adapter's nuts 30 mm below it, like the clutch's own. `directions` puts the clockwise and counter-clockwise
sections side by side, and `sizes` puts the sections of the three shaft presets side by side. The `readme` shot builds each
clutch once, writes all of them, the index image and one 480 x 270 image of each distinct part in `parts/`, including `adapter` and `mount-bolt`:
`go -C tools/render run . -model onewayclutch -shot readme -out ../../models/onewayclutch/images`.

`turbinefan/hero` renders the default right-hand fan (192 mm rotor) from the front, right and above, with blade 1 up, in the
hand-crank fan's hub and blade colours. Every offset below scales with the rotor: s is the blade scale (rotor diameter
/ 128) and j the hub scale (hub radius / 14, at most 1.35), and the numbers in brackets are for 128 mm. The `front`, `back` and `side` presets show other angles. `setscrew` clips the hub, the blade
bolts and nuts and the set screw and its nut by the plane z = 4 and looks from the front; `joint` clips the hub, blade 1, its
bolt and nut and the set screw and its nut by the plane x = 0 and looks from +X. `exploded` pulls each blade 35 s mm out along
its radial line (52.5 mm at 192), each blade bolt its length + 14 mm out of the front face (39 mm; 30 mm at 128), each blade nut
18 j mm along its radial line (24.3 mm), the set screw 32 j mm along -Y (43.2 mm) and its nut 14 mm out of the back face; `exploded-back` shows the same offsets from behind, where the nuts are
visible. `sizes` puts the 608ZZ hub beside the 605ZZ hub, 2 hub radii + 12 mm apart (49.8 mm; 40 mm at 128), from behind. `setscrew`
clips to a box of hub radius + 6 mm, and `joint` frames the hub from hub radius + 11 mm behind blade 1's line to hub radius +
16 mm in front of it. `turbinefan/clutch`, `clutch-back`,
`clutch-cutaway` and `clutch-exploded` draw the clutch-mount fan (`ParamsForClutch` of the default clutch) on the real default
one-way clutch with its adapter ring, both in the clutch frame: `clutch` from the front, right and above, `clutch-back` from behind, `clutch-cutaway`
cuts every body by the plane through the axis and the 330 degree direction, which holds ear 3, adapter bolt 2 and the
clutch's 150 degree bolt, removes the half toward 60 degrees and looks from there. `clutch-exploded` runs the shaft axis left to right and pulls the parts apart along it: the ear nuts
30 mm behind the clutch (also 14 mm out along their ear), the clutch in place, the adapter 20 mm ahead, the adapter bolts 6 mm
ahead (between the clutch and the adapter), the hub and blades 40 s mm ahead (60 mm at 192; each blade 35 s mm and each blade nut
18 j mm out along its radial line, each adapter nut 18 mm out along its line), the blade bolts 40 s + their length + 14 mm
(99 mm at 192; 70 mm at 128) and the ear bolts the larger of 110 mm and that + 40 mm ahead (139 mm at 192). The
`readme` shot writes all of them, the index image and one 480 × 270 image of each part in `parts/`, including `adapter-bolt`
and `ear-bolt`:
`go -C tools/render run . -model turbinefan -shot readme -out ../../models/turbinefan/images`.

`pinring/hero` renders the default pin ring. The `top`, `side` and `low` presets show other angles.
The `readme` shot writes all of them and the index image.

From the repository root, regenerate the spur gear hero image with:

```sh
go -C tools/render run . -model spurgear -shot hero -out ../../models/spurgear/images/hero.png
```

Regenerate the helical gear hero from the verified 17-tooth model with:

```sh
go -C tools/render run . -model helicalgear -shot hero -out ../../models/helicalgear/images/hero.png
```

Regenerate the herringbone gear hero from its verified 17-tooth model with a
3 mm bore and 0.1 mm outer-cap chamfers:

```sh
go -C tools/render run . -model herringbonegear -shot hero -out ../../models/herringbonegear/images/hero.png
```

`-width` and `-height` override the preset's 1600 × 900 pixel size. Paths given
to `-out` are relative to `tools/render`. The renderer writes a temporary PNG
in the output directory and renames it after rendering succeeds.
For other spur gear angles, use `-shot top` or `-shot low` with the matching
filename in `models/spurgear/images`.

Animation output can use the same model scene builders when an animation preset is added.

The `bevelgear/hero` preset renders the 31/31 spiral pair at 35°, with a
post-join 7.75 mm bore in each member, beside a bored 32/16 straight-tooth pair at
Module 4. The preset enlarges the spiral pair fourfold so both pairs have the same
pitch diameter, and it moves the straight pair along X until the two bounding boxes
are 30 mm apart. Regenerate that image from
the repository root with:

```sh
go -C tools/render run . -model bevelgear -shot hero -out ../../models/bevelgear/images/hero.png
```

The `top`, `high`, and `low` presets show only the source-default spiral pair, at
its own size, from three more angles. The `straight-top`, `straight-high`, and
`straight-low` presets show only the 32/16 straight pair. Regenerate the README
images with:

```sh
go -C tools/render run . -model bevelgear -shot top -out ../../models/bevelgear/images/top.png
go -C tools/render run . -model bevelgear -shot high -out ../../models/bevelgear/images/high.png
go -C tools/render run . -model bevelgear -shot low -out ../../models/bevelgear/images/low.png
go -C tools/render run . -model bevelgear -shot straight-top -out ../../models/bevelgear/images/straight-top.png
go -C tools/render run . -model bevelgear -shot straight-high -out ../../models/bevelgear/images/straight-high.png
go -C tools/render run . -model bevelgear -shot straight-low -out ../../models/bevelgear/images/straight-low.png
```

The `bevelgear/exploded-spiral` and `bevelgear/exploded-straight` presets pull each pair apart along its
shafts, the driving gear along +X and the pinion along +Y, each by 0.4 of the pair's bounding box diagonal. The
`readme` shot writes them as 1200 × 1000 pixel `exploded-spiral.png` and `exploded-straight.png`, and writes
480 × 270 pixel `parts/spiral-gear.png`, `parts/straight-driving.png` and `parts/straight-pinion.png` for the
parts table.

The `readme` shot builds each model once and writes a 480 × 270 pixel
`index.png` for `models/README.md`. For helical and herringbone gears it also
writes 1600 × 900 pixel `top.png` and `low.png` views. For bevel gears it
writes a 1600 × 900 pixel `hero.png` and eight 1200 × 1000 pixel views (the two exploded views and `top.png`, `low.png`,
`high.png`, `straight-top.png`, `straight-low.png`, and `straight-high.png`) with outlines
enabled. The hero camera is shifted sideways until the left and right margins match. The hero
scene sets the edge tolerance to 0.12 mm for every image size, so the 480 × 270 pixel index
draws the same outlines as the hero. Each view's camera is shifted and its field of view is set
so the gears sit in the middle of the image with a 6% margin on every side:

```sh
go -C tools/render run . -model spurgear -shot readme -out ../../models/spurgear/images
go -C tools/render run . -model helicalgear -shot readme -out ../../models/helicalgear/images
go -C tools/render run . -model herringbonegear -shot readme -out ../../models/herringbonegear/images
go -C tools/render run . -model bevelgear -shot readme -out ../../models/bevelgear/images
```

The `cycloidal/hero` preset renders the assembled two-disc drive. The `cycloidal/exploded` preset pulls the
housing, both rotor discs, cam, plate and pins apart along the drive axis in Decad's derived explode mode (`parts.Set.Explode`, with the
housing as ground). The `readme` shot writes the
480 × 270 pixel `index.png`, the 1600 × 900 pixel `hero.png` and `exploded.png`, three 1200 × 675 pixel
assembled views (`assembled-output.png`, `assembled-input.png`, `assembled-side.png`), and a 480 × 270 pixel
image in `parts/` for every named body:

```sh
go -C tools/render run . -model cycloidal -shot readme -out ../../models/cycloidal/images
```

The `planetarygearbox/hero` preset renders the default two-stage gearbox fully assembled. The ring and both covers
are cut open with solidlens `Model.Clip` along the plane through the axis that holds the X axis, removing the half
toward negative Y, and the camera looks from that side; every other part is drawn whole. The `readme` shot writes
the 480 × 270 pixel `index.png` and 1600 × 900 pixel `hero.png` from one build. It also writes one 480 × 270 pixel
image of each unique part in `images/parts/`: the ring, one cover, the sun, one planet, one bearing, the input
shaft, the carrier, the carrier pin, the link shaft, the output shaft and one collar ring. It also writes three 1200 × 675 pixel views of the whole two-stage
gearbox with nothing cut, `assembled-output.png`, `assembled-input.png` and `assembled-side.png`, which the
`planetarygearbox/assembled-output`, `assembled-input` and `assembled-side` presets render alone. The `planetarygearbox/fasteners-section` and `fasteners-exploded` presets cut a one-stage housing along the plane through
the first bolt of each cover and draw its bolts, nuts, covers and ring in place and pulled apart along the bolt.
The `planetarygearbox/ear-section` preset cuts the one-stage ring along the plane through the axis and the first
mounting ear, on +Y, and draws the ear's bolt and nut in place, from +X. The `readme` shot writes all three, the last
as `ear-section.png`. The `planetarygearbox/part-ring`, `part-cover`, `part-sun`,
`part-planet`, `part-bearing`, `part-carrier`, `part-carrier-pin`, `part-input-shaft`, `part-link-shaft`,
`part-output-shaft` and `part-collar` presets
render one of them alone. The shafts are seen from almost level with them so the flat and the collar or step show, and the
carrier from below so its through sockets show. A 1600 × 900 pixel exploded view, `exploded.png`, pulls the
whole two-stage gearbox apart along its axis with nothing cut and without its bolts and nuts, the ears' included. The offsets come from
`parts.Set.Explode` over `Gearbox.Set` in Decad's derived mode, which `Gearbox.Explode` runs with the front cover held
in place. Both ends read bearing, cover, shaft from the outside in: each cover bearing sits outside its cover, beyond
its shaft's free end, and each shaft sits on the inner side of its cover. The ring and the front bearing sit 20 mm
from the parts below them. The render also moves each planet, its bearing and its carrier pin 45 mm outward from the axis,
along that planet's own direction, so the three planets of a stage show side by side. The camera looks from nearly
level, 50° round the axis from the first planet, so no planet lines up with the sun and the carriers show edge on. The
`planetarygearbox/exploded` preset renders it alone.

The `motion` shot writes `images/motion.gif`, 960 × 320 pixels and 78 frames, of the two-stage gearbox in three
panels: the hero's cutaway, the input end from below and the output end from above, with a caption strip that reads
the input and output shaft angles. Each frame moves the tessellated bodies rigidly: the input shaft and first sun
turn by the input angle, each carrier by Zs/(Zs+Zr) of its sun's angle, the next sun with the carrier before it, and
each planet is carried around with its carrier while turning by -Zr/Zp of the carrier's angle about its own axis.
Each frame turns the input by half the sun's tooth pitch, so the sun's teeth do not seem to run backward. The 40
input angles play forward and then back, so there is no jump where the animation repeats; a one-way loop would need
the input to turn 7,056 times. All frames share one 255-color palette made from the colors the frames
use most, and each frame after the first stores only the box around the pixels that changed. The `readme` shot
writes it too.

```sh
go -C tools/render run . -model planetarygearbox -shot readme -out ../../models/planetarygearbox/images
go -C tools/render run . -model planetarygearbox -shot motion -out ../../models/planetarygearbox/images/motion.gif
```

The `-kit` flag renders a bearing kit of `planetarygearbox.SupportedCombinations()` instead of the default. It applies
only to `-model planetarygearbox` and any other model with `-kit` is an error. The single-preset shots
(`planetarygearbox/hero`, `assembled-*`, `part-*` and `exploded`) honour it. With `-shot readme` it builds one two-stage
gearbox and writes `hero.png` (1600 × 900), `assembled-output.png`, `assembled-input.png` and `assembled-side.png`
(1200 × 675), `exploded.png` (1600 × 900) and `parts/<part>.png` (480 × 270). It writes
no `index.png`, fastener images or `motion.gif`, because they do not depend on the bearings. A kit whose planet and
shaft bearings differ writes `parts/planet-bearing.png` and `parts/cover-bearing.png` in place of `parts/bearing.png`.
The `part-bearing` preset of such a kit shows the planet bearing.

```sh
go -C tools/render run . -model planetarygearbox -kit planet605-shaft608 -shot readme -out ../../models/planetarygearbox/images/kit-planet605-shaft608
go -C tools/render run . -model planetarygearbox -kit planet608-shaft605 -shot readme -out ../../models/planetarygearbox/images/kit-planet608-shaft605
go -C tools/render run . -model planetarygearbox -kit planet608-shaft608 -shot readme -out ../../models/planetarygearbox/images/kit-planet608-shaft608
```

The `screwgear` presets build the default mechanism once (about 40 seconds and 1.3 GB) and render its three bodies
as separate meshes: ribbon A in blue, ribbon B in orange and the sleeve in grey, each with outlines. The camera frames
the sleeve and the 70 mm of ribbon around it, so the 178.5 mm ribbons run off the image edges. The `hero`, `top` and
`low` presets write one image each; the `readme` shot writes `index.png`, `hero.png`, `top.png`, `low.png`,
`exploded.png`, `cutaway.png` and the 480 × 270 pixel `parts/ribbon-a.png`, `ribbon-b.png` and `sleeve.png` from one
build. The `screwgear/exploded` preset pulls each ribbon 115 mm out along its own axis and lifts the two 40 mm apart
along the sleeve's axis, framed on every vertex, and renders alone. The ribbons are meshed at 0.1 mm and the sleeve at 0.2 mm, the finest chord
Decad accepts for it:

```sh
go -C tools/render run . -model screwgear -shot hero -out ../../models/screwgear/images/hero.png
go -C tools/render run . -model screwgear -shot readme -out ../../models/screwgear/images
```

The `cutaway` preset cuts all three bodies open along the world XZ plane, which holds the sleeve's axis and the
reference direction ê, and removes the half toward negative Y. The camera looks from that side, so the view shows the
inside of the sleeve and the two ribbons meshing in its hollow. It sets solidlens `Model.Clip` on each model, with a
lighter, unlit `Cap` material in each body's colour so the cut faces read as flat solid faces. The `readme` shot also
writes `cutaway.png`, and the preset can be rendered on its own:

```sh
go -C tools/render run . -model screwgear -shot cutaway -out ../../models/screwgear/images/cutaway.png
```

The `escapement/hero` preset renders the default Graham escapement at its default Phase 0.25, with the pendulum at rest, from the
front and a little above and to the right. The front plate is in the picture, and the wheel, anchor and drive show
through its windows. The `readme` shot builds the movement once and writes the 480 × 270 pixel `index.png`, the 1600 × 900 pixel
`hero.png`, `exploded.png`, and the 1200 × 675 pixel `assembled-front.png`, `assembled-angle.png`,
`assembled-side.png`, `closeup.png` and the 1000 × 1000 pixel `assembled-back.png` (from behind, through the back plate's windows, with the lights turned to shine on its back faces), and a 480 × 270 pixel image in `parts/` for each of 20 kinds of body. The
`escapement/closeup` preset is a near-orthographic front view at Phase 0, where the entry pallet holds a tooth, of the pallets and the
teeth between them with the front plate, the crutch, the crutch collar, the knob collar, the pendulum and the other parts in front of the anchor left out. The `escapement/exploded` preset
renders the layout that `Movement.Explode` takes from Decad's derived explode mode (`parts.Set.Explode`), seen from
`(-1, 0, 1.2)`. Each part stays on its own axis, and a layer steps sideways only where a part it joins would hide it.
The preset draws a dotted line from each part of 13 mm or more that left its own line back to the part it joins, from
`Exploded.Guides`. `TestEscapementExploded` checks that no two parts overlap, that the layout stays under 150 × 250 ×
350 mm, and that a part joined to another and covered by it from that camera lies on the other part's axis. The `part-back-plate`, `part-front-plate`, `part-escape-wheel`, `part-anchor`, `part-crutch`,
`part-pendulum`, `part-suspension-pin`, `part-suspension-cap`, `part-great-wheel`, `part-pawl-carrier`, `part-mainspring`, `part-spring-retainer`, `part-spring-anchor-pin`, `part-winding-knob`,
`part-escape-arbor`, `part-pallet-arbor`, `part-great-wheel-arbor`, `part-end-collar`, `part-bearing` and `part-bolt` presets
render one of them alone, the bolt with its nut.

The `motion` shot writes `images/motion.gif`, 960 × 540 pixels and 36 frames, of one pendulum period. Frame `k` is Phase
`k/36`. The build is made once at the default phase and each frame turns the tessellated wheel and arbor, the anchor, pallet arbor, crutch and crutch collar, and the pendulum and its bearing by the change in the angle `Motion` gives, about their own axes. Both plates stay in, because they are fixed.
The front bearings, the bolts and the drive train are left out, because the drive does not repeat in one period. The wheel
advances one tooth in the loop, so the loop has no jump. A readout under the picture gives the pendulum, anchor and wheel
angles.

```sh
go -C tools/render run . -model escapement -shot readme -out ../../models/escapement/images
go -C tools/render run . -model escapement -shot motion -out ../../models/escapement/images/motion.gif
```

The `swisslever/hero` preset renders the default Swiss lever escapement at its default Phase 0.25, with the balance at
rest and the impulse pin on the line of centres, from the front and a little above and to the right. The front plate
is in the picture, and the wheel, lever and balance show through its windows. Every swisslever preset builds the
movement with `swisslever.Build`, the real build. The `readme` shot builds it once and writes the 480 × 270 pixel
`index.png`, the 1600 × 900 pixel `hero.png` and `exploded.png`, the 1200 × 675 pixel `assembled-front.png`,
`assembled-angle.png`, `assembled-side.png` and `closeup.png`, the 1000 × 1000 pixel `assembled-back.png` (from behind,
with the lights turned to shine on the back faces), and a 480 × 270 pixel image in `parts/` for each distinct body.

The `swisslever/closeup` preset is a near-orthographic view from behind, at Phase 0.25, of the pallets, the fork, the
dart, the safety roller and the impulse pin. The pin stands out of the rear face of the roller table, so only a view from
the rear shows it in the fork. The view leaves out the plates, the bearings, the nuts, the bolts, the balance wheel, the
hairspring, its retainer, the knob and the knob collar.

The `swisslever/exploded` preset pulls the movement apart along its depth by fixed offsets, seen from `(1, 0.3, 0.9)`.
The three arbors stay where they are. Every later layer moves forward until it lies in a slab of its own, 16 mm in front
of the layer behind it, and a dotted line joins each part of 13 mm or more back to the place it assembles. The layers,
from the back, are the three arbors; the three back bearings; the back plate; the four nuts; the entry stone with the
safety roller; the exit stone; the lever; the escape wheel with the roller table; the balance wheel; the hairspring;
the hairspring retainer; the three front bearings; the front plate; the four bolts; the knob collar; and the winding
knob. This order is the assembly order for the README's alt text. `TestSwissExploded` checks that every body has an
offset, that the layers keep their gap, that no two bodies overlap or touch, and that the exploded parts fill at least
80% of the image width or height. It skips while `swisslever.Build` is blocked by decad. `TestSwissPresets` checks the
image sizes and that the layers and the parts table name every body once, and `TestSwissMovers` checks the animation's
turn angles. Neither needs a build.

The `part-back-plate`, `part-front-plate`, `part-bearing`, `part-escape-arbor`, `part-pallet-arbor`,
`part-escape-wheel`, `part-lever`, `part-entry-stone`, `part-exit-stone`, `part-safety-roller`, `part-roller-table`,
`part-balance-wheel`, `part-hairspring`, `part-hairspring-retainer`, `part-winding-knob`, `part-knob-collar`,
`part-bolt` and `part-nut` presets render one of them alone. The six bearings, the four bolts and the four nuts are
identical, and the Balance Staff is the Pallet Arbor built twice, so `part-pallet-arbor` stands for both.

The `motion` shot writes `images/motion.gif`, 960 × 540 pixels and 36 frames, of one balance period. Frame `k` is Phase
`k/36`. The build is made once at the default phase, and each frame turns the tessellated escape wheel, escape arbor,
winding knob and knob collar by the change in the wheel angle, the lever and pallet arbor and both stones by the change
in the lever angle, and the balance staff, safety roller, roller table, balance wheel and hairspring retainer by the
change in the balance angle, each about its own axis. The change is measured from the pose `swisslever.Build` backs off
the last contact (`BuildBackoff`). The hairspring stays relaxed. The front plate, the front bearings, the bolts, the
knob and the knob collar are left out so the mechanism shows. A readout under the picture gives the balance, lever and
wheel angles.

Run these once `swisslever.Build` is accepted by decad. Each command builds the movement, which takes about a minute:

```sh
go -C tools/render run . -model swisslever -shot readme -out ../../models/swisslever/images
go -C tools/render run . -model swisslever -shot motion -out ../../models/swisslever/images/motion.gif
go -C tools/render run . -model gallery -shot hero -out ../../docs/images/gallery.png
go -C tools/render test -count=1 -run 'TestSwiss' .
```

A single preset renders with `-model swisslever -shot <name> -out <file>`, for example
`-shot exploded -out ../../models/swisslever/images/exploded.png`.

The `gallery/hero` preset does not include the ball bearing or screw gear.

The `gallery/hero` preset writes the repository's hero image, `docs/images/gallery.png`. It renders the hero
scenes of the spinning wheel, the spur, helical and herringbone gears, the planetary gearbox, the cycloidal
drive, the bevel gear pair, the hand-crank fan, the hand-crank siren, the Graham escapement, the Swiss lever escapement, the crank handle, the
one-way clutch, the spiral spring, the turbine fan and the plaque stand one after another. On a cache miss, it renders each model twice,
once to measure the pixels it draws and once at the scale that fits them to a cell of the 1600 × 900 pixel canvas,
so every model fills its cell alike. The wheel takes a tall cell on the left
(560 pixels wide). The other fifteen models fill a grid beside it: five columns across the top (208 × 300 pixels each)
and two rows of five below (208 × 300 pixels each). The three gears, the bevel gear pair and the plaque stand are across the top. The
planetary gearbox, cycloidal drive, fan, siren and Graham escapement are in the middle, and the Swiss lever escapement,
crank handle, one-way clutch, spiral spring and turbine fan are across the bottom. Building all sixteen models takes
several minutes plus the wheel
build:

```sh
go -C tools/render run . -model gallery -shot hero -out ../../docs/images/gallery.png
```

The renderer stores finished tiles in `.tmp/gallery-cache/` in the checkout. It invalidates every tile when any
renderer or dependency source, module manifest, toolchain or build setting changes. Changing the gallery dimensions
invalidates the tiles too. It skips the cache when `go list` cannot discover the dependencies or the running executable
does not match the current renderer build. Add `-gallery-fresh` to rebuild every tile and update the cache. Deleting
`.tmp/gallery-cache/` also forces a rebuild.

The `handcranksiren/hero` preset builds the default siren once and renders it from the front, left and above (camera at (-250, -650, 330)), with each
part coloured by its group. Operator forward is -X and the crank is on the right (+Y). The `handcranksiren/view-rear`,
`view-side` and `view-top` presets render the whole siren from behind the operator, from the right and from above, with
forward at the top (1200 × 675), and `TestHandCrankSirenView` renders one of them. The `handcranksiren/cutaway` preset
clips the drum, stator, collector and horn by the plane Z = 84 with the top half removed, so the rotor, vanes, ports,
collector cavity and horn bore show. The `handcranksiren/exploded` preset pulls the parts apart by fixed offsets, seen
from behind the operator, on the left and above (camera at (650, -450, 330)): the collector and its nuts and the throat bolts by -135 in Y, the horn and the throat
nuts by -135 in Y and -60 in X, the stator -95, the rotor -65, the grip and its bolts -45, the drum and its bolts and
the collector bolts -35, the front bearings +20, the plane-2 gears +40, the plane-1 gears +60, the frame and the gearbox nuts +85, the rear bearings +110,
the rear plate and its bolts +135 (the shafts lift 160 in Z and the front bolts move -25 in Y). The crank moves +230,
clear of the rear plate, and comes apart from there along the pin axis as the `crankhandle/exploded` preset pulls it:
the pin bolt 18 back, the pin nut 16 and the handle 32 out, and the pin 95 out, clear of the handle. The cross bolts go
with the pin and 14 further along the arm, their nuts 14 back toward the shaft. Each clamp bolt pulls 30 out of the
arm's side face and each clamp nut 25 the other way along its line, past the far jaw, where the arm does not hide it.
The `readme` shot builds the siren once and writes `hero.png`,
`index.png` (480 × 270), `exploded.png`, `cutaway.png`, `view-rear.png`, `view-side.png`, `view-top.png` and one
480 × 270 image per unique part (28) in `parts/`:

```sh
go -C tools/render run . -model handcranksiren -shot readme -out ../../models/handcranksiren/images
```

The `handcrankfan/hero` preset builds the default fan once and renders it from the operator's side, with each part
coloured by its group. The `handcrankfan/exploded` preset renders the same fan pulled apart by Decad's `parts.Set.Explode`, with the
offsets it computes from the model's joints. The `handcrankfan/view-crank`, `view-rear` and `view-below` presets render
the whole fan from the crank side, from behind and above, and from below (1200 × 675); `TestHandCrankFanView` renders
one of them. The `readme` shot builds the fan once and writes `hero.png` (1600 × 900),
`index.png` (480 × 270), `exploded.png` (1600 × 900) and the three views. It also writes one 480 × 270 pixel image of each unique part in
`images/parts/`, which the README's parts table shows: the gearbox, the grip, a grip screw, the three gears and the
pinion, the three shafts, a bearing, the crank arm, pin, pin bolt and handle, the hub, a blade, a blade bolt and the set
screw. The bolt images (`gearbox-bolt`, `gearbox-screw`, `grip-screw`, `crank-pin-screw`, `fan-bolt` and `fan-set-screw`) draw the bolt with its nut,
which the part row names in its `with` field. The
`handcrankfan/part-gearbox`, `part-grip`, `part-grip-screw`, `part-crank-gear`, `part-drive-gear`, `part-driving-gear`,
`part-pinion`, `part-crank-shaft`, `part-drive-shaft`, `part-fan-shaft`, `part-bearing`, `part-crank-arm`,
`part-crank-pin`, `part-crank-pin-screw`, `part-crank-handle`, `part-fan-hub`, `part-fan-blade`, `part-fan-bolt` and
`part-fan-set-screw` presets render one of them alone, each from its own side:

```sh
go -C tools/render run . -model handcrankfan -shot readme -out ../../models/handcrankfan/images
```

The `animation` shot writes `motion.gif`: 72 frames of one crank turn, 1000 × 560, in two panels. The left panel
shows the crank side and the right panel the spur and bevel gears from above and behind, with the crank arm left out. It builds the fan once and meshes each part
from its real body. Each frame turns the crank parts about the crank axis by the crank angle, the drive shaft and its gears the other
way round, and the fan parts about the fan axis by twice the crank angle, so the fan turns twice and the loop has no jump. The 5° step keeps the 32-tooth gear's
11.25° tooth pitch from seeming to run backward. `-frames` shortens it, which `TestHandCrankFanMotion` uses:

```sh
go -C tools/render run . -model handcrankfan -shot animation -out ../../models/handcrankfan/images/motion.gif
```
