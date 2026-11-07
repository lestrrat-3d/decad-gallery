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
The ordinary `spinningwheel` preset still requires the complete assembly. Preview mode does not regenerate
the animation or establish retention, clearance or interference checks.

The `ballbearing/hero` preset renders the 608-2RS nominal mounting envelope from its Decad body.
The `readme` shot writes a 1600 × 900 hero and 480 × 270 index:

```sh
go -C tools/render run . -model ballbearing -shot readme -out ../../models/ballbearing/images
```

`ballbearing/605zz` renders the 5 × 14 × 5 mm 605ZZ envelope through the same builder.
The readme shot also writes `605zz.png` at 1600 × 900.

`m2bolt/hero` renders the default M2x10 bolt lying on its side with its head toward the camera.
The `head`, `side`, `underside` and `lengths` presets show other angles and all six standard lengths.
The `readme` shot writes all of them.

`m2nut/hero` renders the M2 nut. The `top`, `side`, `low` and `withbolt` presets show other angles and the nut on an M2x10 bolt.
The `readme` shot writes all of them.

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

The `readme` shot builds each model once and writes a 480 × 270 pixel
`index.png` for `models/README.md`. For helical and herringbone gears it also
writes 1600 × 900 pixel `top.png` and `low.png` views. For bevel gears it
writes a 1600 × 900 pixel `hero.png` and six 1200 × 1000 pixel views (`top.png`, `low.png`,
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
housing, both rotor discs, cam, plate and pins apart along a staircase. The `readme` shot writes the
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
shaft, the carrier, the link shaft and the output shaft. It also writes three 1200 × 675 pixel views of the whole two-stage
gearbox with nothing cut, `assembled-output.png`, `assembled-input.png` and `assembled-side.png`, which the
`planetarygearbox/assembled-output`, `assembled-input` and `assembled-side` presets render alone. The `planetarygearbox/fasteners-section` and `fasteners-exploded` presets cut a one-stage housing along the plane through
the first bolt of each cover and draw its bolts, nuts, covers and ring in place and pulled apart along the bolt.
The `readme` shot writes both. The `planetarygearbox/part-ring`, `part-cover`, `part-sun`,
`part-planet`, `part-bearing`, `part-carrier`, `part-input-shaft`, `part-link-shaft` and `part-output-shaft` presets
render one of them alone. The shafts are seen from almost level with them so the groove and the collar show, and the
carrier from below so the pins and their collars show. A 900 × 1200 pixel exploded view, `exploded.png`, pulls the
whole two-stage gearbox apart along its axis with nothing cut; the `planetarygearbox/exploded` preset renders it alone.

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

The `screwgear` presets build the default mechanism once (about 40 seconds and 1.3 GB) and render its three bodies
as separate meshes: ribbon A in blue, ribbon B in orange and the sleeve in grey, each with outlines. The camera frames
the sleeve and the 70 mm of ribbon around it, so the 178.5 mm ribbons run off the image edges. The `hero`, `top` and
`low` presets write one image each; the `readme` shot writes `index.png`, `hero.png`, `top.png`, `low.png` and
`cutaway.png` from one build. The ribbons are meshed at 0.1 mm and the sleeve at 0.2 mm, the finest chord
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

The `gallery/hero` preset does not include the ball bearing or screw gear.

The `gallery/hero` preset writes the repository's hero image, `docs/images/gallery.png`. It renders the hero
scenes of the spinning wheel, the spur, helical and herringbone gears, the planetary gearbox, the cycloidal
drive and the bevel gear pair one after another. It places each at its own size on a 1600 × 900 pixel canvas
(the wheel on the left, the gears on the right), and keeps only the pixels that differ from the background
where the tiles overlap. Building all seven models takes about six minutes plus the wheel build:

```sh
go -C tools/render run . -model gallery -shot hero -out ../../docs/images/gallery.png
```
