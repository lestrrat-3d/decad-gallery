# Planetary gearbox steps

These steps map [instructions.md](instructions.md) to Sketch and Decad operations. The instructions own
every dimension and rule; this file owns the call order, the exported API and the checks.

## Exported API

- `Params`, `DefaultParams()`, `(Params).Validate() error`, `(Params).Ratio() float64`.
- `Build(ctx, doc, Params) (Gearbox, error)`. It rejects a nil context or document, calls `Validate`, and wraps
  every later failure with the part it came from.
- `Gearbox{Ring *decad.Body; Stages []Stage}`, `Stage{Sun, Planets, Carrier}`, `(Gearbox).Named()`, and the
  name functions `RingName`, `SunName(k)`, `PlanetName(k, i)`, `CarrierName(k)`.
- `PlanetOrbitAngle(p, i)`, `PlanetToothAngle(p, i)` and `RingToothAngle(p)` from the meshing-phase formulas.

## 1. Parameter table

Write the inputs and the derived names listed under Parameter table into both `doc.Params()` and a fresh
`sketch.World`'s table. Lengths are `units.Millimeters`, the pressure angle `units.Radians`, counts are numbers,
and derived values are expressions over those names. Read the derived radii back from the document's table.
Every sketch circle's diameter is bound to a table name with `Sketch.Bind`.

## 2. Parts built about the origin

Build each distinct part once about the local origin and +Z. A gear union fails on gears built at an offset.

1. **Ring.** (Bolt holes `db'` are 2.6 mm, 0.3 mm radially wider than the M2 shank.) `spurgear.Variant{Define}.Build` with `Teeth = Zr`, thickness `L + 2`, no bore, `Backlash = -j`. `Define` replaces
	`root_radius` by `pitch_radius - module`, `tip_radius` by `pitch_radius + module + c` and `fillet_radius` by
   `0`, so the pipeline builds the tooth-space tool. If `RingToothAngle` is not zero, `decad.Transform` the tool
   by that rotation about +Z. Sketch the blank on the plane z=1: the outline (a circle of diameter `2 * ring_outer_radius`, or with
   `EarCount > 0` the ear outline described under "Mounting ears" below)
   and four bolt-hole circles of diameter `db'` (bound to `bolt_hole_diameter`), each centered on the bolt
   circle at angle `π/4 + iπ/2` and dimensioned from the origin. Select the one valid profile with `4 + EarCount`
   holes and extrude it `L` upward. Cut eight closed round nut cavities from the blank (`cutNutPockets`): one tool, a circle
   of diameter `2 (s / √3 + c)` (`s` = `m2nut.AcrossFlats`, `c = 0.2`), extruded `m2nut.Thickness + 2c` up from its
   plane, copied to the four bolt angles. Place the front cavities' lower face 1 mm above the front face and the rear
   cavities' upper face 1 mm below the rear face. The bolt hole is each cavity's only opening. `decad.Cut` the tooth-space tool from it. Chamfer the ring's two end faces last (see Edge finishing). Check: one solid.
2. **Sun.** `spurgear.Variant{Define}.Build` with `Teeth = Zs`, `Thickness = T`, `Backlash = j`, root radius `pitch_radius - module - c`, tip radius `pitch_radius + module - ε`, and `BoreDiameter = shaft_bore_diameter` (5.1, 0.05 mm larger radially than the 5 mm shaft, a press fit)
   when `Sf = 0`, otherwise no bore. With a flat, extrude the D tool (see below) from `z = -1` up `T + 2` and
   `decad.Cut` it from the sun.
3. **Planet.** `spurgear.Variant{Define}.Build` with `Teeth = Zp`, `Thickness = T`, root radius `pitch_radius - module - c - ε`, tip radius `pitch_radius + module - ε`, `BoreDiameter = lip bore diameter`,
   `Backlash = j`. Then extrude a circle of the pocket diameter on the plane `z = T - Wb`, `Wb + 1` up, and
   `decad.Cut` it from the planet. Then chamfer the pocket's entrance (see **Seat chamfer** below). Check: one solid.
