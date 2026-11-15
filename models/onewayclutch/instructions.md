# One-way clutch instructions

A printed one-way clutch for a D-flat shaft. A hub pressed on the shaft carries three flexing pawls. They engage a ring
with 24 internal sawtooth teeth and three bolt ears, so the ring turns freely one way about the held shaft and is
driven the other way. A back plate and a cover hold the ring on the hub, clamped by three M2 bolts and nuts. Each nut
sits in a hex pocket open to the back plate's outer face. Every part except the bolts and nuts prints on an FDM printer with its axis vertical and no supports. The shaft is not modeled, as in `crankhandle`:
the mechanism owns it and any [Arbor collars](../../docs/model-pipeline.md#arbor-collars) that locate the clutch.

Units are millimeters. The local frame has its origin on the shaft axis in the back plate's outer face (the face toward
the mechanism), +Z runs along the axis to the cover, and the shaft's flat faces +Y. A `Placement` frame maps this
frame after each part's last operation.

## Mechanism

Compliant-pawl ratchet. Each pawl is a thin curved arm printed in one piece with the hub; it ends in a head. The heads
sit in the gaps of the ring's teeth. Turning the hub one way pushes each head against a tooth's radial lock face and
drives the ring. Turning it the other way, the teeth's ramps press the heads inward, the arms bend, and the heads click
over the teeth. The arms flex only in the XY plane, so their layers run along the arm and bending loads the printed
lines. A head leads its arm in the drive direction, so the drive load puts the arm in compression; the arm is curved
about the axis, so that pushes the head outward, deeper into the gap.

### Freewheel direction

`Freewheel` is the direction the ring turns freely about a held shaft, seen from the front (the cover, looking along
-Z). The shaft turning the same way drives the ring with it; the other way slips. One construction serves both
directions. `Direction.Sign()` is `-1` for `Clockwise` and `+1` for `CounterClockwise`. Every hub and ring sketch is
drawn in the counter-clockwise layout below with each point `(x, y)` mapped to `(σx, y)`, a mirror in the plane
`x = 0`. An angle θ of the layout maps to `θ' = 90° + σ(θ - 90°)`. `CreateArc(center, start, end)` runs
counter-clockwise, so each arc's start and end swap when `σ = -1`. The D flat and the three bolts are symmetric about
`x = 0`, so the plates, bolts and nuts are the same in both directions.

## Inputs and defaults

| Input | Default | Meaning |
| --- | --- | --- |
| `Shaft` | `605ZZ` | a preset name from the table below |
| `Freewheel` | `Clockwise` | `Clockwise` or `CounterClockwise`; zero is invalid |
| `BoreClearance` | 0.05 (`ballbearing.PressFitClearance`) | radial clearance of every D bore, on the round and the flat |
| `Placement` | nil | a frame; nil selects the local frame |

| Preset | Diameter | Flat depth | Across the flat | Source |
| --- | ---: | ---: | ---: | --- |
| `605ZZ` | 5 (`ballbearing.Lookup("605ZZ").ShaftDiameter()`) | 0.5 | 4.5 | planetary 605ZZ shafts, `crankhandle.Params605ZZ` |
| `608ZZ` | 8 | 0.75 | 7.25 | hand-crank siren shafts, `crankhandle.Params6082RS` |
| `608ZZ-gearbox` | 8 | 0.5 | 7.5 | planetary gearbox 608ZZ shaft kits |

The body does not change with the preset: only the D bore of the hub and both plates changes. The ring's ears are
the same for every preset and `BoreClearance`. D bore radius
`r_b = D/2 + c`, flat at `f_b = D/2 - d + c` from the axis on the +Y side, where `c` is `BoreClearance`.

### Validation (first failure wins; every message starts `onewayclutch: `)

