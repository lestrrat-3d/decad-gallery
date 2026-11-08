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

1. **Ring.** (Bolt holes `db'` are 2.4 mm for 0.1 mm radial clearance around the M2 shank.) `spurgear.Variant{Define}.Build` with `Teeth = Zr`, thickness `L + 2`, no bore, `Backlash = -j`. `Define` replaces
	`root_radius` by `pitch_radius - module`, `tip_radius` by `pitch_radius + module + c` and `fillet_radius` by
   `0`, so the pipeline builds the tooth-space tool. If `RingToothAngle` is not zero, `decad.Transform` the tool
   by that rotation about +Z. Sketch the blank on the plane z=1: a circle of diameter `2 * ring_outer_radius`
   and four bolt-hole circles of diameter `db'` (bound to `bolt_hole_diameter`), each centered on the bolt
   circle at angle `π/4 + iπ/2` and dimensioned from the origin. Select the one valid profile with four
   holes and extrude it `L` upward. Cut eight closed hex nut cavities from the blank (`cutNutPockets`): one tool, a hexagon
   of `s + 2c` across flats with its flats facing the bolt circle's radius, `c = 0.1`, extruded `m2nut.Thickness + 2c` up from its
   plane, copied to the four bolt angles. Place the front cavities' lower face 1 mm above the front face and the rear
   cavities' upper face 1 mm below the rear face. The bolt hole is each cavity's only opening. `decad.Cut` the tooth-space tool from it. Check: one solid.
2. **Sun.** `spurgear.Variant{Define}.Build` with `Teeth = Zs`, `Thickness = T`, `Backlash = j`, root radius `pitch_radius - module - c`, tip radius `pitch_radius + module - ε`, and `BoreDiameter = shaft_bore_diameter` (8, 0.25 mm larger radially than the 7.5 mm shaft)
   when `Sf = 0`, otherwise no bore. With a keyway, extrude the key tool (see below) from `z = -1` up `T + 2` and
   `decad.Cut` it from the sun.
3. **Planet.** `spurgear.Variant{Define}.Build` with `Teeth = Zp`, `Thickness = T`, root radius `pitch_radius - module - c - ε`, tip radius `pitch_radius + module - ε`, `BoreDiameter = lip bore diameter`,
   `Backlash = j`. Then extrude a circle of the pocket diameter on the plane `z = T - Wb`, `Wb + 1` up, and
   `decad.Cut` it from the planet. Then chamfer the pocket's entrance (see **Seat chamfer** below). Check: one solid.
4. **Bearing.** `ballbearing.Build` with `ballbearing.Params608ZZ()`, nil placement: the annulus from z=0 to `Wb`.
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
   - Pin: circle of diameter `dp = bearing_bore_diameter - 0.5` centered at `(a, 0)` on the plane `z = -(T + g)`, extruded `T + g + Tc/2`.
   - Pin collar, when `g >= 0.75`: circle of diameter `shaft_collar_diameter` at `(a, 0)` on the plane
     `z = -g + 0.5`, extruded `g - 0.5 + Tc/2`. This leaves 0.25 mm between the bearing and collar.
     `decad.Union` the pin and its collar once.
   - `decad.PatternCopies` the joined pin with a `CircularPattern` about the origin and +Z, count `n`, then
     `decad.Union` the plate with the pin and its copies. A union of every pin and collar separately is
     refused by decad (the chain is too deep), and so is a union of separate hub and arm bodies.
   - Shaft hole: `decad.Cut` the key tool, extruded from `z = -1` up `Tc + 2`, from the carrier. The plate keeps a
     tooth its full thickness where the shaft has its groove.
   - Check: one solid, nothing above the plate.
6. **Shafts**, separate bodies. `buildShaft(bottom, top, collarBottom, collarTop, keyBottom, keyTop)`: a round rod
   from `bottom` to `top`, when `collarTop > collarBottom` a round collar of diameter `ds + 2 × collar margin`
   from `collarBottom` to `collarTop`, strictly inside the rod's length, joined to the rod with one union, and
   then the keyway boxes cut out between `keyBottom` and `keyTop`. One box is `Sw` wide about the X axis and
   runs from `Sf` inside the rod's +X edge to 1 mm beyond it, as a rectangle sketch with its corner dimensioned
   from the origin, its width and its height. `decad.PatternCopies` with a `CircularPattern` about +Z, count 3,
   makes the other two, `decad.Union` joins the three into one tool, and one `decad.Cut` removes it. Collars stop `0.5` short of the part they face, so no collar face
   shares a plane with a groove's end.
   - Link shaft, when `K > 1`: `bottom = 0`, `top = T + Tc + g`, collar from `Tc + 0.25` to `Tc + g - 0.25` and grooves
     from `-1` to `Tc` and from `Tc + g` to `top + 1`, so the collar is whole. When `g <= 0.5` it has no collar and
     one groove from `-1` to `top + 1`. The grooves of every span join into one tool and take one cut. Posed with its bottom on a carrier's
     lower face.
   - Output shaft: `bottom = 0`, `top = Tc + g + Tv + Wb + Lo`, collar from `Tc + 0.5` to `Tc + g + Tv - 0.25`, groove from
     `-1` to `Tc`. Posed with its bottom on the last carrier's lower face.
   - Input shaft: `bottom = -(g + Tv + Wb + Li)`, `top = T`, collar from `-(g + Tv)` to `-0.5`, groove from `0` to
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
   - Head pockets (`cutHeadPockets`): a circle of diameter `Dk + 2c` (`Dk` = `m2bolt.HeadDiameter`, `c = 0.1`) on the
     plane `z = Tv - m2bolt.HeadHeight`, extruded up `HeadHeight + 1`, copied to the four bolt angles and cut from
     the cover's +Z face, so the pocket is as deep as the head is tall.
   - Chamfer the boss pocket's entrance (see **Seat chamfer** below). Check: one solid.

   **Fasteners** (`buildFasteners`). Build `m2bolt.Build` with `Length = 6` and `m2nut.Build` once each, and copy
   them into place. A front bolt is turned over about X so its shank points into the ring, with its head's underside
   on the pocket floor `HeadHeight` inside the cover's outer face. A rear bolt keeps its orientation. Each nut sits in its
   cavity with `c` below it, its flats toward the bolt circle's radius. Four bolts and nuts per cover. Cover rim inner radius
   is measured from the head pocket's edge, not the hole's.

