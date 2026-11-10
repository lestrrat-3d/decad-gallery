# Hand-crank siren

## Purpose

Build a hand-cranked siren. A hand crank on the gearbox's rear face turns a two-stage spur step-up train, and the train
spins a slotted rotor disc with paddle vanes in front of a fixed slotted stator disc, inside a drum with an outlet
horn. Each time a rotor port passes a stator port the air flow opens and closes once, so the pitch is
`f = Slots × Ratio × n_crank`. With 20 ports, a ratio of 10 and 2 crank turns per second the rotor turns 20 rev/s
(1200 rpm) and the pitch is 400 Hz. Over 12 to 24 ports the pitch runs from 240 Hz to 480 Hz at that crank speed.

The gears are the `models/spurgear` construction, the bearings are `models/ballbearing`, the bolts are `models/m2bolt`
and the nuts `models/m2nut`, all imported and not copied. Printed parts: everything except the 6 bearings, 16 bolts and
16 nuts.

## Compliance with AGENTS.md

- Sketch constraints: every sketch the build uses (including bearing, bolt, nut and fastener-tool sketches) is fully
  constrained, with DOF 0, no redundant constraint and no conflict, on every parameter branch.
- Fasteners: every head sinks into a pocket and ends flush, every nut sits in a cavity its part encloses, and every
  bolt has a standard M2 length. See Fasteners.
- Part clearances: every pair of separate printed parts keeps a gap of its class. See Fits. A bearing seat takes
  0.05 mm of radial clearance (the bearing-seat rule).
- Gear train clearances: both meshing pairs keep the tip-to-root clearance of 0.5 and the non-meshing gears keep their
  tip circles 0.5 apart. See Gear train clearances.
- Surface edge finishing: see Edge finishing.
- Body containers: see Containers.
- `models/README.md` marks the model **Mixed**.

## API and parameters

Package `handcranksiren`.

- `Ratio = 10.0` is the rotor turns per crank turn, `(50/16)(48/15)`.
- `Params{CrankAngle float64; Slots int}`. `DefaultParams()` is `{5π/4, 20}`.
- `CrankAngle` is the forward turn, clockwise seen from behind (about −Y), from the arm pointing up. `Validate` requires
  `CrankAngle` and `Ratio*CrankAngle` finite.
- `Slots` is the number of ports in the rotor and in the stator and of vanes on the rotor. `Validate` requires it in
  `[12, 24]`.
- `PitchHz(slots int, crankRevPerSecond float64) float64` is `float64(slots) * Ratio * crankRevPerSecond`.
  `PitchHz(20, 2) == 400`, `PitchHz(12, 2) == 240`, `PitchHz(24, 2) == 480`.
- `State{CrankAngle, IntermediateAngle, RotorAngle; Rotor, Intermediate, Crank r3.Vec; ArmDirection r3.Vec}` and
  `Motion(Params) (State, error)`. `IntermediateAngle = −(50/16)·CrankAngle`, `RotorAngle = Ratio·CrankAngle`,
  `ArmDirection = (−sin CrankAngle, 0, cos CrankAngle)`.
- `NewDocument(opts ...decad.DocumentOption)` returns `decad.New(WithChainTolerance(0.01 mm), opts...)`. The rotor's
  unions need the tolerance.
- `Build(ctx, doc, Params) (*Assembly, error)` returns `Assembly{Parts, Sketches, State, Params}`. `Part` is
  `{Name, Group, Motion string; Body *decad.Body}`. The build has 55 parts for every `Slots`.
- Errors read `hand-crank siren <part>: <operation>: <cause>`.

## Frame and orientation

- X runs to the listener's right, Y away from the listener (the horn points to −Y), Z up. The table is Z = 0.
- The operator stands behind (+Y) and turns the crank on the gearbox's rear face.
- `R`, `I` and `C` are the rotor, intermediate and crank axes. All three run along Y.
- Angles about an axis are measured in the XZ plane from +X toward +Z.

## Crank and rotor motion and pitch

- Motions: `fixed` (housing, stator, horn, bearings, every bolt and nut except the crank-pin pair), `crank`
  (crank-gear, crank-shaft, crank-arm, crank-pin, crank-handle, crank-pin-bolt, crank-pin-nut), `intermediate`
  (intermediate-pinion, intermediate-gear, intermediate-shaft) and `rotor` (rotor-pinion, rotor-shaft, rotor).
- The pose turns each part about the line through its axis point along −Y: `crank` by CrankAngle, `intermediate` by
  IntermediateAngle, `rotor` by RotorAngle.