1. `Shaft` names a preset: `unknown shaft preset "<name>"; want one of 605ZZ, 608ZZ, 608ZZ-gearbox`.
2. `Freewheel` is `Clockwise` or `CounterClockwise`.
3. `BoreClearance` is finite and `0.05 <= c <= 0.5`.
4. `BoltCircle - NutCavityRadius - r_b >= PressWall`, that is `r_b <= 4.29`; the message says the bore leaves less than
   1.6 mm of wall between the bore and the nut pockets.
5. `Placement` is nil or valid.
6. Each bolt site passes `m2fastener.Site.Validate()`.

`BuildParts` and `Build` reject a nil context and a nil document, derive, and check `ctx.Err()` before any geometry.

## Fixed dimensions (exported constants)

| Constant | Value | Meaning |
| --- | ---: | --- |
| `FitMargin`; `TightFit`, `RegularFit`, `LooseFit` | 0.05; 0.25, 0.40, 0.55 | class minimum + margin |
| `Pawls`, `Teeth` | 3, 24 | all three heads engage at once |
| `ToothTip`, `ToothRoot` | 14.5, 17.0 | ring tooth tip and root radii |
| `RootRound` | 0.8 | ring tooth root round, `HeadGap + TipRound` |
| `RingCounter`, `RingOuter`, `RingChamfer` | 18.5, 21.0, 0.5 | counterbore, outside radius (OD 42 between the ears), lead-ins of the outer loop and the counterbores |
| `Ears`, `EarAngles` | 3, 90° 210° 330° | ears on the ring, the same in both directions |
| `EarCircle`, `EarRadius`, `EarRound`, `EarReach` | 23.5, 3.6, 1.5, 27.1 | ear bore circle, ear half width and end round, round where an ear side meets the outside, ear tip radius |
| `RingHeight`, `ToothEndGap` | 17.0, 0.85 | ring height; tooth band end to the plate annulus |
| `CoreRadius` | 10.7 | hub core |
| `SlotWidth`, `ArmThickness` | 2.05, 1.2 | slot under each arm; arm (three 0.4 mm lines, the bending-part minimum) |
| `ArmInner`, `ArmOuter` | 12.75, 13.95 | `R_c + s`, `+ t` |
| `WebAngle`, `SlotAngle`, `LockAngle` | -20°, -6°, 90° | pawl 0's web back face, slot end, and ring lock face |
| `WebRound`, `TipRound`, `HeadGap` | 1.0, 0.4, 0.40 | web root round, head tip round, head to its gap (seated) |
| `HubHeight`, `Seam` | 10.0, 0.25 | hub and pawls; hub to each plate boss |
| `PlateRadius`, `Recess` | 18.1, 0.40 | `RingCounter - RegularFit`; annulus below the boss face |
| `BackPlateThickness`, `CoverThickness` | 3.5, 4.0 | outer face to boss face |
| `PlateChamfer` | 0.5 | outer-face chamfer |
| `BoltCircle`, `BoltAngles`, `BoltLength` | 8.4, 30° 150° 270°, 16 | three M2 x 16 through bolts |
| `PressWall` | 1.6 | least wall round a press-fit bore |

### Stack (z along the axis)

| Part | z from | z to | Notes |
| --- | ---: | ---: | --- |
| Back plate | 0 | 3.5 | boss face 3.5, annulus 3.1, rim cylinder 0.5 to 2.0 at r 18.1, hex pockets 0 to 2.0 open to z 0 |
| Hub | 3.75 | 13.75 | 0.25 seams to both bosses; mid-plane 8.75 |
| Cover | 14.0 | 18.0 | boss face 14.0, annulus 14.4, head pockets 16.0 to 18.0 |
| Ring | 0.25 | 17.25 | symmetric about z 8.75, the hub's mid-plane |
| Ring tooth band | 3.95 | 13.55 | 9.6 tall, inside the heads' 3.75 to 13.75 |

The assembly is 18.0 mm long. The ring is 42 mm across between its ears; the ear tips lie on a 54.2 mm circle, so the
clutch spans x -23.95 to 23.95 and y -21.0 to 27.1.

