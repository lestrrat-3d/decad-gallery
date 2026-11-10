# Hand-crank siren

## Purpose

Build a hand-cranked siren. A hand crank on the right of the gearbox turns a two-stage spur step-up train, and the
train spins a slotted rotor disc with paddle vanes in front of a fixed slotted stator disc, inside a drum. A collector
turns the air 90 degrees into a horn that points forward, and a grip on the gearbox's left face gives the other hand
something to hold. Each time a rotor port passes a stator port the air flow opens and closes once, so the pitch is
`f = Slots × Ratio × n_crank`. With 20 ports, a ratio of 10 and 2 crank turns per second the rotor turns 20 rev/s
(1200 rpm) and the pitch is 400 Hz. Over 12 to 24 ports the pitch runs from 240 Hz to 480 Hz at that crank speed.

The gears are the `models/spurgear` construction, the bearings are `models/ballbearing`, the bolts are `models/m2bolt`
and the nuts `models/m2nut`, all imported and not copied. Printed parts: everything except the 6 bearings, 24 bolts and
24 nuts.

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
- `CrankAngle` is the forward turn, clockwise seen from the operator's right (+Y), about −Y, from the arm pointing
  up. `Validate` requires
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
  `{Name, Group, Motion string; Body *decad.Body}`. The build has 73 parts for every `Slots`.
- `GripContainerName = "Grip"` is the one new exported constant: the container of the grip.
- Errors read `hand-crank siren <part>: <operation>: <cause>`.

## Frame and orientation

- The held pose fixes what the build axes mean. The coordinates do not change. The table is Z = 0.

| Operator direction | Build axis | Content |
| --- | --- | --- |
| forward | −X | horn mouth at X −159.25; sound leaves along −X |
| right | +Y | crank arm, pin and handle (Y ≥ 38.6) |
| left | −Y | drum, stator, collector, grip (Y < 0) |
| up | +Z | table at Z 0; only the gearbox pieces reach it |

- The operator holds the grip with the left hand and turns the crank with the right. The listener stands in front,
  at −X. The horn is the frontmost part (smallest X of all parts).
- Seen from +Y, forward (−X) is on the viewer's right, so a forward turn moves the handle forward over the top.
  `DefaultParams` (arm down and toward the operator, +X) does not change.
- `R`, `I` and `C` are the rotor, intermediate and crank axes. All three run along Y.
- Angles about an axis are measured in the XZ plane from +X toward +Z.

## Crank and rotor motion and pitch

- Motions: `fixed` (housing, collector, stator, horn, grip, bearings, every bolt and nut except the crank-pin pair), `crank`
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
| collector | Y [−96.90, −31.90], X [−51, 51], Z [33, 135]; rear wall [−39.90, −31.90] |
| horn | X [−159.25, −51.25] about the axis (Y `throatY` = −65.90, Z 84); flange X [−59.25, −51.25] |
| grip | X [79, 105], Y [−76.25, −0.25], Z [5, 125] |
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

- Outline: rounded rectangle X [−21, 107], Z [0, 131], corner radius 13, drawn counter-clockwise (a
  clockwise winding leaves the sketch fillets over-constrained). Frame cavity X [−14, 100], Z [7, 124], corner
  radius 6. Frame wall 7 thick.
- gearbox-front: the outline from Y 0 for 10. On R, I and C: seat pocket radius 11.05 over [2.6, 10] plus 1 of stock,
  relief radius 9.5 over [1.6, 2.6]. On R: through hole radius 4.2 from −1 to 1.6. Edge finish with offsets {0, 10}.
  Fastener cuts for `gearbox-front-bolt-*`, then the hole and cavity for each `drum-bolt-*`, then the hole and cavity
  for each `grip-bolt-*`.
- gearbox-rear: the outline from Y 27.6 for 10. Seat pockets radius 11.05 over [26.6, 35.0], reliefs radius 9.5 over
  [35.0, 36.0], and on C a through hole radius 4.2 over [36.0, 38.6]. Edge finish offsets {27.6, 37.6}. Fastener cuts
  for `gearbox-rear-bolt-*`.
- gearbox-frame: outline plus cavity on the plane Y 10.25, extruded 17.1. Edge finish offsets {10.25, 27.35}. Hole and
  cavity for every gearbox-front and gearbox-rear bolt.

### Drum, rotor, stator, collector and horn

- drum: a revolved meridian (4.7, −11.50), (48.5, −11.50), (49, −12.00), (49, −21.40), (57.5, −21.40), (58, −21.90),
  (58, −27.90), (57.5, −28.40), (46.5, −28.40), (46, −27.90), (46, −15.50), (4.7, −15.50), (4.2, −15.00),
  (4.2, −12.00) about the axis R along Y; the shaft hole's two 0.5 chamfers are in the meridian.
  Three windows (chord-sided port 14 to 24 radius, half angle 40°, centred at 60°, 180°, 300°... patterned with
  Count 3) are cut through the back plate. Three posts of radius 5 at radius 30 from R, Y [−12.50, −0.25] (angles 0°,
  120°, 240°).
