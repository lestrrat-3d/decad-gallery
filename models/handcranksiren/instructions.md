# Hand-crank siren

## Purpose

Build a hand-cranked siren. A hand crank on the right turns the output (carrier) shaft of the gallery's planetary
gearbox ([`models/planetarygearbox`](../planetarygearbox/), kit `planet608-shaft608`), which runs backwards as a
one-stage step-up: its input (sun) shaft spins a slotted rotor disc with paddle vanes in front of a fixed slotted stator
disc, inside a drum. The crank and the rotor share one axis. A collector turns the air 90 degrees into a horn that points
forward, and a grip on the housing's left face gives the other hand something to hold. Each time a rotor port passes a
stator port the air flow opens and closes once, so the pitch is `f = Slots × Ratio × n_crank`. With 34 ports, the ratio
of 6 and 2 crank turns per second the rotor turns 12 rev/s (720 rpm) and the pitch is 408 Hz. Over 12 to 40 ports the
pitch runs from 144 Hz to 480 Hz at that crank speed.

The gearbox is `models/planetarygearbox`, the crank is `models/crankhandle`, the bearings are `models/ballbearing`, the
bolts are `models/m2bolt` and the nuts `models/m2nut`, all imported and not copied. Printed parts: everything except
the 7 bearings (5 in the gearbox), 34 bolts and 34 nuts (8 bolts and 11 nuts in the gearbox).

The gearbox keeps its own construction, fits and checks. This contract owns how the siren sets its parameters, places
it, holds it and poses it, and every siren part round it.

## Compliance with AGENTS.md

- Sketch constraints: every sketch the siren and the crank build is fully constrained, with DOF 0, no redundant
  constraint and no conflict, on every parameter branch. The gearbox verifies each of its own sketches before use, so a
  gearbox build that returns proves them.
- Fasteners: every head sinks into a pocket and ends flush. Every new siren nut sits in a hex pocket open to a face or in
  a hex slot open to a post's side, held by its bolt. The ear bolts use the gearbox's own embedded ear nuts. The collector
  and throat joints keep their enclosed cavities, which they had before the embedded-nut rule. See Fasteners.
- Part clearances: every pair of separate parts keeps a gap of its class. See Fits.
- Gear train clearances: the siren has no gear pair of its own. The gearbox's meshes are the planetary gearbox's.
- Surface edge finishing: see Edge finishing.
- Corners and roots: see Retained sharp corners.
- Body containers: see Containers.
- Look-alike part marks: the siren has no look-alike parts, so it has no "Look-alike parts" section and no marks.
- Arbor collars: see Arbor collars.
- Split shafts: see Split shafts.
- `models/README.md` marks the model **Mixed**.

## API and parameters

Package `handcranksiren`.

- `Ratio = 6.0` is the rotor turns per crank turn, `1 + Zr/Zs = 1 + 70/14`. It equals the `Ratio()` of
  `planetarygearbox.NewMotion` for the siren's gearbox parameters, driven `planetarygearbox.CarrierDriven`.
- `Params{CrankAngle float64; Slots int}`. `DefaultParams()` is `{4π/3, 34}`.
- `CrankAngle` is the forward turn, clockwise seen from the operator's right (+Y), about −Y, from the arm pointing
  up. `Validate` requires `CrankAngle` and `Ratio*CrankAngle` finite.
- `Slots` is the number of ports in the rotor and in the stator and of vanes on the rotor. `Validate` requires it in
  `[12, 40]`.
- `PitchHz(slots int, crankRevPerSecond float64) float64` is `float64(slots) * Ratio * crankRevPerSecond`.
  `PitchHz(34, 2) == 408`, `PitchHz(12, 2) == 144`, `PitchHz(40, 2) == 480`.
- `GearboxParams() (planetarygearbox.Params, error)` returns the gearbox the siren builds (see Gearbox). Its error is
  `planetarygearbox.DefaultParamsFor`'s, wrapped; `Build` and `Motion` return it.
- `State{CrankAngle, RotorAngle, GearboxInput float64; Axis, ArmDirection r3.Vec}` and `Motion(Params) (State, error)`.
  `RotorAngle = Ratio·CrankAngle`, `GearboxInput = −CrankAngle`, `Axis = (0, 0, 84)`,
  `ArmDirection = (−sin CrankAngle, 0, cos CrankAngle)`.
- `NewDocument(opts ...decad.DocumentOption)` returns `planetarygearbox.NewDocument(opts...)`, a document with a chain
  tolerance of 0.1 mm. The gearbox's ring needs 0.1 mm, and the rotor (0.01 mm) and the crank pin (0.005 mm) need less,
  so one document holds all three.
- `Build(ctx, doc, Params) (*Assembly, error)` returns `Assembly{Parts, Sketches, State, Params, Gearbox}`. `Part` is
  `{Name, Group, Motion string; Body *decad.Body}`. `Gearbox` is the `planetarygearbox.Gearbox` that `BuildParts`
  returned, with `EarBolts` set to nil after the siren removes them. The build has 103 parts for every `Slots`.
- Exported container names: `HousingContainerName` (`Housing`), `StatorContainerName`, `RotorContainerName`,
  `HornContainerName`, `GripContainerName`, `SpacersContainerName` (`Spacers`), `BearingsContainerName`,
  `CrankContainerName`, `BoltsContainerName`, `NutsContainerName` and `GearboxContainerName` (`Gearbox`).
