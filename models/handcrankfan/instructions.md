# Hand-crank fan

## Purpose

Build a handheld fan. The operator holds the grip in the left hand and turns the crank on the fan's right side with
the right hand. The crank turns a straight-tooth bevel gear
pair, the pair turns the fan shaft at twice the crank speed, and the rotor blows air toward the
operator. The blades are shaped like small wind turbine blades and bolt to the hub one by one, so a damaged blade
can be swapped. The whole fan is meant to be 3D-printed; the only bought parts are M2 bolts and nuts. The right-hand crank and the operator-facing rotor are the two orientation rules of this model.

The bevel pair is the `models/bevelgear` construction, imported and not copied.

## API and parameters

Package `handcrankfan`. Expose `Params{CrankAngle, BladePitch float64; Blades int}`.
Defaults: `5*pi/4`, `25*pi/180` and `4`.

- `CrankAngle` is any finite angle in radians. `Ratio*CrankAngle` must also be finite.
- `BladePitch` lies in `[10°, 40°]`, in radians. It is the blade angle at radius 48 (75 % of the tip radius 64),
  measured from the plane of rotation. It turns the whole blade about its radial line, as a pitch bearing does.
- `Blades` is 3 or 4.

Reject an invalid input before the document changes.

All lengths are mm and angles are radians. The frame is the operator's: X runs to the operator's right,
Y runs away from the operator, and Z runs up. The operator therefore looks along +Y.

Expose `DefaultParams`, `Params.Validate`, `Motion(Params) (State, error)`,
`NewDocument(opts ...decad.DocumentOption) *decad.Document` and `Build(ctx, doc, Params) (*Assembly, error)`.
`NewDocument` returns `decad.New` with `decad.WithChainTolerance(0.05 mm)` applied before the caller's options.
`Build` needs a document made by it: the joints between the hub and the lofted blades need that tolerance.

`State` holds `CrankAngle`, `FanAngle`, `Apex` (r3.Vec) and `ArmDirection` (r3.Vec). `Ratio` is the constant 2.
`Assembly` holds `Parts []Part`, `Sketches []*sketch.Sketch` and `State`.
`Part` holds `Name`, `Group`, `Motion` and `Body`. `Motion` is `fixed`, `crank` or `fan`.

## Orientation rules

- The crank is on the right. The crank shaft lies on the line through the apex along X and leaves the gearbox
  toward +X. The crank shaft ends beyond the wall at X 43.8, and the crank pin and grip have larger X than every fixed part.
- The fan faces the operator. The fan shaft lies on the line through the apex along Y and runs toward -Y. The
  rotor has smaller Y than every other part except the fan shaft, and its blades blow air toward -Y.

## Crank turn and fan turn

A forward crank turn is clockwise as seen from the right, a rotation about the -X axis.
`CrankAngle` is the forward turn, measured from the arm pointing straight up: the arm points along
`(0, sin CrankAngle, cos CrankAngle)`.

The bevel pair has a 90° shaft angle. With the crank's outward direction +X and the fan's outward direction -Y,
the two angular velocities satisfy ω_fan = Ratio · ω_crank along -Y. The forward crank turn therefore turns the
fan about -Y by `Ratio * CrankAngle`: counter-clockwise as the operator sees it.

A blade at +X moves toward +Z while the fan turns forward. At radius r its chord, from trailing edge to leading
edge, runs along `(0, sin beta(r), cos beta(r))`, with
`beta(r) = alphaD + atan(tan(p0 - alphaD) * 48 / max(r, 21.5)) + (BladePitch - p0)`, `alphaD = 5°` and
`p0 = 25°`. The leading edge therefore leads the motion and sits upstream (+Y). The concave pressure face leads
the motion and also faces -Y, so it pushes air toward the operator. A blade at the other angles is the same blade
turned about the fan axis.

