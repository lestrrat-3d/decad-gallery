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

`G` is `(0, -45.75)`, the pinion's tooth 0 is at -90° and the great wheel's at 93.75°. `DefaultParams().Phase` is 0.25, where
ψ = 0 and β = 0.

`Layout` computes every constant, derived value and point of instructions.md "Escapement geometry", "Pendulum and
crutch", "Drive train" and the three axes. `Motion` implements "Motion and pose" in closed form. Check against the
default numbers in the tables to 1e-4 and the 4000-sample monotonic and drop checks.

## Bodies

### 1. Back Plate and 2. Front Plate

Each plate is one sketch, extruded once (instructions.md "Open skeleton plates"). Sketch `back plate outline` (`z -6`,
origin `O`) or `front plate outline` (`z 28`): the rounded-rectangle outline (4 axis-aligned lines, 4 corner arcs
`NewRadius 10` tangent to both neighbours, each side placed by one H or V distance from the origin), then two window
loops, then the lip circles `r 5` at `O` (the origin) and at `P`, `G` (polar points), then (back plate) the mounting
circles `r 2.25` at `(±40, 90)` as polar points. The hub centres of `P` and `G` are
polar points on `xRef`, shared by both windows; the hub centre of `O` is the origin.

Window loop (right side, `sx = +1`; the left window mirrors it): 6 lines and 5 arcs, the pieces and points of
instructions.md. Constraints: the vertical spine sides `NewVertical` with `NewHorizontalDistance(origin, A or H or J,
sx 4)`; the bottom edge `NewHorizontal` with `NewVerticalDistance(origin, A, -82)`; the top edge `NewHorizontal` with
`NewVerticalDistance(origin, E, 62)`; the side edge `NewVertical` with `NewHorizontalDistance(origin, C, sx 44)`; each
corner arc `NewRadius 6` and `NewTangent` to both of its lines; each hub arc `NewRadius 10` about its shared centre. The
shared endpoints `F, H, I, J, K, L` are fixed by those lines and arcs, not by distances. The bottom and top edges and the spine
lines are placed by `NewDistancePointLine(origin, line, d)` (so a fillet keeps them), and `sketch.CreateFillet` rounds the seven corners
`A, F, H, I, J, K, L` with radius 1.0; the left window passes the legs in the other order. On the right a clockwise arc from
`X` to `Y` is `CreateArc(centre, Y, X)`; on the left it is `CreateArc(centre, X, Y)`.

Valid profile: 5 holes (front) or 7 holes (back). Extrude 6. Back plate: `Union(plate, four pillars)` (circles `r 5`, `z
-0.5`, 28.5 long, at `(-50, 74)`, `(50, 74)`, `(-50, -88)`, `(50, -88)`). One cut tool, the union of the three pockets
`r 7.05` (back `z -5`, front `z 27`, 6 long), per pillar
`m2fastener.HoleTool` and `CavityTool` (back) or `PocketTool` (front), and on the front plate the D socket about `S`.
Then one `edgefinish.Chamfer` over `{low, low + 6}` with the `keep` filter of instructions.md. Do not union rim, spine
and hub prisms. Check: solid; `TestPlates`, `TestPlateWindows`, `TestPlateMaterial`; volume drop of the chamfer.

### 3-9. Bearings

`ballbearing` 605ZZ (`Params605ZZ`), placed with axis `+Z`: lower caps at `z -5` for `O`, `P`, `G`; `z 28` for `O`,
`P`, `G`; `z 39` at `S`. Names in the Bodies table. Add each model's sketches to `Movement.Sketches`.

### 10-12. Arbors, and 17 Suspension Pin

