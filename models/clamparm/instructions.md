# Clamp arm

## Purpose and source

Construct a printed desk-edge clamp that carries a four-joint friction arm and ends in a documented, replaceable
tool mount interface with one placeholder adapter. The interface at the free end is the deliverable. Real holders (a
webcam holder first) are later items that build against it ("Mount interface" below).

The parts are:

- a C-shaped clamp frame that grips a desk 10 to 40 mm thick, an M5 x 70 clamp screw turned by a screw knob below the
  bottom jaw, and a pad on the screw's tip;
- a swivel, link A, link B and a head, joined by four friction knuckles. J1 pans about a vertical axis (the swivel on
  the frame), J2 is the shoulder, J3 the elbow, J4 the wrist tilt. Each knuckle is a fork, a tongue, an M5 bolt and a
  nut held in a lobed knob;
- a mount interface on the head's top face: a 45 degree cone boss and an M5 stud that rises from below, turned by a
  screw knob under the head;
- a placeholder pad adapter that fits the interface.

Hardware is M5 throughout: one size for the joints, the clamp screw and the stud. The bolts and nuts are
[`models/m5bolt`](../m5bolt/) and [`models/m5nut`](../m5nut/), placed with `internal/m5fastener`. Reference them; do
not copy their geometry. The desk is not modeled.

All dimensions are in millimeters and angles in degrees.

## Symbols

| Symbol | Meaning | Value |
| --- | --- | --- |
| `o` | desk thickness the pad is posed at | `Opening` |
| `L` | axis-to-axis length of each link | `LinkLength` |
| `g` | tongue-in-fork clearance per side | `JointFit` = 0.26 |
| `r_m` | mean friction radius of one knuckle face | `2/3 (10^3 - 2.8^3) / (10^2 - 2.8^2)` = 7.08 |
| `T_k` | knob torque by hand | 0.6 N.m nominal |
| `F` | bolt preload | `T_k / (0.2 d)`, `d` = 5 mm |
| `F_fin` | preload spent closing both fins by `g` | about 49 N |
| `T_hold` | holding torque of one knuckle | `2 mu (F - F_fin) r_m` |
| `R_v` | radius of the knob valleys' centres | 13.857 |
| `h_n` | height of an adapter nut's near face over the mating face | `AdapterNutMin` to `AdapterNutMax` |

## API

Package: `models/clamparm`. Error prefix `clamparm: `. Exported surface:

- Joint constants: `Section` (20), `EndRadius` (10), `TongueThickness` (8), `JointFit` (0.26), `ForkGap` (8.52),
  `FinThickness` (5.74), `TongueShoulder` (20), `ShoulderRound` (4), `SlotDepth` (26), `SlotRound` (4.26).
- Range constants: `J1Max` (125), `JointMax` (115), `ShoulderMin` (-60), `WristMin` (-110), `WristMax` (90),
  `OpeningMin` (10), `OpeningMax` (40), `LinkLengthMin` (80), `LinkLengthMax` (150).
- Layout constants: `FrameWidth` (30), `PanAxisX` (30), `ScrewX` (25), `SwivelLength` (70), `StudOffset` (40),
  `HeadLength` (55), `KnobHeight` (12), `ScrewKnobHeight` (10), `PocketRound` (0.6), `MarkRadius` (8).
- Interface constants: `BossBaseRadius` (6), `BossHeight` (2), `RecessRadius` (6.495), `RecessClearance` (0.35),
  `StudProtrusion` (13.5), `AdapterNutMin` (4.7), `AdapterNutMax` (8.3), `AdapterStudClear` (14.5).
- Container names, in this order: `FrameContainerName` (`Frame`), `PadContainerName` (`Pad`), `SwivelContainerName`
  (`Swivel`), `LinksContainerName` (`Links`), `HeadContainerName` (`Head`), `AdapterContainerName` (`Adapter`),
  `KnobsContainerName` (`Knobs`), `ScrewKnobsContainerName` (`Screw Knobs`), `BoltsContainerName` (`Bolts`) and
  `NutsContainerName` (`Nuts`).