- At the default the rotor angle is 12.5π, a whole number of port pitches for 12, 20 and 24 ports, so the ports line
  up (open).
- Pitch: each rotor port passing a stator port opens and closes the air flow once, so `PitchHz` gives the tone.

## Dimensions

All lengths are mm and all angles radians.

### Layout

| Name | Value |
| --- | --- |
| module, gearThick, backlash | 1.5, 6, 0.15 |
| teeth | crank 50, intermediate pinion 16, intermediate gear 48, rotor pinion 15 |
| a1 (I–C), a2 (R–I) | 49.5, 47.25 |
| R, C | (0, 0, 84), (60, 0, 84) |
| I | x 28.18594 = `(60² + a2² − a1²) / 120`, z 46.07751 = `84 − sqrt(a2² − I.x²)` |
| root radius | `rp − module − 0.5 − 0.02` |
| tip radius | `rp + module − 0.02` |
| tip / root radii | 15T 12.73 / 9.23, 16T 13.48 / 9.98, 48T 37.48 / 33.98, 50T 38.98 / 35.48 |
| tip-to-root | 0.54 on both pairs |

Axial stack (Y ranges):

| Body or feature | Y range |
| --- | --- |
| horn | [−139.90, −31.90], flange [−39.90, −31.90] |
| stator | [−31.65, −28.65] |
| rotor disc | [−27.50, −24.50], hub and vanes drawn from −25.50 to −16.50 |
| drum | flange [−28.40, −21.40], wall [−28.40, −11.50], back plate [−15.50, −11.50], posts [−12.50, −0.25] |
| gearbox-front | [0, 10] |
| gearbox-frame | [10.25, 27.35] |
| gearbox-rear | [27.6, 37.6] |
| gear plane 2 | [12.6, 18.6] (intermediate-gear, rotor-pinion) |
| gear plane 1 | [20.6, 26.6] (crank-gear, intermediate-pinion) |
| front bearings | [2.85, 9.85], pocket [2.6, 10], relief [1.6, 2.6] |
| rear bearings | [27.75, 34.75], pocket [27.6, 35.0], relief [35.0, 36.0] |
| rotor-shaft | [−25.25, 35.70], collar radius 5.5 at [10.1, 12.1] |
| intermediate-shaft | [2.90, 34.70] |
| crank-shaft | [2.90, 45.35], collar radius 5.5 at [19.1, 20.1] |
| crank-arm | [38.6, 46.6] |
| crank-pin | stub [42.1, 47.15], shank [46.85, 94.0], head [93.6, 96.6] |
| crank-handle | [47.0, 97.0] |

### Gear train

- Each gear is a `spurgear.Variant` build with the root deepened by `internal/gearroot` (clearance
  `max(0.25·module, 0.5)` = 0.5, margin 0.02). Pressure angle 20°, Samples 15, Module 1.5, Thickness 6, Backlash 0.15,
  BoreDiameter 0.
- `CapChamfer` is 0.2 for the 48T and 50T gears and 0 for the 15T and 16T gears.
- Placement frame origin: the gear's axis point at the front Y of its plane; normal +Y.

| Part | Axis | Plane | Role | U |
| --- | --- | --- | --- | --- |
| crank-gear (50T) | C | 1 [20.6, 26.6] | stage 1 driver | unit(I − C) |
| intermediate-pinion (16T) | I | 1 | stage 1 driven | unit(C − I) turned by π/16 about +Y |
| intermediate-gear (48T) | I | 2 [12.6, 18.6] | stage 2 driver | unit(R − I) |
| rotor-pinion (15T) | R | 2 | stage 2 driven | unit(I − R) turned by π/15 about +Y |

- Tooth zero is centred on U, so each driver's tooth points at its mate and each driven gear's gap points back.
- After the gear builds, cut each gear's D-bore: a cylinder of radius 3.75 + 0.25 minus a box beyond the flat 3.0 + 0.25
  from the axis toward world +Z. The bore runs from 1 before the gear's front face for 8. The flat faces world +Z.

### Shafts and bearings

- Every shaft is a cylinder of radius 3.75 with a flat 0.75 deep (3.0 from the axis toward +Z) along its whole length.
- rotor-shaft: Y [−25.25, 35.70] with a collar of radius 5.5 over [10.1, 12.1].
- intermediate-shaft: Y [2.90, 34.70].
- crank-shaft: Y [2.90, 45.35] with a collar of radius 5.5 over [19.1, 20.1]. The collar keeps crank-gear in plane 1.
- Six 608-2RS bearings (`ballbearing.DefaultParams()`, 8 × 22 × 7): `bearing-rotor-front`,
  `bearing-intermediate-front`, `bearing-crank-front` at Y 2.85, and `bearing-rotor-rear`,
  `bearing-intermediate-rear`, `bearing-crank-rear` at Y 27.75. Holders are gearbox-front and gearbox-rear.
