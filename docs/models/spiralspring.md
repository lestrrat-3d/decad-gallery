# Spiral spring

The model in `models/spiralspring` constructs a flat spiral spring as one solid.

## Construction and API

`Params` holds the relaxed outer and hub diameters, the strip thickness and height, the smallest gap between
turns, an optional turn count, the arbor and anchor pin sizes with their clearances, and an optional `Placement`
frame. `DefaultParams()` returns the defaults listed in the model README. `Params.Validate()` returns the first
failure in the order that `instructions.md` lists. `Params.Derive()` validates and returns `Dimensions`: the turn
count, pitch, gap, bore and eye sizes, tail length, fillet radius, taper offset, `ActiveLength` and `WindStrain`.

`Build(ctx, doc, params)` returns a `Model` with `Body`, the `Profile` sketch and `Dimensions`. It adds exactly one
body to `doc`. A rejected build leaves `doc` without bodies.

The profile is one fully constrained XY sketch of lines and circular arcs. Its coil is a two-center arc spiral:
half-turn arcs alternate between the axis and a point half a pitch away, so the strip's two edges are concentric
arcs and the gap between turns is the same everywhere. Half turn 0's inner edge is centered at `(d + e, 0)` with
radius `rho0 - e`, so the strip tapers from `t + 2e` at the spoke to `t`. Build extrudes the sketch, rounds five
vertical concave corners with a 3D fillet of radius `min(t/2, g/2)`, chamfers the cap edges as the surface edge finishing rule requires, and applies the
placement.

A spring whose cap is under 400 mm^2 gets no cap chamfer. A larger one chamfers the outer loop of each cap, and
the bore loop and eye-hole loop of a cap when their edge run is 20 mm or longer.

## Build and images

```sh
go test ./models/spiralspring
go -C tools/render run . -model spiralspring -shot readme -out ../../models/spiralspring/images
```

The readme shot writes `hero.png`, `index.png`, `top.png`, `low.png`, `root.png` and `sizes.png`. The presets
`spiralspring/hero`, `top`, `low`, `root` and `sizes` render one view each.

## Verification

Tests check the derived dimensions, validation, and, for the default and seven other parameter branches, the
profile's full constraint, the bounds, the face count and the volume. They mesh every branch but the 40-turn one
with `VerifyAll`. They read the bore and eye-hole radii from the built faces and require the 0.2 mm and 0.35 mm
fit clearances. They slice the mesh at mid-height and require every air gap between turns to equal the derived gap
and every wall to equal the thickness, within the mesh tolerance. The gap between the hub and half turn 0 is
required to be at least `g - 2e`, the root thickness `t + 2e` at the spoke, and the `cornercheck` result empty.

## Known decad limit

After the cap chamfer, `Mesh.VolumeVerified()` is false. Decad publishes no occupied-volume proof for a chamfer
of a cap loop with circular walls. `BoundaryVerified()` stays true, and an unchamfered spring verifies both.
The chamfered mesh's `Bound()` is about 0.8 mm at a requested 0.02 mm, from the chamfer bands. The gap test
slices the walls at mid-height, where they are vertical cylinders and planes, and allows the 0.02 mm chord
tolerance plus 0.001 mm instead of the certified bound.