4. **Bearing.** `ballbearing.Build` with `ballbearing.Params605ZZ()`, nil placement: the annulus from z=0 to `Wb`.
5. **Carrier**, built once; every stage uses the same one.
   - Plate: one sketch on z=0, extruded `Tc`. The outline is the convex hull of the `n` pin bosses: for pin `k`
     at `a (cos φₖ, sin φₖ)` and boss radius `Rb`, edge `k` runs from pin `k` to pin `k+1` with outward normal at
     angle `φₖ + π/n`, from `pinₖ + Rb·normal` to `pinₖ₊₁ + Rb·normal`, and a counterclockwise arc of pin
     `k+1`'s boss circle (center `pinₖ₊₁`) joins its end to the start of edge `k+1`. That is `2n` curves.
     Dimension each pin center from the origin and each edge's end point from the origin. Each arc's other end is
     fixed by the arc's equal-radius relation and one coordinate: x where the direction from the center is
     closer to vertical, otherwise y, so the coordinate always changes as the point moves round the arc (the
     shared `DimensionArcEnd` picks by position and can pick the free one). The plate has no hole yet. Require one
     valid profile with `2n` outer curves.
   - Socket tool: one D section of radius `collar/2 + 0.05` centered at `(a, 0)` with its flat `Sf` inside the rim, on
     the plane `z = -1`, extruded `Tc + 2` so it passes through the plate. `decad.PatternCopies` it with a
     `CircularPattern` about the origin and +Z, count `n`, `decad.Union` the copies and `decad.Cut` the result from
     the plate. With `Sf = 0` the section is round.
   - Centre hole: `decad.Cut` a D section of radius `collar/2 + 0.05`, flat `Sf` inside the rim, extruded from
     `z = -1` up `Tc + 2`, from the carrier. The plate keeps a tooth its full thickness where the shaft has its flat.
     The D tool of the sun bore stays as below.
   - Check: one solid, nothing above the plate.
   - Pin, a separate body built about its own axis in the carrier frame, with `dp = bearing_bore_diameter` and the
     press-fit clearance `0.05`: `thickBottom = -g + 0.5`. A rod of diameter `dp` on the axis from `z = -(T + g)` up to
     `thickBottom + 0.05`, and a D thick part of radius `collar/2` with its flat `Sf` inside the rim from `thickBottom`
     up to `Tc` (with `Sf = 0` a round disc bound to `shaft_collar_diameter`), joined with one `decad.Union`. This
     leaves 0.25 mm between the bearing and the thick part. Pose pin `i` of stage `k` at
     `(a cos φᵢ, a sin φᵢ, kP + T + g)` turned by `φᵢ`, so its flat faces the same way as its socket. Check: one
     solid, its top level with the carrier's top.
6. **Shafts**, separate bodies. `buildSteppedShaft(thickTop, top, flats...)` builds the link and output shafts: a D thick
   part of radius `collar/2` with its flat `Sf` inside the rim from `z = 0` up to `thickTop`, and a round rod of diameter
   `ds` from `thickTop - 0.05` up to `top`, joined with one `decad.Union`; then the flat's box cut out over each of the
   flats' spans. `buildShaft(bottom, top, collarBottom, collarTop, flats...)` builds the input shaft: a round rod
   from `bottom` to `top`, a round collar of diameter `ds + 2 × collar margin` from `collarBottom` to `collarTop`,
   strictly inside the rod's length, joined to the rod with one union, and then the flat's box cut out over each span.
   The box is `ds + 2` wide about the X axis and runs from `Sf` inside the rod's +X edge to 1 mm beyond it, as a
   rectangle sketch with its corner dimensioned from the origin, its width and its height. `decad.Cut` removes it. The
   flats of every span join into one tool and take one cut.
   - Link shaft, when `K > 1`: `thickTop = Tc + g - 0.25`, `top = T + Tc + g`, one flat from `Tc + g` to `top + 1`.
     Posed with its bottom on a carrier's lower face.
   - Output shaft: `thickTop = Tc + g + Tv - 0.25`, `top = Tc + g + Tv + Wb + Lo`, no flat on the rod. Posed with its
     bottom on the last carrier's lower face.
   - Input shaft: `bottom = -(g + Tv + Wb + Li)`, `top = T`, collar from `-(g + Tv)` to `-0.5`, flat from `0` to
     `T + 1`.