- rotor: disc radius 45 over [−27.50, −24.50]; vanes u [8.5, 44], v [−1, 1], Y [−25.50, −16.50], one per slot, centred
  on the lands; hub radius 9 from −25.50 for 9; ports (chord-sided, radii 30 to 42, half angle π/(2N)) cut through the
  disc; D-bore from −25.50 for 10, leaving a floor 2 thick.
- stator: disc radius 58 over [−31.65, −28.65]; the same ports; a hole for each `collector-bolt-*`.
- collector: a rounded-square box coaxial with R, Y [−96.90, −31.90]. Constants: rear face −31.90, ceiling −39.90,
  floor −91.90, end −96.90, outer half-width 51 (corner radius 10), cavity half-width 46 (corner radius 7), bore 44
  across, `throatY` −65.90, throat radius 22, −X wall faces at X −51 (outer) and −46 (inner). Built as the outer prism
  X [−51, 51], Z [33, 135]; the bore (radius 44 cylinder, Y [−40.90, −30.90]) and the cavity (X [−46, 46], Z [38, 130],
  Y [−91.90, −39.90]) cut; 0.5 chamfers on the outer loops of the rear and end faces, the bore rim on the rear face and
  the bore rim on the ceiling; the throat (radius 22, X [−52, −45], axis along −X through (−, −65.90, 84)) cut, with
  0.5 chamfers of its rim on both wall faces; then the hole and cavity for each `collector-bolt-*` and the pocket and
  hole for each `throat-bolt-*`.
- horn: a revolved meridian about the axis O = (−50, `throatY`, 84) along −X (U = +Y radius, V = −X axial, normal +Z),
  half angle 15°, length 100, wall 2.5, mouth inner radius 48.7949. The flange spans X [−51.25, −59.25] (8 thick) and
  the cone runs to X −159.25. Vertices (u radius, v axial): A (22.5, 1.25), B (29.5, 1.25), C (30, 1.75), D (30, 8.75),
  E (29.5, 9.25), F (24.5, 9.25), G (51.1655, 108.7670), H (50.7949, 109.25), I (49.2949, 109.25),
  J (48.6655, 108.7670), K (22, 9.25), L (22, 1.75). Then the hole and cavity for each `throat-bolt-*`.
- Port shape: chord-sided (straight inner and outer edges), not arc-sided. Decad refuses to pattern 20 arc-sided
  instances.
- Air path: air enters through the 11.25 gap behind the drum, passes the three windows, and the vanes throw it outward to
  the port ring. It leaves through the ports whenever a rotor port lines up with a stator port, passes into the
  collector through its 88 mm bore, crosses the 92 × 92 × 52 cavity, turns 90 degrees through the throat (22 radius,
  1520 mm², more than the 1352 mm² of open port area) in the collector's −X wall, and leaves along −X through the horn.

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

### Grip

One body, a vertical D-handle on the left (−Y) face of gearbox-front, toward the operator. Its ring profile lies in the
YZ plane on the frame origin (79, 0, 0), U = +Y, V = +Z, and is extruded along +X from X 79 for 26. The profile is the
outer outline Y [−76.25, −0.25] × Z [5, 125] with corner radius 10 and the finger window Y [−46.25, −16.25] × Z [12, 118]
with corner radius 3 as a hole, both counter-clockwise in (u, v); it has one hole. Then `edgefinish.Chamfer(+X,
{79, 105}, Long(All), 2.0)` on both caps, then the pocket and hole for each `grip-bolt-*`. The parts of the ring are a
pad (Y [−16.25, −0.25], 16 thick, its back face 0.25 off gearbox-front), a bar (Y [−76.25, −46.25], 26 × 30) and two
arms (Z [5, 12] and [118, 125], 7 thick). The window is 30 deep and 106 tall; the 3 mm corners leave 102.4 at 0.25 mm
from the bar face, 104.5 at 1 mm and the full 106 from 3 mm (hand sizes: four-finger breadth up to about 100, finger
depth up to about 25, power-grip diameter 30 to 50). The grip stands 10 mm further out than a thin pad would put it,
so the bent fingers have room before the stator and the drum rim.

- The bar's perimeter after the chamfers is 107.3, a 34.2 mm round bar. The left hand holds the bar thumb up, palm on
  the outer face (Y −76.25), fingers round the front face (X 79).