Meridian sketch (plane `U = +X`, `V = +Z`, origin on the axis): polygon of the staircase points of "Arbors"
(pin: "Pendulum and crutch"); each line `NewHorizontal` or `NewVertical`, the axis edge by
`NewHorizontalDistance(origin, p, 0)`, each step by H distance (radius) and V distance (z) from the origin. Valid
profile: 0 holes. `decad.Revolve` full turn about the sketch line `(0,0)-(0,1)`. The arbor and pin points are the
meridians of instructions.md "Arbors, bearings and axial layout" (the escape, pallet and great wheel arbors have `r 2.8`
and `r 3.05` body sections between their bearing seats). Flat boxes: `flatBox(what, x, y, flatX, z0, z1)`:
`CreateRectangle` at `x flatX to 3.5`, `y -3.5 to 3.5`, corner by H/V distance, width `3.5 - flatX` and height 7 by
distances between corners, extruded over each `flatSpan{X, Z0, Z1}` (`x 2.55` over a body section, `x 2.0` on an end or
the pin); `Union` the boxes of one part and `Cut` once. Check: solid; diameter 5.0 at each bearing seat; flat present at
each span's middle (between the flat plane and the section radius) and absent under each bearing; volume equals the
revolve less the circular segment of each section beyond its flat over each span.

### 13. Escape Wheel

Sketch `escape disc` (`z 22`, `O`): root circle `r 33`; six windows of an inner arc `r 13`, an outer arc `r 28` and
two lines each, the four corners of each window rounded with `sketch.CreateFillet`, radius 1.0; each line `NewDistancePointLine(origin, line, 2.5)` and `NewAngle(xRef, line, σ)`. Valid profile: 6
holes. Extrude 3. Sketch `escape tooth` (`z 22`): `T` by H distance 0 and V distance 40; `L` and `B` on radius 32.5 with
`NewAngle(TO, TL, -10°)` and `NewAngle(TO, TB, -30°)`. Valid profile: 0 holes. Extrude 3. `decad.PatternCopies` (circular, count 30, axis `+Z`
through `O`); `Union(disc, tooth, copies...)`. Front counterbore sketch (`z 23.95`, `O`): circle `r 3.75`, extrude 2.05
(tool `z [23.95, 26.0]`); `Cut` it from the wheel. One `edgefinish.Chamfer` over `{22, 25}`: inner loops with more than
one edge, and the single-edge circle of radius 3.75 about `O` (the counterbore's rim; match the loop's one edge by the
distance of its start from `O`, within 0.01). Pinion: `spurgear.Variant{Define: gearroot.Define(gearroot.Clearance(1.5), 0.02)}`, module 1.5, 13 teeth,
pressure angle 20°, thickness 21, `Backlash`, no bore, placement origin `z 1.5` with `U = (cos -60°, sin -60°)`.
`Union(wheel, pinion)`; there is no boss. Cut the D press bore: sketch (plane at `z -0.75`) arc `r 2.85` and chord
`NewVertical` at `x 2.60` (body family); extrude 27.5. The same cut takes the rear counterbore: sketch (`z 0.5`) circle
`r 3.75`, extrude 2.80 (tool `z [0.5, 3.30]`), unioned with the D bore tool, so one `Cut` makes both. Check: solid; section at
`z 23.5` has 8 loops and 30 tips at radius 40; section at `z 10` shows 13 teeth, tip 11.23; the counterbores are open at
`z 24.2` and `z 2.4` inside radius 3.75 and the part's rearmost face is `z 1.5`.

### 14. Anchor

Sketch `anchor outline` (`z 21`, `P`): the 14 pieces of "Anchor outline" plus `xRef`. Arcs `NewRadius` `r_in`, `r_out`,
`rh`; `C_E`, `F_E`, `C_X`, `F_X` polar points; arm inner lines by `NewAngle(xRef, line, a)` and
`NewPointOnLine(origin, line)`; `K` on the `r_out` arc and on the inner line; arm ends `NewPerpendicular` and
`NewDistancePointLine(M, inner, 6)`; outer sides `NewParallel`, `N` on the top hub arc. Then `sketch.CreateFillet` at the four inside corners: `H_E`, `H_X` (arm
inner side to hub arc) with radius 3.0 and `G_E`, `G_X` (arm inner side to pallet arc) with radius 1.0; the arm inner lines carry their
angle and `NewPointOnLine` before the fillet, and the leg order that verifies is the one the code uses. Valid profile: 0 holes.
Extrude 5. There is no boss. Counterbore sketch (`z 24.70`, `P`): circle `r 3.75`, extrude 2.30 (tool `z [24.70, 27.0]`),
`Cut`. Cut the D press bore (body family: arc `r 2.85`, chord `x 2.60`; tool `z 20`, height 8.75).
`edgefinish.Chamfer` over `{21, 26}` keeping the outer loops and the counterbore's rim (the single-edge circle of radius
3.75 about `P`), last. Check: solid; outline at `z 23.5` passes
within `0.02 + OffSurface` of `C_E`, `F_E`, `C_X`, `F_X`, `M_E`, `M_X`; sketches verified for the three branch cases.

