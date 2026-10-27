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

The tests do not run `decad.Verify` with clearances between the bodies. That call took 108 s on the 16-tooth
model, found no interference, measured 0.548 ± 0.027 mm between the two ribbons, and left both ribbon-versus-cage
pairs undecided.

The package depends on decad capabilities that are on local branches and not upstream yet: long twisted lofts,
twisted sweeps at any angle, twisted-tool cuts, and chained cuts. The gallery `go.mod` has a local `replace` for
`github.com/lestrrat-3d/decad` that points at the `feat-screwgear-decad` worktree. Replace it with a pinned
version before merging.

The [renderer](../rendering.md) builds the screw gear from `DefaultParams()` with `NewDocument()` to produce the
images on the gallery page. The `models/screwgear/images` files come from `tools/render`.
