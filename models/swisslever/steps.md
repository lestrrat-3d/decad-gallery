# Swiss lever escapement Decad steps

`instructions.md` owns every dimension, formula and fit; this file maps them to Sketch and Decad operations. Emit the
package `models/swisslever`. Build every part in its local frame (Graham convention: the model frame at the part's
pose angle 0) and pose it once at the end. Section numbers below are the body numbers of the Bodies table.

## Shared sketch rules

- Create one `sketch.World` per `Build`. Each sketch gets its own plane from `world.CreatePlaneFromFrame`. A flat sketch
  has `U = +X`, `V = +Y` and its origin at the stated point and `z`; a meridian sketch has `U = +X`, `V = +Z` and its
  origin on the axis.
- After the last constraint call `Solve`, then `internal/sketchcheck.VerifyFullyConstrained`; append the sketch to
  `Movement.Sketches`, and only then select a profile. Profile selection: keep the one valid profile whose hole count is
  the one listed; fail when it is missing or not unique.
- `xRef`: one construction line per sketch from the origin to `X1 = (10, 0)` with `NewHorizontal` and
  `NewDistance(origin, X1, 10)`. It anchors every angle.
- Polar point `(r∠a)`: create the point, a construction line from the origin to it, `NewDistance(origin, q, r)` (skip it
  when the point lies on a radius-constrained arc about the origin) and `NewAngle(xRef, line, a)`.
- A point given in `(x, y)` that is a design input (an axis, a round centre) is a polar point of its own radius and angle.
  Do not fix computed points with H/V distances.
- `CreateArc(center, start, end)` runs counter-clockwise from start to end; every arc gets `NewRadius`.
- Round a corner with `sketch.CreateFillet(s, first, second, at, r)` only after both legs carry their own direction and
  position (angle or parallel, plus a distance or a point on them), as the Graham anchor does.
- Overlap every stacked or patterned union by 0.5 (`ov`), so unions cross faces.

Shared builder helpers (copy their behaviour from these rules, not from another package): `flat(what, x, y, z)`,
`meridian(what, x, y)`, `profile(s, holes, what)`, `extrude`, `revolve` (full turn about the sketch line `(0,0)-(0,1)`),
`cut`, `union`, `circleAt(s, r)`, `dLoop(s, r, flatX)` (arc `r` and a vertical chord at `x = flatX`, flat on `+X`),
`dTool(what, x, y, z, r, flatX, length)`, `discAt`, `finish(body, offsets, keep)` (one `edgefinish.Chamfer` along `+Z`,
skipped when the builder is `plain`), `boss(body, face, at, diameter, height)` (one `decad.Boss` with
`RootFillet` 0.5 mm).

## Exported API

`Params`, `DefaultParams`, `Validate`; `Layout(Params) (Geometry, error)`; `Motion(Params) (State, error)`;
`(*Geometry) StateAt`, `LeverAt`, `WheelAt`; `NewDocument(opts...)` (chain tolerance 0.1 mm first);
`Build(ctx, doc, p) (*Movement, error)`; `(*Movement) Named() []NamedBody`; the body name constants and `BoltName(i)`,
`NutName(i)`. `Build` rejects a nil context or document, validates, derives `Layout` and the `State`, and checks
`ctx.Err()` before it touches the document. Every error names the part and the step that failed.

## Layout and motion (no Decad)

`Layout` computes, in this order, every value of `instructions.md`:

1. Escapement: `τ`, `λ`, `α`, `d`, `r_m`, `P`, `B`, the six event angles, `I_L`, `I_R`, `C_E`, `C_X`, `R_h`, `X_E`, `X_X`,
   `F_E`, `F_X`, `u_E`, `u_X`, `φ_E`, `φ_X`; tooth 0's `T`, `H`, `L`, `K`; each stone's `C`, `F`, `G`, `Hs` (`G` is the
   intersection of the line from `F` along `rot(u, β_b)` with the line through `Hs` along `F - C`).
2. Each slot: the stone's four side lines moved outward by 0.05, intersected in turn (outward = away from the
   quadrilateral's centroid).
3. Lever outline: the slot sides moved outward a further 1.95; `VE1` = entry locking ∩ entry top; `VE2` = entry locking ∩
   entry impulse; `VE3` = entry impulse ∩ entry back; `VX1` = exit locking ∩ exit impulse; `VX2` = exit impulse ∩ exit
   back; `VX3` = exit back ∩ exit top; `VX4` = exit top ∩ exit locking. `JL`, `JR`, `r_c`, `K_R`, `K_L`, `r_n`, the gap
   rounds and the dart.