- rotor-shaft floats 0.25 forward and 0.3 backward. If both gearbox seams (T1) close, the rear plate pushes the shaft at
  most 0.2 forward, inside the 0.25.
- intermediate-shaft is captive between two blind seats, 1.3 from each relief floor. crank-shaft's front end sits in a
  blind seat (1.3). The arm is pressed on the rear end.
- The rotor hub, the crank arm and the gears are pressed onto their D-flats.
- Clamping may close the Tight seams T2 and T3 by 0.25 each, which carries the stator 0.5 toward the rotor. The
  rotor-to-stator nominal gap is 1.15. The vane-to-back-plate nominal gap is 1.0, and the worst case is 0.7.

### Gearbox

- Outline: rounded rectangle X [−21, 107], Z [0, 131], corner radius 13. Frame cavity X [−14, 100], Z [7, 124], corner
  radius 6. Frame wall 7 thick.
- gearbox-front: the outline from Y 0 for 10. On R, I and C: seat pocket radius 11.05 over [2.6, 10] plus 1 of stock,
  relief radius 9.5 over [1.6, 2.6]. On R: through hole radius 4.2 from −1 to 1.6. Edge finish with offsets {0, 10}.
  Fastener cuts for `gearbox-front-bolt-*`, then the hole and cavity for each `drum-bolt-*`.
- gearbox-rear: the outline from Y 27.6 for 10. Seat pockets radius 11.05 over [26.6, 35.0], reliefs radius 9.5 over
  [35.0, 36.0], and on C a through hole radius 4.2 over [36.0, 38.6]. Edge finish offsets {27.6, 37.6}. Fastener cuts
  for `gearbox-rear-bolt-*`.
- gearbox-frame: outline plus cavity on the plane Y 10.25, extruded 17.1. Edge finish offsets {10.25, 27.35}. Hole and
  cavity for every gearbox-front and gearbox-rear bolt.

### Drum, rotor, stator and horn

- drum: a revolved meridian (4.2, −11.50), (48.5, −11.50), (49, −12.00), (49, −21.40), (57.5, −21.40), (58, −21.90),
  (58, −27.90), (57.5, −28.40), (46.5, −28.40), (46, −27.90), (46, −15.50), (4.2, −15.50) about the axis R along Y.
  Three windows (chord-sided port 14 to 24 radius, half angle 40°, centred at 60°, 180°, 300°... patterned with
  Count 3) are cut through the back plate. Three posts of radius 5 at radius 30 from R, Y [−12.50, −0.25] (angles 0°,
  120°, 240°).
- rotor: disc radius 45 over [−27.50, −24.50]; vanes u [8.5, 44], v [−1, 1], Y [−25.50, −16.50], one per slot, centred
  on the lands; hub radius 9 from −25.50 for 9; ports (chord-sided, radii 30 to 42, half angle π/(2N)) cut through the
  disc; D-bore from −25.50 for 10, leaving a floor 2 thick.
- stator: disc radius 58 over [−31.65, −28.65]; the same ports; a hole for each `horn-bolt-*`.
- horn: a revolved meridian, half angle 15°, length 100, wall 2.5, mouth inner radius 70.7949. Vertices (r, y):
  A (44.5, −31.90), B (57.5, −31.90), C (58, −32.40), D (58, −39.40), E (57.5, −39.90), F (46.5, −39.90),
  G (73.1655, −139.4170), H (72.7949, −139.90), I (71.2949, −139.90), J (70.6655, −139.4170), K (44, −39.90),
  L (44, −32.40).
- Port shape: chord-sided (straight inner and outer edges), not arc-sided. Decad refuses to pattern 20 arc-sided
  instances.
- Air path: air enters through the 11.25 gap behind the drum, passes the three windows, and the vanes throw it outward to
  the port ring. It leaves through the ports whenever a rotor port lines up with a stator port.

### Crank

- crank-arm: a stadium on the plane Y 38.6 about (60, 84): two arcs of radius 10 at the origin and at 55 toward +Z,
  four tangent lines, and a circle of radius 1.3 at the pin end. Extruded 8, caps chamfered 0.5. D-bore cut from 37.6
  for 8 (floor at Y 45.6). A hexagon 7.43 across flats about the pin axis from 41.85 to 47.6, corners on X. Then the
  crank-pin-bolt head pocket.
