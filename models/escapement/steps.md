# Graham escapement Decad steps

`instructions.md` owns every dimension, formula and fit; this file maps them to Sketch and Decad operations. Emit the
package `models/escapement`. All parts are built in local frames and posed once at the end.

## Shared sketch rules

- Create a fresh `sketch.World` per `Build`. Each sketch gets its own plane from `world.CreatePlaneFromFrame`.
  The plane origin is the part's axis (`O`, `P`, `G`, `S`, a pillar, or the crutch pin centre), so the part's centre
  is `s.Origin()`. A frame plane for a flat sketch has `U = +X`, `V = +Y` and its origin at the stated `z`.
  A meridian plane has `U = +X`, `V = +Z` and its origin on the axis.
- After the last constraint call `Solve`, then `internal/sketchcheck.VerifyFullyConstrained`; append the sketch to
  `Movement.Sketches` and only then select a profile and use it.
- Polar point `Q = (r∠a)`: create the point, a construction line origin to `Q`, `NewDistance(origin, Q, r)` (or place
  `Q` on a radius-constrained arc) and `NewAngle(xRef, line, a)`. `xRef` is one construction line per sketch from the
  origin to `X1` with `NewHorizontal(xRef)` and `NewDistance(origin, X1, 10)`.
- `CreateArc(center, start, end)` runs counter-clockwise from start to end; every arc gets `NewRadius`.
- Do not pin computed points with H/V distances except the single reference points named below.
- Profile selection: `Solve`, keep the valid profile whose hole count is the one listed; fail when it is missing or
  not unique.

## Exported API

`Params`, `DefaultParams`, `Validate`; `Layout(Params) (Geometry, error)`; `Motion(Params) (State, error)`;
`NewDocument(opts...)` (chain tolerance 0.1 mm first); `Build(ctx, doc, p) (*Movement, error)`;
`(*Movement) Named() []NamedBody`. `Build` rejects a nil context or document, validates the parameters, then checks
`ctx.Err()` before it touches the document. Every error carries the part and step that failed.
Check after the step: `TestValidate`, `TestLayout`, `TestMotion`, `TestActionAnalytic`.

## Layout and motion (no Decad)

`Layout` computes every constant, derived value and point of instructions.md "Escapement geometry", "Pendulum and
crutch", "Drive train" and the three axes. `Motion` implements "Motion and pose" in closed form. Check against the
default numbers in the tables to 1e-4 and the 4000-sample monotonic and drop checks.

## Bodies

### 1. Back Plate

Sketch `back plate outline` (`z -6`, origin `O`): 4 axis-aligned lines, 4 corner arcs `NewRadius 10` tangent to both
neighbours, each side placed by one H or V distance from the origin; lip-bore circles `r 5` at `O`, `P`, `G` and
mounting circles `r 2.25` at `(-30, 90)` and `(48, 90)` with their centres located by polar points. Valid profile:
5 holes. Extrude 6. Four pillar sketches (circle `r 5`, `z -0.5`, one per pillar, plane origin at the pillar) extruded
28.5; `Union(plate, pillars...)`. Cut tool: `r 7.05` pocket circles at `O`, `P`, `G` (`z -5`, height 6) unioned with
`m2fastener.HoleTool` and `CavityTool` of the four sites. `edgefinish.Chamfer` over `{-6, 0}`: outer loops, pocket
rims (7.05) and lip-bore rims (5) within 0.01 mm (22 edges), once and last. Check: solid, pockets, cavities, holes
by point classification, volume drop.

### 2. Front Plate

Sketch `front plate outline` (`z 28`): as the back plate with `y` top 82, the three lip bores only. Valid profile:
3 holes. Extrude 6. Cut tool: pocket circles `r 7.05` at `z 27` (height 6), `m2fastener.HoleTool` and `PocketTool`
of the four sites, and the D press outline about `S` (`r 2.55`, chord `x = S_x + 2.05`, plane origin at `S`, `z 31`,
height 4). One chamfer over `{28, 34}` (22 edges).

### 3-9. Bearings

`ballbearing` 605ZZ (`Params605ZZ`), placed with axis `+Z`: lower caps at `z -5` for `O`, `P`, `G`; `z 28` for `O`,
`P`, `G`; `z 39` at `S`. Names in the Bodies table. Add each model's sketches to `Movement.Sketches`.

### 10-12. Arbors, and 17 Suspension Pin

Meridian sketch (plane `U = +X`, `V = +Z`, origin on the axis): polygon of the staircase points of "Arbors"
(pin: "Pendulum and crutch"); each line `NewHorizontal` or `NewVertical`, the axis edge by
`NewHorizontalDistance(origin, p, 0)`, each step by H distance (radius) and V distance (z) from the origin. Valid
profile: 0 holes. `decad.Revolve` full turn about the sketch line `(0,0)-(0,1)`. Flat boxes: `CreateRectangle` at
`x 2.0 to 3.5`, `y -3.5 to 3.5`, corner by H/V distance, width 1.5 and height 7 by distances between corners,
extruded over each flat span; `Union` the boxes of one part and `Cut` once. Check: solid; diameter 5.0 at each bearing
seat; flat present at each span's middle and absent under each bearing.

### 13. Escape Wheel