4. Fork action: `θ1`, `θ2`, the jump angle, `θ_d`, `θ3` (each by a 0.05° scan from the previous event, then bisection to
   1e-12 rad on the sign change); `ψ_g`; the corner freedoms; the crescent centre and radius.
5. Balance and hairspring: `SpringParams()` and `Derive`; `γ_s`, `γ_f`; the eye centre; `SpringRate`, the inertia and the
   period of Balance and hairspring.
6. Banking post centres, pillar centres and the fastener sites.

`WheelAt(psi, plus)` implements "Action of wheel and pallets"; `LeverAt(theta, plus)` implements "Pin-in-fork action";
`StateAt(t)` composes them with `θ = -Θ cos 2πt`. Each circle–circle and line–circle intersection returns an error when
it has no root (it never does inside the parameter range). The wheel angle of swing − is the swing + angle of the same
lever angle plus the exit table; keep the tooth indices `k_E = 2`, `k_X = 0`, and `k_E + 1` for the entry landing.

Check after this step: `TestValidate`, `TestLayout`, `TestMotion`, `TestActionAnalytic`, `TestErrorStudy`, `TestGuard`.

## Bodies

### 1. Back Plate

1. Sketch `back plate` (`z -6`, origin `O`): the rounded-rectangle outline (4 lines, each `NewHorizontal` or
   `NewVertical` and placed by one H or V distance from the origin; 4 corner arcs `NewRadius 10` and `NewTangent` to both
   neighbours); windows W1, W2, W4 (below), W3 left and right (each a rounded rectangle: 4 lines placed by H/V distances
   from the origin, 4 corner arcs `r 6` tangent to their lines); lip circles `r 5` at `O` (the origin) and at `P`, `B`
   (polar points, shared with the window hub arcs). Valid profile: 8 holes. Extrude 6.
