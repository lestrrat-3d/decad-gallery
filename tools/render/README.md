# Model renderer

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

The `bevelgear/hero` preset renders the source-default 31/31 spiral pair,
with a post-join 7.75 mm bore in each member. Regenerate that image from
the repository root with:

```sh
go -C tools/render run . -model bevelgear -shot hero -out ../../models/bevelgear/images/hero.png
```

The `top`, `high`, and `low` presets use the same source-default pair and
show its tooth bands from three more angles. Regenerate their README images
with:

```sh
go -C tools/render run . -model bevelgear -shot top -out ../../models/bevelgear/images/top.png
go -C tools/render run . -model bevelgear -shot high -out ../../models/bevelgear/images/high.png
go -C tools/render run . -model bevelgear -shot low -out ../../models/bevelgear/images/low.png
```

The `readme` shot builds each model once and writes a 480 × 270 pixel
`index.png` for `models/README.md`. For helical and herringbone gears it also
writes 1600 × 900 pixel `top.png` and `low.png` views:

```sh
go -C tools/render run . -model spurgear -shot readme -out ../../models/spurgear/images
go -C tools/render run . -model helicalgear -shot readme -out ../../models/helicalgear/images
go -C tools/render run . -model herringbonegear -shot readme -out ../../models/herringbonegear/images
go -C tools/render run . -model bevelgear -shot readme -out ../../models/bevelgear/images
```
