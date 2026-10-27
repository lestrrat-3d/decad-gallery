# Screw gear package

The `models/screwgear` package builds the mechanism that the
[screw gear gallery page](../../models/screwgear/) shows. Its source is
[`instructions.md`](../../models/screwgear/instructions.md), and the
[steps](../../models/screwgear/steps.md) map that contract to Sketch and Decad operations. The geometry and the
numbers follow the Fusion add-in's screw gear specification. The add-in's bore identification marks and print
diagnostics are not part of this model. Lengths are in millimeters and angles are in radians.

## Using the package

`NewDocument()` returns a decad document that the sleeve's chain of six cuts needs. `Build` with a document made
by plain `decad.New` can fail at a window cut that meets a bore rim. `NewDocument` sets
`decad.WithChainTolerance` to 0.1 mm and applies any options you pass after it.

`Build(ctx, doc, Params)` adds three bodies to the document and returns them as a `*Mechanism` with fields `GearA`,
`GearB` and `Cage`. `DefaultParams()` returns the default 68-tooth mechanism. The table in `instructions.md` lists
every field. The three bodies are separate solids. The two ribbons are not joined to the sleeve and are not
operands of any boolean.

`Params.Validate()` checks the inputs in the order `instructions.md` lists and returns the first failure, naming
the input and its bound. `Build` calls it first. `Params` also has methods for the derived values, such as
`Lambda`, `Beta`, `AxisOffset`, `Length`, `Travel`, `AxialWindow`, `SleeveInner`, `SleeveOuter` and `SleeveCut`.

Set `Params.Placement` (an `r3.Frame`) to anchor the mechanism. The frame origin is the centre `C` of the
mechanism, its U axis is the reference direction `ê`, and its normal `U × V` is the common perpendicular `n̂` of
the two gear axes, which is also the sleeve's axis. A nil `Placement` puts the mechanism on the world XY frame
at the origin, with the sleeve's axis along world Z. `Build` constructs everything in the unplaced frame and
moves the three bodies onto `Placement` last.

## Checks

Run `go test ./models/screwgear` to check the parameter cases. Building the default 68-tooth mechanism takes
about 40 s, and the build cases use 16 teeth, which takes about 15 s each. A long ribbon loft takes about 12 s and
1.3 GB of memory, so run the cases in sequence on a small machine. The tests check:

- the defaults, the derived values and every input check;
- the pair frame and the roof side of each bore;
- the wall separation of 5.310 mm and the window search numbers at the defaults;
- the default build's volumes and bounds, a placed build, and three build branches;
- the distance from the cage to each ribbon at the bores' crossings, measured with `DistanceToPoint`.

`Mechanism.SectionPair(ctx, height, tol)` cuts both ribbons with the plane `Mechanism.MidPlane(height)`, which is
parallel to the mid-plane between the gear axes and `height` millimeters from it along `n̂`. It returns each ribbon's
`decad.SectionOutline`, read from the body's verified mesh at chord tolerance `tol`, and `Clearance`, the
`decad.SectionOutline.Clearance` between the two outlines: the least distance, the closest pair of points, whether
the outlines overlap, and `Lower()`, the part of the distance decad proves. `Clearance` is nil when a ribbon misses
the plane, as a gear does 1 mm from the mid-plane, where gear A's teeth end. On the default pair, the planes from
−0.4 mm to +0.4 mm read a least gap of 0.64 to 0.75 mm, proven to at least 0.60 mm, with no overlap. Each section
takes about 30 ms once the mechanism is built.

The section tests hold the proven lower bound above 0.55 mm on five planes of the 16-tooth build, check every
outline point lies on the plane, and check a plane 1 mm toward gear B reads empty for gear A.

The tests do not run `decad.Verify` with clearances between the bodies. That call took 108 s on the 16-tooth
model, found no interference, measured 0.548 ± 0.027 mm between the two ribbons, and left both ribbon-versus-cage
pairs undecided.

The package needs the decad revision `v0.0.0-20261027045205-cfe5a7e9e746` or later, which has long twisted lofts,
twisted sweeps at any angle and off the axis, twisted-tool cuts and chained cuts (decad pull requests 2629 to 2634).

The [renderer](../rendering.md) builds the screw gear from `DefaultParams()` with `NewDocument()` to produce the
images on the gallery page. Its `cutaway` shot cuts the bodies open with solidlens `Model.Clip`. The `models/screwgear/images` files come from `tools/render`.
