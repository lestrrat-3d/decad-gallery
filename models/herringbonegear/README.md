# Herringbone gear

![A bored herringbone gear with opposing tooth slopes](images/hero.png)

## More views

| From above | Low side angle |
| --- | --- |
| ![Top face, tooth outline, and centered bore](images/top.png) | ![Mirrored tooth slopes meeting at the gear midpoint](images/low.png) |

`Build(ctx, doc, Params)` lofts an involute tooth to a twisted section at the
body midpoint, mirrors that half, and joins the two halves. It patterns the
chevron tooth around a full-thickness root disc, then applies the inherited
helical root fillet and optional spur bore and outer-cap chamfer.

Set `Params.Placement` (an `r3.Frame`) to anchor the gear: the frame origin is the
reference point on the gear axis, its U and V axes span the reference plane that
holds the lower cap, and the gear grows along U×V. `Build` constructs the gear at
the origin and moves the finished body onto the frame. A nil `Placement` keeps the world XY plane.

The [instructions](instructions.md) define the source parameters and operation
order. The [steps](steps.md) map them to Sketch and Decad.

The hero renders the default 17-tooth gear with its 3 mm bore and 0.1 mm cap chamfers.
The gallery accepts positive helix angles from 1° through 45°. Run
`go test ./models/herringbonegear` to check the supported parameter cases.
