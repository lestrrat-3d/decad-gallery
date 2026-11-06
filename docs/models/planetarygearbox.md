# Planetary gearbox package

The `models/planetarygearbox` package builds the gearbox that the
[planetary gearbox gallery page](../../models/planetarygearbox/) shows. Its source is
[`instructions.md`](../../models/planetarygearbox/instructions.md), and the
[steps](../../models/planetarygearbox/steps.md) map that contract to Sketch and Decad operations.
Lengths are in millimeters and angles are in radians.

## Using the package

`Build(ctx, doc, Params)` adds the gearbox's bodies to a decad document and returns a `Gearbox`. The housing is the
`Ring`, a `FrontCover` and a `RearCover`, each with a 608ZZ bearing (`FrontBearing`, `RearBearing`); the `InputShaft`
passes through the front bearing into the first sun. Each entry of `Stages` holds a `Sun`, the `Planets`, one
`Bearings` entry per planet and a `Carrier`. `Named` lists every body with a
display name. `DefaultParams()` returns the 13/29/71 gearbox with three planets and one stage.

Set `Params.Stages` to stack more stages in the same ring. Each carrier's stub shaft runs through the next stage's
sun bore, and the last carrier's stub runs through the rear cover's bearing and is the output shaft. Every shaft is
as wide as the bearing bore, 8 mm. `Params.Ratio` returns the reduction with the ring held.
`Validate` applies the input checks, and `Build` calls it first.

`Backlash` thins every sun and planet tooth by that much at the pitch circle and passes the negative of it to the ring's tooth-space tool, so each meshing pair has twice `Backlash` of play. The default is 0.05 mm.

`Placement` (an `r3.Frame`) anchors the gearbox. The frame origin is a point on its axis and the first stage's gears
start on the frame's plane.

## How it is built

The sun and planets come from `spurgear.Build`. A planet is built with the lip bore, then a bearing pocket is cut from
its upper face. Each planet's bearing is `ballbearing.Build` with `Params608ZZ()`, kept as its own body in
`Stage.Bearings`. The carrier's hub and arms are one 12-curve outline extruded once; its pins and stub shaft are
unioned to it, because decad refuses a union of separate hub and arm bodies. A cover is one extruded frame (a rim
with four bolt holes, three spokes and a hub, so its section has eight holes) joined to a bearing boss; the ring blank
has the same four bolt holes as inner loops of its section, so no boolean cuts them. The ring is a cylinder with the tooth spaces cut out by a 71-tooth
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

The [renderer](../rendering.md) draws the hero from a two-stage gearbox with the ring and covers cut open, and the
animation from a one-stage gearbox.