### 15. Crutch

Sketch `crutch outline` (`z 35.30`, `P`): hub arc `r 6`, vertical lines at `x = ±2.5` (each `NewVertical` and `NewDistancePointLine(origin, line, 2.5)`), end arc `r 4` centred by H/V
distance `(0, -45)`. Round the two arm-to-hub corners with `sketch.CreateFillet`, radius 2.5. Valid profile: 0 holes. Extrude 3. Pin sketch (`z 37.80`, origin at the end centre) circle `r 1.5`,
extrude 6.5; `Union`. Cut the keyed rear counterbore: `dTool` D outline (arc `r 3.75`, chord `x 3.35`) about `P` at `z 34.30`,
extrude 3.0, flat on `+X` like the bore. Cut the D press socket (end family: `r 2.55`, flat `2.05`; tool `z 34.30`, height 5). Check: solid, area under 400 mm².

### 16. Pendulum

Sketch `pendulum outline` (`z 38.80`, `S`): eye arc `r 10`, lines at `x = ±4`, bob arc `r 25` centred by H/V
distance `(0, -Lp)`, each rod line `NewVertical` and `NewDistancePointLine(origin, line, 4)`, the four rod corners rounded with
`sketch.CreateFillet` radius 4.0, lip circle `r 5`, slot (`CreateSlot`) with arc centres by H/V distance `(0, -61.25)` and
`(0, -56.75)`, radius 1.75. Valid profile: 2 holes. Extrude 6. Pocket sketch (`z 39.80`) circle `r 7.05`, extrude
6, `Cut`. One chamfer over `{38.80, 44.80}`: outer loop, pocket rim (7.05), lip-bore rim (5). Check: solid, slot
and pocket present, chamfered volume drop.

### 18. Suspension Cap

Circle `r 3.7` at `S`, `z 45.30`, extrude 1.5; cut the D press socket (`z 44.30`, height 3).

### 19. Great Wheel