**Seat chamfer.** `decad.Chamfer` of the pocket's circular rim in the plane `z = T` (planet) or `z = Tv + Wb` (cover)
by `seat_chamfer`, skipped when it is zero. A boolean moves a circle's seam, so find the rim by its radius and height
among the body's edges, then select it with `decad.Edges(decad.Circular(), decad.EndpointAt(its start))` asserted
`Exactly(1)`. Chamfer the planet and the cover as built, before they are copied.

**Shaft section.** A circle of diameter `shaft_diameter = bearing_bore_diameter - 0.5`, the section of every rod.
**Key tool.** The section of the sun bore and carrier hole: one outline with 0.25 mm around the round shaft and
normal to each key face, extruded once so the tool is not itself a boolean result. With
`R = shaft_diameter/2 + 0.25`, half width `h = (Sw - 0.5)/2`, floor `x = R - Sf` and wall
`w = sqrt(R² - h²)`, groove `k` lies at angle
`2πk/3`. Going counterclockwise the outline passes, for each groove, a line in along its lower wall from the rim
point `(w, -h)` to `(R - Sf, -h)`, a line across the floor to `(R - Sf, +h)` and a line out along its upper wall to
`(w, +h)`, all turned by the groove's angle, and then an arc about the origin to the next groove's lower rim
point. Dimension each upper rim point and each floor point from the origin and each lower rim point by the one
coordinate that changes along the arc there, so the arc's radius relation fixes the rest; require one valid
profile with twelve outer curves. With `Sf = 0` the tool is the round section.

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
  x is `a + boss/2`, its top is the plate's top and its bottom is the pins' ends; the output shaft starts at the
  carrier's lower face and ends `Lo` beyond the rear boss.
- Bearings: for each planet, points on three radii, three heights and three angles inside the bearing's
  annulus are inside the bearing body and outside the planet; the bearing is 22 mm across, `Wb` wide and
  its top is at `T`; the planet's volume is the plain gear's with the lip bore minus
  `π/4 (pocket² - lip²) Wb`, within the summed volume bounds plus 1.
- Carrier: with `LocatePoint`, at mid-plate height the plate is inside at radius 12 and 15 and 60 degrees (between
  two pins) and outside at radius 20 on that line; a pin boss is inside and the shaft hole open at the axis and at
  `(-2.5, 0)`; a pin is inside below the plate and the point 6 mm beside it outside; the plate is solid at
  `(6, 0)`. Nothing stands above the plate.
- Mesh: for every planet, the outline vertices of the planet taken at the mid-height of the sun, and of the ring,
  and the reverse, are never reported inside by `Body.LocatePoint` with a 0.02 mm tolerance. The gears are
  straight prisms, so cap vertices stand for every height.
- Seat chamfer: a planet and a cover each weigh `π (2 r) Sc² / 2` less than the `Sc = 0` ones, with `r` the pocket
  radius, within the summed volume bounds plus 1, and stay solid.
- Phase: a planet copy turned by `π/Zp` about its own axis has such vertices inside both the sun and the ring.
- Shaft keys: with `LocatePoint` at radius 3.5, mid-height of the sun, the input shaft is outside and the sun inside
  at 0°, 120° and 240°, and the reverse at 60°, 180° and 300°; at `(3.5, 0)` under the front bearing the
  shaft is inside; the shaft's volume equals a round rod's plus the collar's less the three grooves', within the volume
  bound plus 0.5; in the two-stage case the link shaft's grooves and the second sun's teeth relate the same way.
- Housing: the covers' and bearings' bounds sit at the axial positions of the contract; the input shaft starts `Li`
  below the front boss and ends at `T`, and `LocatePoint` finds it inside and the sun outside at the axis; at
  each bolt angle the ring and both covers are inside beside the bolt hole; a cover
  has material on a spoke and none in a window and weighs less than a disc; points are 1.5 mm or more from any surface.
- Fasteners: 8 bolts and 8 nuts, all solids. A front bolt's head ends level with the front cover's outer face and a
  rear bolt's with the rear cover's. Each nut sits `1 + c` mm behind the ring's end face. A bolt's tip
  is `HeadHeight + 6` from the cover's outer face. No bolt overlaps its cover or the ring and no nut overlaps the ring
  (volume of the intersection of copies). A hole narrower than the bolt shank, a cover thinner than head height plus
  1 mm, and pockets that leave less than 1 mm of ring wall are rejected.
- Two stages: 40 bodies, stage offsets by `P`, the link shaft's top equal to the second sun's top, the output shaft
  `Lo` beyond the rear boss, and the second stage's planets mesh as the first's do.
- Placed: bounds follow the frame, and each body's volume matches the unplaced one within the volume bounds.
- Backlash: a gearbox with `j = 0.2` has a smaller sun, planet and ring volume than the default beyond the volume bounds.
- Invalid: every listed input error names its rule.
