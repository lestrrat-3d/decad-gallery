# One-way clutch instructions

A printed one-way clutch for a D-flat shaft. A hub pressed on the shaft carries three flexing pawls. They engage a ring
with 24 internal sawtooth teeth, so the ring turns freely one way about the held shaft and is driven the other way. A
back plate and a cover hold the ring on the hub, clamped by three M2 bolts and nuts. Every part except the bolts and
nuts prints on an FDM printer with its axis vertical and no supports. The shaft is not modeled, as in `crankhandle`:
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

The body does not change with the preset: only the D bore of the hub and both plates changes. D bore radius
`r_b = D/2 + c`, flat at `f_b = D/2 - d + c` from the axis on the +Y side, where `c` is `BoreClearance`.

### Validation (first failure wins; every message starts `onewayclutch: `)

1. `Shaft` names a preset: `unknown shaft preset "<name>"; want one of 605ZZ, 608ZZ, 608ZZ-gearbox`.
2. `Freewheel` is `Clockwise` or `CounterClockwise`.
3. `BoreClearance` is finite and `0.05 <= c <= 0.5`.
4. `BoltCircle - NutCavityRadius - r_b >= PressWall`, that is `r_b <= 4.29`; the message says the bore leaves less than
   1.6 mm of wall between the bore and the nut cavities.
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
| `RingBand` | 13.5 | blank bore radius in the tooth band, cut away by the tooth tool |
| `RingCounter`, `RingOuter`, `RingChamfer` | 18.5, 21.0, 0.5 | counterbore, outside radius (OD 42), OD end lead-ins |
| `CoreRadius` | 10.7 | hub core |
| `SlotWidth`, `ArmThickness` | 2.05, 1.2 | slot under each arm; arm (three 0.4 mm lines, the bending-part minimum) |
| `ArmInner`, `ArmOuter` | 12.75, 13.95 | `R_c + s`, `+ t` |
| `WebAngle`, `SlotAngle`, `LockAngle` | -20°, -6°, 90° | pawl 0's web back face, slot end, and ring lock face |
| `WebRound`, `TipRound`, `HeadGap` | 1.0, 0.4, 0.40 | web root round, head tip round, head to its gap (seated) |
| `HubHeight`, `Seam` | 10.0, 0.25 | hub and pawls; hub to each plate boss |
| `PlateRadius`, `Recess` | 18.1, 0.40 | `RingCounter - RegularFit`; annulus below the boss face |
| `BackPlateThickness`, `CoverThickness` | 5.0, 4.0 | outer face to boss face |
| `PlateChamfer` | 0.5 | outer-face chamfer |
| `BoltCircle`, `BoltAngles`, `BoltLength` | 8.4, 30° 150° 270°, 16 | three M2 x 16 through bolts |
| `PressWall` | 1.6 | least wall round a press-fit bore |

### Stack (z along the axis)

| Part | z from | z to | Notes |
| --- | ---: | ---: | --- |
| Back plate | 0 | 5.0 | boss face 5.0, annulus 4.6, nut cavities 1.5 to 3.5 |
| Hub | 5.25 | 15.25 | 0.25 seams to both bosses |
| Cover | 15.5 | 19.5 | boss face 15.5, annulus 15.9, head pockets 17.5 to 19.5 |
| Ring | 1.25 | 19.25 | symmetric about z 10.25, the hub's mid-plane |
| Ring tooth band | 5.45 | 15.05 | 9.6 tall, inside the heads' 5.25 to 15.25 |

The assembly is 42 mm across and 19.5 mm long.

## Construction

### Ring (revolve, then one cut)

Meridian `(r, z)`, closed, revolved a full turn about Z: (20.5, 1.25), (21.0, 1.75), (21.0, 18.75), (20.5, 19.25),
(18.5, 19.25), (18.5, 16.55), (17.0, 15.05), (13.5, 15.05), (13.5, 5.45), (17.0, 5.45), (18.5, 3.95), (18.5, 1.25).
The counterbores take the plate rims; the 45° cones from `RingCounter` down to `ToothRoot` hold the ring axially; the
flats at z 5.45 and 15.05 become the teeth's end faces.