- crank-pin: a hexagon 6.93 across flats over [42.1, 47.15], a shank of radius 4.5 over [46.85, 94.0], a head of
  radius 5.8 over [93.6, 96.6], then the hole and cavity for crank-pin-bolt.
- crank-handle: revolved about the pin axis (60, y, 139), Y [47.0, 97.0], meridian (axial, radius) from the fan's
  profile shifted +1: (1.5, 4.9), (47.2, 4.9), (47.2, 6.4), (50.5, 6.4), (51, 6.9), (51, 8.3), (49.8, 9.3), (44, 8),
  (37, 7.25), (23, 8), (9, 7.8), (2.2, 7.2), (1, 6), (1, 5.4).

### Fasteners

16 bolts (12 M2 × 16, 3 M2 × 18, 1 M2 × 10) and 16 nuts. The pocket and cavity sizes come from `internal/m2fastener`:
head pocket radius 2.0 and depth 2.0, hole radius 1.3, nut cavity 4.2 across flats and 1.8 high, tool stock 1.0.

| Bolts | Nuts | Len | Head top | Dir | Corner | Cavity | Head host | Nut host |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `gearbox-front-bolt-1..4` | `gearbox-front-nut-1..4` | 16 | (−17.5, 0, 74), (103.5, 0, 74), (30, 0, 127.5), (30, 0, 3.5) | +Y | Z, Z, X, X | 11.75 | gearbox-front | gearbox-frame |
| `gearbox-rear-bolt-1..4` | `gearbox-rear-nut-1..4` | 16 | (−17.5, 37.6, 50), (103.5, 37.6, 50), (75, 37.6, 127.5), (75, 37.6, 3.5) | −Y | Z, Z, X, X | 11.75 | gearbox-rear | gearbox-frame |
| `drum-bolt-1..3` | `drum-nut-1..3` | 18 | R + 30(cos φ, 0, sin φ) at Y −15.50, φ = 0°, 120°, 240° | +Y | (cos φ, 0, sin φ) | 17.0 | drum | gearbox-front |
| `horn-bolt-1..4` | `horn-nut-1..4` | 16 | R + 53.5(cos ψ, 0, sin ψ) at Y −21.40, ψ = 45° + 90°(k−1) | −Y | (−sin ψ, 0, cos ψ) | 12.0 | drum | horn |
| `crank-pin-bolt` | `crank-pin-nut` | 10 | (60, 38.6, 139) | +Y | X | 4.75 | crank-arm | crank-pin |

Derived: front-bolt cavities at Y [11.75, 13.55] with tips at 18. Rear-bolt cavities at [24.05, 25.85] with tips at 19.6.
Drum cavities at [1.5, 3.3] with tips at 4.5. Horn cavities at [−33.40, −35.20] with tips at −39.40, 0.5 inside the horn
flange's front face. The pin cavity is at [43.35, 45.15] with its tip at 50.6, inside the shank.

#### Print pause

- `gearbox-frame` has two pauses (front nuts near the bed, rear nuts near the top).
- `gearbox-front` prints inner face down, and its drum nuts sit near the top.
- `horn` prints flange down and mouth up.
- `crank-pin` prints head down.
- `drum` prints standing on its flange's front face, and its back plate needs slicer support.

### Fits

The nominal gap is the class minimum plus 0.05. The test proves each lower bound with section outlines.