## Construction

### Ring (prism, chamfer, two cuts)

1. Outline sketch on z = 0.25: the outside, radius 21.0, with three ears as the outer loop; as holes the 24-tooth loop
   and the three ear bores. Extrude `RingHeight` 17.0 along +Z. The extrusion is not a Boolean.
2. Chamfer the whole outer loop of both end faces by 0.5 in one call (`edgefinish.Chamfer`, `Long(Outer)`).
3. Cut two counterbore tools, revolved meridians `(r, z)`. Back: (13.0, -0.75), (20.0, -0.75), (18.5, 0.75),
   (18.5, 2.45), (17.0, 3.95), (13.0, 3.95). Front, the back one mirrored in z = 8.75: (13.0, 18.25), (20.0, 18.25),
   (18.5, 16.75), (18.5, 15.05), (17.0, 13.55), (13.0, 13.55). The counterbores take the plate rims; the 45° lead-in
   from the end face and the 45° cones from `RingCounter` down to `ToothRoot` hold the ring axially; the flats at z 3.95
   and 13.55 are the teeth's end faces.

The tooth loop is the whole void inside the teeth, one closed loop. Counter-clockwise layout: lock face `j` lies on the
radial axis at `φ_j = 90° + 15°j`, from `Q_j = R_root∠φ_j` in to `P_j = R_tip∠φ_j`; the ramp of tooth `j` runs from
`P_j` to `Q_{j+1}`. Each `Q_j` is rounded by `RootRound`. The tooth occupies the angles just counter-clockwise of each
lock face; the gap where a head sits lies just clockwise. The ramp line is at distance 13.289 from the axis
(`RampDistance`), at 141.417° (`RampAngle`).

Drawing the teeth in the outline leaves the ring two Boolean cuts. A tooth cut after the counterbore cuts is refused
by decad, because it would meet the counterbore cuts' rims.

Ears: three full-height lugs at 90°, 210° and 330° (symmetric about `x = 0`, so not mirrored), each 7.2 mm wide with a
full end round of 3.6 about a point 23.5 from the axis, the sides parallel to the ear's axis and tangent to the end
round. Each side meets the outside circle in a round of 1.5 (a stressed root, above half the 2.5 mm wall). An M2
clearance bore (radius 1.3) runs through each ear along the axis, 2.3 mm from its sides and end and 3.7 mm from the
counterbore. The model places no bolt in the ears: the driven part decides between bolts and a tight fit in a housing
with three notches.

### Plates (prism, chamfer, cuts)

Profile on the plate's inner-side plane (back plate z = 0, cover z = 14.0): the circle `PlateRadius` about the origin,
the D bore (arc of `r_b` the long way through -Y, flat at `y = f_b`), and three circles of `m2fastener.HoleRadius` (1.3)
at `BoltCircle` on 30°, 150° and 270°. Extrude 3.5 or 4.0 along +Z. Then:

1. Chamfer the outer face (back plate z = 0, cover z = 18.0) by 0.5 in one call: its outer circle, plus the whole D loop
   when `BoreLoop >= 20` (the 8 mm presets; the 5 mm loop is 15.8). The bolt-hole and head-pocket loops are short.
2. Cut the inner-face tool, a revolved meridian `(r, z)`. Back plate: (9.8, 4.5), (9.8, 3.9), (10.6, 3.1), (17.0, 3.1),
   (19.1, 1.0), (19.1, 4.5). Cover: (9.8, 13.0), (9.8, 13.6), (10.6, 14.4), (17.0, 14.4), (19.1, 16.5), (19.1, 13.0).
   It leaves the boss (radius 10.2 at the face, a 0.4 mm 45° bevel), the recessed annulus out to r 17.0 and a 45° cone
   to the rim.