7. **Cover.** One extruded frame, then a bearing boss, joined by one union.
   - Frame: one sketch on z=0. Circle of diameter `2 * ring_outer_radius`; circle of diameter `lip_bore_diameter`
     at the origin; the four bolt-hole circles as for the ring. For each spoke `k`, with spoke angle
     `ψ = π/6 + 2πk/3`, half spoke width `h`, hub radius `Rh = cover hub diameter / 2` and rim radius
     `Rr = cover rim inner radius`: a window between spoke `k` and spoke `k + 1` bounded by a line along spoke
     `k`'s upper edge from the hub circle to the rim circle, an arc of the rim circle (about the origin)
     to spoke `k + 1`'s lower edge, a line back to the hub circle, and an arc of the hub circle back to the
     start. Dimension the start and rim-start points from the origin and the other two arc ends by one
     coordinate each. Require one valid profile with `3 + 1 + 4` holes and extrude it `Tv`.
   - Boss: a sketch on the plane `z = Tv - 1` with two circles about the origin, diameters
     `cover_boss_diameter` and `pocket_diameter`; extrude the annulus `Wb + 1` up.
   - `decad.Union` the frame and the boss.
   - Head pockets (`cutHeadPockets`): a circle of diameter `Dk + 2c` (`Dk` = `m2bolt.HeadDiameter`, `c = 0.3`) on the
     plane `z = Tv - m2bolt.HeadHeight`, extruded up `HeadHeight + 1`, copied to the four bolt angles and cut from
     the cover's +Z face, so the pocket is as deep as the head is tall.
   - Chamfer the boss pocket's entrance (see **Seat chamfer** below). Check: one solid.

   **Fasteners** (`buildFasteners`). Build `m2bolt.Build` with `Length = 6` and `m2nut.Build` once each, and copy
   them into place. A front bolt is turned over about X so its shank points into the ring, with its head's underside
   on the pocket floor `HeadHeight` inside the cover's outer face. A rear bolt keeps its orientation. Each nut sits in its
   cavity with `c = 0.2` below it, its flats toward the bolt circle's radius. Four bolts and nuts per cover. Cover rim inner radius
   is measured from the head pocket's edge, not the hole's.

**Seat chamfer.** `decad.Chamfer` of the pocket's circular rim in the plane `z = T` (planet) or `z = Tv + Wb` (cover)
by `seat_chamfer`, skipped when it is zero. A boolean moves a circle's seam, so find the rim by its radius and height
among the body's edges, then select it with `decad.Edges(decad.Circular(), decad.EndpointAt(its start))` asserted
`Exactly(1)`. Chamfer the planet and the cover as built, before they are copied.

**Mounting ears.** With `EarCount > 0`, replace the blank's circle by one closed loop (`addRingOutline`). For ear `i` at angle
`a = π/2 + 2πi/EarCount`, with `we = EarBoreDiameter/2 + earWall` (`earWall` = 3) and `Re = ring_outer_radius + we`, create:
a construction line `axis` from the origin to the bore's center point; a bore circle at that point; a construction line `across`
between the half circle's two ends; two lines from those ends to the outer circle; and a half-circle arc centered on the bore,
counter-clockwise from the clockwise end to the counter-clockwise end. Between ear `i` and ear `i+1` add an arc centered on the
origin, counter-clockwise from ear `i`'s counter-clockwise root to ear `i+1`'s clockwise root. Constrain with relations:
`Distance(origin, bore center) = Re` (bound to `ring_outer_radius + ear_bore_diameter / 2 + ear_wall`); the first axis `Vertical`,
each later axis `Angle` of `360/EarCount` degrees from the one before; the bore `Diameter` (bound to `ear_bore_diameter`); the half
circle `Radius = we` (bound to `ear_bore_diameter / 2 + ear_wall`); `Perpendicular(axis, across)` and `PointOnLine(bore center,
across)`; both straight edges `Parallel` to the axis; each outer arc `Radius = ring_outer_radius`. Place each initial point from
the angle `a`. The sketch must report fully constrained with DOF 0. Ear bores are inner loops of the section.

