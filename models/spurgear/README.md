# Spur gear

![The Decad spur gear rendered at an angle](images/hero.png)

## More views

| From above | Low side angle |
| --- | --- |
| ![Top face, tooth outline, and centered bore](images/top.png) | ![Tooth flanks and gear thickness from a low angle](images/low.png) |

This item builds one straight-toothed involute gear. Its source is
[instructions.md](instructions.md). The [steps](steps.md) map the contract to
Sketch and Decad operations; `spurgear.go` constructs the model with
decad. Dimensions are in millimeters, and the pressure angle is in radians.

The Go package exposes `Build(ctx, doc, Params)`. Its caller supplies a decad
document and receives one gear body. Set `Params.Placement` (an `r3.Frame`) to anchor the gear: the frame origin is
the reference point on the gear axis, its U and V axes span the reference
plane that holds the lower cap, and the gear grows along U×V. Leaving it nil
places the gear on the world XY plane at the origin. Helical gears take the
same placement, and so do herringbone and bevel gears. Run `go test ./models/spurgear` to check
the supported parameter cases.

`Build` extrudes a root disc and extrudes one tooth section with fit-spline
flanks, repeats that tooth `N` times with a circular pattern about the gear
axis, and joins the disc and the teeth in one union. For a gear whose flanks
begin inside the root circle, the tooth is the natural involute tooth closed by
an arc just inside the root circle, and its flanks cross the root circle in the
union. It then asks decad to fillet the axial root corners.
The default model uses 15 fit points per flank and has 17 teeth and 34 root
fillets.

`BoreDiameter` cuts a centered through-bore after the root fillets.
`CapChamfer` chamfers the outer loops of both end caps, with or without a
bore. When present, the bore rims stay sharp. When a flank begins inside the
root circle, Sketch trims its original fit spline and joins the gear regions
into one profile. A nonpositive derived fillet radius skips root fillets.

The [model renderer](../../tools/render/README.md) produces these images
from a 17-tooth, 5 mm thick gear with a 3 mm bore and a 0.1 mm cap chamfer.