Sketch `escape disc` (`z 22`, `O`): root circle `r 33`; six windows of an inner arc `r 13`, an outer arc `r 28` and
two lines each; each line `NewDistancePointLine(origin, line, 2.5)` and `NewAngle(xRef, line, σ)`. Valid profile: 6
holes. Extrude 3. Sketch `escape tooth` (`z 22`): `T` by H distance 0 and V distance 40; `L` and `B` on radius 32.5 with
`NewAngle(TO, TL, -10°)` and `NewAngle(TO, TB, -30°)`. Valid profile: 0 holes. Extrude 3. `decad.PatternCopies` (circular, count 30, axis `+Z`
through `O`); `Union(disc, tooth, copies...)`. One `edgefinish.Chamfer` over `{22, 25}`: inner loops with more than
one edge. Pinion: `spurgear.Variant{Define: gearroot.Define(gearroot.Clearance(1.5), 0.02)}`, module 1.5, 13 teeth,
pressure angle 20°, thickness 21, `Backlash`, no bore, placement origin `z 1.5` with `U = (cos -60°, sin -60°)`.
Sketch `rear boss` (`z 0.25`): circle `r 3.7`, extrude 1.75. `Union(wheel, pinion, boss)`. Cut the D press bore: sketch
(plane at `z -0.75`) arc `r 2.55` and chord `NewVertical` at `x 2.05`; extrude 26.75. Check: solid; section at
`z 23.5` has 8 loops and 30 tips at radius 40; section at `z 10` shows 13 teeth, tip 11.23.

### 14. Anchor

Sketch `anchor outline` (`z 21`, `P`): the 14 pieces of "Anchor outline" plus `xRef`. Arcs `NewRadius` `r_in`, `r_out`,
`rh`; `C_E`, `F_E`, `C_X`, `F_X` polar points; arm inner lines by `NewAngle(xRef, line, a)` and
`NewPointOnLine(origin, line)`; `K` on the `r_out` arc and on the inner line; arm ends `NewPerpendicular` and
`NewDistancePointLine(M, inner, 6)`; outer sides `NewParallel`, `N` on the top hub arc. Valid profile: 0 holes.
Extrude 5. Boss sketch (`z 25.5`) circle `r 3.7`, extrude 2.25, `Union`. Cut the D press bore (tool `z 20`, height
8.75). `edgefinish.Chamfer` over `{21, 26}` with `Outer` (28 edges), last. Check: solid; outline at `z 23.5` passes
within `0.02 + OffSurface` of `C_E`, `F_E`, `C_X`, `F_X`, `M_E`, `M_X`; sketches verified for the three branch cases.

### 15. Crutch

Sketch `crutch outline` (`z 34.25`, `P`): hub arc `r 6`, vertical lines at `x = ±2.5`, end arc `r 4` centred by H/V
distance `(0, -45)`. Valid profile: 0 holes. Extrude 3. Pin sketch (`z 36.75`, origin at the end centre) circle `r 1.5`,
extrude 6.5; `Union`. Cut the D press socket (`r 2.55`, flat `2.05`; tool `z 33.25`, height 5). Check: solid, area under 400 mm².

### 16. Pendulum

Sketch `pendulum outline` (`z 37.75`, `S`): eye arc `r 10`, lines at `x = ±4`, bob arc `r 25` centred by H/V
distance `(0, -Lp)`, lip circle `r 5`, slot (`CreateSlot`) with arc centres by H/V distance `(0, -61.25)` and
`(0, -56.75)`, radius 1.75. Valid profile: 2 holes. Extrude 6. Pocket sketch (`z 38.75`) circle `r 7.05`, extrude
6, `Cut`. One chamfer over `{37.75, 43.75}`: outer loop, pocket rim (7.05), lip-bore rim (5). Check: solid, slot
and pocket present, chamfered volume drop.

### 18. Suspension Cap

Circle `r 3.7` at `S`, `z 44.25`, extrude 1.5; cut the D press socket (`z 43.25`, height 3).

### 19. Great Wheel

`spurgear` with the `gearroot` variant: module 1.5, 48 teeth, thickness 7, `BoreDiameter 5.8`, `Backlash`, placement
origin `z 1.5`. Ratchet tool: disc sketch (`z 5.0`, `G`) circle `r 11.5`; triangle sketch with polar points
`(11.0∠0°)`, `(13.5∠30°)`, `(11.0∠30°)`; extrude both 4.5; pattern the triangle 12 times; `Union(disc, tooth,
copies...)`; `Cut` from the wheel. Check: solid; outline at `z 7` shows 12 ratchet teeth; outline at `z 3` shows 48 teeth.

### 20. Drum

Meridian sketch of the drum polygon (instructions "Drum"), revolve, cut the D press bore (tool `z 7.75`, height 21).
Pawl layer: core sketch (`z 5.25`) circle `r 7.5` with the D relief loop (`r 2.8`, flat `x 2.3`): valid profile with 1
hole, extrude 4; two pawl sketches (the pawl loop, polar points, arcs `NewRadius` 7.0, 9.0, 12.5, 10.5) extruded 4;
`Union(core, pawl0, pawl1)`; `Union(drum, layer)`. Check: solid; pawls present at `z 7`; flange front face under 400 mm².

### 21. Winding Knob

Circle `r 10` at `G`, `z 34.25`, extrude 3; D press socket cut (`z 33.25`, height 5).

### 22-29. Bolts and nuts

`m2fastener.PlaceBolt` and `PlaceNut` on the four sites. Check: head top at `z 34`, nut inside its cavity, tip inside
the pillar hole, no overlap with a plate.

## Pose, containers and placement

Pose each body once as in "Motion and pose" (rotation about its axis, then `Placement`), then create the containers
of the Bodies table with `ChildNamed` and `AddBody`; `doc.VerifyContainers().Err()` must be nil. The 0.05° back-off of
the wheel avoids contact at the build pose. Check: `TestInventory`, `TestSketches` (three branch cases),
`TestPlacement`, `TestGearMesh`, `TestCoupling`, `TestFits`, `TestEdgeFinishing`, `TestPoseNoOverlap`.
