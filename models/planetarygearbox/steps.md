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
   by that rotation about +Z. Sketch a circle of diameter `2 * ring_outer_radius` on the plane z=1 and extrude it
   `L` upward. `decad.Cut` the tool from it. Check: one solid.
2. **Sun.** `spurgear.Build` with `Teeth = Zs`, `Thickness = T`, `BoreDiameter = ds`, `Backlash = j`.
3. **Planet.** `spurgear.Build` with `Teeth = Zp`, `Thickness = T`, `BoreDiameter = dp + 2 cp`, `Backlash = j`.
4. **Carrier**, once per distinct stub top. A multi-stage gearbox needs a middle carrier (stub top
   `Tc + g + T` above the plate bottom) and a last carrier (stub top `Tc + Lo`). A single stage needs only the last.
   - Plate: circle of diameter `2 * carrier_radius` at the origin of the plane z=0, extruded `Tc`.
   - Pin: circle of diameter `dp` centered at `(a, 0)` on the plane `z = -(T + g)`, extruded `T + g + Tc/2`.
     `decad.PatternCopies` with a `CircularPattern` about the origin and +Z, count `n`.
   - Stub: circle of diameter `ds` at the origin of the plane `z = Tc/2`, extruded `stubTop - Tc/2`.
   - `decad.Union` the plate, pins and stub. Pins and stub reach into the plate, so the union overlaps.
   - Check: one solid.

Each circle sketch is solved and checked with `sketchcheck.VerifyFullyConstrained` before extruding, and must
yield exactly one valid profile. The sketch is anchored by its origin; a center away from it gets
`sketchcheck.DimensionPoint`.

## 3. Placing the parts

Compose each pose as `turn about +Z`, then `translate`, then the placement frame (`r3.FromFrame`, or the identity
for a nil placement). For every stage `k` with `z = k P`:

- Sun `k`: translate `(0, 0, z)`.
- Planet `i` of stage `k`: turn by `PlanetToothAngle(p, i)`, translate
  `(a cos φᵢ, a sin φᵢ, z)`.
- Carrier `k`: translate `(0, 0, z + T + g)`; the last stage takes the last carrier and the rest the middle one.
- Ring: translate `(0, 0, -1 - g)`, which puts its blank's z=1 start at `-g`.

For each part, `decad.Copy` with `WithCopyPlacement` makes every instance but the last, and `decad.Transform` moves
the master itself into the last, so no stray master remains. The document then holds exactly `1 + K (2 + n)`
bodies.

## 4. Acceptance checks

The tests in `planetarygearbox_test.go` implement the acceptance cases:

- Default: bodies, solidity, volumes, ring and sun and planet and carrier bounds.
- Mesh: for every planet, the outline vertices of the planet taken at the mid-height of the sun, and of the ring,
  and the reverse, are never reported inside by `Body.LocatePoint` with a 0.02 mm tolerance. The gears are
  straight prisms, so cap vertices stand for every height.
- Phase: a planet copy turned by `π/Zp` about its own axis has such vertices inside both the sun and the ring.
- Two stages: 11 bodies, stage offsets by `P`, the first carrier's top equal to the second sun's top, and the
  second stage's planets mesh as the first's do.
- Placed: bounds follow the frame, and each body's volume matches the unplaced one within the volume bounds.
- Backlash: a gearbox with `j = 0.2` has a smaller sun, planet and ring volume than the default beyond the volume bounds.
- Invalid: every listed input error names its rule.
