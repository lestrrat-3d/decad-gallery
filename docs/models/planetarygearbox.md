# Planetary gearbox package

The `models/planetarygearbox` package builds the gearbox that the
[planetary gearbox gallery page](../../models/planetarygearbox/) shows. Its source is
[`instructions.md`](../../models/planetarygearbox/instructions.md), and the
[steps](../../models/planetarygearbox/steps.md) map that contract to Sketch and Decad operations.
Lengths are in millimeters and angles are in radians.

## Using the package

`Build(ctx, doc, Params)` adds the gearbox's bodies to a decad document and returns a `Gearbox`. Its `Ring` is the
housing, and each entry of `Stages` holds a `Sun`, the `Planets` and a `Carrier`. `Named` lists every body with a
display name. `DefaultParams()` returns the 13/29/71 gearbox with three planets and one stage.

Set `Params.Stages` to stack more stages in the same ring. Each carrier's stub shaft runs through the next stage's
sun bore, and the last carrier's stub is the output shaft. `Params.Ratio` returns the reduction with the ring held.
`Validate` applies the input checks, and `Build` calls it first.

`Backlash` thins every sun and planet tooth by that much at the pitch circle and passes the negative of it to the ring's tooth-space tool, so each meshing pair has twice `Backlash` of play. The default is 0.05 mm.

`Placement` (an `r3.Frame`) anchors the gearbox. The frame origin is a point on its axis and the first stage's gears
start on the frame's plane.

## How it is built

The sun and planets come from `spurgear.Build`. The ring is a cylinder with the tooth spaces cut out by a 71-tooth
spur gear whose root circle is the ring's tip circle and whose tip circle is the ring's root circle (`ringTool` in
`gears.go`). `phase.go` turns each planet, and the ring tool, so their teeth enter the sun's and the ring's gaps.
Each distinct part is built once at the origin and copied into place, because the spur union fails on gears built
at an offset.

`spurgear.Build` writes `teeth`, `thickness` and the other spur names into the document's parameter table each
time it runs, so those names hold the last gear built. The gearbox's own entries (`sun_teeth`, `orbit_radius` and
the others listed in `instructions.md`) are not overwritten.

## Checks

Run `go test ./models/planetarygearbox`; it takes about 90 seconds. The tests build the default, two-stage and
placed gearboxes, and check that no planet vertex lies inside the sun or the ring and that a planet turned half a
tooth does. The sketches are checked for full constraint inside the build.

The [renderer](../rendering.md) draws the page's image from a two-stage gearbox, pulled apart along its axis.