**Edge finishing.** After a ring, a cover or a carrier is complete, chamfer the boundary loops it keeps by `edge_chamfer`
(see Edge finishing in `instructions.md`). For each plane `z`, find the body's planar faces whose plane normal is along Z
at that height and whose area is at least 400 mm², then take the kept loops of those faces. Select their edges with
`decad.Edges(...).Or(...)`, each edge named by `decad.EndpointAt(its start)` plus `decad.Circular()` for a circle or arc, or
`decad.ParallelTo(its direction)` for a line, asserted `Exactly(count)`, and make one `decad.Chamfer` call per plane. Ring
(`finishFacesEdges`, one `decad.Chamfer` call for both planes, after the tooth-space cut): the outer loop at `z = ringMargin`
and at `z = ringMargin + ring length`, ears included. Cover (after the seat chamfer): at `z = 0` the
outer loop, every loop of more than one edge and the single-circle loop of radius `lip_bore_diameter / 2`; at `z = Tv` the
outer loop and every loop of more than one edge. Carrier (after the centre hole is cut): the outer loop at `z = 0` and at
`z = Tc`. A body that has no large face in the plane, or a chamfered result that is not a solid, is an error.

**Shaft section.** A circle of diameter `shaft_diameter = bearing_bore_diameter`, the section of every rod.
**D tool.** The section of the sun bore: one outline with 0.05 mm (`shaft_bore_clearance`) around the D shaft, its flat
0.05 mm beyond the shaft's flat, extruded once so the tool is not itself a boolean result. With
`R = shaft_diameter/2 + 0.05`, flat `x = f = R - Sf` and half width `h = sqrt(R² - f²)`, the outline is an arc about the
origin from the flat's upper end `(f, +h)` counterclockwise round to its lower end `(f, -h)`, and the line across the
flat that closes it. Dimension the upper end from the origin and the lower end by the one coordinate that changes along
the arc there, so the arc's radius relation fixes the rest; require one valid profile with two outer curves. With
`Sf = 0` the tool is the round section.

Each circle sketch is solved and checked with `sketchcheck.VerifyFullyConstrained` before extruding, and must
yield exactly one valid profile. The sketch is anchored by its origin; a center away from it gets
`sketchcheck.DimensionPoint`.

## 3. Placing the parts

Compose each pose as `turn about +Z`, then `translate`, then the placement frame (`r3.FromFrame`, or the identity
for a nil placement). For every stage `k` with `z = k P`:

- Sun `k`: translate `(0, 0, z)`.
- Planet `i` of stage `k`: turn by `PlanetToothAngle(p, i)`, translate
  `(a cos φᵢ, a sin φᵢ, z)`.
- Bearing `i` of stage `k`: translate `(a cos φᵢ, a sin φᵢ, z + T - Wb + 0.25)`.
- Carrier `k`: translate `(0, 0, z + T + g)`.
- Shaft of stage `k`: translate `(0, 0, z + T + g)`; stage `K - 1` takes the output shaft and the rest the link shaft.
- Ring: translate `(0, 0, -1 - g)`, which puts its blank's z=1 start at `-g`.
- Rear cover: translate `(0, 0, K P)`. Front cover: turn over about the X axis by `π`, then translate `(0, 0, -g)`.
- Rear bearing: translate `(0, 0, K P + Tv)`. Front bearing: translate `(0, 0, -(g + Tv + Wb))`.
- Input shaft: no motion.

For each part, `decad.Copy` with `WithCopyPlacement` makes every instance but the last, and `decad.Transform` moves
the master itself into the last, so no stray master remains. The document then holds exactly `6 + K (3 + 2n)`
bodies: the ring, two covers, two cover bearings and the input shaft, and per stage a sun, `n` planets, `n`
bearings, a carrier and a shaft.

## 4. Acceptance checks

The tests in `planetarygearbox_test.go` implement the acceptance cases:

- Default: bodies, solidity, volumes, ring and sun and planet and carrier bounds. The carrier's largest
  x is `a + boss/2`, its top is the plate's top and its bottom is its lower face; the output shaft starts at the
  carrier's lower face and ends `Lo` beyond the rear boss.