Teeth: one sketch on z = 0.25 holding the whole void inside the teeth, extruded 20 along +Z and cut from the blank.
Counter-clockwise layout: lock face `j` lies on the radial axis at `φ_j = 90° + 15°j`, from `Q_j = R_root∠φ_j` in to
`P_j = R_tip∠φ_j`; the ramp of tooth `j` runs from `P_j` to `Q_{j+1}`. Each `Q_j` is rounded by `RootRound`. The tooth
occupies the angles just counter-clockwise of each lock face; the gap where a head sits lies just clockwise. The ramp
line is at distance 13.289 from the axis (`RampDistance`), at 141.417° (`RampAngle`).

### Plates (prism, chamfer, cuts)

Profile on the plate's inner-side plane (back plate z = 0, cover z = 15.5): the circle `PlateRadius` about the origin,
the D bore (arc of `r_b` the long way through -Y, flat at `y = f_b`), and three circles of `m2fastener.HoleRadius` (1.3)
at `BoltCircle` on 30°, 150° and 270°. Extrude 5.0 or 4.0 along +Z. Then:

1. Chamfer the outer face (back plate z = 0, cover z = 19.5) by 0.5 in one call: its outer circle, plus the whole D loop
   when `BoreLoop >= 20` (the 8 mm presets; the 5 mm loop is 15.8). The bolt-hole and head-pocket loops are short.
2. Cut the inner-face tool, a revolved meridian `(r, z)`. Back plate: (9.8, 6.0), (9.8, 5.4), (10.6, 4.6), (17.0, 4.6),
   (19.1, 2.5), (19.1, 6.0). Cover: (9.8, 14.5), (9.8, 15.1), (10.6, 15.9), (17.0, 15.9), (19.1, 18.0), (19.1, 14.5).
   It leaves the boss (radius 10.2 at the face, a 0.4 mm 45° bevel), the recessed annulus out to r 17.0 and a 45° cone
   to the rim.
3. Back plate: cut `m2fastener.CavityTool` at each site (z 1.5 to 3.5). Cover: cut `m2fastener.PocketTool` at each
   site (z 17.5 to 19.5). The holes are already in the profile.

### Hub (one prism)

One sketch on z = 5.25 holds the whole outline, the D bore and the three bolt holes; extrude `HubHeight` along +Z. The
hub has no Boolean. Its faces are not chamfered yet (see "Surface edge finishing").

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

Site `i` at angle `a_i` in {30°, 150°, 270°}: `m2fastener.Site{Head: (8.4 cos a, 8.4 sin a, 19.5), Dir: (0, 0, -1),
Corner: (cos a, sin a, 0), Length: 16, Cavity: 16.0}`. The head lies flush with the cover's outer face in a 2 mm pocket.
The bolt passes the cover, the hub and the back plate. The nut cavity (z 1.5 to 3.5) is enclosed by the back plate
except for the bolt hole. The tip ends at z 1.5, the cavity's far face. M2 x 16 is in `m2bolt.StandardLengths()`.

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
| P1 | hub D bore / shaft | press fit | `c` (0.05) | 0.040 (at `c` 0.35: 0.340) | section at mid-height |
| P2 | back plate, cover D bores / shaft | press fit | `c` | 0.049 (at `c` 0.35: 0.349) | as P1 |
| T1 | hub / each plate boss | Tight 0.2 | 0.25 | 0.237 | mesh gap, placed meshes |
| R1 | pawls / plate annuli | Regular 0.35 | 0.65 | 0.642 | axial section, hub bounds |
| R2 | ring counterbores / plate rims | Regular | 0.40 radial | 0.390 | section mid-rim |
| R3 | ring cones / plate cones | Regular | 0.60 normal | 0.591 | axial section |
| R4 | ring tooth ends / plate annuli | Regular | 0.85 axial | 0.840 | axial section |
| R5 | each head / its tooth gap, seated | Regular | 0.40 | 0.387 | `meshgap` on the placed meshes |
| L1 | arm outer arcs / tooth tips | Loose 0.5 | 0.55 | 0.543 | section at mid-height |
| F | bolts, nuts | fastener rules | pocket 0.3, hole 0.3, cavity 0.2 | | bounds and no overlap |

