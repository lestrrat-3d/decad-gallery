# Clamp arm Decad steps

Follow `instructions.md` for the API, the dimensions, the fits and the checks. Every error message starts with
`clamparm: `. All dimensions are in mm, angles in degrees. The package has seven Go files:

| File | Owns |
| --- | --- |
| `params.go` | constants, `Params`, `DefaultParams`, `Validate`, `Derive`, `Dimensions`, the pose chain and the sites |
| `document.go` | `NewDocument` and the shared run helpers (sketch, profile, extrude, outline, slot and tongue tools) |
| `knob.go` | the lobed outline, `Knob`, `ScrewKnob`, the part-mark tool |
| `clamp.go` | the clamp frame and the pad |
| `arm.go` | the joint tools, the swivel, the link and the head |
| `adapter.go` | the interface constants, `InterfaceRecessTool` and the placeholder adapter |
| `clamparm.go` | package doc, container names, `Part`, `Parts`, `Model`, `BuildParts`, `Build` |

Each section below is one part and names its file. Section 13 records decad notes found while writing this file.

## Conventions

### Sketches

Every sketch is on a fresh `sketch.NewWorld()` plane from an `r3.Frame`, appended to the run's `sketches`, solved, and
checked with `sketchcheck.VerifyFullyConstrained` before its profile is used. Every sketch is anchored to its origin
(a dimensioned point, a point on a line through the origin, or an origin-centred arc) and dimensions its geometry. No
sketch fixes every computed point. Profile selection: `profile(s, holes)` takes the one valid profile with exactly
`holes` holes and fails on none or on several. A sketch with a hole circle has two valid profiles, the ring with one
hole and the hole's disc, so the hole count is always given.

Idioms, each checked to solve fully constrained with one valid profile and the expected area:

- **Axis-aligned outline.** `CreatePoint` for each vertex, `CreateLine` between consecutive vertices. Each line gets
  `NewHorizontal` or `NewVertical` and `NewDistancePointLine(origin, line, |c|)`, where `c` is its coordinate. A line
  with `c = 0` gets `NewPointOnLine(origin, line)` instead, because a corner rounded by `CreateFillet` loses its point's
  constraints. Round a corner with `sketch.CreateFillet(s, previousLine, nextLine, cornerPoint, r)`; it works for convex
  and concave corners.
- **Round-ended slot cutter** (`slotCutter(cx, xEnd, z0)`): points `a = DimensionPoint(xEnd, +4.26)` and `d =
  DimensionPoint(xEnd, -4.26)`; `b = CreatePoint(cx, +4.26)` and `c = CreatePoint(cx, -4.26)` each with
  `NewHorizontalDistance(origin, ., cx)`; `centre = DimensionPoint(cx, 0)`; `arc = CreateArc(centre, b, c)`
  (counter-clockwise from top to bottom, through -X) with `NewRadius(arc, 4.26)`; lines a-b, c-d, d-a. Placing b and c by
  height instead would meet the circle at a tangent, a singular solve.
- **Half-circle end** (head and swivel plan): `top = CreatePoint(0, 10)` and `bottom = CreatePoint(0, -10)` each with
  `NewHorizontalDistance(origin, ., 0)`; `arc = CreateArc(origin, top, bottom)` with `NewRadius(arc, 10)`; the side
  lines horizontal; the far end vertical with `NewDistancePointLine`. The side lines' heights come from the circle.
- **Stadium** (link): `CreateSlot(0, 0, L, 0, 10)` with `NewCoincident(slot.C1, origin)`, `NewVerticalDistance(origin,
  slot.C2, 0)`, `NewHorizontalDistance(origin, slot.C2, L)` and `NewRadius(slot.A1, 10)`.
- **Revolved cutter or boss**: `fastenertool.RoundedRevolve(ctx, doc, f, pts)` with `f.U` along the axis and `f.V` the
  radius; `pts` are `(along, radius)`, every vertex placed by `sketchcheck.DimensionPoint`.

### Features

- `decad.Extrude(ctx, doc, s, profile, decad.Distance{D: units.Millimeters(d), Dir: decad.Along})` goes along the sketch
  plane's normal `U x V`; `decad.Against` goes the other way; `decad.Symmetric{D: units.Millimeters(half)}` goes `half`
  each side.
- Cuts use `decad.Cut(ctx, body, tool)`, one tool at a time, in the order each part lists.
- Edge finishing uses `internal/edgefinish` with size `edgefinish.MinChamfer` (0.5). `Chamfer(ctx, body, normal, offsets,
  keep, size)` and `ChamferExcept(ctx, body, normal, offsets, keep, skip, size)` chamfer every kept loop of every planar
  face whose normal is `+-normal`, whose plane lies at `offsets[i]` along `normal` from the world origin, and whose net
  area is at least 400. `keep` is `edgefinish.Outer` everywhere here, so hole rims stay. Both return `(body, count,
  err)`; the count must be at least 1. `skipParallel(axis)` is the `skip` that returns true for a `decad.Line3` edge
  whose end points differ along `axis` only; it leaves out the edge where a face meets a tangent round.
- Every part is solid (`body.IsSolid()`) before it is placed, and is placed once, last, with
  `placement.Place(ctx, body, &frame)`.

### Dependencies