The shape follows a wind turbine blade. A round root carries the joint. The section then turns into a thick oval
and an airfoil at the maximum chord (r 25), and the chord tapers to the tip along the Schmitz optimum. The
thickness ratio falls from 100 % at the root to 26 % at maximum chord and 17 to 18 % outboard. The blade angle
follows the flow angle of a uniform axial inflow plus a design angle of attack of 5°, so it falls from the root to
the tip, and it stays constant inboard of r 21.5, as on a turbine root. A fan adds energy where a turbine takes
it, so the angle is flow plus attack, not flow minus attack. The sections are NACA 4-digit mean lines with 3 to
4 % camber outboard, blended with an ellipse at the root.

## Dimensions

The apex is the point where the crank axis meets the fan axis: `(0, 0, 170)`.

### Bevel pair

`bevelgear.BuildPair` with Module 2, 32 driving teeth, 16 pinion teeth, 90° shaft angle, straight teeth and
default pressure angle and samples. The driving bore is 8 and the pinion bore is 6. The placement frame has origin
`(24, 0, 170)`, U = -Y and V = +Z, so the driving gear's back face lies in the plane X = 24 and grows toward -X.
The pinion's back face lies in the plane Y = -36.

The heel distances 24 and 36 follow from the pair: the driving heel is `Module*16/2 + Module*32/8` and the
pinion heel is `Module*32/2 + Module*16/8`.

### Fixed parts

- Handle, `handle`: an oval grip held in the left hand, a smooth loft through 14 oval sections, raked 15° like a
  pistol grip with its base toward the operator (-Y). It spans Z [8, 127.9], so its top is 0.1 below the gearbox
  floor, and the centre of its top is at X 7, Y -28. Section `z`, width `W` (along X) and length `L` (along Y), in
  mm, centred on `X 0, Y (z - 68) tan 15°` before the loft is placed:

  | z | 8 | 9 | 11 | 14 | 18 | 26 | 45 | 65 | 85 | 100 | 114 | 122 | 126 | 127.9 |
  | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
  | W | 28 | 33 | 37 | 36 | 32 | 31 | 32 | 33.5 | 34 | 33 | 31 | 31.5 | 33 | 35 |
  | L | 38 | 43 | 47 | 46 | 41 | 39 | 40.5 | 42 | 42.5 | 41 | 38 | 36.5 | 35.5 | 35.5 |

  The base flares to stop the little finger sliding off, the palm swell is at Z 85, and the neck under the
  gearbox is the narrowest part. The grip is 35 to 38 mm across when measured round, inside the 30 to 50 mm range
  for a power grip, and the hand zone is about 108 mm long. There are no finger grooves, which make pressure
  points for other hand sizes. Each section is a fit spline through 25 points that leaves a 2° gap at the back,
  closed by two short lines, as the blade sections are. Two pilot holes of diameter 1.8 run from Z 116 up through the
  top, at X -2 and X 16 on Y -28.
- `handle-bolt-1`, `handle-bolt-2`, group `hardware`: two M2 x 16 screws, one at each pilot. Each stands on
  Y -28 with its head 2.0 high on the floor's top face (0.1 above it) and a shank of radius 0.8, which is the
  thread root, down into the pilot. They cut their own thread in the printed pilot. Each is one plain body.
- Gearbox, one part, `gearbox`: a block minus a cavity, with rounded edges and nothing that sticks out. Each
  shaft turns in a hole through a flat wall about 10 thick, so a printed shaft has a long bearing and the housing has
  no boss or sleeve. Printed floor down, every wall is vertical and the housing needs no support.
  - Block: the outline Y [-46, 20], Z [128, 186] with four corners of radius 3, on the plane X -20, extruded
    54 toward +X, with both end faces rounded with radius 2.
  - Cavity: the outline Y [-36.2, 21], Z [134, 187] with the corner at (Y -36.2, Z 134) rounded with radius 3,
    on the plane X -21, extruded 45.2 toward +X, with its end face (X 24.2) rounded with radius 2. Each wall's inner
    face is 0.2 from its gear's back face, so a gear needs no spacer. Cutting the cavity leaves three plates:
    - floor plate: X [-20, 34], Y [-46, 20], Z [128, 134];
    - front plate: X [-20, 34], Y [-46, -36.2], Z [128, 186];
    - right plate: X [24.2, 34], Y [-46, 20], Z [128, 186].
  - A bore of diameter 8.4 on the crank axis through the right plate, a bore of diameter 6.4 on the fan axis through
    the front plate, and two holes of diameter 2.4 through the floor at X -2 and X 16, Y -28, for the grip screws.

