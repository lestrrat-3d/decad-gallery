# Spur gear package

The `models/spurgear` package builds the gear that the [spur gear gallery page](../../models/spurgear/)
shows. Its source is [`instructions.md`](../../models/spurgear/instructions.md), and the
[steps](../../models/spurgear/steps.md) map that contract to Sketch and Decad operations.
`spurgear.go` constructs the model with decad. Lengths are in millimeters and angles are in radians.

## Using the package

`Build(ctx, doc, Params)` takes a decad document and adds one gear body to it. `DefaultParams()` returns
Module 1, 17 teeth, a 20° pressure angle, 10 mm thickness and 15 fit points per flank. The table in
`instructions.md` lists every field.

`BoreDiameter` cuts a centered through-bore. `CapChamfer` chamfers the outer loops of both end caps,
with or without a bore, and leaves the bore rims sharp.

Set `Params.Placement` (an `r3.Frame`) to anchor the gear. The frame origin is the reference point on the
gear axis, its U and V axes span the reference plane that holds the lower cap, and the gear grows along
U×V. A nil `Placement` puts the gear on the world XY plane at the origin. The helical, herringbone and
bevel gear packages take the same placement.

## Checks

Run `go test ./models/spurgear` to check the supported parameter cases. How `Build` assembles the gear
is described in the doc comment on `buildLocal` in `variant.go`.

The [renderer](../rendering.md) produces the page's images from a 17-tooth, 5 mm thick gear with a 3 mm
bore and a 0.1 mm cap chamfer.
