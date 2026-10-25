# Helical gear package

The `models/helicalgear` package builds the gear that the
[helical gear gallery page](../../models/helicalgear/) shows. It follows the Fusion add-in's
helical instructions, recorded in [`instructions.md`](../../models/helicalgear/instructions.md), and
reuses the spur involute tooth calculation from [`models/spurgear`](spurgear.md).

## Using the package

`Build(ctx, doc, Params)` takes a decad document and adds one gear body to it. The helix angle is the
whole twist across the thickness. `Build` requires a twist angle from 1° through 45°; use
`spurgear.Build` for zero twist.

Set `Params.Placement` (an `r3.Frame`) to anchor the gear. The frame origin is the reference point on the
gear axis, its U and V axes span the reference plane that holds the lower cap, and the gear grows along
U×V. A nil `Placement` keeps the world XY plane.

## Checks

Run `go test ./models/helicalgear` to check the supported parameter cases. The test file lists the
slower real-body checks and the command for each.

The [renderer](../rendering.md) builds the hero image from the 17-tooth gear with a 3 mm bore and 0.1 mm
outer-cap chamfers. The bore matches the spur gear hero's diameter on the same Module 1, 17-tooth outline.