The model is **Mixed**: the bore clearance is a parameter, and the bolts and nuts keep their nominal size. The ring's
outside fit in the user's housing belongs to the consumer. `TestOneWayClutchFits` measures P1 to R5 and L1 for every
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
- Ring tooth sketch: 24 axes at `φ_j`. `P_j` is `NewPointOnLine` on axis `j` with `NewDistance(origin, P_j, R_tip)`. The
  lock line `Q_j → P_j` is `NewParallel` to axis `j`. The ramp `P_j → Q_{j+1}` is `NewDistancePointLine(origin, ramp,
  13.289)`. `CreateFillet` at every `Q_j`, 0.8. One closed loop, no holes.
- Meridians (ring blank, plate inner tools): the plane holds the axis, U along +Z and V radial; every vertex
  `DimensionPoint`; revolve with `decad.FullRevolution{}`.
- Plate profile: the hub's bore and holes, plus `NewRadius(PlateRadius)` on a circle about the origin.
- Fastener tools: `m2fastener` returns their sketches; they are recorded in `Parts.Sketches`.

## Surface edge finishing

| Part | Face | Net area | Long runs | Action |
| --- | --- | ---: | --- | --- |
| Back plate | outer face | 964 to 994 | outer circle 113.7; D loop 25.1 to 25.3 at 8 mm | chamfer 0.5 (one call) |
| Cover | outer face | 934 to 964 | as above; head pockets 13.8, short | chamfer 0.5 (one call) |
| Plates | annulus | 555 | outer edge meets the 45° cone | the 1.1 mm cone is that edge's finish |
| Plates | boss face | 260 to 290 | none | small, none |
| Hub | each face | 409 to 439 | per pawl: arm outer arc 22.9, inner run 47.8; head run 7.1, short | pending (below) |
| Ring | end faces | 245 net | none | none; the OD end lead-ins are in the meridian |

The rule asks for 0.5 mm on the hub's long runs only. Decad refuses a chamfer of part of a cap loop, and refuses the
whole loop at 0.5 because of the head's 0.4 tip round, so the hub's finish waits on that decision. The hub is built
without a chamfer for now; this is a known gap, not a retained sharp edge.

## Corners and roots

All four printed parts print with their axis vertical. Rounded as the rule asks: web roots 1.0, slot ends full round
1.025, the plate boss step as a 45° bevel, every cone-to-cylinder or cone-to-flat turn at 45°. The arm (1.2) meets the
bending-part minimum; every other wall is 0.8 or more. The hub, back plate and cover are press-fit rings with walls of
1.6 or more round the bore.

Retained sharp corners (the corner test is a later task and must match each entry to a finding):

| Part | Corner | Reason |
| --- | --- | --- |
| Hub | each head's tip (0.4 round) and lock-face edges | Engagement corners of a pawl tooth. The 51.4° tip is blunted by the rule's minimum 0.4 round. |
| Hub, back plate, cover | the two corners of the D bore | Mating corners of the shaft's flat. |
| Ring | the 24 tooth roots, 0.8 round | A ratchet tooth root, rounded below half its section: 0.8 is the largest round that keeps the head's 0.40 gap. |
| Ring | the 24 tooth tips | Engagement corners. |
| Cover | the floor rims of the three head pockets (between layers) | The 0.3 mm head clearance leaves no room. |
| Back plate | the floor and roof rims of the three nut cavities (between layers) | A round would cut into the nut's 0.2 mm face clearance. |

Not applicable: Split shafts (no printed shaft or pin), Gear train clearances (no gears), Arbor collars (the shaft is
the mechanism's).

## Printing and assembly

- Material: PETG is recommended for the hub, since its arms flex at every click; PLA works at 0.65 % strain.
- Hub and ring: either face down, no supports. The ring's tooth ends overhang 2.5 mm at the lower end; the 0.85 mm gap
  to the plate absorbs a little droop.
- Back plate: outer face down. Pause at z 3.5, drop an M2 nut into each cavity, resume.
- Cover: outer face down; the head pockets open on the bed.

Assembly: press the back plate onto the shaft, outer face toward the mechanism; press the hub on; push each pawl head in
with a finger and slide the ring over the hub, turning it in the freewheel direction; press the cover on; drive the
three M2 x 16 bolts into the nuts and tighten.