### Crank parts, motion `crank`

- `driving-gear`: the driving bevel gear.
- `crank-shaft`: a round shaft of radius 4 on the crank axis, X [14, 37], a collar of radius 6 at X [34.3, 37.8]
  (0.3 from the wall's outer face) and a D-shaped stub, a circle of radius 4 with a flat 3 below the axis (toward -Z),
  X [37.5, 43.8]. A nut slot 4.3 wide and 1.9 deep at X [40.2, 42.1], open at the flat, and a screw hole of diameter
  2.4 along the axis from X 37 to the stub end, join the stub. The hole continues into the shaft to X 35.
- `crank-arm`: a stadium 20 wide, X [38, 46], with the axis and the pin 55 apart (end radius 10, so the arm
  is 75 long), both faces rounded with radius 1.5, and a hole of diameter 2.4 at each end. A D pocket (radius
  4.15, flat 3.15 toward -Z) at X [37, 44] opens on the inner face for the shaft stub, and one (flat toward +Z)
  at X [40, 47] opens on the outer face for the pin stub. Each pocket leaves a 2 mm floor with the screw hole.
- `crank-pin`: a D-shaped stub (radius 4, flat 3 toward +Z, away from the shaft) at X [40.2, 46.5] with a nut
  slot at X [42, 43.9] and a screw hole of diameter 2.4 from X 39, a round shank of radius 4.5 from X 46.1 (the
  shoulder, 0.1 clear of the arm) to X 93, and a head of radius 5.8 at X [92.6, 95.6]. The pin is 55 from the
  crank axis. The handle slides on over the stub, so nothing on the pin is wider than the handle's bore except
  the head.
- `crank-grip`: the crank handle, a small revolved barrel X [46.2, 96.2], 50 long, for the thumb, index and middle
  fingers. Its outer diameter is 14.4 at the inner end, 15.6 at 8 from it, 16 at 22, 14.5 at 36 (the waist), 16 at
  43, 18.6 at 48.8 (the flare) and 16.6 at the outer end. This is inside MIL-STD-1472G's light-load handle (10 to 16
  mm thick, 25 to 75 mm long). It has a bore of diameter 9.4 (0.4 clearance on the shank, so it spins freely and tilts
  at most 0.5°) and a counterbore of diameter 12.8, 3.8 deep, at the outer end for the pin head. It is made of cones
  and cylinders, with chamfers of 0.5 to 1.5 on its corners: Decad cannot decide a handle with rounded corners
  against its pin.
- `crank-arm-bolt` and `crank-pin-bolt`, group `hardware`: M2 x 10 screws with nuts, each one plain body. The
  arm bolt has its head on the arm's outer face and runs down the crank axis through the shaft stub; the nut sits
  in the shaft's slot. The pin bolt has its head on the arm's inner face and runs up the pin axis through the pin
  stub; the nut sits in the pin's slot. Together with the D stubs they hold the arm to the shaft and the pin to the
  arm, so nothing is loose, and the handle spins on the pin, held by the pin head.

Each joint just before it is tightened keeps a gap of 0.1 to 0.2 mm, because Decad cannot decide a pair of
parts whose faces touch.

### Fan parts, motion `fan`