- The hand box X [57, 140], Y [−110, −46.25], Z [12, 118] holds no other part. The finger window box
  X [79, 105], Y [−46.15, −16.35], Z [12, 118] holds no part. No other part comes within 25 mm of the bar's inner front
  edge (X 79, Y −46.25) on the X < 79 side, at any Z across the window, and the window is at least 100 tall at the bar
  face. The grip is at Y ≤ −0.25 and the crank at Y ≥ 38.6, so the
  gearbox sits between the hands. The grip is at X ≥ 79 and the horn at X ≤ −51.25. Its lowest point is Z 5.
- The pad covers the head of `gearbox-front-bolt-2` (103.5, 0, 74) with a 0.25 gap. Remove the grip to reach it.

### Fasteners

24 bolts (12 M2 × 16, 3 M2 × 18, 5 M2 × 10, 4 M2 × 20) and 24 nuts. The pocket and cavity sizes come from `internal/m2fastener`:
head pocket radius 2.0 and depth 2.0, hole radius 1.3, nut cavity 4.2 across flats and 1.8 high, tool stock 1.0.

| Bolts | Nuts | Len | Head top | Dir | Corner | Cavity | Head host | Nut host |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `gearbox-front-bolt-1..4` | `gearbox-front-nut-1..4` | 16 | (−17.5, 0, 74), (103.5, 0, 74), (30, 0, 127.5), (30, 0, 3.5) | +Y | Z, Z, X, X | 11.75 | gearbox-front | gearbox-frame |
| `gearbox-rear-bolt-1..4` | `gearbox-rear-nut-1..4` | 16 | (−17.5, 37.6, 50), (103.5, 37.6, 50), (75, 37.6, 127.5), (75, 37.6, 3.5) | −Y | Z, Z, X, X | 11.75 | gearbox-rear | gearbox-frame |
| `drum-bolt-1..3` | `drum-nut-1..3` | 18 | R + 30(cos φ, 0, sin φ) at Y −15.50, φ = 0°, 120°, 240° | +Y | (cos φ, 0, sin φ) | 17.0 | drum | gearbox-front |
| `collector-bolt-1..4` | `collector-nut-1..4` | 16 | R + 53.5(cos ψ, 0, sin ψ) at Y −21.40, ψ = 45° + 90°(k−1) | −Y | (−sin ψ, 0, cos ψ) | 12.0 | drum | collector |
| `throat-bolt-1..4` | `throat-nut-1..4` | 10 | (−46, `throatY` + 26 cos a, 84 + 26 sin a), a = 45° + 90°(k−1) | −X | (0, cos a, sin a) | 6.75 | collector | horn |
| `grip-bolt-1..4` | `grip-nut-1..4` | 20 | (85, −16.25, 30), (99, −16.25, 30), (85, −16.25, 100), (99, −16.25, 100) | +Y | X | 17.75 | grip | gearbox-front |
| `crank-pin-bolt` | `crank-pin-nut` | 10 | (60, 38.6, 139) | +Y | X | 4.75 | crank-arm | crank-pin |

Derived: front-bolt cavities at Y [11.75, 13.55] with tips at 18. Rear-bolt cavities at [24.05, 25.85] with tips at 19.6.
Drum cavities at [1.5, 3.3] with tips at 4.5. Collector cavities at Y [−33.40, −35.20] inside the collector's rear wall
(8 thick), with tips at −39.40, 0.5 inside the cavity ceiling; the holes run Stock past the tip into the cavity.
Throat pockets go into the collector's −X wall from the cavity side, X [−46, −48], with 3 of material to the outer face;
the bolts cross the 0.25 seam into the horn flange. Throat nut cavities are at X [−52.75, −54.55], with tips at X −58,
1.25 inside the flange's outer face at −59.25. Throat bolts go in from inside the collector, through its 88 mm bore,
before the collector meets the stator. Grip pockets go into the pad's window face, Y [−16.25, −14.25], with 14 of material
under them. Grip nut cavities are in gearbox-front at Y [1.5, 3.3], the same layers as the drum cavities, with tips at
Y 5.75 and holes ending at 6.75, inside the 10-thick plate. The pin cavity is at [43.35, 45.15] with its tip at 50.6, inside the shank.

#### Print pause

- `gearbox-frame` has two pauses (front nuts near the bed, rear nuts near the top).
- `gearbox-front` prints inner face down, and its seven nuts sit near the top: three drum nuts and four grip nuts, in
  one pause.
- `collector` prints with its end face (Y −96.90) down; the ceiling bridges at most about 18 mm at the cavity corners.
  Its four collector nuts sit near the top.
- `horn` prints with its flange mating face (X −51.25) down and the mouth up. Its four throat nuts sit near the bed.
- `grip` prints with one X cap down. The ring profile is the layer shape, so it needs no support and no pause.
- `crank-pin` prints head down.
- `drum` prints standing on its flange's front face, and its back plate needs slicer support.

### Fits