3. Back plate: cut `m2fastener.HexPocketTool` at each site (z 0 to 2.0, open to the outer face; the chamfer in step 1
   runs first, so the hex loops are never chamfered). Cover: cut `m2fastener.PocketTool` at each site (z 16.0 to
   18.0). The holes are already in the profile.

### Hub (one prism)

One sketch on z = 3.75 holds the whole outline, the D bore and the three bolt holes; extrude `HubHeight` along +Z. The
hub has no Boolean. Both faces are then chamfered 0.5 along their long runs (see "Surface edge finishing").

### Hub outline, counter-clockwise layout, pawl 0 (pawl `k` is turned by 120°k)

The gap where pawl 0's head sits is bounded by the ring lock face at 90° (the line `x = 0`) and the previous tooth's
ramp, from `P = 14.5∠75°` to `Q = 17∠90°`.

- Lock line: `x = HeadGap`, parallel to the 90° axis, on the gap side.
- Flank line: parallel to the ramp, `FlankDistance = RampDistance - HeadGap` = 12.889 from the axis, on the axis side.
- Apex `A` = lock ∩ flank = (0.400, 16.169). `TipRound` rounds it; its centre is (0.800, 15.338).
- `HeadReach` (outermost head radius) = 15.759; `HeadTravel` = `HeadReach - ToothTip` = 1.259.
- `LockEnd` `H_l` = lock ∩ circle `ArmInner` = (0.400, 12.744). `FlankEnd` `H_f` = flank ∩ circle `ArmOuter` = (3.867, 13.403),
  the root nearer `A`.

Outline of one period, counter-clockwise, material on the left:

1. Core arc, radius `CoreRadius`, from the slot round of pawl `k-1` to the web root round of pawl `k`.
2. Web root round, radius `WebRound`, tangent to the core arc and the web back face (a stressed root).
3. Web back face on the axis at `WebAngle`, out to `W = ArmOuter∠WebAngle`, a sharp 90° corner.
4. Arm outer arc, radius `ArmOuter`, from `W` to `H_f`.
5. Flank from `H_f` toward `A`, tip round, lock face down to `H_l` (a sharp 90° corner).
6. Arm inner arc, radius `ArmInner`, from `H_l` clockwise back to the slot round.
7. Slot-end full round, centre on the axis at `SlotAngle`, tangent to the arm inner arc (internal) and the core arc
   (external); radius `SlotWidth/2` = 1.025 follows from the tangencies. It bulges toward the web.
8. Core arc again, on to the next pawl's web root round.

The head's lock face bears on the ring's lock face over 0.84 mm radially and 9.6 mm axially, on each of three heads. The
flexible arm runs 79.9° from the slot end to `H_f`, 18.6 mm at radius 13.35; full travel bends it to 0.65 % strain. The
slot leaves `SlotWidth - HeadTravel` = 0.79 mm under a fully pressed head. The thinnest hub walls are 3.05 from the bore
to a bolt hole and 1.0 from a bolt hole to the core arc.

### Bolts and nuts

Site `i` at angle `a_i` in {30°, 150°, 270°}: `m2fastener.Site{Head: (8.4 cos a, 8.4 sin a, 18.0), Dir: (0, 0, -1),
Corner: (cos a, sin a, 0), Length: 16, Cavity: 16.0}`. The head lies flush with the cover's outer face in a 2 mm pocket.
The bolt passes the cover, the hub and the back plate. The nut sits in a hexagonal pocket (z 0 to 2.0) open to the back plate's outer face, with its corners on
`Corner` at radius 2.51 and its flats 2.174 from the bolt line; the nut sits from z 0.2 to 1.8. The tip ends at z 0,
flush with the back plate's outer face, past the nut's outer face, so the nut is fully engaged. The bolt holds the
nut in its pocket, and the pocket's flats stop it turning. The pocket's floor is 1.1 mm below the annulus. M2 x 16 is in `m2bolt.StandardLengths()`.

## Bodies and containers

