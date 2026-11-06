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

1. **Ring.** `spurgear.Variant{Define}.Build` with `Teeth = Zr`, thickness `L + 2`, no bore, `Backlash = -j`. `Define` replaces
   `root_radius` by `pitch_radius - module`, `tip_radius` by `pitch_radius + 1.25 * module` and `fillet_radius` by
   `0`, so the pipeline builds the tooth-space tool. If `RingToothAngle` is not zero, `decad.Transform` the tool
   by that rotation about +Z. Sketch the blank on the plane z=1: a circle of diameter `2 * ring_outer_radius`
   and four bolt-hole circles of diameter `db'` (bound to `bolt_hole_diameter`), each centered on the bolt
   circle at angle `π/4 + iπ/2` and dimensioned from the origin. Select the one valid profile with four
   holes and extrude it `L` upward. `decad.Cut` the tool from it. Check: one solid.
2. **Sun.** `spurgear.Build` with `Teeth = Zs`, `Thickness = T`, `Backlash = j` and `BoreDiameter = ds` (8, the bearing bore)
   when `Sf = 0`, otherwise no bore. With a flat, extrude the shaft section (see below) from `z = -1` up `T + 2` and
   `decad.Cut` it from the sun.
3. **Planet.** `spurgear.Build` with `Teeth = Zp`, `Thickness = T`, `BoreDiameter = lip bore diameter`,
   `Backlash = j`. Then extrude a circle of the pocket diameter on the plane `z = T - Wb`, `Wb + 1` up, and
   `decad.Cut` it from the planet. Check: one solid.