- Errors read `hand-crank siren <part>: <operation>: <cause>`.

## Frame and orientation

- The held pose fixes what the build axes mean. The table is Z = 0.

| Operator direction | Build axis | Content |
| --- | --- | --- |
| forward | −X | horn mouth at X −159.25; sound leaves along −X |
| right | +Y | crank arm, pin and handle (Y ≥ 38.30) |
| left | −Y | drum, stator, collector, grip (Y ≤ −11.5) |
| up | +Z | table at Z 0; only housing-front and housing-rear reach it |

- The operator holds the grip with the left hand and turns the crank with the right. The listener stands in front,
  at −X. The horn is the frontmost part (smallest X of all parts).
- Seen from +Y, forward (−X) is on the viewer's right, so a forward turn moves the handle forward over the top.
- `A` is the one shaft axis, the line X 0, Z 84 along Y. The gearbox, the rotor, the drum and the crank all turn about
  it. Y is the gearbox's own height above its reference plane, so every gearbox dimension in
  `models/planetarygearbox/instructions.md` reads directly as a siren Y.
- Angles about A are measured in the XZ plane from +X toward +Z and written φ.

## Crank and rotor motion and pitch

- Motions: `fixed` (housing-front, housing-rear, drum, collector, stator, horn, grip, both siren bearings, every siren
  bolt and nut except the crank's), `crank` (crank-arm, crank-pin, crank-handle, crank-spacer and the crank's four bolts
  and four nuts), `rotor` (rotor, rotor-spacer) and `gearbox` (every gearbox body).
- The pose turns each `crank` part about A along −Y by CrankAngle and each `rotor` part about A along −Y by RotorAngle.
  Each `gearbox` body takes `motion.Transform(name, GearboxInput)` of `planetarygearbox.NewMotion` for the siren's
  gearbox parameters, driven `planetarygearbox.CarrierDriven`. That turns the output shaft, the rear
  collar, the carrier, its pins and its planet bearings with the crank, rolls each planet on its pin, and turns the input
  shaft and the front collar with the rotor. The ring, the covers, the cover bearings and the gearbox's bolts and nuts
  stay fixed.
- The planetary's motion turns about its local +Z, which is world +Y, so its input is −CrankAngle. A rotor part's
  transform therefore equals the Input Shaft's, and a crank part's equals the Output Shaft's.
- At the default the rotor angle is 8π, a whole number of turns, so the ports line up (open) for every `Slots`.
- Pitch: each rotor port passing a stator port opens and closes the air flow once, so `PitchHz` gives the tone.

## Dimensions

All lengths are mm and all angles radians unless written in degrees.

### Layout

Axial stack (Y ranges, at crank angle 0). Radii are about A.

| Body or feature | Y range | Radius or extent |
| --- | --- | --- |
| collector | [−106.70, −41.70]; rear wall [−49.70, −41.70] | X [−51, 51], Z [33, 135] |
| horn | about the throat axis (Y `throatY` = −75.70, Z 84); flange X [−59.25, −51.25] | cone to X −159.25 |
| grip | [−85.25, −11.50] | X [79, 105], Z [5, 125] |
| stator | [−41.45, −38.45] | 58 |
| rotor disc | [−37.30, −34.30] | 45 |
| rotor hub and vanes | [−35.30, −26.30] | hub 9; vanes from `rv` to 44 |
| rotor-spacer | [−26.25, −22.75] | 5.75 |
| drum flange | [−38.20, −31.20] | 49 to 58 |
| drum wall | [−38.20, −21.30] | 46 to 49 |
| drum back plate | [−25.30, −21.30] | 9 to 49 |
| drum bearing hub | [−25.30, −15.50]; seat [−22.75, −15.50] | hub 13.55; seat 11.05; lip bore 9 |
| bearing-rotor (608ZZ) | [−22.50, −15.50] | 4 to 11 |
| drum posts | [−21.30, −11.50] | 5, at radius 30 |
| gearbox front collar ring | [−15.25, −11.25] | 5.75 |
| housing-front plate | [−11.25, −1.25] | outline X [−68, 107], Z [0, 152] |
| housing-front boss hole | [−11.25, −5.30] | 14.55 |
| housing-front recess | [−5.30, −1.25] | 44.1 |
| gearbox front cover | boss [−11, −4], frame [−4, −1] | boss 14.05, frame 43.6 |
| gearbox ring and ears | [−1, 13] | 43.6; ears to 52.2 |
| gearbox rear cover | frame [13, 16], boss [16, 23] | frame 43.6, boss 14.05 |
| housing-front posts | [−1.25, 26.00] | 5, at (±52, 32) and (±52, 136) |
| gearbox rear collar ring | [23.25, 27.25] | 5.75 |
| housing-rear plate | [26.25, 34.50]; lip [26.25, 27.25]; seat [27.25, 34.50] | outline X [−68, 68], Z [0, 152] |
| bearing-crank (608ZZ) | [27.50, 34.50] | 4 to 11 |
| crank-spacer | [34.75, 38.25] | 5.75 |
| crank-arm | [38.30, 46.30]; D bore floor at 45.30 | throw 55 |
| crank-handle | [46.70, 96.70] | about the pin axis |
| input shaft | [−35.05, 8] | gearbox |
| output shaft | [9, 45.05] | gearbox |

### Gearbox

