# Planetary gearbox package

The `models/planetarygearbox` package builds the gearbox that the
[planetary gearbox gallery page](../../models/planetarygearbox/) shows. Its source is
[`instructions.md`](../../models/planetarygearbox/instructions.md), and the
[steps](../../models/planetarygearbox/steps.md) map that contract to Sketch and Decad operations.
Lengths are in millimeters and angles are in radians.

## Using the package

`Build(ctx, doc, Params)` adds the gearbox's bodies to a decad document and returns a `Gearbox`. The housing is the
`Ring`, a `FrontCover` and a `RearCover`, each with a shaft bearing (`FrontBearing`, `RearBearing`; 605ZZ by default); the `InputShaft`
passes through the front bearing into the first sun. Each entry of `Stages` holds a `Sun`, the `Planets`, one
`Bearings` entry per planet and a `Carrier`. `Named` lists every body with a
display name. `DefaultParams()` returns the 13/29/71 gearbox with three planets and one stage.

`Gearbox.Set(doc, params)` returns the build as a `parts.Set` with its materials and joints; it needs the document and
parameters `Build` used. `Gearbox.Explode(ctx, set)` pulls that set apart along its axis with `Set.Explode` in Decad's
derived mode (`parts.WithDerivedExplode`), keeping the front cover in place. Decad reads from the tessellated bodies
which parts block which and derives the removal order and directions, with one exception: a `parts.WithRemoval` hint
makes the input shaft leave toward the rear, after the ring, so the layout draws it on the inner side of the front
cover. `parts.WithPartGap` gives the ring and the front bearing a 20 mm gap in place of 6 mm. In the set the input and
output shafts turn in their cover bearings and each planet turns on its carrier
pin; every other joint is fixed. Each pin-in-bore joint declares its fit's radial clearance: zero for a shaft or pin
in a bearing bore and for a bolt in its nut, `BearingClearance` for a bearing in its seat, 0.05 mm for a part pressed
into a sun or carrier and 0.3 mm for a bolt in its hole. For one stage, `Set.Linkage` with the ring as ground gives
five turning links: the input shaft with its sun, the carrier with its pins and the output shaft, and each planet with
its bearing. With more stages, each carrier before the last turns in no bearing: only the gear meshes, which are
contact joints, hold it. `Set.Linkage` then leaves that carrier with its pins, its link shaft and the next sun, and the
carrier's planets, static.

`Params.PlanetBearing` and `Params.ShaftBearing` name the catalog bearing (`ballbearing.Lookup`) of each role, `"605ZZ"` or
`"608ZZ"`, and every dependent dimension follows. `Validate` rejects a name that is not in the catalog.

`Problems(p)` returns every way `p` is invalid, and `Validate` returns the first. A failure forced by a bearing says
what it needs: for a 608ZZ shaft on the default teeth it reports `SunTeeth must be at least 14 for the 8.1 mm shaft at
module 1; try 14/28/70`. It never changes teeth itself.

`SupportedCombinations()` lists four kits: every pair of 605ZZ and 608ZZ over the planet and shaft roles, named
`planet<NNN>-shaft<NNN>`. `DefaultParamsFor(kit)` returns `DefaultParams()` with that kit's bearings and a tooth set that
passes `Validate` (13/29/71, or 14/28/70 when a 608ZZ shaft needs a larger sun). It never adjusts teeth for you: an
unknown kit name is an error that lists the supported kits.

Set `Params.Stages` to stack more stages in the same ring. Each stage's shaft is a separate body: a link shaft
through the carrier's hole and the next stage's sun bore, and for the last stage the output shaft, which runs through the
rear cover's bearing. The link and output shafts are thin-thick: a D thick part of the shaft collar diameter, pressed
through the carrier's hole, and a thin rod of the shaft diameter. The rod is as wide as the shaft bearing's bore (5 mm for
the default 605ZZ) where it passes through a bearing, with no clearance. A carrier pin is as wide as the planet
bearing's bore and has a D thick part of the pin collar diameter.
The parameter table names each bearing by role: `planet_bearing_bore_diameter`, `planet_bearing_outer_diameter`,
`planet_bearing_width` and the same three with `shaft_bearing_`. It holds separate planet and cover pocket and lip bore
diameters (`planet_pocket_diameter`, `planet_lip_bore_diameter`, `cover_pocket_diameter`, `cover_lip_bore_diameter`) and
`pin_collar_diameter` for a pin's thick part.
Each shaft has a flat `Params.FlatDepth` deep, which gives it a D section, where a sun or carrier sits, and
round where a bearing sits; the sun's bore is cut with a tool of the shaft's D section grown by the fit clearance, and
the carrier's hole is the thick D grown by the press-fit clearance, so each keeps a tooth at the flat and each shaft
drives them. `Params.SeatChamfer` chamfers the entrance of each bearing pocket, in the planets and the covers, with
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
its upper face. Each planet's bearing is `ballbearing.Build` with the planet bearing's catalog `Spec.Params()`, kept as its own body in
`Stage.Bearings`. The carrier's plate is one outline, the convex hull of the pin bosses extruded once. A pattern of one D socket tool is cut through it, then the press-fit D centre hole. Each pin is its own body, `buildPin`: a rod and a D thick part joined in one union, built once and posed for every planet in `Stage.Pins`.
`buildSteppedShaft` joins a D thick part and a thin rod for the link and output shafts, then cuts the flat. `buildShaft` joins a round rod and a collar for the input shaft, then cuts the flat. A cover is one extruded frame (a rim
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

Run `go test ./models/planetarygearbox -timeout 60m`; it takes about 15 minutes and 3.6 GB, plus about 6 minutes for each kit that `TestKitGearboxes` builds. That test skips the two kits with a 608ZZ planet bearing until a Decad explode fix lands. `TestExplodedGearbox`
alone takes about 30 seconds. The tests build the default, two-stage and placed gearboxes, each once for all the
tests that read it, and check that no planet vertex lies inside the sun or the ring and that a planet turned half a
tooth does. The sketches are checked for full constraint inside the build.

The [renderer](../rendering.md) draws the hero from a two-stage gearbox with the ring and covers cut open, three
assembled views of it, and an animation of it turning.
