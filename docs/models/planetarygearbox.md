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

Set `Params.Stages` to stack more stages in the same ring. Each stage's shaft is a separate body: a link shaft
through the carrier's hole and the next stage's sun bore, and for the last stage the output shaft, which runs through the
rear cover's bearing and has a collar against it. Every shaft is
7.5 mm, with 0.25 mm radial clearance in the nominal 8 mm bearing bore. Each shaft has three keyway grooves, 120° apart, each `Params.KeyDepth` deep and `Params.KeyWidth` wide, where a sun or carrier sits, and
round where a bearing sits; the sun's bore and the carrier's hole are cut with a tool of the shaft's section less the
grooves, so each keeps three teeth and each shaft drives them. `Params.SeatChamfer` chamfers the entrance of each bearing pocket, in the planets and the covers, with
`decad.Chamfer` on the rim found by its radius and height. `Params.Ratio` returns the reduction with the ring held.
`Validate` applies the input checks, and `Build` calls it first.

`NewDocument()` returns a decad document with `decad.WithChainTolerance` set to 0.1 mm, and `Build` needs a document made
with it while the ring has mounting ears. A plain `decad.New()` document fails at the chamfer of the ring's end faces,
because the chamfer meets tooth-space facets held to about 0.0034 mm. With `EarCount = 0` a plain document works.

`EarCount` (default 3) and `EarBoreDiameter` (default 4 mm) set the mounting ears on the ring. The ears are spread
evenly, the first on +Y, and each runs the whole ring length with a bore parallel to the axis. The model adds no
bolt, head pocket or nut cavity for an ear; the bore is only a hole of the size you give. `Validate` rejects a negative
count, a bore that is not positive when there are ears, ears too close to each other, and an ear count that puts an ear
over one of the housing's bolt sites, such as 8.

`Backlash` thins every sun and planet tooth by that much at the pitch circle and passes the negative of it to the ring's tooth-space tool, so each meshing pair has twice `Backlash` of play. The default is 0.05 mm.

`Placement` (an `r3.Frame`) anchors the gearbox. The frame origin is a point on its axis and the first stage's gears
start on the frame's plane.

## How it is built

The sun and planets come from `spurgear.Build`. A planet is built with the lip bore, then a bearing pocket is cut from
its upper face. Each planet's bearing is `ballbearing.Build` with `Params608ZZ()`, kept as its own body in
`Stage.Bearings`. The carrier's plate is one outline, the convex hull of the pin bosses extruded once, and the keyed shaft hole is cut last.
One pin joined to its collar is patterned and unioned to the plate in a single step, because decad refuses a union
of every pin and collar separately. `buildShaft` joins a round rod and an optional collar, then cuts the three grooves. A cover is one extruded frame (a rim
with four bolt holes, three spokes and a hub, so its section has eight holes) joined to a bearing boss; the ring blank
has the same four bolt holes as inner loops of its section. A cover then has four round pockets cut for the M2 bolt heads,
and the ring blank has eight closed hex cavities cut for the M2 nuts, before the tooth spaces are cut. The bolts and nuts are built
once by `models/m2bolt` and `models/m2nut` and copied into place (`fasteners.go`). The ring is a cylinder, with three mounting ears when `EarCount` is above zero, and has the tooth spaces cut out by a 71-tooth
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

The [renderer](../rendering.md) draws the hero from a two-stage gearbox with the ring and covers cut open, three
assembled views of it, and an animation of it turning.