- `internal/fastenertool`: `Cylinder(ctx, doc, f, radius, length)` (circle on `f`, extruded along `f`'s normal);
  `HexPrism(ctx, doc, f, cornerRadius, depth)` (first corner on `f`'s U, sharp corners); `RoundedRevolve(ctx, doc, f,
  pts)`; `QuarterRound(t, r, radius, start, chords)`. Each builder returns `(*decad.Body, *sketch.Sketch, error)`.
- `models/m5nut`: ISO 4032 M5, `Params{Placement *r3.Frame}`, `Build(ctx, doc, p) (*Model, error)`, `Model{Body,
  Hexagon, LowerChamfer, UpperChamfer *sketch.Sketch}`. Local frame: lower face centre, +Z up, corners on X. Thickness
  4.7, across flats 8, `AcrossCorners()` 9.2376.
- `models/m5bolt`: ISO 4017 M5 hex bolt, `Params{Length, Placement}`, `StandardLengths()` (16, 20, 25, 30, 35, 40, 45,
  50, 60, 70, 80), `Build` returning `Model{Body, Head, Shank}`. Local frame: centre of the head's underside, head along
  +Z, shank along -Z, head corners on X. Head height 3.5, shank diameter 5.
- `internal/m5fastener`: constants `HeadClearance` 0.3, `NutClearance` 0.2, `HoleRadius` 2.8, `HeadPocketRadius` 4.9188,
  `HeadPocketDepth` 3.5, `NutPocketRadius` 4.8188, `NutCavityHeight` 5.1, `PressPocketRadius` 4.6688, `PocketRound` 0.6,
  `PressPocketRound` 0.5, `Stock` 1. `Site{Head, Dir, Corner r3.Vec; Length float64}` with `Validate`, `At(t)`, `Across(t)` (frame at `t` from
  `Head`, normal `Dir`, U `Corner`) and `Tip()` (`HeadPocketDepth + Length` from `Head`). `Head` is the centre of the
  head's top face. Pocket tools take a floor frame `f` (origin at the pocket floor's centre, +Z toward the opening, U the
  hex corner direction), extrude from the floor out through the opening plus `Stock`, and fillet the cutter's
  `CapStart` outer loop by `PocketRound`: `HeadPocketTool(ctx, doc, f)` (pocket depth 3.5), `NutPocketTool(ctx, doc, f,
  depth)` (`depth` is the pocket's depth from the floor to the face it opens on; the extrusion is `depth + Stock`),
  `PressPocketTool(ctx, doc, f)` (depth 3.5, floor rounded by `PressPocketRound` 0.5). `HoleTool(ctx, doc, site)` is a cylinder of radius `HoleRadius` from `Stock`
  before `Head` to `Stock` past the tip. `RoundedSocketTool(ctx, doc, f, radius, depth, round)` is a blind round socket
  whose floor corner is rounded by `round` with 8 chords: `f`'s origin is the floor's centre, +Z points toward the
  opening, and the socket runs `depth + Stock` along +Z. `PlaceBolt(ctx, doc, site)
  (*m5bolt.Model, error)` puts the bolt's head top face at `site.Head`: its frame origin is `site.At(3.5)`, U is
  `Corner` and V is `(-Dir) x Corner`. `PlaceNut(ctx, doc, f) (*m5nut.Model, error)` puts the nut's lower face on `f`
  with its corners on `f`'s U.
- `pocketFrame(site)` (a `params.go` helper) is the head pocket's floor frame: `r3.NewFrame(site.At(3.5), site.Corner,
  site.Dir.Scale(-1).Cross(site.Corner))`, whose normal is `-Dir`, toward the opening.

## 1. Params, validation and dimensions (params.go)

1. Constants, each with its value in a comment. The exported ones are those of `instructions.md` "API": `Section` 20,
   `EndRadius` 10, `TongueThickness` 8, `JointFit` 0.26, `ForkGap` 8.52, `FinThickness` 5.74, `TongueShoulder` 20,
   `ShoulderRound` 4, `SlotDepth` 26, `SlotRound` 4.26, `J1Max` 125, `JointMax` 115, `ShoulderMin` -60, `WristMin` -110,
   `WristMax` 90, `OpeningMin` 10, `OpeningMax` 40, `LinkLengthMin` 80, `LinkLengthMax` 150, `FrameWidth` 30,
   `PanAxisX` 30, `ScrewX` 25, `SwivelLength` 70, `StudOffset` 40, `HeadLength` 55, `KnobHeight` 12, `ScrewKnobHeight`
   10, `PocketRound` 0.6, `MarkRadius` 8, the interface constants and the container names. The unexported ones are
   `swivelRise` 14.26 (J1's swivel bottom face over the desk top), `jawBottom` -73.5, `jawTop` -59.5, `finTop` 28.26,
   `gapLow` 14, `gapHigh` 22.52, `gapEnd` 46, `jawEnd` 50, `padLength` 12, `padSocketRadius` 2.85, `padSocketDepth` 8,
   `padSocketRound` 0.5, `jawNutFloor` -64.6, `jawNutDepth` 5.1, `knobNutFloor` 3, `knobNutDepth` 9, `pressFloor` 6.5,
   `markDiameter` 1.5, `markFirst` 30, `markStep` 25, `lobeRadius` 4, `lobeCentre` 11, `valleyRadius` 3, `valleyCentre`
   13.857, `jointBolt` 25, `panBolt` 35, `studBolt` 40, `screwBolt` 70.
2. `Params` has `LinkLength`, `Opening`, `Pan`, `Shoulder`, `Elbow`, `Wrist` (`float64`) and `Placement *r3.Frame`.
   `DefaultParams()` returns 120, 25, 0, 70, -90, 20 and nil.
3. `Derive` checks in this order and returns the first failure:
   1. each of the six floats is finite, in field order: `clamparm: <Field> must be finite`;
   2. `clamparm: LinkLength must be from 80 to 150 mm, got %g`;
   3. `clamparm: Opening must be from 10 to 40 mm, got %g`;
   4. `Pan`, `Shoulder`, `Elbow`, `Wrist` in that order, each `clamparm: <Field> must be from <min> to <max> degrees, got
      %g` with the limits of "Params and defaults" (`Pan` +-`J1Max`, `Shoulder` `ShoulderMin` to `JointMax`, `Elbow`
      +-`JointMax`, `Wrist` `WristMin` to `WristMax`);
   5. `Placement` is nil or `Placement.IsValid()`: `clamparm: Placement is not a valid frame`.
   `Validate` returns `Derive`'s error. `Derive` makes no geometry call.
4. Pose chain with `r3` transforms. `T(v)` is `r3.Translation`; `Rz(a)` is `r3.Rotation((0,0,1), units.Degrees(a))`;
   `R(a)` is `r3.Rotation((0,-1,0), units.Degrees(a))`, which lifts local +X toward +Z. `compose(t1, ..., tn)` is the
   product `t1 . ... . tn`, which applies `tn` first: `tn.Then(...).Then(t1)`. `frameOf(t)` is `r3.NewFrame(t.Apply(0),
   t.ApplyDir(X), t.ApplyDir(Y))`.
   `cs(a), sn(a)` are the cosine and sine of `a` degrees, exactly `(1,0)`, `(0,1)`, `(-1,0)`, `(0,-1)` when `a` is a
   multiple of 90 and `math.Cos`, `math.Sin` of `a` in radians otherwise. `R(a)` is built as `r3.FromFrame` of
   `NewFrame(0, (cs(a), 0, sn(a)), (0,1,0))`, which equals the `-Y` rotation and is exact at multiples of 90. Each
   pose is one translation to the part's joint and one rotation by the summed joint angle, never a product of
   rounded rotations, so a level head (`Shoulder + Elbow + Wrist` = 0) is the identity rotation. Let `c1 = Shoulder`,
   `c2 = Shoulder + Elbow` and `c3 = Shoulder + Elbow + Wrist`.
   - `S = compose(T(30,0,swivelRise), Rz(Pan))`; `base = compose(S, T(70,0,10))`;
   - `A = compose(base, R(c1))`; `B = compose(base, T(L cs(c1), 0, L sn(c1)), R(c2))`; `H = compose(base, T(L cs(c1),
     0, L sn(c1)), T(L cs(c2), 0, L sn(c2)), R(c3))`; `I = compose(H, T(40,0,10))`. The frames equal the chained
     `A . T(L,0,0) . R(Elbow)` form to within 3e-14 mm;
   - `Dimensions.Swivel`, `LinkA`, `LinkB`, `Head`, `Interface` are `frameOf` of those, in the model's world frame
     before `Placement`.
5. `Joints[0] = (30, 0, 14.26)`; `Joints[1..3]` are the origins of `A`, `B` and `H`. `Axes[0] = (0,0,1)`; `Axes[1..3]` are
   `S`, `A` and `B` applied to `(0,1,0)` as directions (the fork's +Y, the knob side).
6. Fixed local sites (`params.go`, unexported): `forkSite(lf, z)` is `Site{Head: (lf,-10,z), Dir: (0,1,0), Corner:
   (0,0,1), Length: 25}`; `tongueSite(z)` is `forkSite(0, z)`; `studLocal` is `Site{Head: (40,0,-20), Dir: (0,0,1),
   Corner: (1,0,0), Length: 40}`; `panLocal` is `Site{Head: (0,0,-14.26), Dir: (0,0,1), Corner: (1,0,0), Length: 35}`,
   the swivel's J1 hole in its own frame.
7. World sites, each `Site{Head, Dir, Corner}` moved by `Apply` for `Head` and `ApplyDir` for `Dir` and `Corner`:
   `PanSite = {(30,0,0), (0,0,1), (0,1,0), 35}`; `JointSites[0] = S` applied to `forkSite(70, 10)`; `JointSites[1] = A`
   applied to `forkSite(L, 0)`; `JointSites[2] = B` applied to `forkSite(L, 0)`; `StudSite = H` applied to `studLocal`;
   `ScrewSite = {(25,0,-o-77.5), (0,0,1), (0,1,0), 70}`. `Derive` returns each site's `Validate()` error as
   `clamparm: <name>: %w`.
8. Frames, in the same world frame (`NewFrame(origin, U, V)`, normal `U x V`):
   - `KnobFrames[0] = PanSite.Across(28.26)`; `KnobFrames[1+i] = JointSites[i].Across(20)`. Each U is its site's
     `Corner` and each normal is the site's `Dir`.
   - `ScrewKnobFrames[0] = NewFrame((25,0,-o-67.5), (0,1,0), (1,0,0))` (normal -Z); `ScrewKnobFrames[1]` is `H` applied
     to the frame `NewFrame((40,0,-10), (1,0,0), (0,-1,0))` (normal -Z in head coordinates). Each U is its site's
     `Corner`.
   - `KnobNutFrames[0] = PanSite.Across(31.26)`; `KnobNutFrames[1+i] = JointSites[i].Across(23)`.
   - `JawNutFrame = NewFrame((25,0,-64.6), (0,1,0), (-1,0,0))`; `AdapterNutFrame` is `I`'s frame moved 4.7 along its
     normal; `PadFrame = NewFrame((25,0,-o), (1,0,0), (0,1,0))`.
   - `ScrewKnobGap = o - 6`.
9. `Dimensions` is returned by value. Nothing in `Derive` touches a document.

## 2. Document and run helpers (document.go)

1. `NewDocument(opts ...decad.DocumentOption)` returns `decad.New(append([]decad.DocumentOption{
   decad.WithChainTolerance(units.Millimeters(0.01))}, opts...)...)`. The chain tolerance matters for every cut that
   follows a union or an intersection (head, swivel).
2. `run` holds `ctx`, `doc`, `p Params`, `dm Dimensions` and `sketches []*sketch.Sketch`. One `run` builds one part.
3. Helpers on `run`: `sketchOn(f)`, `profile(s, holes)`, `extrude(s, holes, d)`, `extrudeAgainst(s, holes, d)`,
   `extrudeSymmetric(s, holes, half)`, `outline(s, pts, radii)` (the axis-aligned idiom; `radii[i] == 0` keeps corner `i`
   sharp), `cuts(body, tools...)`, `frame(origin, u, v)` (wraps `r3.NewFrame` with the error), `slotCutter`,
   `tongueCutter`, `holeTool(site)`, `pocketTool(site)`, `mark(h, angles...)`, `skipParallel`.
4. `slotCutter(cx, xEnd, z0)`: sketch on `frame((0,0,z0), X, Y)`, the round-ended slot idiom of "Conventions", extruded
   22 along +Z (`z0` to `z0 + 22`).
5. `tongueCutter(side)` (`side` is +1 or -1): sketch on `frame((0,0,-11), X, Y)`. Outline `(-11, 4s)`, `(20, 4s)`, `(20,
   11s)`, `(-11, 11s)` with `s = side`; `radii` 0, 4, 0, 0, so corner `(20, 4s)` takes a round of radius 4 whose
   centre is `(16, 8s)`. Lines: `y = 4s` (distance 4), `x = 20`, `y = 11s`, `x = -11` (distance 11). Extrude 22 along +Z.
   The cut leaves the tongue `|y| <= 4` for `x < 20` with the shoulder rounds of radius 4.
6. `holeTool(site)` is `m5fastener.HoleTool(ctx, doc, site)`; the sketch is appended. `pocketTool(site)` is
   `m5fastener.HeadPocketTool(ctx, doc, pocketFrame(site))`; the sketch is appended.
7. `mark(h, angles...)` builds one cutter per angle `a`: sketch on `frame((0,0,h+1), X, Y)`, one circle whose centre
   is `sketchcheck.DimensionPoint(s, 8 cos a, 8 sin a)` with `NewRadius(circle, 0.75)`, extruded `decad.Against` by
   `h + 2`. Each hole has its own sketch. The angles are measured counter-clockwise seen from the outer face.
8. Optional sketches: none. Every sketch listed below exists for every valid `Params`. The part marks are the only
   sketches whose count differs by part: 1 for the knob, 2 for the screw knob.

## 3. Knob (knob.go)

Local frame: origin at the inner face's centre on the bolt axis, +Z outward, the hex corner on +X.

1. `lobedOutline(s)` on a sketch whose frame is `frame((0,0,0), X, Y)`, shared with the screw knob. For `k` = 0..5, `a_k =
   60 k` degrees, `L_k = DimensionPoint(11 cos a_k, 11 sin a_k)` (lobe centre) and `V_k = CreatePoint(13.857 cos(a_k + 30),
   13.857 sin(a_k + 30))` (valley centre, left free). The tangent points sit on the line between neighbouring centres:
   `P_k` is 4 from `L_k` toward `V_k` and `Q_{k+1}` is 4 from `L_{k+1}` toward `V_k` (indices mod 6); create them with
   `CreatePoint`, as the shared end points of the arcs. Arcs, all counter-clockwise: `lobe_k = CreateArc(L_k, Q_k, P_k)`
   and `valley_k = CreateArc(V_k, Q_{k+1}, P_k)`. Constraints: `NewRadius(lobe_k, 4)`, `NewRadius(valley_k, 3)`,
   `NewTangentCircles(lobe_k, valley_k, false)` and `NewTangentCircles(valley_k, lobe_{k+1}, false)` (external
   tangency). Lobe tips reach radius 15 and valley floors 10.857. Net area 545.5 mm^2 before the hole.
2. Sketch `knob`: the outline and `CreateCircle(origin, 2.8)` with `NewRadius(circle, 2.8)`. Profile with 1 hole
   (`profile(s, 1)`, net area 520.9). Extrude 12 along +Z.
3. `edgefinish.Chamfer(ctx, body, (0,0,1), []float64{0, 12}, edgefinish.Outer, 0.5)`: both cap outlines, 24 edges.
4. Cut the nut pocket: `m5fastener.NutPocketTool(ctx, doc, f, 9)` with `f = frame((0,0,3), (1,0,0), (0,1,0))`. Corner
   radius 4.8188, floor at z = 3, open to the outer face, floor outline rounded 0.6.
5. Cut the part mark, one hole at 30 degrees: `mark(12, 30)`.
6. Return `Part{Body, Frame: placed KnobFrames[i], PrintUp: placed +Z, Sketches}`. Require solid. Four knobs, J1 to J4.
7. Each knob records 3 sketches: the outline, the nut pocket's hexagon and the mark.

## 4. Screw knob (knob.go)

Local frame as the knob. Used for the clamp screw (`ScrewKnobFrames[0]`) and the stud (`ScrewKnobFrames[1]`).

1. Sketch `screwknob` as the knob's: `lobedOutline` and the hole circle; `profile(s, 1)`; extrude 10 along +Z.
2. `edgefinish.Chamfer(ctx, body, (0,0,1), []float64{0, 10}, edgefinish.Outer, 0.5)`.
3. Cut the head pocket: `m5fastener.PressPocketTool(ctx, doc, f)` with `f = frame((0,0,6.5), (1,0,0), (0,1,0))`: corner
   radius 4.6688, depth 3.5, floor at z = 6.5, open to the outer face, floor outline rounded 0.5 (`PressPocketRound`; the
   Task 5 contract: `PressPocketTool` now uses `PressPocketRound` 0.5, not `PocketRound`). The head ends flush.
4. Cut the part marks, two holes: `mark(10, 30, 55)` (two sketches, 25 degrees apart).
5. Return the `Part` as the knob, `PrintUp` placed +Z. Wall round the press pocket is at least 1.6.

## 5. Clamp frame (clamp.go)

Local frame is the world frame before `Placement`. `PrintUp` is (0, 1, 0): the part prints on its -Y face.

1. Sketch `frame` on `frame((0,0,0), X, Z)` (normal -Y), an outline with these vertices counter-clockwise, `(x, z)`,
   radius in brackets: `(-14,-73.5)` [2], `(45,-73.5)` [2], `(45,-59.5)` [2], `(0,-59.5)` [7], `(0,0)` [7], `(50,0)` [2],
   `(50,14)` [2], the slot's bottom end `S1 = (-0.74, 14)`, the slot's top end `S2 = (-0.74, 22.52)`, `(46,22.52)` [2],
   `(46,28.26)` [2], `(-14,28.26)` [2]. Construction:
   - lines between consecutive vertices except S1 to S2, placed by the axis-aligned idiom: `x = 0` (from `(0,-59.5)` to
     `(0,0)`) and `z = 0` (from `(0,0)` to `(50,0)`) use `NewPointOnLine(origin, line)`;
   - the lines S1's predecessor `(50,14)` to S1 and S2 to `(46,22.52)` are horizontal only; their heights come from the
     slot's round;
   - `centre = DimensionPoint(-0.74, 18.26)`; `slot = CreateArc(centre, S2, S1)` (counter-clockwise from the top end,
     through -X) with `NewRadius(slot, 4.26)`; S1 and S2 each get `NewHorizontalDistance(origin, ., -0.74)`;
   - `CreateFillet` at each vertex with a radius in brackets. The two radius-7 corners are the jaw roots.
   Net area 2998.0 mm^2, one valid profile, no hole.
2. `extrudeSymmetric(s, 0, 15)`: the frame is 30 wide, y -15 to 15.
3. `edgefinish.Chamfer(ctx, body, (0,1,0), []float64{-15, 15}, edgefinish.Outer, 0.5)`: both Y cap outlines, 44 edges.
4. Cuts, in this order:
   1. J1 hole: `holeTool(PanSite)`;
   2. J1 head pocket: `pocketTool(PanSite)`, floor at z = 3.5, open to the top jaw's underside (z = 0);
   3. clamp screw hole: `fastenertool.Cylinder(ctx, doc, frame((25,0,-74.5), X, Y), m5fastener.HoleRadius, 16)`;
   4. jaw nut pocket: `m5fastener.NutPocketTool(ctx, doc, f, 5.1)` with `f = frame((25,0,-64.6), (0,1,0), (-1,0,0))`
      (U is +Y), open to the bottom jaw's top face at z = -59.5.
5. Return `Part{Body, Frame: Placement or identity, PrintUp (0,1,0)}`. No edge finishing after the cuts: hole and pocket
   rims are runs under 20 mm.
6. Failure checks: each cut returns a solid; the body's bounds are x -14 to 50, y -15 to 15, z -73.5 to 28.26.

## 6. Pad (clamp.go)

Local frame: origin at the top face's centre on the screw axis, +Z up; the body spans z -12 to 0. `PrintUp` is (0, 0,
-1): the top face is on the bed.

1. Sketch `pad` on `frame((0,0,-12), X, Y)`: outline `(-24.5,-15)`, `(15,-15)`, `(15,15)`, `(-24.5,15)`, all radius 2.
   Net area 1181.6. Extrude 12 along +Z.
2. `edgefinish.Chamfer(ctx, body, (0,0,1), []float64{-12, 0}, edgefinish.Outer, 0.5)`: both cap outlines, 16 edges.
3. Cut the socket: `m5fastener.RoundedSocketTool(ctx, doc, frame((0,0,-4), X, -Y), 2.85, 8, 0.5)`: radius 2.85
   (diameter 5.7), 8 deep, floor corner rounded 0.5 with 8 chords. The floor is at z = -4.
4. Return `Part{Body, Frame: placed PadFrame, PrintUp placed (0,0,-1)}`. The pad's -X face stands 0.5 from the spine.

## 7. Joint tools and the swivel (arm.go)

Local frame: origin on J1's axis in the swivel's bottom face, +X toward J2, +Z up. `PrintUp` is +Z.

1. Plan prism: sketch on `frame((0,0,-1), X, Y)`: `top = (0,10)`, `bottom = (0,-10)`, `c1 = (81,10)`, `c2 = (81,-10)`, the
   half-circle idiom with radius 10 about the origin and the far end at distance 81. Extrude 22 along +Z (z -1 to 21).
   Net area 1777.1. The prism is taller and longer than the swivel so that none of its faces lies in a plane of the
   elevation prism or touches its curve (Section 13, item 1).
2. Elevation prism: sketch on `frame((0,11,0), X, Z)` (normal -Y): `a = (-11,0)`, `b = (70,0)`, `c = (70,20)`, `d = (-11,20)`;
   `centre = DimensionPoint(70, 10)`; `arc = CreateArc(centre, b, c)` (through +X) with `NewRadius(arc, 10)`;
   `NewHorizontalDistance(origin, b, 70)` and `(origin, c, 70)`; the lines a-b and c-d horizontal, d-a vertical with
   `NewDistancePointLine(origin, d-a, 11)`. Net area 1777.1. Extrude 22 along the normal: y 11 to -11.
3. `decad.Intersect(ctx, plan, elevation)`: the swivel's body, 20 thick, round about J1 in plan and about J2 in
   elevation.
4. Edge finishing before the cuts: one call, `edgefinish.ChamferExcept(ctx, body, (0,0,1), []float64{0, 20},
   edgefinish.Outer, skipParallel((0,1,0)), 0.5)`. With the exact cross-plane chain support in decad 5a61e0681424, this
   call chamfers the whole closed chain, so it also covers the Y-face rims and the J2 round-end arcs. A second call
   on the Y faces returns `ErrNoEdge`, so there is none.
5. J1 tongue tool: sketch on `frame((0,12,0), X, Z)` (normal -Y): outline `(-12,8)`, `(30,8)`, `(30,21)`, `(-12,21)`
   with radii 0, 4, 0, 0, so the corner `(30,8)` takes a round of radius 4 whose centre is `(26,12)`. Lines: z = 8
   (distance 8), x = 30, z = 21, x = -12. Extrude 24 along the normal (through y 12 to -12). Cut it. The tongue is 8
   thick, z 0 to 8, for x < 30.
6. J1 hole: `holeTool(panLocal)`. Cut.
7. J2 fork slot: `slotCutter(70 - 21.74, 81, -1)`: the slot `|y| < 4.26`, from x = 81 back to x = 44, ended by a round of
   radius 4.26 centred at x = 48.26. Cut.
8. J2 hole: `holeTool(forkSite(70, 10))`. J2 head pocket: `pocketTool(forkSite(70, 10))`, floor at y = -6.5. Cut both.
9. Edge finishing after the cuts, passing with decad 5a61e0681424:
   `edgefinish.ChamferExcept(ctx, body, (0,0,1), []float64{8}, edgefinish.Outer, skipParallel((0,1,0)), 0.5)` for the
   tongue's top face, and `edgefinish.ChamferExcept(ctx, body, (0,1,0), []float64{-4.26, 4.26}, edgefinish.Outer,
   skipParallel((0,0,1)), 0.5)` for the fork's inner faces.
10. Return `Part{Body, Frame: placed S, PrintUp placed (0,0,1)}`. Failure checks: bounds `-10..80`, `-10..10`, `0..20`;
    the tongue is `z <= 8` for x < 26, full width (y +-10).

## 8. Link (arm.go)

Local frame: the tongue's axis at the origin, the fork's axis at `(L, 0, 0)`, joint axes along Y. The link prints on its
-Z face, so `PrintUp` is (0, 0, 1). It is built twice, as link A and link B, each with its own sketches.

1. Sketch `link` on `frame((0,0,0), X, Z)` (normal -Y): the stadium idiom with `L = LinkLength`. Net area `20 L + 314.2`.
   `extrudeSymmetric(s, 0, 10)`: y -10 to 10.
2. `edgefinish.Chamfer(ctx, body, (0,1,0), []float64{-10, 10}, edgefinish.Outer, 0.5)`: both Y cap outlines, 8 edges.
3. Cuts, in this order: `tongueCutter(+1)`, `tongueCutter(-1)`, `slotCutter(L - 21.74, L + 11, -11)`, the tongue hole
   `holeTool(tongueSite(0))`, the fork hole `holeTool(forkSite(L, 0))`, the fork's head pocket
   `pocketTool(forkSite(L, 0))` (floor at y = -6.5).
4. Edge finishing after the cuts, the design's call (passes with decad 5a61e0681424): `edgefinish.ChamferExcept(ctx, body, (0,1,0),
   []float64{-4.26, -4, 4, 4.26}, edgefinish.Outer, skipParallel((0,0,1)), 0.5)`: the tongue faces and the fork's
   inner faces.
5. Return `Part{Body, Frame: placed LinkA or LinkB, PrintUp placed (0,0,1)}`. Failure checks: bounds `-10..L+10`,
   `-10..10`, `-10..10`; the tongue is 8 thick for x < 16; the slot is 8.52 wide.

## 9. Head (arm.go)

Local frame: the J4 axis at the origin, +X along the head, +Z up. `PrintUp` is (0, 0, 1).

1. Sketch `head` on `frame((0,0,0), X, Z)` (normal -Y): the half-circle idiom with `top = (0,10)`, `bottom = (0,-10)`,
   corners `c1 = (55,10)`, `c2 = (55,-10)` and the far line at distance 55, then `CreateFillet` at `c1` and `c2` with
   radius 2. Net area 1255.4. `extrudeSymmetric(s, 0, 10)`.
2. `edgefinish.Chamfer(ctx, body, (0,1,0), []float64{-10, 10}, edgefinish.Outer, 0.5)`: both Y cap outlines, 12 edges.
   This is before the boss union.
3. Boss: `fastenertool.RoundedRevolve(ctx, doc, frame((40,0,9), (0,0,1), (1,0,0)), [][2]float64{{0,0}, {0,6}, {1,6}, {3,4},
   {3,0}})`: along the axis from z = 9 (1 mm inside the head) to z = 12, radius 6 at z = 10 narrowing at 45 degrees to
   radius 4 at z = 12. `decad.Union(ctx, body, boss)`.
4. Cuts, in this order: `tongueCutter(+1)`, `tongueCutter(-1)`, the tongue hole `holeTool(tongueSite(0))`, the stud hole
   `holeTool(studLocal)` (diameter 5.6 along Z at x = 40, through the boss and the head).
5. Edge finishing after the cuts, the design's call (passes with decad 5a61e0681424): `edgefinish.ChamferExcept(ctx, body, (0,1,0),
   []float64{-4, 4}, edgefinish.Outer, skipParallel((0,0,1)), 0.5)`: the tongue faces.
6. Return `Part{Body, Frame: placed H, PrintUp placed (0,0,1)}`. Failure checks: bounds `-10..55`, `-10..10`, `-10..12`.

## 10. Mount interface (adapter.go)

1. Constants `BossBaseRadius` 6, `BossHeight` 2, `RecessRadius` 6.495, `RecessClearance` 0.35, `StudProtrusion` 13.5,
   `AdapterNutMin` 4.7, `AdapterNutMax` 8.3, `AdapterStudClear` 14.5. Package doc names this the "Mount interface".
2. `InterfaceRecessTool(ctx, doc, f r3.Frame) (*decad.Body, *sketch.Sketch, error)`: the meridian frame is
   `r3.NewFrame(f.Origin(), f.N(), f.U())` (axis along `f`'s normal). Call `fastenertool.RoundedRevolve` with the
   `(along, radius)` points `(-1, 0)`, `(-1, 7.495)`, `(0, 6.495)`, `(3.695, 2.8)`, `(16, 2.8)`, `(16, 0)`. Every vertex
   is a `DimensionPoint`; the sketch is fully constrained. The 45 degree cone runs from radius 6.495 at the face to 2.8
   at 3.695 above it.
3. The recess rim is the 45 degree register and stays sharp.

## 11. Placeholder adapter (adapter.go)

Local frame is `I`: origin at the interface face's centre on the stud axis, U = head +X, normal = head +Z. The disc spans
z 0 to 15. `PrintUp` is (0, 0, 1): the mating face is on the bed.

1. Sketch `adapter` on `frame((0,0,0), X, Y)`: `CreateCircle(origin, 20)` with `NewRadius(circle, 20)`. `profile(s, 0)`.
   Extrude 15 along +Z.
2. `edgefinish.Chamfer(ctx, body, (0,0,1), []float64{0, 15}, edgefinish.Outer, 0.5)`: both disc rims, 2 edges. This is
   before the cuts, so the recess rim stays sharp.
3. Cut `InterfaceRecessTool(ctx, doc, frame((0,0,0), X, Y))`.
4. Cut the nut pocket: `m5fastener.NutPocketTool(ctx, doc, f, 10.3)` with `f = frame((0,0,4.7), (1,0,0), (0,1,0))`:
   floor at z = 4.7, open to the top face (z = 15), floor outline rounded 0.6.
5. Return `Part{Body, Frame: placed I, PrintUp placed (0,0,1)}`. The stud's tip is at z = 13.5, 4.1 past the nut
   (z 4.7 to 9.4) and 1.5 inside the top face.

## 12. Assembly (clamparm.go)

1. Package doc, container names in this order (`Frame`, `Pad`, `Swivel`, `Links`, `Head`, `Adapter`, `Knobs`,
   `Screw Knobs`, `Bolts`, `Nuts`), `Part{Body *decad.Body; Frame r3.Frame; PrintUp r3.Vec; Sketches
   []*sketch.Sketch}`, `Parts{Clamp, Pad, Swivel Part; Links [2]Part; Head, Adapter Part; Knobs [4]Part; ScrewKnobs
   [2]Part; Dimensions Dimensions}`, `Model{Parts; Bolts [6]*m5bolt.Model; Nuts [6]*m5nut.Model}`.
2. `BuildParts(ctx, doc, p)`: reject a nil `ctx` and a nil `doc` (`clamparm: context must not be nil`, `clamparm: document
   must not be nil`), call `p.Derive()`, check `ctx.Err()` (`clamparm: before creating geometry: %w`). Build in this
   order, each with its own `run`: frame, pad, swivel, link A, link B, head, adapter, knobs J1 to J4, screw knobs (clamp
   screw, stud). Each body must be solid (`clamparm: <name> is not a solid`). Place each body once with `placement.Place(ctx,
   body, &placed)` where `placed` is the part's frame (Sections 1, 3 to 11) composed with `Placement`: with `xf =
   r3.FromFrame(*Placement)`, `placed = frameOf(compose(xf, r3.FromFrame(frame)))`; with a nil `Placement`, `placed` is
   the frame. Each `Part.Frame` is `placed`, each `PrintUp` is `r3.FromFrame(placed).ApplyDir(local up)`. `Parts.Dimensions`
   is the value `Derive` returns, before `Placement`. A rejected build adds no body:
   every check precedes the first geometry call.
3. `Build(ctx, doc, p)`: `BuildParts`, then the placed sites (`Head` by `Apply`, `Dir` and `Corner` by `ApplyDir`) and
   placed frames. Bolts in the order J1 (`PanSite`, M5x35), J2 to J4 (`JointSites`, M5x25), stud (`StudSite`, M5x40),
   screw (`ScrewSite`, M5x70): `m5fastener.PlaceBolt(ctx, doc, site)`. Nuts in the order `KnobNutFrames[0..3]`,
   `JawNutFrame`, `AdapterNutFrame`: `m5fastener.PlaceNut(ctx, doc, frame)`.
4. Containers: after every boolean, `doc.Root().ChildNamed(name)` then `AddBody`, in container order: `Frame` (1),
   `Pad` (1), `Swivel` (1), `Links` (2, A then B), `Head` (1), `Adapter` (1), `Knobs` (4, J1 to J4), `Screw Knobs` (2,
   clamp screw then stud), `Bolts` (6), `Nuts` (6). The document holds 25 bodies. `doc.VerifyContainers().Err()` is nil.
5. Placement: the pad is posed at `(25, 0, -o)`, the screw knob at `-o - 67.5`; the pad and the screw knob move with
   `Opening`; nothing else does.
6. Failure checks: every part sketch fully constrained; every part solid; 25 bodies; the screw knob gap at least 0.5.

## 13. Decad notes (decad 001517e320ed)

1. `decad.Intersect` of the plan and elevation prisms refuses coplanar faces ("two operand facets overlap in one
   plane") when the plan prism spans z 0 to 20, and a tangent contact when its end is at x = 80. The steps use z
   -1 to 21 and x = 81 for the plan prism. The result is the same shape as the design's z 0 to 20, x = 80 prism, and
   `instructions.md` "Swivel" states the swivel's outline, not the operand. This is an operand choice, not a workaround.
2. Edge finishing: the refusals recorded earlier ("selected edge ends at a tangent junction", and the link and head
   tongue-face chamfer after the cap chamfer and the tongue cuts) are fixed by decad 5a61e0681424. With that pin every
   edge-finishing call in Sections 7 to 9 builds, in section order, with the real `CreateSlot` stadium at L = 80, 120
   and 150. The swivel's before-cut call covers the Y-face arcs, so a second call returns `ErrNoEdge`.
3. Nut pocket depth: the J1 to J4 knob pockets are 9 mm deep and the adapter pocket runs z 4.7 to 16. AGENTS.md asks
   for a pocket as deep as a nut cavity is high (5.1 mm for M5). The design keeps the deeper pockets so the bolt tip
   ends inside the pocket and never stands outside a face. This is a recorded departure from that rule.
4. The chamfers that pass: the frame (44 edges), the pad (16), the link (8 on the Y caps, plus the tongue and fork
   faces in one call), the head (12 on the Y caps, plus the tongue faces) and the knobs (24) and the adapter (2).
5. Assembly checks (decad 001517e320ed, which reads placed seat planes exactly in `Intersect`). Every hardware body
   against every printed body or other hardware body whose box meets it proves empty with `decad.Intersect` and
   `BooleanEmpty`, hardware first or part first, in all 15 cases of `asmCases` (`TestClampArmFastenersNoOverlap`, with
   every body moved to the document root, since `Intersect` across containers returns `decad: operands sit in
   different containers`). A refusal outside the list below fails the test. The pairs below are refused in both
   operand orders, so `asmMeshOverlap` in `fasteners_test.go` decides them (`asmMeshOnly` lists them). It requires that
   no vertex of the hardware's mesh lies inside the part's mesh, that no vertex of the part's mesh lies inside the
   hardware's mesh (a ray-parity test, deeper than 0.01 mm), that sections normal to the hardware's axis at seven
   stations along it do not overlap, and that sections through the axis at 0, 45, 90 and 135 degrees do not overlap
   (`SectionOutline` loops, as in `crankhandle`, tested by vertex depth because `SectionClearance.Overlaps` reads true
   for outlines 0.3 mm apart where the plane meets a rim on a rotated part).
   `TestClampArmFastenersMeshOverlapControl` shows that each of the four tests catches a bolt moved 0.6 mm and 0.35 mm
   (diagonal) off its hole in link A's tongue, and passes one moved 0.1 mm.
   - Bolt J2 against the swivel in `pan-max` and `pan-min`: `Intersect` returns `not supported by the current
     evaluator: intersect's ... operand holds its mesh, where this pair meets, only within a bound of ... mm, coarser
     than this pair's chord tolerance` in both orders. The swivel's held mesh (built by the J1 tongue cut, the J2 slot,
     hole and pocket cuts and two chamfer calls) is coarser than the pair's chord tolerance. At the other 13 cases the
     pair proves empty.
   - Nut J4 against knob J4 in `raised` (`Shoulder` 110, `Elbow` -70, `Wrist` -40): refused in both orders with `the
     operands' held facets come within the chord tolerance without provably interpenetrating deeper than it`. At the
     other 14 cases the pair proves empty.
   - A bolt against its own nut (bolt J1 to J4 with nut J1 to J4, the stud with the adapter nut, the clamp screw with
     the jaw nut) is refused in both orders in every case. The nut's bore is the shank, so the cylinders coincide and
     the refusal is correct. `TestClampArmFastenersNoOverlap` leaves these pairs out and
     `TestClampArmFastenersBoltNutEngagement` checks them analytically.
   - The adapter nut against the adapter proves empty hardware first in every case, although the part-first order is
     still refused, which the check accepts.
   - `Body.DistanceToPoint` on placed bodies. On link A at `default`, `DistanceToPoint(ctx, p, tol)` for
     `p = (100, 0, 27.36)` (0.3 mm outside its J2 tongue hole's wall) with `tol` 0.005 returns `requested tolerance
     0.005 mm is below the faceted body's minimum mesh bound 0.0175 mm`; with `tol` 0.02 or 0.05 it returns 0.1588
     mm, `Approximate`, error bound 0.1588 mm, so the result cannot tell 0.3 from 0.14. Bodies moved by `Transform`
     (the placed case, and every body after a pose) hold meshes, and their bound runs from 0.05 to 0.23 mm. The fit
     checks measure from `meshgap.Mesh.PointDistance` over a mesh tessellated as finely as the body allows instead.