- `pinion`: the pinion bevel gear.
- `fan-shaft`: a cylinder of radius 3 on the fan axis, Y [-80, -22].
- `fan-hub`, group `rotor`: a disc of radius 14 on the fan axis, Y [-76, -56], mid plane Y -66. It has a bore
  of radius 3 and, for each blade k at angle `2 pi (k-1) / Blades` about the fan axis (the first along +X, turning
  toward +Z):
  - a radial socket of radius 3.15 on the hub's mid plane, from r 5.6 outward through the hub surface;
  - a bolt hole of radius 1.15 parallel to the fan axis at r 9.5, through the hub;
  - a head counterbore of radius 2.15 in the front face, 2.2 deep (floor at Y -73.8);
  - a hex nut trap of 4.3 across flats in the back face, 4.0 deep (floor at Y -60), with its corners on the
    radial line.
- `fan-blade-1` .. `fan-blade-N`, group `blade`: one blade each, with N = `Blades`. A blade is a smooth loft
  through the ten sections of the table below, stacked on its radial line through the hub's mid plane. The loft
  starts at r 14.3, just outside the hub. A root stud of radius 3, from r 6 to r 16 on the radial line, is joined
  to it, with a bolt hole of radius 1.15 parallel to the fan axis at r 9.5. The stud engages the socket from r 6
  to r 14 with a 0.15 sliding clearance. The first blade lies along +X at fan angle 0.

  | r | 14.3 | 16.3 | 19 | 21.5 | 25 | 30 | 38 | 46 | 55 | 64 |
  | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
  | chord c | 8.0 | 8.0 | 11.5 | 15.5 | 17.8 | 16.5 | 14.5 | 12.5 | 10.5 | 8.0 |
  | thickness ratio t/c | 1.00 | 1.00 | 0.60 | 0.38 | 0.26 | 0.22 | 0.19 | 0.18 | 0.17 | 0.18 |
  | camber m (of c) | 0 | 0 | 0.010 | 0.025 | 0.035 | 0.040 | 0.040 | 0.040 | 0.035 | 0.030 |
  | ellipse weight w | 1 | 1 | 0.60 | 0.25 | 0 | 0 | 0 | 0 | 0 | 0 |

  Each section, at chord fraction x from the leading edge, has the NACA 4-digit mean line `yc(x)` with camber
  `m` at `p = 0.40`. Its half thickness is `(1 - w) naca(x) + w ell(x)`, where
  `naca(x) = (t/2 - 0.15 tTE) / 0.10002 * n(x) + tTE x / 2`,
  `n(x) = 0.2969 sqrt(x) - 0.1260 x - 0.3516 x^2 + 0.2843 x^3 - 0.1036 x^4`,
  `ell(x) = t sqrt(x (1 - x)) + tTE x / 2`, `t = (t/c) c` and the trailing-edge thickness `tTE = 0.5`.
  It is laid off normal to the chord on both sides of `c yc(x)`. The section is turned to `beta(r)` at
  `BladePitch = 25°` about its stacking point, the mean-line point at chord fraction `0.40 + 0.10 w`, which lies on
  the radial line. The loft always uses the sections at 25°. For another pitch the finished loft is turned by
  `BladePitch - 25°` about the radial line before the stud joins it, because Decad's twisted loft panics on
  some section tables that depend on the pitch. Outboard camber is held to 3 to 4 %, because Decad's loft refuses
  thin sections with more.
- `fan-bolt-1` .. `fan-bolt-N`, group `hardware`: each is one M2 x 16 socket head cap screw and its M2 nut as one
  plain body without a thread: a head of radius 1.9 and height 2.0, a shank of radius 1.0, and a hex nut of
  4.0 across flats and 1.6 thick. The bolt lies on the blade's bolt line with its head 0.1 above the counterbore
  floor and its nut 0.1 below the trap floor, which is the joint just before tightening. The thread end lies
  inside the hub. Undoing the bolt from the front frees the blade; the nut stays in its trap.

The hub is 3D-printed with its front face down. The head counterbore, the nut trap and the sockets are sized for
a printed fit and may need a drill or a test print first.

## Parts set and exploded view