- `GearboxParams()` is `planetarygearbox.DefaultParamsFor("planet608-shaft608")` with these overrides:
  - `InputShaftLength` 24.05: the boss face is at −11, and the shaft ends at −35.05, 0.25 short of the rotor bore's
    floor (Tight).
  - `OutputShaftLength` 22.05: the boss face is at 23, and the shaft ends at 45.05, 0.25 short of the crank arm bore's
    floor.
  - `Placement` the frame with origin (0, 0, 84), U (0, 0, 1) and V (1, 0, 0), so local z is world Y, local +X is
    world +Z and local +Y is world +X.
  - Every other field keeps the kit's value: module 1, 14/28/70 teeth, three planets, one stage, `FlatDepth` 0.5,
    608ZZ planet and shaft bearings, three ears with 2.6 mm bores.
- Build it with `planetarygearbox.BuildParts(ctx, doc, GearboxParams())` in the siren's document, before any siren part.
- Local +X is where every flat faces at angle 0, so the shafts' flats and the D bores pressed on them face world +Z at
  crank angle 0, as the crank arm's bore does.
- The three ears lie at local angles 90°, 210° and 330°, which are φ 0°, 240° and 120°. Their bores are at radius 47.9:
  ear 1 (47.9, Y, 84), ear 2 (−23.95, Y, 42.518) and ear 3 (−23.95, Y, 125.482). Each ear runs Y [−1, 13] and holds the
  gearbox's own ear nut in a closed cavity Y [0.8, 2.8], the nut resting at Y [1.0, 2.6].
- The gearbox's ear bolts clamp nothing in the siren. Remove each of `Gearbox.EarBolts` with `doc.Remove` and leave it
  out of the parts. The siren's ear bolts take their place (see Fasteners). The ears' rear-face head pockets (Y
  [11, 13]) stay empty.
- The gearbox's 8 housing bolts (M2 × 6) and 8 nuts, its 3 ear nuts, its 5 bearings and its printed parts are posed and
  filed as Containers says, under their `Gearbox.Parts()` names.
- The ear joint follows the gearbox contract: housing-front lies against the ears' front faces and takes an M2 bolt from
  its own side into each ear's nut. The ears need no change.

### Housing

The housing is two printed parts that stand on the table and hold the gearbox by its ears.

- housing-front: the rounded rectangle X [−68, 107], Z [0, 152], corner radius 13, drawn counter-clockwise, with a
  circle of radius 14.55 about A as its one hole, extruded from Y −11.25 for 10.
  - Cut the recess, a cylinder of radius 44.1 about A over Y [−5.30, −1.25] (with stock out through the +Y face). The
    gearbox front cover sits in it: 0.5 radially from the cover frame and 1.3 axially from its −Y face, and the boss
    sits in the boss hole 0.5 radially.
  - Chamfer the edges (see Edge finishing).
  - Join four posts, each a cylinder of radius 5 along +Y from Y −2.25 (1 inside the plate) to 26.00, at
    (52, 32), (−52, 32), (52, 136) and (−52, 136) in (X, Z).
  - Round each post's root on the plate's +Y face (Y −1.25) by 5, half the post's 10 mm section (a stressed root).
  - Round the recess floor's outer corner (radius 44.1, Y −5.30) by 0.5.
  - Cut the fastener features of the ear bolts, the drum nuts, the grip nuts and the housing nuts (see Fasteners).
- housing-rear: the rounded rectangle X [−68, 68], Z [0, 152], corner radius 13, counter-clockwise, extruded from Y 26.25
  for 8.25.
  - Cut the lip bore, a cylinder of radius 9 about A through the plate.
  - Cut the crank bearing's seat, a cylinder of radius 11.05 about A over Y [27.25, 34.50], open to the +Y face. The
    1 mm under it, Y [26.25, 27.25], is the lip.
  - Chamfer the edges (see Edge finishing), then cut the housing bolts' head pockets and holes.
- The posts end 0.25 short of housing-rear's −Y face. Four M2 × 16 housing bolts from housing-rear's +Y face join them.
- Print housing-front with its −Y face (Y −11.25) down: the recess, the drum and grip nut pockets and the posts open
  upward, and the ear bolts' head pockets open to the bed with 4.4 mm bridges. Print housing-rear with its −Y face down,
  so the seat opens upward. Neither needs a pause or support.

### Bearings and spacers

- Two 608ZZ bearings, `ballbearing.Params608ZZ()` (8 × 22 × 7), the gearbox's bearing: `bearing-rotor` over Y
  [−22.50, −15.50] in the drum's seat and `bearing-crank` over Y [27.50, 34.50] in housing-rear's seat. Each sits 0.25
  off its seat floor and flush with the seat's open face.
- Each siren bearing sits on the shaft's free end, inside the D flat that runs from 1.25 outside the cover boss to the end.
  Its bore touches the 8 mm round of the shaft (no clearance, the bearing-shaft rule); the flat leaves a segment of
  the bore free.
- The gearbox's front collar ring faces bearing-rotor's inner ring across 0.25 (Y −15.25 to −15.50), and its rear collar
  ring faces bearing-crank's inner ring across 0.25 (27.25 to 27.50). Both rings are 11.5 across, inside the 608ZZ's
  12 mm inner race diameter.