| ID | Parts | Class (min) | Nominal | How measured |
| --- | --- | --- | --- | --- |
| T1 | gearbox-front / gearbox-frame, gearbox-frame / gearbox-rear | Tight 0.2 | 0.25 | axial gap (Y) |
| T2 | drum posts / gearbox-front | Tight | 0.25 | axial bounds |
| T3 | stator / drum, horn / stator | Tight | 0.25 | axial bounds |
| T4 | each shaft / each of its bearings | Tight | 0.25 | section at the bearing's mid-plane |
| T5 | each D-bore / its shaft (4 gears, rotor hub, crank-arm) | Tight | 0.25 | section at the bore's mid-plane |
| T6 | shaft ends / bore floors and blind relief floors | Tight | 0.25 / 0.3 (rotor rear) / 1.3 | axial section |
| T7 | rotor-shaft collar / bearing-rotor-front | Tight | 0.25 | axial section through R |
| T8 | crank-pin stub / crank-arm pocket | Tight | 0.25 | section at Y 44 |
| T9 | rotor-pinion / rotor-shaft collar | Tight | 0.5 | axial section through R |
| T10 | crank-gear / crank-shaft collar | Tight | 0.5 | axial section through C |
| S1 | each bearing / its seat | bearing-seat rule, proven ≥ 0.02 | 0.05 | section at mid-plane |
| S2 | each bearing / its seat floor | Tight | 0.25 | holder section 0.2 from the floor-side face has no material at radius 10.25 on 12 angles |
| R1 | rotor-shaft / gearbox-front hole and drum hole; crank-shaft / gearbox-rear hole | Regular 0.35 | 0.45 | section mid-hole |
| R2 | rotor / stator | Regular plus clamp budget, proven ≥ 1.10 | 1.15 | axial bounds |
| R3 | crank-handle / crank-pin | Regular | 0.4 | sections at Y 70 and 95; axial plane |
| R4 | crank-arm / crank-handle | Regular | 0.4 | axial bounds |
| L1 | rotor / drum | Loose 0.5 | 1.0 | section at Y −26.0 and axial plane through R |
| L2 | each gear / gearbox-front, gearbox-rear, gearbox-frame | Loose | ≥ 1.0 | axial bounds; section at each of 5 mesh samples |
| L3 | plane-2 gears / plane-1 gears; crank-shaft / intermediate-gear; rotor-shaft / crank-gear | Loose | 2.0 / 8.27 / 17.3 | axial bounds; sections |
| L4 | each gear / nearest bearing | Loose | ≥ 1.15 | axial plane section |
| L5 | crank-arm / gearbox-rear | Loose | 1.0 | axial bounds |

Nominal exceptions: a bolt shank in its own nut; a bolt head on its pocket floor (proved with `doc.ContactPair`, gap at
most 0.05). The two meshing pairs fall under Gear train clearances.

### Gear train clearances

For each pair, turn the driver by θ_k = k·(2π/Z_driver)/4 for k = 0..4 and the driven gear by −θ_k·Z_driver/Z_driven,
each about its own axis. Take the section at the plane's mid-Y (23.6 for stage 1, 15.6 for stage 2), keep each gear's
outer loop, and require:

- reach − root ≥ 0.5 in both directions;
- each root loop within 0.05 of the gear's root radius;
- no overlap of the two outlines;
- the centre distance equals `a1` or `a2` within 1e-9;
- the L2 tip-to-frame gaps at that sample are at least 0.5.

### Edge finishing

| Body | Large planar faces | Finishing | Retained sharp |
| --- | --- | --- | --- |
| gearbox-front, gearbox-rear | both faces | `edgefinish` Long(All) 0.5 | bolt pockets and holes (loops < 20 mm) |
| gearbox-frame | both ends | `edgefinish` Long(All) 0.5 | bolt holes (short) |
| drum | back plate rear face, flange faces | meridian chamfers; `edgefinish` Long(Inner) on windows and shaft hole | concave junctions |
| stator | both faces | cap outer loops 0.5 | ports: functional chopping edge |
| rotor | front face, disc back face | cap outer loops 0.5 | ports: functional chopping edge |
| horn | flange faces, mouth lip | meridian chamfers | none |
| crank-arm | both faces | cap outer loops 0.5 | pin bolt hole; D-bore and hex pocket openings (Tight fits) |
| crank-gear, intermediate-gear | both caps | `CapChamfer` 0.2 | 0.5 consumes the tooth tip land at module 1.5 |
| intermediate-pinion, rotor-pinion | none: net cap areas 381 and 327 mm² | none | not qualifying |

### Containers

`Housing` (gearbox-front, gearbox-frame, gearbox-rear, drum), `Stator`, `Rotor`, `Horn`, `Gear Pairs` with children
`Pair` (crank-gear, intermediate-pinion) and `Pair 2` (intermediate-gear, rotor-pinion), `Shafts`, `Bearings`, `Crank`
(crank-arm, crank-pin, crank-handle), `Bolts`, `Nuts`. A container exists only when it holds a body.
`doc.VerifyContainers().Err()` is nil.

## Pose

Build every part at crank angle 0, pose it by its motion, move it into its container last.

## Verification

`go test ./models/handcranksiren` covers the default and the `Slots` 12 and 24 variants: parts, sketches, containers,
ports, orientation, overlap, fits R2 and L1 on the variants, and on the default also the mesh, clearances, fasteners,
sweep, tessellate and edges checks.