| Body | Count | Container | Printed |
| --- | ---: | --- | --- |
| Hub (core and three pawls) | 1 | `Hub` | yes |
| Ring | 1 | `Ring` | yes |
| Back plate, cover | 1 + 1 | `Plates` | yes |
| M2 x 16 bolt | 3 | `Bolts` | bought |
| M2 nut | 3 | `Nuts` | bought |

Ten bodies, five containers, each a child of the document root, filed with `doc.Root().ChildNamed` and `AddBody`. The
body set does not change with any parameter. There is no "Look-alike parts" section, so no part marks.

## Part fits

| ID | Parts | Class (min) | Modeled | Measured, proven lower bound | Measure |
| --- | --- | --- | ---: | ---: | --- |
| P1 | hub D bore / shaft | press fit | `c` (0.05) | 0.049 (at `c` 0.35: 0.349) | section at mid-height |
| P2 | back plate, cover D bores / shaft | press fit | `c` | 0.049 (at `c` 0.35: 0.349) | as P1 |
| T1 | hub / each plate boss | Tight 0.2 | 0.25 | 0.237 | mesh gap, placed meshes |
| R1 | pawls / plate annuli | Regular 0.35 | 0.65 | 0.601 | axial section, hub bounds |
| R2 | ring counterbores / plate rims | Regular | 0.40 radial | 0.381 | section mid-overlap (z 1.375 and 16.125) |
| R3 | ring cones / plate cones | Regular | 0.60 normal | 0.583 | axial section |
| R4 | ring tooth ends / plate annuli | Regular | 0.85 axial | 0.832 | axial section |
| R5 | each head / its tooth gap, seated | Regular | 0.40 | 0.387 | `meshgap` on the placed meshes |
| L1 | arm outer arcs / tooth tips | Loose 0.5 | 0.55 | 0.515 | section at mid-height |
| F | bolts, nuts | fastener rules | head pocket 0.3, hole 0.3, nut 0.17 at each flat, 0.20 past each corner and to the floor | | bounds, sections and no overlap |

The model is **Mixed**: the bore clearance is a parameter, and the bolts and nuts keep their nominal size. The ring's
ears and outside fit in the user's driven part or housing belong to the consumer. `TestOneWayClutchFits` measures P1 to R5 and L1 for every
preset in both directions, for `BoreClearance` 0.35 on `605ZZ`, and the mesh gaps for a placed default. The measured
column is the least value over those cases, less each section's or mesh's proven displacement bound, so it sits a
little under the modeled value. A D bore's corners stand more than `c` from the shaft's corners, since `c` is radial
on the round and on the flat. In the head-on-pocket-floor and bolt-in-nut pairs the meshes touch by design (nominal
pair).

`TestOneWayClutchPawls` checks the arm (1.2 mm thick, 0.65 % strain at full travel), the slot (2.05 mm, leaving 0.79
mm under a fully pressed head), the web (1.80 mm to the slot end's round, wider than the arm) and the lock face (0.82
mm radial contact, 9.6 mm axial), from `Dimensions` and from sections of the built hub.

## Sketch constraints

Every sketch is solved, then `sketchcheck.VerifyFullyConstrained` before its profile is used. Use geometric
constraints; place computed points with `sketchcheck.DimensionPoint` only in meridians.

- Polar axes: a horizontal construction reference line from the origin, and one construction axis per angle, a line from
  the origin with `NewAngle(ref, axis, θ')` and `NewDistance(origin, end, 20)`, using the mapped angle.