- spacer: one printed ring, built once about the origin and placed twice. It is 3.5 long, 11.5 across (radius 5.75,
  inside the 12 mm inner race diameter), with a D bore that is the shaft's D grown 0.05 radially: radius 4.05 with its
  flat 3.55 from the axis. Its wall is 1.7, over the 1.6 press-fit wall. It prints with its axis vertical.
  - rotor-spacer over Y [−26.25, −22.75], between bearing-rotor (0.25) and the rotor hub (0.05).
  - crank-spacer over Y [34.75, 38.25], between bearing-crank (0.25) and the crank arm (0.05).
- Each spacer is pressed on the flat. The carried part sets its place: the rotor and the crank arm are pressed until
  their blind bores' floors meet the shaft's end. That closes the 0.25 floor gap and the spacer's 0.25 gap to its
  bearing's inner ring together, and the inner ring turns with the spacer. The model shows both gaps open.
- The input shaft floats ±0.25 in the gearbox (between the input shaft collar and the front collar ring). At either
  limit the rotor-spacer or the front collar ring touches bearing-rotor's inner ring, which turns with them.

### Arbor collars

[Arbor collars](../../docs/model-pipeline.md#arbor-collars) applies. The gearbox owns the collars inside it (see
`models/planetarygearbox/instructions.md`). Outside it:

| Collar | Shaft | Carried part | Flush side | Y | Len | Dia | Option |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Front collar ring (gearbox) | Input shaft | bearing-rotor's inner ring | Front cover bearing's outer face, Y −11 | [−15.25, −11.25] | 4.0 | 11.5 | 1 |
| rotor-spacer | Input shaft | rotor, which turns with the shaft | bearing-rotor's −Y face, set in the drum's seat, Y −22.50 | [−26.25, −22.75] | 3.5 | 11.5 | 1: the 3.8 gap less 0.25 at the bearing and 0.05 at the rotor |
| Rear collar ring (gearbox) | Output shaft | bearing-crank's inner ring | Rear cover bearing's outer face, Y 23 | [23.25, 27.25] | 4.0 | 11.5 | 1 |
| crank-spacer | Output shaft | crank arm, which turns with the shaft | bearing-crank's +Y face, flush with housing-rear's +Y face, Y 34.50 | [34.75, 38.25] | 3.5 | 11.5 | 1: as the rotor-spacer |

The gearbox's collar rings keep their shape and place; each now faces the next bearing's inner ring with a Tight gap
instead of a pressed part with a press-fit gap.

### Split shafts