The nominal gap is the class minimum plus 0.05. The test proves each lower bound with section outlines. Fits between
round parts (T4, T5, T7, T9, T10, T8, R3) use a section tolerance of 0.01, because at 0.02 the intermediate shaft's bound
against its bearing is 0.192, below the Tight minimum. Fits that include a printed plate use 0.02, the plates being
faceted with a mesh bound of 0.01615. S1 sections the bearing at 0.01 and the plate at 0.0162.

| ID | Parts | Class (min) | Nominal | How measured |
| --- | --- | --- | --- | --- |
| T1 | gearbox-front / gearbox-frame, gearbox-frame / gearbox-rear | Tight 0.2 | 0.25 | axial gap (Y) |
| T2 | drum posts / gearbox-front | Tight | 0.25 | axial bounds |
| T3 | stator / drum, collector / stator | Tight | 0.25 | axial bounds (Y): `axialGap(collector, stator, 1)` |
| T11 | horn / collector | Tight | 0.25 | axial bounds (X): `axialGap(horn, collector, 0)` |
| T12 | grip / gearbox-front; grip / gearbox-front-bolt-2 head | Tight | 0.25 / 0.25 | axial bounds (Y) |
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
| L6 | collector / drum; horn / stator; horn / drum | Loose | 3.5 / 4.25 / ≥ 4.25 | `axialGap(collector, drum, 1)`; section in the plane Z 84 for the horn rows |
| L7 | grip / collector, stator, drum | Loose | 28 / 21 / 21 | axial bounds (X) |

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
| drum | back plate rear face, flange faces | meridian chamfers (outer corners and the shaft hole); `edgefinish` Long(Inner) on the windows | concave junctions (the posts) |
| stator | both faces | cap outer loops 0.5 | ports: functional chopping edge |
| rotor | front face, disc back face | cap outer loops 0.5 | ports: functional chopping edge |
| collector | rear face, end face, cavity ceiling, −X wall faces | `edgefinish` 0.5: Long(All) on Y −31.90 and −96.90; Long(Inner) on Y −39.90 and on X −51, −46 | concave cavity corners and the horn-side junctions; bolt pockets and holes (loops < 20 mm) |
| horn | flange faces, mouth lip | meridian chamfers 0.5 (B/C, D/E, A/L, lip G to J) | flange outer face to cone junction (concave); throat-bolt holes (short) |
| grip | both X caps | `edgefinish` Long(All) **2.0** (deviation A7) | bolt pockets and holes (short); the window and outline corners are tangent arcs |
| crank-arm | both faces | cap outer loops 0.5 | pin bolt hole; D-bore and hex pocket openings (Tight fits) |
| crank-gear, intermediate-gear | both caps | `CapChamfer` 0.2 | 0.5 consumes the tooth tip land at module 1.5 |
| intermediate-pinion, rotor-pinion | none: net cap areas 381 and 327 mm² | none | not qualifying |

The grip's chamfer is the one deviation from the 0.5 minimum of "Surface edge finishing" (A7). It is 2.0, so the 26 × 30
bar rounds into a hand-sized grip. It is larger than the rule asks, and the edge test proves the 2.0 setback.

### Containers

`Housing` (gearbox-front, gearbox-frame, gearbox-rear, drum, collector), `Stator`, `Rotor`, `Horn` (horn), `Grip` (grip), `Gear Pairs` with children
`Pair` (crank-gear, intermediate-pinion) and `Pair 2` (intermediate-gear, rotor-pinion), `Shafts`, `Bearings`, `Crank`
(crank-arm, crank-pin, crank-handle), `Bolts`, `Nuts`. A container exists only when it holds a body.
`doc.VerifyContainers().Err()` is nil.

## Pose

Build every part at crank angle 0, pose it by its motion, move it into its container last.

## Verification

`go test ./models/handcranksiren` covers the default and the `Slots` 12 and 24 variants: parts, sketches, containers,
ports, outlet, orientation, overlap, fits R2 and L1 on the variants, and on the default also the grip, mesh,
clearances, fasteners, sweep, tessellate and edges checks.

- `outlet` (default and variants) sections the collector and the horn at `fitTolerance` 0.02: no material at radius 36
  about R in the collector at Y −35.90; no material at the throat centre or at radius 20 about it, and material at
  radius 23 on 0°, 90°, 180° and 270°, in the collector at X −48.5; no material at radius 30 and material at radius
  34.17 on 8 angles in the horn at X −100.
- `grip` (default only) proves the hand box and the finger window box hold no other part, the grip is left of the
  gearbox and the crank right of it, the grip is behind the horn in X, and the grip's bounds.
- `knuckle room` (default only) proves no other part comes within 25 mm of the bar's inner front edge on the X < 79 side.
- `window height` (default only) sections the grip at X 92 and proves the window is at least 100 tall 0.25 mm from the
  bar face.