`spurgear` with the `gearroot` variant: module 1.5, 48 teeth, thickness 7, `BoreDiameter 6.9`, `Backlash`, placement
origin `z 1.5`. Counterbore sketch (`z 0.5`, `G`): circle `r 4.10`, extrude 3.0 (tool `z [0.5, 3.50]`); `Cut` it from
the gear, then one `edgefinish.Chamfer` over `{1.5}` that keeps only the single-edge circle of radius 4.10 about the
origin (the counterbore's rim), so the tooth flanks stay sharp. Ratchet tool: disc sketch (`z 5.0`, `G`) circle `r 11.5`; triangle sketch with polar points
`(11.0∠0°)`, `(14.0∠30°)` (a plain point), `(11.0∠30°)`; the radial side `NewPointOnLine(origin, line)` and the slope side `NewAngle`; round the
`14.0` corner with `sketch.CreateFillet`, radius 0.5; extrude both 4.5; pattern the triangle 12 times; `Union(disc, tooth,
copies...)`; `Cut` from the wheel. Check: solid; outline at `z 7` shows 12 ratchet teeth; outline at `z 3` shows 48 teeth and the counterbore (`r 4.10`) as a hole; the rim is chamfered.

### 20. Pawl Carrier

Built about the origin and moved onto `G` by the pose. Pawl layer sketch (`z 5.25`): the one closed loop of instructions "Pawl
Carrier" for both pawls (`g = 90° - δb` and `270° - δb`), valid profile with 0 holes, extrude 4. Core arcs `NewRadius 7.5`; the slot-end
round is `CreateArc(centre, top, start)` with `NewTangentCircles(core, round, false)` and both end points polar (angle `c`), and no radius
dimension; arcs `NewRadius` 9.0, 12.5, 10.5; the contact and back lines `NewAngle(xRef, line, a)`; then `sketch.CreateFillet` at the tooth root
(0.5) and the block back (1.0). Boss sketch (`z 8.75`): circle `r 4.2`, extrude 2.25. `Union(layer, boss)`; cut the D press bore
(sketch `z 4.25`, drive family: arc `r 3.10` and a vertical chord at `x 2.60`, extrude 7.75). Check: solid; pawl tips at the old drum's
world angles; slot-end round present; boss present at `z 10`, absent at `z 11.1`.

### 21. Mainspring

`spiralspring.Build(ctx, doc, SpringParams())` with `Placement` origin `(G_x, G_y, 11.25)`, `U = (cos γs, sin γs, 0)`,
`V = (-sin γs, cos γs, 0)`. `Layout` derives the dimensions with `Derive`. The item chamfers its caps and verifies its own
profile sketch; `Model.Profile` goes into `Movement.Sketches`. Check: solid; bounds `x ∈ [-32.0, 31.22]`, `y ∈ [-83.16, -13.75]`,
`z ∈ [11.25, 17.25]`; the eye's hole at `z 14.25` lies 1.9 mm round `Q`.

### 22. Spring Spacer and 23. Spring Anchor Pin

Spacer sketch (`z 17.5`): circle `r 3.7`, extrude 10.25; D press bore (body family: arc `r 2.85`, chord `x 2.60`; sketch `z 16.5`, extrude 12.25). Anchor pin sketch (plane
origin at `Q`, `z 10.25`): circle `r 1.5`, extrude 20.7. The front plate's socket tool (sketch at `Q`, `z 27`): circle
`r 1.55`, extrude 4, appended to the front plate's tool list (`spring.go`). Check: solids; `TestFrontPlateSpringAnchor`.

### 24. Winding Knob

Circle `r 9` at `G`, `z 35.30`, extrude 3; cut the keyed rear counterbore (`dTool`, arc `r 3.75`, chord `x 3.35`, at `z 34.30`,
extrude 3.0) and the D press socket (`z 34.30`, height 5).

### 25-26. Escape Rear Collar and Pallet Front Collar

Each ring is a disc sketch `r 3.7` (about `O` at `z 0.25`; about `P` at `z 24.75`), extruded 3.0, with the body-family D bore
cut through (sketch 1 mm below the ring, arc `r 2.85`, chord `NewVertical` at `x 2.60`, extrude 5.0), the pattern of the
Spring Spacer. Both rings are built where they stand and posed with their arbor (`poses()`: `O` by `-wheel`, `P` by
`-ψ`). They go in the container `Collars`. Check: solid; z extent 3.0; outer radius 3.7; `TestArborCollars`.

### 27-28. Crutch Collar and Knob Collar

Each ring is a keyed D: sketch `dTool` outline (arc `r 3.7`, chord `x 3.3`, flat on `+X`) about `P` (crutch) or the origin
(knob) at `z 34.25`, extruded 3.0, with the end-family D bore cut through (sketch at `z 33.25`, arc `r 2.55`, chord
`x 2.05`, extrude 5.0). The Crutch Collar is posed with the anchor (`P` by `-ψ`), the Knob Collar with the knob (`G` by
the arbor turn). Both go in the container `Collars`. Check: solid; rear face `z 34.25`; length 3.0; outer chord 3.3.

### 29-36. Bolts and nuts

`m2fastener.PlaceBolt` and `PlaceNut` on the four sites. Check: head top at `z 34`, nut inside its cavity, tip inside
the pillar hole, no overlap with a plate.

## Pose, containers and placement

Pose each body once as in "Motion and pose" (rotation about its axis, then `Placement`), then create the containers
of the Bodies table with `ChildNamed` and `AddBody`; `doc.VerifyContainers().Err()` must be nil. The 0.05° back-off of
the wheel avoids contact at the build pose. Check: `TestInventory`, `TestSketches` (three branch cases),
`TestPlacement`, `TestGearMesh`, `TestCoupling`, `TestFits`, `TestEdgeFinishing`, `TestPoseNoOverlap`.