- Hub sketch: three axes per pawl (web, slot, lock) and one per bolt. Arcs `CreateArc(origin, ...)` with `NewRadius`.
  Web back face: `NewParallel(webAxis, web)` and `W` on the axis. Lock line: `NewParallel(lockAxis, lock)` and
  `NewDistancePointLine(origin, lock, 0.40)`. Flank: `NewAngle(ref, flank, RampAngle')`, `NewDistancePointLine(origin,
  flank, 12.889)`. Rounds: `sketch.CreateFillet` for the web root (core arc, web line) and the tip (flank, lock). Slot
  round: centre `NewPointOnLine` on the slot axis, `NewTangentCircles` with the arm inner arc (internal) and the next
  core arc (external); no radius constraint. The D bore and bolt holes follow the plate profile. A fillet removes its
  corner point and the constraints on it, so each line is held by constraints on the line and its far end.
- Ring outline sketch: three construction axes at the ear angles with `NewAngle(ref, axis, e_k)`; each ear centre
  `NewPointOnLine` on its axis with `NewDistance(origin, centre, 23.5)`; the end round `NewRadius(3.6)`; each side
  `NewParallel(axis, side)` and `NewTangent(side, endRound)`; the three outside arcs `NewRadius(21.0)` about the origin;
  the bores `NewRadius(1.3)` sharing the ear centres; then `CreateFillet` at all six side-to-outside corners, 1.5. The
  tooth loop is drawn in the same sketch (next item). Select the profile with four holes.
- Ring tooth loop: 24 axes at `φ_j`. `P_j` is `NewPointOnLine` on axis `j` with `NewDistance(origin, P_j, R_tip)`. The
  lock line `Q_j → P_j` is `NewParallel` to axis `j`. The ramp `P_j → Q_{j+1}` is `NewDistancePointLine(origin, ramp,
  13.289)`. `CreateFillet` at every `Q_j`, 0.8. One closed loop, a hole of the ring's outline.
- Meridians (ring counterbore tools, plate inner tools): the plane holds the axis, U along +Z and V radial; every vertex
  `DimensionPoint`; revolve with `decad.FullRevolution{}`.
- Plate profile: the hub's bore and holes, plus `NewRadius(PlateRadius)` on a circle about the origin.
- Fastener tools: `m2fastener` returns their sketches; they are recorded in `Parts.Sketches`.

## Surface edge finishing

| Part | Face | Net area | Long runs | Action |
| --- | --- | ---: | --- | --- |
| Back plate | outer face | about 931 to 961 | outer circle 113.7; D loop 25.1 to 25.3 at 8 mm; hex loops 15.06, short | chamfer 0.5 (one call) |
| Cover | outer face | 934 to 964 | as above; head pockets 13.8, short | chamfer 0.5 (one call) |
| Plates | annulus | 555 | outer edge meets the 45° cone | the 1.1 mm cone is that edge's finish |
| Plates | boss face | 260 to 290 | none | small, none |
| Hub | each face | 409 to 439 | per pawl: arm outer arc 22.9, inner run 47.8; head run 7.1, short; D loop 25.1 to 25.3 at 8 mm | chamfer 0.5 on the long runs; head run sharp (below) |
| Ring | each end face | 413.3: outline 1504.5 less counterbore 1075.2 less bores 15.9 | outer loop (outside arcs, rounds, sides, end rounds, all tangent) 158.7; counterbore rim 116.2; ear bores 8.2, short | chamfer the outer loop 0.5; the counterbore rim by the tool's lead-in |