[Split shafts](../../docs/model-pipeline.md#split-shafts) applies. The build does not model any split.

| Part | Split plane (normal at crank angle 0) | What holds the halves |
| --- | --- | --- |
| Input Shaft (gearbox) | Holds A and the flat's normal (+Z): normal ±X, turned by RotorAngle | front cover bearing, sun bore, front collar ring, bearing-rotor, rotor-spacer, rotor bore |
| Output Shaft (gearbox) | The same: normal ±X, turned by CrankAngle | carrier hole, rear cover bearing, rear collar ring, bearing-crank, crank-spacer, clamped arm bore |
| crank-pin | Holds the pin axis and the hand's driving force: normal +Z, along the arm, turned to `ArmDirection` | its two M2 × 6 cross bolts (models/crankhandle) |

The gearbox's other shafts and pins keep the gearbox's own split planes and checks. The siren checks the two shafts it
lengthens.

### Drum, rotor, stator, collector and horn

Each is the earlier siren's part moved 9.80 toward −Y, except the drum's centre and the rotor's vanes.

- drum: a revolved meridian about A (U the radius, V = Y), counter-clockwise from the seat's rim:
  (11.55, −15.50), (11.05, −16.00), (11.05, −22.75), (9, −22.75), (9, −24.80), (9.50, −25.30), (46, −25.30),
  (46, −37.70), (46.50, −38.20), (57.50, −38.20), (58, −37.70), (58, −31.70), (57.50, −31.20), (49, −31.20),
  (49, −21.80), (48.50, −21.30), (13.55, −21.30), (13.55, −15.50). The 0.5 segments are the seat chamfer, the lip bore's
  rim chamfer and the outer chamfers.
  - Round the hub's root, the circle of radius 13.55 at Y −21.30, by 1.25 (half the 2.5 hub wall), with
    `decad.Fillet`.
  - Cut three windows through the back plate: `port(21, 31, 40°, round 1)` centred at φ 60°, 180° and 300°. Each
    window's inner chord lies 21·cos 40° = 16.09 from A, 1.29 outside the hub's round.
  - Chamfer the windows (see Edge finishing).
  - Join three posts of radius 5 at radius 30 from A, Y [−21.30, −11.50], at φ 0°, 120° and 240°.
  - Cut the drum bolts' head pockets and holes and the collector bolts' head pockets and holes.
- rotor: disc radius 45 over [−37.30, −34.30], both cap outer loops chamfered 0.5; vanes u [`rv`, 44], v [−1, 1],
  Y [−35.30, −26.30], one per slot, centred on the lands; hub radius 9 from −35.30 for 9; ports
  `port(30, 42, π/(2N), round 1)` cut through the disc; D bore from the hub's +Y face (−26.30) toward −Y, 9 deep,
  radius 4.05 with its flat 3.55 from A toward +Z, leaving a floor 2 thick.
  - `rv = max(11, 0.51·Slots)`: 11 up to 21 slots, 17.34 at 34 and 20.40 at 40. It keeps at least 1.2 between the
    inner corners of neighbouring vanes (1.208 at 34, 1.207 at 40, 1.214 at 22). The vanes lose 9.9 % of their throw
    at 34 slots, which scales with `44² − rv²`.
- stator: disc radius 58 over [−41.45, −38.45]; the same ports; a hole for each `collector-bolt-*`.
- collector: the earlier collector moved 9.80 toward −Y. Constants: rear face −41.70, ceiling −49.70, floor −101.70,
  end −106.70, outer half-width 51 (corner radius 10), cavity half-width 46 (corner radius 7), bore 44 across,
  `throatY` −75.70, throat radius 22, −X wall faces at X −51 and −46. The outer prism, the bore (radius 44 cylinder,
  Y [−50.70, −40.70]), the cavity (X [−46, 46], Z [38, 130], Y [−101.70, −49.70]), the chamfers, the throat (radius 22,
  X [−52, −45], axis along −X through (−, −75.70, 84)), then the hole and cavity for each `collector-bolt-*` and the
  pocket and hole for each `throat-bolt-*`.
- horn: the earlier horn's meridian about O = (−50, `throatY`, 84) along −X, unchanged in its own frame: half angle 15°,
  length 100, wall 2.5, flange X [−51.25, −59.25], cone to X −159.25, vertices A to L as before, then the hole and
  cavity for each `throat-bolt-*`.
- Port shape: chord-sided with a 1 mm round at each of the four corners, rounded with `sketch.CreateFillet`, the cut
  tool patterned `Slots` times. At 40 slots the inner chord is 2.36 long and the two rounds take 1.92 of it.
- Air path: air enters through the 10.05 gap between the drum's back plate and housing-front, passes the three
  windows, and the vanes throw it outward to the port ring. It leaves through the ports whenever a rotor port lines up
  with a stator port, then crosses the collector and leaves through the throat and the horn, as before.

### Crank

The crank is [`models/crankhandle`](../crankhandle/)'s `Params6082RS()` with `FlatDepth` set to the gearbox's `FlatDepth`
(0.5), so its D bore fits the gearbox's output shaft: radius 4.05, flat 3.55 from the axis. It is built by
`crankhandle.BuildParts` with a placement that maps its origin to (0, 38.30, 84), its +Z to +Y and its +Y to +Z (U
(−1, 0, 0), V (0, 0, 1)). The siren files its three parts in its own `Crank` container and places the crank's four
bolts and four nuts at the sites crankhandle cut them for. `models/crankhandle/instructions.md` owns every crank
dimension; at crank angle 0 they read:

- crank-arm: 20 wide and 8 thick from Y 38.30, the pin end at Z 139, the end arc behind A. Its blind D bore runs from
  the back face (38.30) to its floor at 45.30. The M2 × 16 clamp bolt crosses the hub along +X at Z 77.6, mid thickness
  (Y 42.30): head pocket in the −X side face (X −10), nut in a hexagonal pocket open to the +X side face (X 10).
- crank-pin: about (0, Y, 139), the stub from Y 41.80, the shank to Y 91.05 and the head over [93.30, 96.30]; cross
  bolts at Y 55.30 and 87.30.
- crank-handle: revolved about the pin axis, Y [46.70, 96.70].
- With the 0.5 flat the arm's D bore corners turn acos(3.55/4.05) = 28.8°, under the checker's limit.

### Grip

One body, a vertical D-handle on housing-front's −Y face, toward the operator. Its ring profile lies in the YZ plane on
the frame origin (79, 0, 0), U = +Y, V = +Z, and is extruded along +X from X 79 for 26. The profile is the outer outline
Y [−85.25, −11.50] × Z [5, 125] with corner radius 10 and the finger window Y [−55.25, −23.25] × Z [12, 118] with corner
radius 3 as a hole, both counter-clockwise in (u, v). Then `edgefinish.Chamfer(+X, {79, 105}, Long(All), 2.0)` on both
caps, then the pocket and hole for each `grip-bolt-*`. The ring is a pad (Y [−23.25, −11.50], 11.75 thick, its back
face 0.25 off housing-front), a bar (Y [−85.25, −55.25], 26 × 30) and two arms (Z [5, 12] and [118, 125]). The window
is 32 deep and 106 tall.

- The bar's perimeter after the chamfers is 107.3, a 34.2 mm round bar. The left hand holds the bar thumb up, palm on
  the outer face (Y −85.25), fingers round the front face (X 79).
- The hand box X [57, 140], Y [−119.25, −55.25], Z [12, 118] holds no other part. The finger window box
  X [79, 105], Y [−55.15, −23.35], Z [12, 118] holds no part. No other part comes within 25 mm of the bar's inner front
  edge (X 79, Y −55.25) on the X < 79 side, at any Z across the window: the nearest is the stator's rim at
  (58, −41.45, 84), 25.13 away. The window is at least 100 tall at the bar face.
- The grip is at Y ≤ −11.50 and the crank at Y ≥ 34.75, so the housing sits between the hands. The grip is at X ≥ 79
  and the horn at X ≤ −51.25. Its lowest point is Z 5.

### Fasteners

34 bolts and 34 nuts. The siren places 26 bolts (12 M2 × 16, 3 M2 × 18, 5 M2 × 10, 4 M2 × 20, 2 M2 × 6) and 23 nuts;
the gearbox brings 8 M2 × 6 bolts, 8 nuts and the 3 ear nuts. The pocket and cavity sizes come from
`internal/m2fastener`: head pocket radius 2.2 and depth 2.0, hole radius 1.3 (2.6 across), round nut cavity of radius
2.51 and 2.0 high, hex pocket with its corners on radius 2.51 (4.35 across its flats) and 2.0 deep, tool stock 1.0.

| Bolts | Nuts | Len | Head top | Dir | Corner | Cavity | Head host | Nut host | Nut seat |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `ear-bolt-1..3` | gearbox `Ear Nut 1..3` | 16 | (47.9 cos φ, −11.25, 84 + 47.9 sin φ), φ = 0°, 240°, 120° | +Y | (cos φ, 0, sin φ) | 12.05 | housing-front | Ring Gear | the gearbox's closed cavity, Y [0.8, 2.8] |
| `housing-bolt-1..4` | `housing-nut-1..4` | 16 | (52, 34.5, 32), (−52, 34.5, 32), (52, 34.5, 136), (−52, 34.5, 136) | −Y | (sx, 0, sz)/√2 | 12.0 | housing-rear | housing-front (post) | hex slot open to the post's outer side |
| `drum-bolt-1..3` | `drum-nut-1..3` | 18 | A + 30(cos φ, 0, sin φ) at Y −25.30, φ = 0°, 120°, 240° | +Y | (cos φ, 0, sin φ) | 18.0 | drum | housing-front | hex pocket open to the recess floor (Y −5.30) |
| `collector-bolt-1..4` | `collector-nut-1..4` | 16 | A + 53.5(cos ψ, 0, sin ψ) at Y −31.20, ψ = 45° + 90°(k−1) | −Y | (−sin ψ, 0, cos ψ) | 12.0 | drum | collector | closed cavity (unchanged joint) |
| `throat-bolt-1..4` | `throat-nut-1..4` | 10 | (−46, `throatY` + 26 cos a, 84 + 26 sin a), a = 45° + 90°(k−1) | −X | (0, cos a, sin a) | 6.75 | collector | horn | closed cavity (unchanged joint) |
| `grip-bolt-1..4` | `grip-nut-1..4` | 20 | (85, −23.25, 30), (99, −23.25, 30), (85, −23.25, 100), (99, −23.25, 100) | +Y | X | 20.0 | grip | housing-front | hex pocket open to the +Y face (Y −1.25) |
| `crank-pin-bolt` | `crank-pin-nut` | 10 | (0, 38.30, 139) | +Y | −X | 4.75 | crank-arm | crank-pin | crankhandle |
| `crank-cross-bolt-1..2` | `crank-cross-nut-1..2` | 6 | (0, 55.30, 143), (0, 87.30, 143) | −Z | −X | 6.0 | crank-pin | crank-pin | crankhandle |
| `crank-clamp-bolt` | `crank-clamp-nut` | 16 | (−10, 42.30, 77.6) | +X | +Y | 16.0 | crank-arm | crank-arm | crankhandle |

`sx` and `sz` are the signs of the post's X and of its Z − 84, so each slot opens away from A. The crank's sites are
crankhandle's (`Dimensions.Site`, `CrossSites`, `ClampSite`) carried through the placement; the table gives them at crank
angle 0, and the pose turns them with the crank. crankhandle cuts their pockets, holes and cavities.

Derived, along each bolt from its head's top face:

- Ear bolts: the head pocket is Y [−11.25, −9.25] in housing-front, the hole runs through the plate, and the bolt crosses
  the 0.25 seam into the ear, passes the whole ear nut (Y [1.0, 2.6]) and ends at Y 6.75, inside the ear's bore and
  4.25 short of the empty head pocket's floor (Y 11). The nut bears on the ear's 1.8 roof, the joint the gearbox
  contract describes.
- Housing bolts: the head pocket is Y [32.5, 34.5] in housing-rear. The bolt crosses housing-rear and the 0.25 seam into
  the post and ends at Y 16.5. The hex slot spans Y [20.5, 22.5]: a hex pocket (`HexPocketTool`, depth 2.0) joined to a
  box 4.35 wide (the pocket's flats) running from the bolt's line along Corner out through the post's side. The nut
  slides in flat-first, the slot's walls stop it turning, and the bolt runs through it. The post keeps 2.49 round the
  hexagon's far corner and 3.5 between the slot and its end.
- Drum bolts: the head pocket is Y [−25.30, −23.30] in the drum back plate's inner face. The bolt crosses the back
  plate, the post and the 0.25 seam, and ends flush with the recess floor at Y −5.30. The hex pocket is Y
  [−7.30, −5.30], 3.95 above housing-front's −Y face. Each pocket lies under a window of the gearbox's front cover (its
  spokes are at φ 60°, 180° and 300°), so the nut can be set or held through the window.
- Grip bolts: the head pocket is Y [−23.25, −21.25] in the pad's window face, with 9.75 of pad under it. The bolt ends
  flush with housing-front's +Y face at Y −1.25, and the hex pocket is Y [−3.25, −1.25].
- Collector and throat bolts: the earlier geometry moved 9.80 toward −Y. Collector cavities at Y [−43.20, −45.20] in
  the collector's rear wall, tips at −49.20. Throat pockets in the collector's −X wall from the cavity side, nut
  cavities in the horn flange, X [−52.75, −54.75], tips at X −58.
- Hex pockets and slots: a bolt's tip ends at the pocket's open face or inside the post, never outside a face. Every
  pocket keeps 1 mm or more of material round it.

#### Print pause

- `collector` prints with its end face (Y −106.70) down; its four collector nuts sit near the top, in one pause.
- `horn` prints with its flange mating face (X −51.25) down; its four throat nuts sit near the bed, in one pause.
- The gearbox ring's housing and ear nuts drop in as `models/planetarygearbox` says.
- `housing-front`, `housing-rear`, `drum`, `grip`, `rotor`, `stator`, the spacers and the crank parts need no pause.
  Their nuts go in at assembly.
- `crank-pin` prints as two halves, each on its cut face; `crank-arm` with its back face down, axis vertical;
  `crank-handle` standing on an end; `rotor` with its disc's front face down; `stator` with either face down; each
  spacer standing on an end, axis vertical.
- `drum` prints standing on its flange's front face; its back plate needs slicer support. The seat opens upward.

#### Assembly order

1. Set the three drum nuts in housing-front's recess-floor pockets, with the plate on its −Y face.
2. Seat the gearbox's front cover in the recess, ears on the plate's +Y face, and drive the three ear bolts from the
   −Y face.
3. Set the four housing nuts in the posts' slots. Press bearing-crank into housing-rear, slide housing-rear over the
   output shaft onto the posts and drive the four housing bolts.
4. Press the crank-spacer and the crank arm on the output shaft and fit the crank as `models/crankhandle` says.
5. Press bearing-rotor into the drum's seat, slide the drum over the input shaft onto housing-front and drive the
   three drum bolts from inside the drum.
6. Press the rotor-spacer and the rotor on the input shaft.
7. Fit the stator, the collector and the horn as before: the throat bolts first, from inside the collector.
8. Set the four grip nuts in housing-front's +Y face pockets and drive the grip bolts through the finger window.

### Fits

The nominal gap is the class minimum plus 0.05. A bearing shaft equals the bearing bore (no gap), and a press fit is a
0.05 radial clearance whose proven bound must reach 0.03. Fits between round parts use a section tolerance of 0.01. The
crank arm is sectioned at 0.0102 and its partner at 0.01. Fits that include a printed plate use 0.02.

| ID | Parts | Class (min) | Nominal | How measured |
| --- | --- | --- | --- | --- |
| T1 | housing-front posts / housing-rear | Tight 0.2 | 0.25 | axial gap (Y) |
| T2 | drum posts / housing-front | Tight | 0.25 | axial bounds |
| T3 | stator / drum, collector / stator | Tight | 0.25 | axial bounds (Y) |
| T11 | horn / collector | Tight | 0.25 | axial bounds (X) |
| T12 | grip / housing-front | Tight | 0.25 | axial bounds (Y) |
| T13 | gearbox ears / housing-front | Tight | 0.25 | axial gap from each ear's front face |
| T4 | input shaft / bearing-rotor, output shaft / bearing-crank | none: shaft equals the bore | 0 | sections touch at the bearing's mid-plane, on the round side |
| T5 | rotor D bore, crank-arm D bore, each spacer's D bore / its shaft | Press fit | 0.05 | section at the bore's mid-plane |
| T6 | input shaft end / rotor bore floor, output shaft end / arm bore floor | Tight | 0.25 | axial section |
| T7 | each spacer / its bearing's inner ring; each gearbox collar ring / the next bearing's inner ring | Tight | 0.25 | Y gap from the face found in the axial section through A |
| T8 | crank-pin stub / crank-arm pocket | Tight | 0.25 | section in the stub |
| T9 | rotor-spacer / rotor hub, crank-spacer / crank arm | Press fit | 0.05 | Y gap |
| S1 | each siren bearing / its seat | bearing-seat rule, proven ≥ 0.02 | 0.05 | section at mid-plane |
| S2 | each siren bearing / its seat floor | Tight | 0.25 | Y gap |
| R1 | rotor-spacer / the drum's lip bore; rear collar ring / housing-rear's lip bore | Loose | 3.25 | section |
| R2 | rotor / stator | Regular plus clamp budget, proven ≥ 1.10 | 1.15 | axial bounds |
| R3 | crank-handle / crank-pin | Regular | 0.4 | sections; axial plane |
| R4 | crank-arm / crank-handle | Regular | 0.4 | axial bounds |
| L1 | rotor / drum | Loose 0.5 | 1.0 | vanes to the back plate's inner face; disc to the wall |
| L5 | crank-arm / housing-rear | Loose | 3.8 | axial bounds |
| L6 | collector / drum; horn / stator; horn / drum | Loose | 3.5 / 4.25 / ≥ 4.25 | as before |
| L7 | grip / collector, stator, drum | Loose | 28 / 21 / 21 | axial bounds (X) |
| L8 | gearbox front cover / housing-front recess and boss hole | Loose | 0.5 radial / 1.3 axial | sections |
| L9 | gearbox rear cover boss / housing-rear | Loose | 3.25 | axial bounds |
| L10 | drum hub / gearbox front cover boss; drum posts / front cover | Loose | 4.5 / ≥ 10 | axial bounds; sections |

Clamping may close T2 and T3 by 0.25 each, which carries the stator 0.5 toward the rotor; the input shaft's ±0.25
float takes another 0.25, so R2's worst case is 0.40. The drum moves away from the vanes when T2 closes, so L1's worst
case is 0.75, from the float.

Nominal exceptions: a bolt shank in its own nut; a bolt head on its pocket floor (proved with `doc.ContactPair`, gap at
most 0.05); each siren bearing's bore on the round of the D-flat shaft.

### Retained sharp corners

The rule is "Corners and roots" in `docs/model-pipeline.md`. Each siren part is read in its print orientation (Print
pause): the axis is Y for housing-front, housing-rear, the drum, the rotor, the stator, the collector and the spacers, X
for the horn and the grip, and the split normal for the crank pin. Bolts, nuts and bearings are purchased parts. The
gearbox's parts follow `models/planetarygearbox/instructions.md`. The crank's rows match
`models/crankhandle/instructions.md`.

| Part | Corner | Reason |
| --- | --- | --- |
| housing-front | the floor rims of the three ear bolts' head pockets | Fasteners rule: the head's flat underside needs the whole floor. |
| housing-front | the six corners of each drum and grip nut's hex pocket, and the three of each housing nut's slot | Mating corners: they match the M2 nut's sharp corners. |
| housing-rear | the bearing seat's floor corner (radius 11.05, Y 27.25) | Mating corner: the bearing's outer ring sits 0.25 off the floor with a 0.15 to 0.3 corner chamfer, which leaves no room for a 0.5 round. |
| housing-rear | the floor rims of the four housing bolts' head pockets | Fasteners rule. |
| drum | the bearing seat's floor corner (radius 11.05, Y −22.75) | As housing-rear's. |
| drum | the posts' roots on the back plate's +Y face; the wall's root on the back plate's inner face (r 46, Y −25.30); the flange step (r 49, Y −31.20) | Between layers. The posts are clamped by the drum bolts and carry little load. |
| drum | the floor rims of the drum bolts' head pockets | Fasteners rule. |
| rotor, crank-arm, rotor-spacer, crank-spacer | the two corners of each D bore | Mating corners with the shaft's flat. They turn 28.8°, so the checker does not flag them. |
| rotor | the roots of the vanes and the hub on the disc's back face | Between layers, opening upward when the disc prints face down; air drag only. |
| crank-arm | the stub pocket's six corners; the clamp head pocket's floor rim; the slanted cut of the clamp nut pocket's 0.6 floor round; the 1.05 web at the slit; the clamp nut pocket's six corners | As in `models/crankhandle`. |
| crank-pin | the six corners of each cross nut's hexagonal pocket | As in `models/crankhandle`. |

Rounds below half their section:

- The port corners: 1.0 on lands 2.36 (40 slots) to 7.8 (12 slots) wide at radius 30, `2·30·sin(π/(2N))`. The rotor's
  centrifugal stress at 720 rpm is under 0.01 MPa and the stator is clamped, so the lands carry almost no load. The
  drum window corners: 1.0 on lands over 14 wide; the posts stand at r 30 between the windows and carry the load.

The press-fit walls: the rotor hub keeps 4.95 round its bore, each spacer 1.7 and the crank arm 1.6 (1.05 at the slit,
as crankhandle lists), each at least the 1.6 of the press-fit clause.

### Edge finishing

| Body | Large planar faces | Finishing | Retained sharp |
| --- | --- | --- | --- |
| housing-front | −Y face, +Y face, recess floor | `edgefinish` Long(All) 0.5 on Y −11.25, −1.25 and −5.30 (outer loop, boss hole rim, recess rim) | pockets, holes and slots (loops < 20 mm); post roots and recess corner (concave) |
| housing-rear | both faces | `edgefinish` Long(All) 0.5 on Y 26.25 (outer loop, lip bore rim) and 34.50 (outer loop, seat rim) | head pockets and holes (loops < 20 mm) |
| drum | back plate faces, flange faces | meridian chamfers (outer corners, seat rim, lip bore rim); `edgefinish` Long(Inner) on the windows on Y −21.30 and −25.30 | concave junctions (the posts, the wall, the hub's round) |
| stator, rotor, collector, horn, grip | as before | as before | as before |
| crank-arm | both faces; the bore's rim in the inner face | cap outer loops 0.5; the bore rim's two arcs and flat 0.5 (`crankhandle`'s `chamferBoreRim`) | pin bolt hole; hex pocket opening (Tight fit); the slit's walls and end round in both faces (the 1 mm clamp gap, as `models/crankhandle` lists) |
| rotor-spacer, crank-spacer | none: each face is about 55 mm² | none | not qualifying |

The grip's 2.0 chamfer is the one deviation from the 0.5 minimum of "Surface edge finishing" (A7), as before.

### Containers

`Housing` (housing-front, housing-rear, drum, collector), `Stator`, `Rotor`, `Horn`, `Grip`, `Spacers` (rotor-spacer,
crank-spacer), `Bearings` (bearing-rotor, bearing-crank), `Crank` (crank-arm, crank-pin, crank-handle), `Bolts` (26),
`Nuts` (23), and `Gearbox`, whose children are the containers `Gearbox.Parts()` names (`Housing`, `Cover Bearings`,
`Sun Gears`, `Planet Gears`, `Planet Bearings`, `Carriers`, `Carrier Pins`, `Shafts`, `Shaft Collars`, `Bolts`, `Nuts`).
A gearbox part's `Group` is `Gearbox/<its container>`. A container exists only when it holds a body.
`doc.VerifyContainers().Err()` is nil.

## Pose

Build every part at crank angle 0, pose it by its motion, move it into its container last.

## Verification

`go test ./models/handcranksiren` covers the default and the `Slots` 12 (CrankAngle 0) and 40 (CrankAngle 4π/3)
variants. The tests, one per rule, are named in [steps.md](steps.md) ("Checks").