4. **Bearing.** `ballbearing.Build` with `ballbearing.Params608ZZ()`, nil placement: the annulus from z=0 to `Wb`.
5. **Carrier**, once per distinct stub top. A multi-stage gearbox needs a middle carrier (stub top
   `Tc + g + T` above the arms' lower face) and a last carrier (stub top `Tc + g + Tv + Wb + Lo`). A single
   stage needs only the last.
   - Spider: one sketch on z=0 holding the hub and all `n` arms as a single closed outline of `4n` curves,
     extruded `Tc`. With half the arm width `h`, hub radius `Rh`, boss radius `Rb` and orbit radius `a`,
     arm `i` points along `φᵢ`. It contributes, counterclockwise, a line along its lower edge from the
     hub circle (distance `sqrt(Rh² - h²)` along the axis) out to the boss circle (distance
     `a - sqrt(Rb² - h²)`), a long arc of the boss circle (center at distance `a` on the axis) to the upper
     edge, and a line back along the upper edge. A short arc of the hub circle about the origin joins an
     arm's upper hub point to the next arm's lower hub point. Dimension every computed point from the
     origin, except one end of each arc, which the arc's equal-radius relation fixes: the next arm's lower hub
     point, and for the last arm its upper hub point, and each boss arc's upper point. Verify full
     constraint and require one valid profile with `4n` outer curves. A union of separate hub and arm
     bodies is refused by decad (coincident or crossing faces); draw the one outline.
   - Pin: circle of diameter `dp` centered at `(a, 0)` on the plane `z = -(T + g)`, extruded `T + g + Tc/2`.
     `decad.PatternCopies` with a `CircularPattern` about the origin and +Z, count `n`.
   - Stub: the shaft section on the plane `z = Tc/2`, extruded `stubTop - Tc/2`.
   - `decad.Union` the spider, pins and stub. Pins and stub reach into the spider, so the union overlaps.
   - Check: one solid.

Each circle sketch is solved and checked with `sketchcheck.VerifyFullyConstrained` before extruding, and must
yield exactly one valid profile. The sketch is anchored by its origin; a center away from it gets
`sketchcheck.DimensionPoint`.

6. **Cover.** One extruded frame, then a bearing boss, joined by one union.
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
   - `decad.Union` the frame and the boss. Check: one solid.
7. **Input shaft.** The shaft section on the plane `z = -(g + Tv + Wb + Li)`, extruded up to `z = T`.

**Shaft section.** A circle of diameter `ds` when `Sf = 0`. Otherwise a D: with radius `R = ds/2`, flat line
`x = R - Sf` and half-chord `h = sqrt(R² - (R - Sf)²)`, two curves, a long arc about the origin from
`(R - Sf, +h)` counterclockwise to `(R - Sf, -h)` and the line between them. Dimension the upper point
from the origin and the lower point by one coordinate, so the arc's radius relation fixes the rest; require
one valid profile with two outer curves. The input shaft, every carrier stub and the sun bore tool use it.

## 3. Placing the parts

Compose each pose as `turn about +Z`, then `translate`, then the placement frame (`r3.FromFrame`, or the identity
for a nil placement). For every stage `k` with `z = k P`:

- Sun `k`: translate `(0, 0, z)`.
- Planet `i` of stage `k`: turn by `PlanetToothAngle(p, i)`, translate
  `(a cos φᵢ, a sin φᵢ, z)`.
- Bearing `i` of stage `k`: translate `(a cos φᵢ, a sin φᵢ, z + T - Wb)`.
- Carrier `k`: translate `(0, 0, z + T + g)`; the last stage takes the last carrier and the rest the middle one.
- Ring: translate `(0, 0, -1 - g)`, which puts its blank's z=1 start at `-g`.
- Rear cover: translate `(0, 0, K P)`. Front cover: turn over about the X axis by `π`, then translate `(0, 0, -g)`.
- Rear bearing: translate `(0, 0, K P + Tv)`. Front bearing: translate `(0, 0, -(g + Tv + Wb))`.
- Input shaft: no motion.

For each part, `decad.Copy` with `WithCopyPlacement` makes every instance but the last, and `decad.Transform` moves
the master itself into the last, so no stray master remains. The document then holds exactly `6 + K (2 + 2n)`
bodies: the ring, two covers, two cover bearings and the input shaft, and per stage a sun, `n` planets, `n`
bearings and a carrier.

## 4. Acceptance checks

The tests in `planetarygearbox_test.go` implement the acceptance cases:

- Default: bodies, solidity, volumes, ring and sun and planet and carrier bounds. The carrier's largest
  x is `a + boss/2`.
- Bearings: for each planet, points on three radii, three heights and three angles inside the bearing's
  annulus are inside the bearing body and outside the planet; the bearing is 22 mm across, `Wb` wide and
  its top is at `T`; the planet's volume is the plain gear's with the lip bore minus
  `π/4 (pocket² - lip²) Wb`, within the summed volume bounds plus 1.
- Carrier: with `LocatePoint`, a point on arm 0's and arm 1's axis at radius 15 and mid-thickness is
  inside, and a point at radius 15 between them is outside; the volume is below that of a plate of
  radius `a + boss/2` and thickness `Tc`.
- Mesh: for every planet, the outline vertices of the planet taken at the mid-height of the sun, and of the ring,
  and the reverse, are never reported inside by `Body.LocatePoint` with a 0.02 mm tolerance. The gears are
  straight prisms, so cap vertices stand for every height.
- Phase: a planet copy turned by `π/Zp` about its own axis has such vertices inside both the sun and the ring.
- Shaft flats: with `LocatePoint` at `x = 3.5`, mid-height of the sun, the input shaft is outside and the sun inside
  at `(3.5, 0)`, and the reverse at `(-3.5, 0)`, `(0, 3.5)` and `(0, -3.5)`; the shaft's volume equals a round rod's
  less the flat's circular segment within the volume bound plus 0.5; in the two-stage case the first carrier's stub
  and the second sun relate the same way.
- Housing: the covers' and bearings' bounds sit at the axial positions of the contract; the input shaft starts `Li`
  below the front boss and ends at `T`, and `LocatePoint` finds it inside and the sun outside at the axis; at
  each bolt angle the ring and both covers are outside (hole) at the bolt circle and inside beside it; a cover
  has material on a spoke and none in a window and weighs less than a disc; points are 1.5 mm or more from any surface.
- Two stages: 22 bodies, stage offsets by `P`, the first carrier's top equal to the second sun's top, and the
  second stage's planets mesh as the first's do.
- Placed: bounds follow the frame, and each body's volume matches the unplaced one within the volume bounds.
- Backlash: a gearbox with `j = 0.2` has a smaller sun, planet and ring volume than the default beyond the volume bounds.
- Invalid: every listed input error names its rule.
