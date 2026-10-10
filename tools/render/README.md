# Model renderer

This separate Go module builds gallery models with Decad and renders their
tessellated meshes with SolidLens. Each image preset names a model and a shot.
Add a scene builder and a preset entry in `main.go` for another model or shot.

From the repository root, regenerate the spur gear hero image with:

```sh
go -C tools/render run . -model spurgear -shot hero -out ../../models/spurgear/images/hero.png
```

`-width` and `-height` override the preset's 1600 × 900 pixel size. Paths given
to `-out` are relative to `tools/render`. The renderer writes a temporary PNG
in the output directory and renames it after rendering succeeds.

Run `go -C tools/render test ./...` to build the actual spur gear and check that
the output PNG contains it. Animation output can use the same model scene
builders when an animation preset is added.

The bevel gear has a `bevelgear/hero` preset. Regenerate its image from the
repository root with:

```sh
go -C tools/render run . -model bevelgear -shot hero -out ../../models/bevelgear/images/hero.png
```