2. `boss` six times on the face `decad.Faces(decad.FaceCreatedBy(decad.CapEnd(base))).Exactly(1)` (the plate's inner
   face at `z 0`; `base` is the extruded body): pillars, diameter 10, height 27, at `(±48, -40, 0)` and `(±48, 136, 0)`;
   banking posts, diameter 4, height 9.80, at `(±7.1564, 61.8279, 0)` (use `Layout`'s values).
3. One cut tool, the union of: pockets `r 7.05` at `O`, `P`, `B` over `z ∈ [-5, 1]`; per site `m2fastener.HoleTool` and
   `m2fastener.CavityTool`. Cut once.
4. `finish(body, {-6, 0}, keep)`: outer loops; inner loops of more than 4 edges (windows); single-edge circle loops of
   radius 7.05 or 5 (±0.01) centred within 0.01 of `O`, `P` or `B` (pocket rims on the inner face, lip rims on the rear
   face). The pillar and post root circles are single-edge loops of other radii and stay out.

Window loop W2 (W1, W4 and the front plate's W3 follow the same pattern): bottom line `y = 4` from the left corner arc
to `(-h, 4)`; hub arc `r 10` about `O` from `(-h, 4)` over `(0, 10)` to `(h, 4)` (`CreateArc(O, (h,4), (-h,4))`); bottom
line on to the right corner arc; right side `x = 45`; top line `y = 40.5714` from the right to `(h, 40.5714)`; hub arc
`r 10` about `P` from `(h, 40.5714)` under through `(0, 34.5714)` to `(-h, 40.5714)`; top line on; left side `x = -45`.
Corner arcs `r 6` tangent to their lines. Each straight edge is `NewHorizontal` or `NewVertical` and placed by
`NewDistancePointLine(origin, line, d)`, so a fillet keeps it; the hub arcs share their centre with the lip circle
(the origin, or the polar point `P` or `B`). Then `sketch.CreateFillet` radius 1.0 at the four junctions of a bar edge and
a hub arc. W1 has only its top hub arc (`O`), W4 only its bottom one (`B`), the front W3 has `P` at the bottom and `B` at
the top; the back W3 halves have none.

Check: solid; 6 posts present at their heights; `TestCentreLine`, `TestFasteners`; chamfer volume drop.

### 2. Front Plate

Sketch `front plate` (`z 27`, origin `O`): outline, W1, W2, W3, W4, lip circles at `O`, `P`, `B`; valid profile: 7 holes;
extrude 6. `boss` on `decad.Faces(decad.FaceCreatedBy(decad.CapStart(base))).Exactly(1)` (inner face `z 27`, outward
`-Z`): the hairspring stud, diameter 3.0, height 8.05, at `(-32.0106, 92.5714, 27)`. One cut tool: pockets `r 7.05` over
`z ∈ [26, 32]`; per site `m2fastener.HoleTool` and `m2fastener.PocketTool`. `finish(body, {27, 33}, keep)` as the back
plate. Check: stud tip at `z 18.95`; `TestHairspring` (eye hole round the stud).

### 3-8. Bearings

`ballbearing.Params605ZZ()` with axis `+Z`, lower caps at `z -5` and `z 27` for `O`, `P`, `B`. Append each model's
sketch.

### 9-11. Arbors

Meridian sketch per arbor (origin on its axis): the polygon of `instructions.md` (Arbors table). Each horizontal or
vertical line `NewHorizontal` or `NewVertical`, the axis edge by `NewHorizontalDistance(origin, p, 0)`, each corner by
an H distance (radius) and a V distance (`z`) from the origin; a cone is the slanted line between two such corners. Valid
profile: 0 holes; revolve. Flat tools: a span with no ramp is a box `x ∈ [flat, 3.5]`, `y ∈ [-3.5, 3.5]` over its `z`
span (`CreateRectangle` with one corner by H/V distance and its sides by distances between corners). A span with a ramp
is a meridian sketch through the axis with the polygon `(X, Z0) (X, Z1) (4.0, Z1 ± 1.732(4.0 - X)) (4.0, Z0)`, each
corner by H and V distance, extruded `Symmetric` 4.0 each way. Union the tools of one arbor, cut once. The Pallet Arbor
and the Balance Staff call the same builder.

Escape Arbor flats: `x 2.55` box `[0.20, 13.35]` plus the ramp past 13.35; `x 1.5` box `[33.25, 38.25]`. Pallet Arbor and
Balance Staff: the ramp behind 3.30 and the box `[3.30, 26.80]`.

Check: solid; diameter 5.0 at each seat; flat present mid-span and absent under each bearing; volume equals the revolve
less each section's circular segment beyond its flat; `TestSplitShafts`, `TestArborCollars`.

### 12. Escape Wheel

1. Sketch `escape disc` (`z 10.35`, origin `O`): circle `r 32`; five windows, window `k` between the spokes at
   `σ_k = 90° + 72° k` and `σ_{k+1}`: inner arc `r 10`, outer arc `r 27.5`, side lines `NewAngle(xRef, line, σ)` and
   `NewDistancePointLine(origin, line, 2.5)`, the four corners rounded with `sketch.CreateFillet`, radius 2.5. Valid
   profile: 5 holes. Extrude 3.
2. Sketch `escape tooth` (`z 10.35`, origin `O`): `T` by `NewHorizontalDistance(origin, T, 0)` and
   `NewVerticalDistance(origin, T, 38.6)`; `H` a polar point `(40.1546∠93°)` (use `Layout`'s `R_h`); construction lines
   `T→O` and `H→O`; `L` with `NewDistance(origin, L, 31.5)` and `NewAngle(T→O, T→L, -ρ)`; `K` with
   `NewDistance(origin, K, 31.5)` and `NewAngle(H→O, H→K, -χ)` (signs chosen so `L` and `K` fall at the `instructions.md`
   points); lines `L–T`, `T–H`, `H–K`, `K–L`. Valid profile: 0 holes. Extrude 3.
3. `decad.PatternCopies(tooth, decad.CircularPattern{Center: O, Axis: +Z, Count: 15})`; `Union(disc, tooth, copies...)`.
4. Root fillets: `decad.Fillet(ctx, wheel, decad.Edges(decad.ParallelTo(+Z), decad.Concave()).Exactly(30), 1.4 mm)`, as
   `models/spurgear` rounds its roots.
5. D press bore: `dTool` body family (arc `r 3.35`, chord `x 2.60`) at `z 9.60`, length 4.5; cut.
6. `finish(body, {10.35, 13.35}, keep inner loops of more than 4 edges)` (the five windows).

Check: solid; section at `z 11.85` has 7 loops, 15 toes at radius 38.6 and 15 heels at 40.1546 (± bound); the roots read
a 1.4 round; `TestEscapementAction`.

### 13. Lever

1. Sketch `lever` (`z 3.30`, origin `P`): `xRef`; the two slot quadrilaterals (holes): `C'` and `F'` polar points; the
   locking side `C'→H'` with `NewAngle(xRef, line, ∠u)` and `NewDistance(C', H', |H' - C'|)`; the back side `F'→G'` with
   `NewAngle(xRef, line, ∠u + β_b)`; the top `G'–H'` with `NewParallel(top, C'F')`. The outline (band S version):
   - `JR`, `JL` on the arm sides; each arm side `NewVertical` with `NewDistancePointLine(origin, side, 3.0)`; `JR` and
     `JL` by `NewVerticalDistance(origin, J, 12)`.
   - Corner rounds: arcs about `K_R`, `K_L` (polar points), `NewRadius r_c`, each tangent to its arm side
     (`NewTangent`); each runs over the top from the arm side to its inner end on `y = K.y`.
   - Gap rounds: arcs `NewRadius 0.53384`, each `NewTangentCircles` to its corner round and `NewTangent` to its dart side,
     running below `y = K.y` from the corner round's inner end to the dart side.
   - Dart sides: `NewVertical`, `NewDistancePointLine(origin, side, 0.6)`; tip arc about the polar point `T_d` (0,
     42.15), `NewRadius 0.6`, tangent to both sides.
   - Frame lines: `JL→VE1`, `VE1→VE2`, `VE2→VE3`, `VE3→VX1`, `VX1→VX2`, `VX2→VX3`, `VX3→VX4`, `VX4→JR`. Each line that
     follows a slot side is `NewParallel` to it with `NewDistancePointLine(slot corner, line, 1.95)`. The two offset lines
     that meet the outline only at vertices (entry top through `VE1`, entry back through `VE3`, exit locking side through
     `VX1` and `VX4`) are construction lines with the same two constraints, and each such vertex is `NewPointOnLine` of
     both its lines. The lower line
     `VE3→VX1` and the upper lines `JL→VE1`, `VX4→JR` are fixed by their end points.
   - `sketch.CreateFillet` radius 3.0 at `JL` and `JR`, radius 1.5 at `VX1`.
   Valid profile: 2 holes. Extrude 6.5 (`z 3.30` to `9.80`).
2. Notch tool, sketch `lever notch` (`z 5.80`, origin `P`): the notch semicircle `CreateArc(N_c, (-r_n, K.y), (r_n, K.y))`
   (centre `N_c` a polar point, `NewRadius r_n`, running under the centre); lines from `(r_n, K.y)` to `(1.0, 44.0)`, to
   `(-1.0, 44.0)`, to `(-r_n, K.y)`, the upper corners by H/V distance. Valid profile: 0 holes. Extrude 5.0; cut. This
   removes the dart and the filled notch above `z 5.80`.
3. Dart root: `decad.Fillet(ctx, lever, decad.Edges(decad.Concave(), decad.Circular()).Exactly(1), 0.6 mm)`: the
   semicircular edge where the notch wall meets the top of band S at `z 5.80`.
4. D press bore: `dTool` body family at `(P, z 2.55)`, length 8.0; cut.
5. `finish(body, {3.30, 9.80}, outer loops)`.

Check: solid; section at `z 7.8` holds the notch semicircle and both rounds and no dart; section at `z 4.8` holds the
dart to `y` 42.75 (lever frame) and no notch below `y` `K.y`; slots run through; `TestForkAction`, `TestStonesKeyed`.

### 14-15. Entry Stone, Exit Stone

Sketch `entry stone` / `exit stone` (`z 3.30`, origin `P`): `C` and `F` polar points; `Hs` with
`NewAngle(xRef, C→Hs, ∠u)` and `NewDistance(C, Hs, 7.0)`; `G` with `NewAngle(xRef, F→G, ∠u + β_b)` and
`NewParallel(G–Hs, C–F)`. Valid profile: 0 holes. Extrude 10.6. Check: section at `z 11.85` within the bound of the four
corners; `TestStonesKeyed`.

### 16. Safety Roller

Built about the origin with its bore flat on local `+X`, posed onto `B`. Sketch `safety roller` (`z 3.30`, origin
`(0, 0)`): the roller arc `r 6.444` and the crescent arc (centre the polar point `(9.0∠a_p)`, `a_p = -90° - γ_f`,
`NewRadius 4.03`), meeting at the two lips; `sketch.CreateFillet` radius 0.4 at both lips. Valid profile: 0 holes.
Extrude 7.5. D press bore `dTool` body family at `z 2.55`, length 9.0; cut. Check: wall 1.62 between the crescent and the
bore; `TestPressFitWalls`.

### 17. Roller Table

Sketch `roller table` (`z 10.85`, origin `(0, 0)`): circle `r 11`. Extrude 2.5. `boss` on
`decad.Faces(decad.FaceCreatedBy(decad.CapStart(table))).Exactly(1)` (rear face, outward `-Z`): the impulse pin,
diameter 3.0, height 4.5, at local `(8.5∠a_p, 10.85)`. D press bore `dTool` body family at `z 10.10`, length 4.0; cut.
Check: pin tip at `z 6.35`; pin root round present.

### 18. Balance Wheel

Sketch `balance wheel` (`z 13.40`, origin `(0, 0)`): circle `r 42`; four windows, window `k` between the arms at
`45° + 90° k` and `135° + 90° k`: hub arc `r 7`, rim arc `r 36`, side lines `NewAngle(xRef, line, arm angle)` and
`NewDistancePointLine(origin, line, 2.0)`, the four corners `sketch.CreateFillet` radius 2.0. Valid profile: 4 holes.
Extrude 5. `boss` on `CapEnd` (front face `z 18.40`): diameter 14, height 1.0, at `(0, 0, 18.40)`. D press bore `dTool`
body family at `z 12.65`, length 7.5; cut. `finish(body, {13.40, 18.40}, outer loops or inner loops of more than 4
edges)`. Check: mass 13.1 g at 1.24 g/cm³ (± 2 %); inertia within 2 % of the `Layout` value.

### 19. Hairspring

`spiralspring.Build(ctx, doc, SpringParams())` with `Placement` origin `(0, 92.5714, 19.65)` (use `Layout`'s `B`),
`U = (cos γ_s, sin γ_s, 0)`, `V = (-sin γ_s, cos γ_s, 0)`. Append `Model.Profile`. Check: solid; `TestHairspring`.

### 20. Hairspring Retainer

Circle `r 5.0` at `(0, 0, 22.90)`, extrude 3.0; D press bore `dTool` body family at `z 22.15`, length 4.5.

### 21-22. Winding Knob and Knob Collar

Knob: circle `r 11` at `(0, 0, 34.30)`, extrude 3; keyed counterbore `dTool` (arc `r 3.75`, chord `x 3.35`) at `z 33.30`,
length 3.0; D press socket `dTool` end family at `z 33.30`, length 5.0. Knob Collar: `dTool` outline (arc `r 3.7`, chord
`x 3.3`) at `z 33.25`, extruded 3.0, with the end-family bore cut through (sketch at `z 32.25`, length 5.0).

### 23-30. Bolts and nuts

`m2fastener.PlaceBolt` and `PlaceNut` on the four sites: head `(px, py, 33)`, `Dir -Z`, `Corner +X`, length 10, cavity
8. Append each model's sketches.

## Fasteners: why the nuts are embedded

The Fasteners rules embed a nut only when no other solution exists. The pillars belong to the back plate, so each bolt
clamps only the front plate to its pillar: the head sits flush in the front plate's front face and the nut must lie in
the pillar, behind 1 mm or more of pillar material, because the tightened bolt pulls the nut toward the head. A hex
pocket open to the pillar's top face would put the nut on the head's side of that material: the bolt would pull the nut
out against the front plate and clamp nothing. The only face beyond the material is the back plate's rear face, 39 mm
from the head; no standard length (6 to 25 mm) reaches it, and splitting the pillar into shorter bolts from each end
puts each nut inside the pillar again. A side-entry hex slot would open the pillar's side, but `internal/m2fastener` has
no such tool and the rules forbid sizing a pocket by hand. So each nut sits in an enclosed `CavityTool` cavity, dropped in
during the print pause at 31 mm. This is the Graham model's arrangement.

## Pose, containers and placement

Pose each body once as in `instructions.md` (Motion and pose), then `Placement`. Bodies built about the origin (safety
roller, roller table, balance wheel, retainer, knob, Knob Collar) are turned about the origin and then moved to their
axis; bodies built at their axis (plates, arbors, wheel, lever, stones) are turned about their
axis. Create the containers of the Bodies table with `ChildNamed`, move each body with `AddBody`, and require
`doc.VerifyContainers().Err()` to be nil. Check: `TestInventory`, `TestSketches`, `TestPlacement`, `TestFits`,
`TestEdgeFinishing`, `TestPoseNoOverlap`, `TestCorners`, `TestPressFitWalls`, `TestInstallability`.

## Decad operations to prove first

Three operations are new to this repository or used in a new way. Run each smallest case first; when decad refuses it,
stop and hand off that case as the blocker (`docs/model-pipeline.md` step 4). Do not replace the geometry.

| Operation | Smallest real case |
|---|---|
| `decad.Boss` with `RootFillet` on an extruded cap | the Roller Table: the `r 11` × 2.5 disc and the one pin boss |
| several `decad.Boss` calls on one cap, selected with `FaceCreatedBy(CapEnd(base))` after the first | the back plate's extrude with two of its bosses |
| `decad.Fillet` of one concave circular edge between a plane and a cylinder, ending where the cylinder meets tangent walls | the lever: base extrude, notch cut, dart-root fillet |

The escape wheel's root fillet uses the selection and call of `models/spurgear` on a union of a disc and patterned teeth.