- `Params`, `DefaultParams()`, `Params.Validate() error`, `Params.Derive() (Dimensions, error)` and `Dimensions`.
  `Dimensions` calls no geometry. It holds:
  - the five pose frames `Swivel`, `LinkA`, `LinkB`, `Head`, `Interface` (`r3.Frame`, in the world);
  - `Joints [4]r3.Vec` (J1 on the swivel's bottom face at `(30, 0, 14.26)` at `Pan`; J2 to J4 on each axis in the
    part's mid-plane) and `Axes [4]r3.Vec` (unit, world; the knob side for J2 to J4, +Z for J1);
  - every bolt `m5fastener.Site`: `PanSite`, `JointSites [3]` (J2 to J4), `StudSite`, `ScrewSite`;
  - the knob frames `KnobFrames [4]` and `ScrewKnobFrames [2]`, where each knob frame's U equals its site's `Corner`,
    so the press pocket lines up with the head's hexagon;
  - the nut frames `KnobNutFrames [4]`, `JawNutFrame`, `AdapterNutFrame`, the `PadFrame r3.Frame`, and
    `ScrewKnobGap float64` (the screw knob's gap to the bottom jaw at this `Opening`).
- `NewDocument(opts ...decad.DocumentOption) *decad.Document` returns `decad.New(WithChainTolerance(0.01 mm), opts...)`.
- `Part` has `Body *decad.Body`, `Frame r3.Frame` (local to world), `PrintUp r3.Vec` (world) and
  `Sketches []*sketch.Sketch`.
- `BuildParts(ctx, doc, p) (*Parts, error)` adds the 13 printed bodies, posed, in no container. `Parts` holds `Clamp`
  (the frame part), `Pad`, `Swivel`, `Links [2]`, `Head`, `Adapter`, `Knobs [4]`, `ScrewKnobs [2]` (each a `Part`) and
  `Dimensions`.
- `Build(ctx, doc, p) (*Model, error)` adds the parts and the hardware, each in its container. `Model` embeds `Parts`
  and holds `Bolts [6]*m5bolt.Model` (order J1, J2, J3, J4, stud, screw) and `Nuts [6]*m5nut.Model` (order J1 to J4
  knobs, jaw, adapter).
- `InterfaceRecessTool(ctx, doc, f r3.Frame) (*decad.Body, *sketch.Sketch, error)` ("Mount interface" below).

## Params and defaults

| Field | Default | Valid range | Meaning |
| --- | --- | --- | --- |
| `LinkLength` | 120 | 80 to 150 | axis-to-axis length of each link |
| `Opening` | 25 | 10 to 40 | desk thickness the pad is posed at |
| `Pan` | 0 | -`J1Max` to `J1Max` | J1, swivel about +Z |
| `Shoulder` | 70 | `ShoulderMin` to `JointMax` | J2, link A from the swivel |
| `Elbow` | -90 | -`JointMax` to `JointMax` | J3, link B from link A |
| `Wrist` | 20 | `WristMin` to `WristMax` | J4, head from link B |
| `Placement` | nil | nil or a valid frame | maps the world frame; nil selects the local frame |

`J1Max` is 125, `JointMax` 115, `ShoulderMin` -60, `WristMin` -110 and `WristMax` 90. `ShoulderMin` is separate
because link B swings into the clamp below about -75. The default pose puts the head level: 70 - 90 + 20 = 0.
`DefaultParams()` returns the defaults above.

## Validation

`Derive` checks in this order and returns the first failure:

1. Every float field is finite: `<Field> must be finite`.
2. `LinkLength` is in 80 to 150.
3. `Opening` is in 10 to 40.
4. Each angle is in its range. The message names the field and the range.
5. `Placement` is nil or a valid frame.

`Validate()` returns `Derive`'s error. `Build` and `BuildParts` reject a nil context and a nil document, derive the
dimensions, and check `ctx.Err()` before any geometry. A rejected build adds no body.

## World frame and pose

- The origin is at the desk edge: x = 0 on the spine's inner face, z = 0 on the top jaw's underside (the desk top),
  y = 0 on the frame's mid-plane. +X runs over the desk and +Z is up.
- A joint angle `theta` is a rotation about the joint's -Y axis in its local frame, so +`theta` lifts local +X toward
  +Z.
- The pose chain maps local frames to world, with `Params.Placement` applied last:

| Frame | Definition |
| --- | --- |
| Frame | identity |
| Swivel `S` | `T(30, 0, 14.26) . Rz(Pan)` |
| Link A `A` | `S . T(70, 0, 10) . R(-Y, Shoulder)` |
| Link B `B` | `A . T(L, 0, 0) . R(-Y, Elbow)` |
| Head `H` | `B . T(L, 0, 0) . R(-Y, Wrist)` |
| Interface `I` | `H . T(StudOffset = 40, 0, 10)`; the adapter's local frame is `I` |

Each printed part is built in its local frame and placed once, after its last operation, with
`internal/placement.Place`.

## Shared joint geometry

Every knuckle is the same fork and tongue, except J1 whose tongue is longer and bottom-flush ("Swivel" below).

| Name | Value | Note |
| --- | --- | --- |
| `Section` | 20 | square section of swivel, links and head (Y by Z) |
| `EndRadius` | 10 | round end about each joint axis |
| `TongueThickness` | 8 | |
| `JointFit` | 0.26 | per side, Tight class |
| `ForkGap` | 8.52 | `TongueThickness + 2 JointFit`; equals the hex head pocket's across-flats |
| `FinThickness` | 5.74 | `(Section - ForkGap) / 2` |
| `TongueShoulder` | 20 | joint axis to the tongue's shoulder face |
| `ShoulderRound` | 4 | shoulder inside corners; half the tongue (stressed root) |
| `SlotDepth` | 26 | joint axis to the fork slot's deepest point |
| `SlotRound` | 4.26 | `ForkGap / 2`, full round at the slot end |

- **Tongue.** The local frame has the joint axis along Y through the origin and the part running along +X. Keep
  `|y| <= 4` for `x < 20`. The two inside corners at `(20, +-4)` take rounds of radius 4 (centres `(16, +-8)`), with
  their edges along Z. Cut a hole of diameter 5.6 along Y.
- **Fork.** The joint axis runs along Y at `x = Lf`. Cut a slot `|y| < 4.26` from beyond the end back to
  `x = Lf - 26`, ended by a full round of radius 4.26 centred at `x = Lf - 21.74`, cut through Z. Cut a hole of
  diameter 5.6 along Y. Cut the hex head pocket in the -Y fin with `m5fastener.HeadPocketTool`, Corner +Z. The knob
  seats on the +Y fin face.
- **Slot depth proof.** The tongue's material inside the slot reaches `sqrt(20^2 + 10^2)` = 22.36 from the axis. The
  slot's round end at `|y| = 4` lies `21.74 + sqrt(4.26^2 - 4^2)` = 21.74 + 1.47 = 23.21 away, a margin of 0.85.
- **Fold limit proof.** The tongue part's material in the fins' band `|y|` in 4.26 to 10 starts at
  `x = 16 + sqrt(16 - 3.74^2)` = 17.42. A Loose clearance of 0.5 to the fin band needs
  `17.42 sin(beta) - 10 cos(beta) >= 10.5`, which holds for `beta >= 61.5`, the angle between the two parts'
  directions. `JointMax` = 115 keeps the margin.

## Joint capacity

Each knuckle holds by friction on two faces, the tongue's two faces against the two fins. `T_hold = 2 mu (F - F_fin)
r_m`, with `r_m` = 7.08 (annulus radii 10 and 2.8) and `F = T_k / (0.2 * 5)`. `F_fin` is the preload spent closing
both fins by 0.26: each fin is a cantilever 26 long, 20 wide and 5.74 thick, E = 3500 MPa, and needs about 49 N for
0.26.

| Inputs | `mu` | `T_k` (N.m) | `F` (N) | `T_hold` (N.m) |
| --- | --- | --- | --- | --- |
| Low | 0.2 | 0.4 | 400 | 1.00 |
| Nominal | 0.3 | 0.6 | 600 | 2.34 |
| High | 0.4 | 0.8 | 800 | 4.25 |

The gravity moment at J2, the worst joint, with the arm horizontal, a 200 g camera and a 30 g adapter at the stud and
the parts at their centres, is about 0.89 N.m at `LinkLength` 120 and 1.09 N.m at 150. A centre of mass 10 mm off the
stud axis adds 0.02 N.m. The nominal margin is 2.6 at 120 and 2.1 at 150, at least the required 2.

Bench test: lock the clamp, straighten the arm horizontally, hang masses at the stud and record the slip mass at each
joint. `TestClampArmCapacity` pins the geometric inputs: `r_m`, the face count and the fin size.

## Clamp frame

One XZ profile extruded along Y from -15 to 15 (`FrameWidth` 30). Print it on its -Y face. Material regions:

| Region | x | z |
| --- | --- | --- |
| Spine | -14 to 0 | -73.5 to 28.26 |
| Top jaw (J1 lower fin) | 0 to 50 | 0 to 14 |
| Upper fin (J1) | 0 to 46 | 22.52 to 28.26 |
| Bottom jaw | 0 to 45 | -73.5 to -59.5 |

- **J1 gap.** The gap between the top jaw and the upper fin runs z 14 to 22.52 (8.52), is open at x = 46, and ends in
  a full round of radius 4.26 whose deepest point is x = -5 (centre `(-0.74, 18.26)`). Proof: the swivel's material
  in the gap reaches `sqrt(30^2 + 10^2)` = 31.6 from J1 at most; the round at the tongue's faces lies
  `30 + 0.74 + sqrt(4.26^2 - 4^2)` = 32.2 away.
- **Concave corners.** `(0, 0)` and `(0, -59.5)` take radius 7 (jaw roots, half the 14 mm jaw).
- **Convex corners.** Every other profile corner takes a tangent round of radius 2, so no sharp edge runs along Y.
- **`OpeningMax`.** 40 sets the bottom jaw: its top face is `z_b = -(40 + 7.5 + 12)` = -59.5, so the pad clears the
  jaw-root round by 0.5 at its lowest position.
- **Cuts.**
  - J1: a hole of diameter 5.6 along Z at `(30, 0)` through the top jaw and the upper fin, and a hex head pocket in
    the top jaw's underside, depth 3.5 (`PanSite`: Head `(30, 0, 0)`, Dir +Z, Corner +Y, Length 35).
  - Clamp screw: a hole of diameter 5.6 along Z at `(25, 0)` through the bottom jaw, and a nut pocket open to the
    bottom jaw's top face (hex corner radius 4.8188, Corner +Y) from z = -59.5 down 5.1 (floor at -64.6), the floor's
    outline rounded 0.6.
- **Edge finishing.** Chamfer both Y cap outlines by 0.5 before the cuts.
- **Strength.** 500 N at the pad acts on a 32 mm lever to the spine's centreline, 16 N.m. The jaw and spine section of
  30 by 14 carries it at 16.3 MPa along the layers.

## Pad

- Local frame: origin at the top face's centre on the screw axis, +Z up. The footprint is x -24.5 to 15, y -15 to 15,
  z -12 to 0, with the plan corners rounded 2. The pad is posed at world `(25, 0, -o)`.
- A blind socket of diameter 5.7 (Regular on the M5 shank) runs from the bottom face up 8, so its floor is at z = -4.
  The floor corner is rounded 0.5 by a revolved meridian with 8 chords per round (`fastenertool` rounded revolve, as
  `m2fastener.RoundedHoleTool`).
- The pad's -X face stands 0.5 from the spine face (Loose). The pad cannot turn, so it does not scrub the desk. It
  rests on the screw's tip and falls off only when the clamp is turned over with no desk.
- Print it top face down. Chamfer both plan cap outlines by 0.5.

## Clamp screw and screw knob placement

At `Opening` `o`:

- the screw's tip is at z = -(o + 4), the pad socket's floor;
- the M5 x 70's underhead is at z = -o - 74;
- the screw knob's outer face is at z = -o - 77.5 and its inner face at z = -o - 67.5;
- `ScrewKnobGap`, from the knob's inner face to the bottom jaw's underside (z = -73.5), is `o - 6`: 4 at `o` = 10 and
  34 at `o` = 40. Require at least 0.5;
- `ScrewSite`: Head `(25, 0, -o - 77.5)`, Dir +Z, Corner +Y, Length 70;
- the jaw nut, seated on its floor, occupies z -64.6 to -59.9. The screw passes through it at every valid `o`.

## Swivel

- Local frame: origin on J1's axis in the swivel's bottom face, +X toward J2, +Z up (the J1 axis).
- Body: the `decad.Intersect` of a plan prism and an elevation prism.
  - Plan (XY) outline: a half circle of radius 10 about the origin, sides y = +-10 to x = 80, and a square end.
    Extrude z 0 to 20. Decad refuses an Intersect whose operands share coplanar faces or touch tangent, so the
    build uses a plan prism of z -1 to 21 and end x = 81; the swivel's shape is unchanged.
  - Elevation (XZ) outline: x from -11 to 70 at z 0 to 20, closed by a half circle of radius 10 about `(70, 10)`.
    Extrude y -11 to 11.
- J1 tongue, bottom-flush: cut z 8 to 20 for x < 30, with the inside corner `(30, 8)` rounded 4 (centre `(26, 12)`),
  the tool sketched in XZ and extruded through Y. Cut a hole of diameter 5.6 along Z at the origin. The tongue is
  8 thick, z 0 to 8.
- J2 fork at x = 70, z = 10: the shared fork with `Lf = 70` (slot through Z, deepest point x = 44).
  `JointSites[0]`: Head `(70, -10, 10)`, Dir +Y, Corner +Z, Length 25.
- Print it bottom face down, the tongue on the bed.
- J1 range proof: the swivel's material above the gap starts `26 + sqrt(16 - 3.74^2)` = 27.42 from J1. It must stay
  outside the frame's box `x' <= 16.5, |y| <= 15.5` (upper fin end, knob and spine, Loose added). The inequality
  `s sin(phi) + 10 cos(phi) >= 15.5` with `s` = 27.42 holds to `phi` of about 127.8, so `J1Max` = 125.

## Link

- Local frame: the tongue's axis at the origin and the fork's axis at `(L, 0, 0)`, both joint axes along Y.
- The XZ outline is a stadium: round ends of radius 10 about both axes and straight sides at z = +-10. Extrude y -10
  to 10. The tongue is at x = 0 and the fork has `Lf = L`. `JointSites[i]` (fork): Head `(L, -10, 0)`, Dir +Y, Corner
  +Z, Length 25.
- Print it on its -Z face. The lowest 2.9 mm of each round end overhangs more than 45 degrees from the bed's tangent
  line, as the underside of any horizontal cylinder does.
- Mass is about 45 g at `L` = 120 with 20% infill. "Joint capacity" uses it.
- Links A and B are identical copies.

## Head

- Local frame: the J4 axis at the origin, +X along the head, +Z up. The XZ outline is a half circle of radius 10 about
  the origin and sides z = +-10 to x = 55 (`HeadLength`), with the front corners rounded 2. Extrude y -10 to 10. The
  tongue is at x = 0.
- Union a cone boss on the top face at `(40, 0, 10)`: a revolved meridian from base radius 6 at z = 10, at 45 degrees
  to radius 4 at z = 12.
- Cut the stud hole, diameter 5.6 along Z at x = 40, through the boss and the head.
- `StudSite`: Head `(40, 0, -20)`, Dir +Z, Corner +X, Length 40. The screw knob's inner face is at z = -10 (the head's
  bottom) and its outer face at z = -20. The bolt's tip is at z = 23.5, so `StudProtrusion` = 13.5 above the
  interface face.
- Print it bottom face down.

## Knobs

Both knobs share one lobed outline: six convex lobes of radius 4 centred on radius 11 at 0, 60, 120, ... degrees, and
six concave valleys of radius 3 centred on radius `R_v` = 13.857 at 30, 90, ... degrees. `|c_l - c_v|` = 7 = 4 + 3,
so each lobe joins its neighbours tangent. Lobe tips lie at radius 15 and valley floors at 10.857. Sketch every arc
with `NewTangentCircles` between neighbours.

- **Knob** (J1 to J4). Local origin at the inner face's centre on the axis, +Z outward. Extrude 12 (`KnobHeight`).
  Cut a hole of diameter 5.6 through. Cut a hex nut pocket open to the outer face: corner radius 4.8188, Corner +X,
  floor at z = 3 (9 deep), floor outline rounded 0.6. The nut sits on the floor, pulled toward the seat by the bolt,
  so the knob is clamped between the nut and the seat. The pocket is 9 deep, deeper than a 5.1 nut cavity, so the
  bolt's tip ends inside it. The adapter's pocket is deep for the same reason. Part mark: 1 hole.
- **Screw knob** (clamp screw, stud). Local origin at the inner face, +Z outward. Extrude 10 (`ScrewKnobHeight`).
  Cut a hole of diameter 5.6 through. Cut a hex head pocket open to the outer face, press fit: corner radius
  `4.6188 + 0.05` = 4.6688, depth 3.5 (the head ends flush), Corner +X, floor outline rounded 0.6. The wall round the
  press pocket is at least 1.6 (6.19 to the valleys). Print it with the axis vertical. Part marks: 2 holes.
- Print each knob inner face down. Chamfer both cap outlines by 0.5.
- Part marks sit on the outer face, which touches no part, and run through. The holes have diameter 1.5 on radius
  `MarkRadius` = 8, the first at 30 degrees, then 55 degrees (25 degrees apart). Each sketch places the centre by
  horizontal and vertical distances from the origin.

Knob placement at J2 to J4, in the fork's local frame: the inner face is at y = 10 (seated), the nut on the floor
spans y 13.0 to 17.7, the bolt's tip is at y = 18.5 (0.8 past the nut) and the outer face at y = 22. The tip is 3.5
inside the outer face, and 3.0 once the fins close.

Knob placement at J1, in the world frame: the inner face is at z = 28.26, the nut spans z 31.26 to 35.96, the tip is
at z = 38.5 (2.54 past the nut) and the outer face at z = 40.26 (the tip 1.76 inside).

## Mount interface

The interface is the deliverable. Future adapters build against it by importing `models/clamparm` constants and
tools. NEVER copy its geometry.

- **Frame `I`.** The origin is the interface face's centre on the stud axis (head `(40, 0, 10)`). N is the head's +Z
  (out of the face), U the head's +X (away from J4), and V = N x U.
- **The arm side provides:**
  - a flat face, x 20 to 55 by y +-10 in head coordinates (U -20 to 15, V +-10);
  - a cone boss: base diameter 12 at the face, 45 degrees, height 2, top diameter 8;
  - an M5 stud along N that protrudes `StudProtrusion` = 13.5 when the screw knob is seated;
  - the screw knob under the head (radius 15, 10 tall), which turns the stud.
- **The adapter must provide:**
  - a flat mating face with normal -N that covers at least the annulus of diameters 13 to 20 about the stud;
  - a cone recess at 45 degrees, radius `RecessRadius` = `6 + 0.35 sqrt(2)` = 6.495 at the mating face, running to the
    stud hole of diameter 5.6 (0.35 normal clearance, Regular);
  - an ISO 4032 M5 nut held against rotation (a hex pocket open to a face, per AGENTS.md), its near face at a height
    `h_n` over the mating face with `AdapterNutMin` = 4.7 <= `h_n` <= `AdapterNutMax` = 8.3. The lower bound leaves
    1 mm over the recess; the upper bound leaves the tip 0.5 past the nut's far face;
  - a hole of diameter 5.6 on the axis from the face to at least `AdapterStudClear` = 14.5 above it. The tip must not
    stand outside any face;
  - a total adapter and camera mass of at most 230 g, with the centre of mass within 10 mm of the stud axis
    ("Joint capacity");
  - clearance: nothing below the face plane within U < -20, where link B and the J4 knob pass when the wrist tilts.
- The adapter may sit at any angle about N. The stud's preload and the face friction hold it, with about 1 N.m
  capacity at 400 N.
- Exported constants: `BossBaseRadius` 6, `BossHeight` 2, `RecessRadius` 6.495, `RecessClearance` 0.35,
  `StudProtrusion` 13.5, `AdapterNutMin` 4.7, `AdapterNutMax` 8.3, `AdapterStudClear` 14.5.
- `InterfaceRecessTool(ctx, doc, f r3.Frame) (*decad.Body, *sketch.Sketch, error)` builds the recess and stud-hole
  cutter for an adapter whose mating face lies on `f` with `f`'s +Z = N. The meridian `(r, h)` has the vertices
  `(0, -1)`, `(7.495, -1)`, `(6.495, 0)`, `(2.8, 3.695)`, `(2.8, 16)`, `(0, 16)`, each placed by
  `sketchcheck.DimensionPoint`, revolved about `f`'s Z.
- With no adapter fitted, the stud bolt and its screw knob are loose in the head and can drop out. This is a known
  limitation: a retainer would need a part or hardware this model does not have.

## Placeholder adapter

- A disc of diameter 40 and thickness 15, local frame `I` (z 0 to 15). Cut `InterfaceRecessTool` at z = 0. Cut a nut
  pocket open to the top face: hex corner radius 4.8188 from z = 4.7 to 15 + 1, floor rounded 0.6. The nut sits at
  z 4.7 to 9.4. The stud's tip is at z = 13.5, 4.1 past the nut and 1.5 inside the top face.
- The top face is a flat pad for double-sided foam tape.
- Print it with the mating face down: the recess walls are at 45 degrees and the pocket opens upward.
- Chamfer both disc rims by 0.5. The recess rim stays sharp: it is the 45 degree register.
- The pocket is deep because the stud's tip ends inside it, not because the nut needs the depth.

## Hardware

| Part | Qty | Item | Site |
| --- | --- | --- | --- |
| M5 x 35 hex bolt | 1 | `m5bolt` | J1 |
| M5 x 25 hex bolt | 3 | `m5bolt` | J2, J3, J4 |
| M5 x 40 hex bolt | 1 | `m5bolt` | stud |
| M5 x 70 hex bolt | 1 | `m5bolt` | clamp screw |
| M5 nut | 6 | `m5nut` | 4 knobs, bottom jaw, adapter |

- Bolts are hex head (ISO 4017, fully threaded), so a hex pocket stops the head turning while a knob turns the nut.
  Nuts are ISO 4032 M5. Thread is not modeled. Every length is in `m5bolt.StandardLengths()`.
- A head pocket is the corner circle plus 0.3 mm radial, as deep as the head is tall (3.5). A nut pocket is the
  corner circle plus 0.2 mm radial, 5.1 high where it is a cavity (4.7 plus 0.2 on each face). The clearance hole is
  diameter 5.6.
- The head's 0.6 floor round assumes the modeled 30 degree chamfer on the head's underside (the ISO 4017 washer
  face). A plain underside would sit about 0.1 mm proud at the flats.
- Hex pockets keep sharp corners (mating corners) and round their floor outline by `PocketRound` = 0.6.
- The clamp screw is a purchased bolt. A printed screw would be a printed shaft with a thread, which
  [Split shafts](../../docs/model-pipeline.md#split-shafts) forbids.
- Joint bolts have their head in the fork's -Y fin and the knob and nut on the +Y fin, so turning the joint never turns
  a bolt in its nut. The nuts sit in pockets open to the knob's outer face and the bolt holds them there.
- The J1 bolt's head sits in the top jaw's underside, flush with the desk face. No bolt tip stands outside a face.

## Containers and bodies

`Build` files 25 bodies in children of the document root: `Frame` (1), `Pad` (1), `Swivel` (1), `Links` (2), `Head`
(1), `Adapter` (1), `Knobs` (4), `Screw Knobs` (2), `Bolts` (6) and `Nuts` (6). Use `ChildNamed` and `AddBody` after
each body's last boolean. Names follow
[Body containers](../../docs/model-pipeline.md#body-containers).

## Part fits

| Fit | Class | Clearance |
| --- | --- | --- |
| Tongue in fork (J1 to J4) | Tight | 0.26 per side, free state |
| Bolt shank in holes | Fastener rule | 0.3 radial (diameter 5.6) |
| Hex head in pocket | Fastener rule | 0.3 past corners, flush |
| Nut in pocket | Fastener rule | 0.2 past corners |
| Bolt head in screw knob | Press fit | 0.05 past corners |
| Pad socket on screw | Regular | 0.35 radial |
| Pad -X face to spine | Loose | 0.5 |
| Pad to jaw-root rounds at the `Opening` limits | Loose | at least 0.5 |
| Screw knob to bottom jaw | Loose | at least 0.5 (4 at `o` = 10) |
| Cone boss in recess | Regular | 0.35 normal |
| Jointed parts across each joint at its limits | Loose | at least 0.5 |

The knob closes the joint's free-state fit when tightened. Clamped faces are modeled in contact and are interfaces,
not fits: the bolt head on its pocket floor, the knob on its seat, the nut on its pocket floor and the adapter on the
interface face. The model is **Mixed**: the custom fits are adjustable and the bolts and nuts keep their nominal size.

## Edge finishing

[Surface edge finishing](../../docs/model-pipeline.md#surface-edge-finishing) applies with 0.5 chamfers through
`internal/edgefinish` (`Chamfer`, and `ChamferExcept` to skip edges tangent to a round), in this order per part:

| Part | Before cuts | After cuts |
| --- | --- | --- |
| Clamp frame | both Y cap outlines | none (hole and pocket rims are runs under 20 mm) |
| Pad | both Z cap outlines | none |
| Swivel | Z = 0 and Z = 20 cap outlines; Y = +-10 face outlines (the J2 round-end arcs) | J1 tongue top face (z = 8) outline, skipping the shoulder round; fin inner faces (y = +-4.26) outlines, skipping the slot round |
| Link | Y = +-10 cap outlines (stadium) | tongue faces (y = +-4) and fin inner faces (y = +-4.26) outlines, skipping the tangent rounds |
| Head | Y = +-10 cap outlines, before the boss union | tongue faces (y = +-4), skipping the shoulder rounds |
| Knob, screw knob | both cap outlines | none |
| Adapter | both disc rims | none (the recess rim stays sharp) |

The +-Z faces of links and head meet the round ends tangent, so their only convex edges are the cap-outline edges. The
tongue and fin rims lose 0.5 of a 10 mm friction ring; that is accepted. A decad refusal of any listed chamfer stops
the work: do not substitute a retained edge without handing it back.

## Rules that do not apply

- [Arbor collars](../../docs/model-pipeline.md#arbor-collars) and
  [Split shafts](../../docs/model-pipeline.md#split-shafts): the model has no printed shaft, arbor, axle or pin. Every
  pivot is a steel bolt.
- [Gear train clearances](../../docs/model-pipeline.md#gear-train-clearances): the model has no gears.
- Felt or rubber pads, cable routing, thread geometry on the hardware, desk geometry and a motion GIF are not
  modeled.

## Look-alike parts

Two sets can be mixed up. The marks follow
[Look-alike part marks](../../docs/model-pipeline.md#look-alike-part-marks).

| Set | Parts | Reason | Holes |
| --- | --- | --- | --- |
| Lobed knobs | Knob | same lobed outline, hex nut pocket | 1 |
| Lobed knobs | Screw Knob | same lobed outline, press head pocket | 2 |

Links A and B are identical copies, not look-alikes, and carry no marks. Fasteners are not marked.

## Retained sharp corners and edges

The rule is "Corners and roots" in `docs/model-pipeline.md`. Print orientations: frame on its -Y face, pad top face
down, swivel bottom face down, links on their -Z face, head bottom face down, knobs inner face down, adapter mating
face down.

| Part | Corner or edge | Reason |
| --- | --- | --- |
| Every printed part with a hex pocket | The six corners of each head or nut pocket | Mating corners with the sharp corners of the M5 hardware. |
| Every printed part with a hex pocket | The pocket floor's 0.6 round where a layer cuts it obliquely | A slanted section reads the round as turns; it is the round the rule asks for, so only that band is exempt. |
| Adapter | The recess rim | The 45 degree register. |
| Link, swivel fork, head | Round ends near the bed | An overhang note ("Link"), not a corner. |

Only these kinds may be added when the corner test finds more: further mating corners with hardware, chamfer faces
that edge finishing adds, and a root rounded below half its section with the reason stated. Any other `cornercheck`
finding is a geometry defect: fix it here and in `steps.md`, then re-emit the Go.

## Printing and assembly order

Print every part in the orientation its section states. The nuts go into their pockets at assembly.

1. Seat the J1 bolt's head into the top jaw's pocket from below, slide the swivel's tongue into the gap, and fit the J1
   knob with its nut on top.
2. Drop the jaw nut into its pocket. Press the screw knob onto the M5 x 70's head, push the screw up through the jaw
   and set the pad on its tip.
3. Slide link A's tongue into the swivel's fork, push the J2 bolt in from -Y and fit its knob on +Y.
4. Fit link B and the J3 bolt and knob the same way.
5. Fit the head and the J4 bolt and knob.
6. Press the stud's screw knob onto the M5 x 40's head and push the stud up through the head.
7. Set the nut in the adapter, set the adapter on the boss, and turn the stud's screw knob.

## Valid range

- `LinkLength` 80 to 150 and `Opening` 10 to 40.
- The clamp holds desks 10 to 40 mm thick. The 7 mm jaw-root round keeps a square desk edge up to 7 mm off the spine.
- `Pan` +-125, `Shoulder` -60 to 115, `Elbow` +-115 and `Wrist` -110 to 90.
- Mount load: adapter plus camera at most 230 g, centre of mass within 10 mm of the stud axis.
- `Validate` does not check collisions between non-adjacent parts. Only the listed poses are checked.

### Range of motion

- For each limit case below, build it and require the `meshgap` clearance between the two sides of that joint (all
  bodies proximal against all bodies distal, hardware included) to be at least 0.5. `shoulder-min` runs at `Opening`
  40, where the screw knob hangs lowest.
- A constant that fails is reduced in magnitude by whole degrees until it passes. Change it here and in `steps.md`
  first, then re-emit the Go. NEVER hand-edit emitted Go.
- If `JointMax` would fall below 115, the default pose, `raised` and the folded render pose stay valid down to 115
  only. Stop and hand back instead of changing them.

## Representative cases

- `default`.
- `short`: `LinkLength` 80.
- `long`: `LinkLength` 150.
- `thin-desk`: `Opening` 10.
- `thick-desk`: `Opening` 40.
- `raised`: `Shoulder` 110, `Elbow` -70, `Wrist` -40.
- `placed`: the frame with origin `(10, -5, 7)`, U `(0, 1, 0)` and V `(0, 0, 1)`.
- Limit cases, the other fields at their defaults:
  - `pan-max`: `Pan` 125, and `pan-min`: `Pan` -125.
  - `shoulder-min`: `Shoulder` `ShoulderMin`, `Opening` 40, and `shoulder-max`: `Shoulder` 115.
  - `elbow-min`: `Elbow` -115, and `elbow-max`: `Elbow` 115.
  - `wrist-min`: `Wrist` `WristMin`, and `wrist-max`: `Wrist` `WristMax`.

## Checks and images

Check every case above:

- every sketch fully constrained (`sketch.FullyConstrained`, DOF 0, no redundant constraints, no conflicts), including
  each optional sketch;
- 25 bodies, the container names and their members, and `doc.VerifyContainers().Err()` nil;
- every fit in "Part fits", each bolt's head flush and inside its pocket, each nut inside its pocket and no fastener
  overlapping a part it passes through;
- the range limits ("Range of motion");
- `internal/cornercheck` on every printed part with its `PrintUp`, exempting only the listed corners, each matching a
  finding;
- the edge finishing sections, the look-alike hole counts (1 against 2, each hole through the part) and
  `TestClampArmCapacity`;
- the default's bounds, `VerifyAll` on the frame, swivel, links and head, and the placed case against the default
  under the frame; invalid inputs add no body.

Images come from real bodies through `tools/render/` ([Model images](../../AGENTS.md#model-images)): hero, side, back,
top, poses, exploded, clamp section, joint section, interface section, interface close-up and one image per distinct
part. The README's exploded view and parts table follow
[Exploded views](../../docs/model-pipeline.md#exploded-views).
