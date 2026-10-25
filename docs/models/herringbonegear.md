# Herringbone gear package

The `models/herringbonegear` package builds the gear that the
[herringbone gear gallery page](../../models/herringbonegear/) shows. Its contract is in
[`instructions.md`](../../models/herringbonegear/instructions.md) and its
[steps](../../models/herringbonegear/steps.md) map that contract to Sketch and Decad.

## Using the package

`Build(ctx, doc, Params)` lofts an involute tooth to a twisted section at the body midpoint, mirrors
that half, and joins the two halves. It patterns the chevron tooth around a full-thickness root disc,
then applies the inherited helical root fillet and the optional spur bore and outer-cap chamfer. The
gallery accepts positive helix angles from 1° through 45°.

Set `Params.Placement` (an `r3.Frame`) to anchor the gear. The frame origin is the reference point on the
gear axis, its U and V axes span the reference plane that holds the lower cap, and the gear grows along
U×V. A nil `Placement` keeps the world XY plane.

## Checks

Run `go test ./models/herringbonegear` to check the supported parameter cases. The hero image shows the
default 17-tooth gear with its 3 mm bore and 0.1 mm cap chamfers.
