# Model renderer

This page documents `tools/render`.

This separate Go module builds gallery models with Decad and renders their
tessellated meshes with SolidLens. Each image preset names a model and a shot.
Add a scene builder and a preset entry in `main.go` for another model or shot.

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

Run `go -C tools/render test ./...` to build the actual spur gear and check that
the output PNG contains it. Animation output can use the same model scene
builders when an animation preset is added.

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

The `cycloidal/hero` preset renders the default two-disc drive pulled apart along a staircase: the housing,
both rotor discs, the eccentric cam, and the output plate with its pins. The `readme` shot writes its
480 × 270 pixel `index.png` and 1600 × 900 pixel `hero.png`:

```sh
go -C tools/render run . -model cycloidal -shot readme -out ../../models/cycloidal/images
```

The `screwgear` presets build the default mechanism once (about 40 seconds and 1.3 GB) and render its three bodies
as separate meshes: ribbon A in blue, ribbon B in orange and the sleeve in grey, each with outlines. The camera frames
the sleeve and the 70 mm of ribbon around it, so the 178.5 mm ribbons run off the image edges. The `hero`, `top` and
`low` presets write one image each; the `readme` shot writes `index.png`, `hero.png`, `top.png` and `low.png` from one
build. The ribbons are meshed at 0.1 mm and the sleeve at 0.2 mm, the finest chord Decad accepts for it:

```sh
go -C tools/render run . -model screwgear -shot hero -out ../../models/screwgear/images/hero.png
go -C tools/render run . -model screwgear -shot readme -out ../../models/screwgear/images
```

The `section` shot builds the default mechanism once and writes three 1200 × 900 pixel plane sections of the two
ribbons, `section-below.png`, `section-mid.png` and `section-above.png`, for planes −0.4 mm, 0 and +0.4 mm from the
mid-plane between the gear axes. It reads each outline with `Body.SectionOutline` at a 0.1 mm chord tolerance, draws
gear A in blue and gear B in orange with solidlens `RenderSection`, paints any overlap red, and puts the least gap
from `SectionOutline.Clearance` and its proven part in the caption:

```sh
go -C tools/render run . -model screwgear -shot section -out ../../models/screwgear/images
```

The `gallery/hero` preset does not include the screw gear.

The `gallery/hero` preset writes the repository's hero image, `docs/images/gallery.png`. It renders the hero
scenes of the spur, helical, herringbone, cycloidal and bevel gear models one after another, places each at
its own size on a 1600 × 900 pixel canvas, and keeps only the pixels that differ from the background where
the tiles overlap. Building all five models takes about six minutes:

```sh
go -C tools/render run . -model gallery -shot hero -out ../../docs/images/gallery.png
```