The hub's two faces are chamfered 0.5 mm along the runs of 20 mm or more, and the head run stays sharp so the head keeps
its 0.4 tip round and its lock contact. `edgefinish.ChamferRuns` chooses the edges on the extruded prism, then chamfers in
two decad calls: the partial runs (the arm's outer arc and the inner run of every pawl, 36 edges on both faces), then,
at the 8 mm presets, the whole D loop of each face. Each run ends at a convex corner (the web, the lock face's root)
or at the concave 22.5 degree corner beside the head's flank, where the bevel tapers out on the flank's plane over
about 1.5 mm. The 605ZZ's D loop is 15 mm long and stays sharp. The result is faceted, so its checks are sections 0.25 mm
inside each face (`TestOneWayClutchEdgeFinishing`, and `TestOneWayClutchRingEdgeFinishing` for the ring), not volumes.

## Corners and roots

All four printed parts print with their axis vertical. Rounded as the rule asks: web roots 1.0, slot ends full round
1.025, the plate boss step as a 45° bevel, every cone-to-cylinder or cone-to-flat turn at 45°. The arm (1.2) meets the
bending-part minimum; every other wall is 0.8 or more. The hub, back plate and cover are press-fit rings with walls of
1.6 or more round the bore.

Retained sharp corners. `TestOneWayClutchCorners` reads sections at every layer run of each part, for every preset
and both directions, and exempts the back plate's hex pocket corners only, each exemption matching a finding in every case. Each corner below is kept sharp by design. It
either passes the rule's limits (inside turn up to 50°, tip turn up to 120°) or lies between layers, where no section
reads it.

| Part | Corner | Reason | Test reading |
| --- | --- | --- | --- |
| Hub | each head's tip (0.4 round) and lock-face edges | Engagement corners of a pawl tooth. The 51.4° tip is blunted by the rule's minimum 0.4 round. | passes the tip limit |
| Hub, back plate, cover | the two corners of the D bore | Mating corners of the shaft's flat. | passes the inside limit (turn 28.8° to 36.5°) |
| Ring | the 24 tooth roots, 0.8 round | A ratchet tooth root, rounded below half its section: 0.8 is the largest round that keeps the head's 0.40 gap. | passes the inside limit |
| Ring | the 24 tooth tips | Engagement corners. | passes the tip limit |
| Cover | the floor rims of the three head pockets (between layers) | The 0.3 mm head clearance leaves no room. | between layers, not read |
| Back plate | the six corners of each of the three hex pockets (18), edges vertical | Mating corners: they match the M2 nut's sharp corners, as in `models/crankhandle`. | read in the z 0 to 2.0 run: 60° inside turns; exempt with `cornercheck.Annulus` about each site, radius 2.51 ± 0.3 |
| Back plate | the ceiling rim of each hex pocket at z 2.0 | A 45° chamfer of 0.5 mm would fill the corner the nut's corners sit in. | between layers, not read |
| Ring | the six ear root rounds (1.5), ear end rounds and bores | Rounded as the rule asks: 1.5 is above half the 2.5 mm wall. | passes |

The test also checks each press-fit ring's wall: in every layer run, the hub's, back plate's and cover's bore lies
1.6 mm or more from every other outline.

Not applicable: Split shafts (no printed shaft or pin), Gear train clearances (no gears), Arbor collars (the shaft is
the mechanism's).

## Printing and assembly

- Material: PETG is recommended for the hub, since its arms flex at every click; PLA works at 0.65 % strain.
- Hub and ring: either face down, no supports. The ears are vertical prisms with no overhang. The ring's tooth ends overhang 2.5 mm at the lower end; the 0.85 mm gap
  to the plate absorbs a little droop.
- Back plate: outer face down, no pause. The hex pockets open on the bed; each pocket's ceiling bridges at most 5.02 mm
  around the 2.6 mm bolt hole.
- Cover: outer face down; the head pockets open on the bed.

Assembly. The nuts are not held until their bolts reach them, so assemble off the shaft with the bolts loose:

1. Push the three M2 x 16 bolts through the cover's head pockets. Stand the cover on its outer face, bolts up.
2. Slide the hub over the bolts onto the cover.
3. Push each pawl head in and slide the ring over the hub, turning it the way it turns freely.
4. Lay the back plate on, inner face down, with the bolts through its holes.
5. Drop an M2 nut into each hex pocket. Lift the stack, holding the plates together, and turn each bolt from the
   cover's side until it catches its nut. Leave them loose.
6. Press the clutch onto the shaft, back plate first, its outer face toward the mechanism. The loose bolts let the three
   D bores line up on the shaft.
7. Tighten the three bolts from the front. The pockets stop the nuts turning.