- Bearings: for each planet, points on three radii, three heights and three angles inside the bearing's
  annulus are inside the bearing body and outside the planet; the bearing is 14 mm across, `Wb` wide and
  its top is at `T`; the planet's volume is the plain gear's with the lip bore minus
  `π/4 (pocket² - lip²) Wb`, within the summed volume bounds plus 1.
- Carrier pins: each pin is solid, runs from the gears' lower face to the carrier's upper face, has a D thick part 0.05 mm
  from its socket (proven to 0.02 mm) and starts 0.2 to 0.3 mm above its bearing; its volume is the rod's plus the D prism's.
- Carrier shafts: the output shaft's thick part starts at the carrier's lower face, ends 0.25 mm short of the rear
  bearing, and its volume is the D prism's plus the thin rod's.
- Press-fit rims: each carrier face has `n + 1` inner loops and none has a bevel beside it.
- Carrier: with `LocatePoint`, at mid-plate height the plate is inside at radius 12 and 15 and 60 degrees (between
  two pins) and outside at radius 20 on that line; a pin boss is inside and the shaft hole open at the axis and at
  `(-2.5, 0)`; a pin socket is empty at a quarter and three quarters of the plate's height and the plate beside it is solid; the plate is solid at
  `(6, 0)`. Nothing stands above the plate.
- Mesh: for every planet, the outline vertices of the planet taken at the mid-height of the sun, and of the ring,
  and the reverse, are never reported inside by `Body.LocatePoint` with a 0.02 mm tolerance. The gears are
  straight prisms, so cap vertices stand for every height.
- Seat chamfer: a planet and a cover each weigh `π (2 r) Sc² / 2` less than the `Sc = 0` ones, with `r` the pocket
  radius, within the summed volume bounds plus 1, and stay solid.
- Edge finishing: the ring, both covers and each carrier have at least two large flat faces whose whole outer boundary
  has a 45° bevel beside it (a plane tilted 45° or a cone of 45° half angle); the circular outer edges of the ring and
  the covers lie `Ec` inside the widest cylinder; an edge chamfer below 0.5 or above 1 is rejected.
- Phase: a planet copy turned by `π/Zp` about its own axis has such vertices inside both the sun and the ring.
- Shaft flat: with `LocatePoint` at mid-height of the sun, the sun is inside at `(2.4, 0.3)` and outside at `(-1, 0.3)`,
  and the input shaft is inside at `(1, 0.3)`; the input shaft's volume equals a round rod's plus the collar's less the flat's
  circle segment over the sun's thickness, within the volume bound plus 0.5; in the two-stage case the second sun has
  the same tooth for the link shaft.
- Housing: the covers' and bearings' bounds sit at the axial positions of the contract; the input shaft starts `Li`
  below the front boss and ends at `T`, and `LocatePoint` finds it inside and the sun outside at the axis; at
  each bolt angle the ring and both covers are inside beside the bolt hole; a cover
  has material on a spoke and none in a window and weighs less than a disc; points are 1.5 mm or more from any surface.
- Fasteners: 8 bolts and 8 nuts, all solids. A front bolt's head ends level with the front cover's outer face and a
  rear bolt's with the rear cover's. Each nut sits `1 + c` mm behind the ring's end face. A bolt's tip
  is `HeadHeight + 6` from the cover's outer face. No bolt overlaps its cover or the ring and no nut overlaps the ring
  (volume of the intersection of copies). A hole narrower than the bolt shank, a cover thinner than head height plus
  1 mm, and pockets that leave less than 1 mm of ring wall are rejected.
- Two stages: 46 bodies, stage offsets by `P`, the link shaft's top equal to the second sun's top, the output shaft
  `Lo` beyond the rear boss, and the second stage's planets mesh as the first's do. The link shaft's thick part is a
  0.02 mm or wider press fit in the first carrier and ends 0.25 mm short of the second sun; a build with `Sf = 0` and
  `g = 0.75` succeeds.
- Placed: bounds follow the frame, and each body's volume matches the unplaced one within the volume bounds.
- Backlash: a gearbox with `j = 0.2` has a smaller sun, planet and ring volume than the default beyond the volume bounds.
- Invalid: every listed input error names its rule.