`Assembly.Set(doc)` returns a `parts.Set` (Decad's `parts` package) over the build's document. The printed parts
are `Manufactured` in PLA, and the screws and nuts (`handle-bolt-*`, `crank-arm-bolt`, `crank-pin-bolt`,
`fan-bolt-*`) are `Purchased` steel, grouped by `Spec` for the bill of materials: M2 x 16 self-tapping screws for the
grip, M2 x 10 screws with nuts for the crank and M2 x 16 screws with nuts for the blades. Every joint is a
`PinInBore`, with its axis stated in the posed coordinates, because the faceted bores cannot state it:

- the crank shaft in the wall (turns), in the arm and in the driving gear, the arm screw in the arm and the shaft,
  the pin and the pin screw in the arm and in each other, and the handle on the pin (turns);
- the fan shaft in the wall (turns), in the pinion and in the hub, each blade's stud in the hub, and each blade bolt
  in the hub and in its blade;
- each grip screw in the floor and in the grip.

`Assembly.Explode(ctx, set)` runs `Set.Explode` with the gearbox as the ground. The pin screw's joint declares a pull
toward the shaft, because its head is on the arm's inner face. The stages run in this order:

1. the shaft, the arm and the pin screw;
2. the handle, then the pin, then the arm screw, each along +X: the handle slides over the pin's stub end, so along
   the crank axis the arm, the handle and the pin follow in that order with gaps between them;
3. the driving gear, which slides off to the left along its own axis;
4. the pinion, which lifts out of the open top;
5. the two grip screws, which lift out of the open top, and then the grip, which drops.

The fan shaft, the hub, the blades and the blade bolts come off along the fan axis in the last stage. After the
explosion no two parts share space.

## Pose

Build every part at `CrankAngle = 0`. Then turn each `crank` part about the line through the apex along -X by
`CrankAngle`, and each `fan` part about the line through the apex along -Y by `Ratio*CrankAngle`.
The gearbox and handle do not move.

## Verification

- `Build` rejects an invalid `Params` before the document changes.
- Every returned part is a native solid, every part name is unique, and the document holds no other body.
- Every sketch passes `Sketch.Verify` as fully constrained: DOF 0, no redundant constraint, no conflict.
- The crank is on the right and the fan faces the operator, by the bounds rules above.
- The grip hangs under the gearbox floor, is longer than 100, and no part reaches below its lower end. Its rings at
  Z 26, 85 and 114 have the widths and lengths of the table, and their centres follow the 15° rake.
- The blade pitch pushes air toward -Y. In the mesh of `fan-blade-1` with the pose undone, at radii near 30, 40,
  50 and 60, the chord runs from the midpoint of the trailing edge to the leading edge. The leading edge has a
  larger Y than the trailing edge, and the chord angle `atan2(dY, dZ)` lies within 0.5° of `beta(r)`.
- The blade is thick at the root. The extent of a ring across the chord over its extent along it lies within 0.03
  of the thickness ratio of the table at those radii, and is at least 0.9 near r 15.
- The hub is empty at a socket and at a nut trap. No two of the finished parts overlap or are undecided in
  `doc.Verify`. The finished parts are the hub, the blades and the bolts of the rotor, the handle and its screws,
  the gearbox, and the crank's shaft, arm, pin, handle and screws. That proves each stud fits its socket and each
  bolt its holes and pockets. The gear pair and the fan shaft are left out because they touch.
- The bolts lie inside the hub's front and back faces. Blades 2 to N and bolts 2 to N are copies of the first,
  turned by `2 pi (k-1) / Blades`.
- The exploded view puts the arm, the handle and the pin in that order along the crank axis, slides the driving gear
  to the left, lifts the pinion, and leaves no two parts sharing space.
- The crank and fan parts turn by the angles above.
- The gears mesh at the default pose: the driving gear's back face lies at X = 24 and the pinion's at Y = -36.
- No two parts overlap except the meshing gear pair, which touches at its teeth.
