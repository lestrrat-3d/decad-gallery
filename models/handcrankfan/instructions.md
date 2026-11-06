# Hand-crank fan

## Purpose

Build a handheld fan. The operator holds the grip in the left hand and turns the crank on the fan's right side with
the right hand. The crank shaft sits 70.4 below the fan's axis, so the crank handle swings clear of the blades. A pair
of equal spur gears carries the crank's turn up to the drive shaft, a straight-tooth bevel gear pair turns the fan
shaft at twice that speed, and the rotor blows air toward the operator. The blades are shaped like small wind turbine blades and bolt to the hub one by one, so a damaged blade
can be swapped. The whole fan is meant to be 3D-printed; the only bought parts are M2 bolts and nuts. The right-hand crank and the operator-facing rotor are the two orientation rules of this model.

The bevel pair is the `models/bevelgear` construction and the spur gears are the `models/spurgear` construction,
imported and not copied.

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

`State` holds `CrankAngle`, `FanAngle`, `Apex` (r3.Vec), `Crank` (r3.Vec) and `ArmDirection` (r3.Vec). `Ratio` is the constant 2.
`Assembly` holds `Parts []Part`, `Sketches []*sketch.Sketch` and `State`.
`Part` holds `Name`, `Group`, `Motion` and `Body`. `Motion` is `fixed`, `crank`, `drive` or `fan`.

## Orientation rules

- The crank is on the right. The crank shaft lies on a line along X at Y 0, Z 99.6, 70.4 below the drive shaft, and
  leaves the gear case toward +X. It ends beyond the cover at X 67, and the crank pin and grip have larger X than every
  fixed part.
- The fan faces the operator. The fan shaft lies on the line through the apex along Y and runs toward -Y. The
  rotor has smaller Y than every other part except the fan shaft and the grip's raked base, and its blades blow air
  toward -Y.
- The crank handle never meets a blade. The arm, the pin and the handle reach 65 from the crank axis. Turned through a
  full turn about the crank axis, none of their vertices comes within 15 of the disc the blades sweep.

## Crank turn and fan turn

The crank shaft turns 70.4 below the drive shaft, and a pair of equal spur gears joins them. The pair turns the drive
shaft the opposite way at the same speed, and the bevel pair turns the fan shaft at twice that speed.

A forward crank turn is counter-clockwise as seen from the right, a rotation about the +X axis. `CrankAngle` is
the forward turn, measured from the arm pointing straight up: the arm points along
`(0, -sin CrankAngle, cos CrankAngle)`. The default is 3 pi / 4, the arm pointing down and toward the operator. The
drive shaft and the bevel gear on it turn about -X by `CrankAngle`.

The bevel pair has a 90° shaft angle. With the drive shaft's outward direction +X and the fan's outward direction -Y,
the two angular velocities satisfy ω_fan = Ratio · ω_drive along -Y. The forward crank turn therefore turns the
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

The apex is the point where the drive shaft's axis meets the fan axis: `(0, 0, 170)`. The crank axis is 70.4 below it.

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
  floor, and the centre of its top is at X 2, Y -28. Section `z`, width `W` (along X) and length `L` (along Y), in
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
  top, at X -7 and X 11 on Y -28.
- `handle-bolt-1`, `handle-bolt-2`, group `hardware`: two M2 x 16 screws, one at each pilot. Each stands on
  Y -28 with its head 2.0 high on the floor's top face (0.1 above it) and a shank of radius 0.8, which is the
  thread root, down into the pilot. They cut their own thread in the printed pilot. Each is one plain body.
- Gearbox, one part, `gearbox`: an L-shaped tray, a floor and a fan-side wall with rounded edges, open at the top,
  the back and the left, with a square bearing block in front of the fan-side wall. It has no boss or sleeve. Printed
  floor down, every wall and the block are vertical, so it needs no support.
  - Block: the outline Y [-46, 11], Z [128, 181] with four corners of radius 3, on the plane X -20, extruded
    44 toward +X (to X 24), with both end faces rounded with radius 2.
  - Cavity: the outline Y [-36.2, 12], Z [134, 182] with the corner at (Y -36.2, Z 134) rounded with radius 3, on the
    plane X -21, extruded 46 toward +X, so it runs through both ends. It leaves the floor, X [-20, 24], Y [-46, 11],
    Z [128, 134], and the fan-side wall, X [-20, 24], Y [-46, -36.2], Z [128, 181]. Y -36.2 is 0.2 from the pinion's
    back face.
  - Bearing block: a rounded square 22 wide (X [-11, 11]) and 53 tall (Z [128, 181]), corners of radius 3, from
    1 inside the wall to Y -60, with its end face rounded with radius 2. Its bore and the wall's make a bearing
    23.8 long for the fan shaft.
  - A bore of diameter 6.4 on the fan axis through the wall and the block, two holes of diameter 2.4 through the floor
    at X -7 and X 11, Y -28 for the grip screws, and five pilot holes of diameter 1.8, 9 deep, in the tray's right end
    face (X 24), at (Y, Z) = (-30, 131), (0, 131), (-41.1, 145), (-41.1, 160) and (-41.1, 175), for the gear case's
    screws.
- `gear-case`: the gearbox's crank-side wall and the housing of the spur gears, one block X [24.2, 42] with rounded
  edges: a plate X [24.2, 34] (inner face 0.2 from the driving gear's back face and from the tray's end) and a rim
  around a chamber X [34, 42], open at the top and on the +X side. The outline is Y [-46, 46], Z [56.4, 181], corners of
  radius 3, with the inner face's edge rounded with radius 1.5. The chamber is Y [-38.2, 38.2], Z [61.4, 182]. It
  has a bore of diameter 8.4 for the drive shaft, a pocket of diameter 8.4 for the crank shaft's end (X [28, 34]),
  five holes of diameter 2.3 through the plate with counterbores of diameter 4.3 (X [31.7, 35]) for the five case
  screws, and four pilot holes of diameter 1.8 (X [30.5, 42]) in the rim at (Y, Z) = (-42.1, 70), (-42.1, 120),
  (42.1, 70) and (42.1, 120) for the cover's screws. Printed with its plate down, the plate and the rim need no support.
- `gear-cover`: a plate X [42.2, 49.2], Y [-46, 46], Z [56.4, 181] with a square bearing block 22 wide around the crank
  axis, X [49.2, 57.2], 1 inside the plate, with its end face rounded with radius 2. It has a bore of diameter 8.4 for
  the crank shaft through the plate and the block (a bearing 15 long), a pocket of diameter 8.4, 5 deep, for the drive
  shaft's end, and four holes with counterbores for the cover screws. Printed with its inner face down, it needs no
  support.
- `case-bolt-1` to `case-bolt-5` and `cover-bolt-1` to `cover-bolt-4`, group `hardware`: nine M2 x 16 self-tapping
  screws. Each head, 2 high, sits in its counterbore 0.1 clear of the floor and the shank, radius 0.8, runs 16 toward
  -X into its pilot hole. The case screws hold the case to the tray; the cover screws hold the cover to the rim.
- `handle-bolt-1`, `handle-bolt-2` as above.

### Crank parts, motion `crank`

The crank parts turn about the crank axis, which lies at Y 0, Z 99.6, along X.

- `crank-gear`: a spur gear of Module 2, 35 teeth, 6 thick, bore 8, on the crank axis at X [35, 41]. Its roots are
  sharp, because Decad cannot fillet them ("loft fillets consume section segment").
- `crank-shaft`: a round shaft of radius 4 on the crank axis, X [28.2, 67], from the plate's pocket through the crank
  gear, the cover and its block. It has no collar, so the cover can slide over it, and the crank gear between the
  plate and the cover holds it. A flat 3 below the axis (toward -Z) at X [60.7, 67] makes the D-shaped stub, with a nut
  slot 4.3 wide and 1.9 deep at X [63.4, 65.3], open at the flat, and a screw hole of diameter 2.4 along the axis from
  X 58.2 to the end. The stub is cut from the round shaft, because a union of a round shaft and a D stub of the same
  radius shares a cylinder, which Decad refuses.
- `crank-arm`: a stadium 20 wide, X [61.2, 69.2], with the axis and the pin 55 apart (end radius 10, so the arm
  is 75 long), both faces rounded with radius 1.5, and a hole of diameter 2.4 at each end. A D pocket (radius
  4.15, flat 3.15 toward -Z) at X [60.2, 67.2] opens on the inner face for the shaft stub, and one (flat toward +Z)
  at X [63.2, 70.2] opens on the outer face for the pin stub. Each pocket leaves a 2 mm floor with the screw hole.
- `crank-pin`: a D-shaped stub (radius 4, flat 3 toward +Z, away from the shaft) at X [63.4, 69.7] with a nut
  slot at X [65.2, 67.1] and a screw hole of diameter 2.4 from X 62.2, a round shank of radius 4.5 from X 69.3 (the
  shoulder, 0.1 clear of the arm) to X 116.2, and a head of radius 5.8 at X [115.8, 118.8]. The pin is 55 from the
  crank axis. The handle slides on over the stub, so nothing on the pin is wider than the handle's bore except
  the head.
- `crank-grip`: the crank handle, a small revolved barrel X [69.4, 119.4], 50 long, for the thumb, index and middle
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

### Drive parts, motion `drive`

The drive parts turn about the drive shaft's axis, along X through the apex, the other way round from the crank.

- `driving-gear`: the driving bevel gear.
- `drive-gear`: a spur gear like the crank gear, on the drive axis at X [35, 41], turned half a tooth about its axis so
  that its tooth meets the crank gear's gap on the line between the two axes. The centre distance is 70.4, so the
  teeth have 0.4 of backlash, and the two gears' solids do not intersect.
- `drive-shaft`: a round shaft of radius 4 on the drive axis, X [14, 47], through the bevel gear, the plate, the drive
  gear and into the cover's pocket.

### Fan parts, motion `fan`

- `pinion`: the pinion bevel gear.
- `fan-shaft`: a cylinder of radius 3 on the fan axis, Y [-90, -22].
- `fan-hub`, group `rotor`: a disc of radius 14 on the fan axis, Y [-86, -66], mid plane Y -76. The hub's back face is 6 in front of the bearing block's end (Y -60). The crank is
  lower than the fan axis, so the blades stay clear of its reach without a long shaft. It has a bore
  of radius 3 and, for each blade k at angle `2 pi (k-1) / Blades` about the fan axis (the first along +X, turning
  toward +Z):
  - a radial socket of radius 3.15 on the hub's mid plane, from r 5.6 outward through the hub surface;
  - a bolt hole of radius 1.15 parallel to the fan axis at r 9.5, through the hub;
  - a head counterbore of radius 2.15 in the front face, 2.2 deep (floor at Y -83.8);
  - a hex nut trap of 4.3 across flats in the back face, 4.0 deep (floor at Y -70), with its corners on the
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
are `Manufactured` in PLA, and the screws and nuts (`handle-bolt-*`, `case-bolt-*`, `cover-bolt-*`, `crank-arm-bolt`,
`crank-pin-bolt`, `fan-bolt-*`) are `Purchased` steel, grouped by `Spec` for the bill of materials: M2 x 16
self-tapping screws for the grip, M2 x 16 self-tapping screws for the gear case and its cover, M2 x 10 screws with
nuts for the crank and M2 x 16 screws with nuts for the blades. Every joint is a `PinInBore`, with its axis stated in
the posed coordinates, because the faceted bores cannot state it:

- the crank shaft in the plate and in the cover (turns), in the crank gear and in the arm, the arm screw in the arm and
  the shaft, the pin and the pin screw in the arm and in each other, and the handle on the pin (turns);
- the drive shaft in the plate and in the cover (turns), in the drive gear and in the driving gear;
- the fan shaft in the wall (turns), in the pinion and in the hub, each blade's stud in the hub, and each blade bolt
  in the hub and in its blade;
- each grip screw in the floor and in the grip, each case screw in the plate and in the gearbox, and each cover screw
  in the cover and in the case.

`Assembly.Explode(ctx, set)` runs `Set.Explode` with the gearbox as the ground. The pin screw's joint declares a pull
toward the shaft, because its head is on the arm's inner face. Decad plans the explosion from boxes, and a chain of
nested parts along one axis makes it jump parts over each other, so the stage order is the result of a search over the
orders. It leaves no two parts sharing space at the default pose and puts the arm, the handle and the pin in that order.
The stages run in this order, all along +X unless noted:

1. the five case screws, then the case;
2. the drive shaft, the crank shaft, the drive gear and the crank gear;
3. the four cover screws, then the cover;
4. the arm, the handle, the pin and the arm screw: the handle slides over the pin's stub end, so along the crank axis
   the arm, the handle and the pin follow in that order with gaps between them; the pin screw goes toward -X;
5. the driving gear, which slides off to the left along its own axis;
6. the pinion, which lifts out of the open top;
7. the two grip screws, which lift out of the open top, and then the grip, which drops.

The fan shaft, the hub, the blades and the blade bolts come off along the fan axis in the last stage.

## Pose

Build every part at `CrankAngle = 0`. Then turn each `crank` part about the line along +X through `State.Crank` by
`CrankAngle`, each `drive` part about the line through the apex along -X by `CrankAngle`, and each `fan` part about
the line through the apex along -Y by `Ratio*CrankAngle`. The gearbox, the gear case, the cover, the handle and the
screws that hold them do not move.

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
  the gearbox, the gear case and its cover, and the crank's shaft, arm, pin, handle and screws, with the case and cover
  screws. That proves each stud fits its socket and each bolt its holes and pockets. The bevel pair, the spur pair and
  the fan shaft are left out because they touch.
- A hand on the crank handle cannot meet a blade. Every vertex of the arm, the pin and the handle is turned through a
  full turn about the crank axis in steps of 3°, and none comes within 15 of the blades' disc, taken as the Y range of
  the blades and the largest distance of a blade vertex from the fan axis.
- The bolts lie inside the hub's front and back faces. Blades 2 to N and bolts 2 to N are copies of the first,
  turned by `2 pi (k-1) / Blades`.
- The exploded view, checked at the default pose, puts the arm, the handle and the pin in that order along the crank
  axis, moves the gear case, its cover, the gears and the shafts, slides the driving gear to the left, lifts the
  pinion, and leaves no two parts sharing space. The bill of materials counts 2 + 2 + 5 + 4 screws and one for each
  blade.
- The crank and fan parts turn by the angles above.
- The bevel gears mesh at the default pose: the driving gear's back face lies at X = 24 and the pinion's at Y = -36.
- No two parts overlap except the meshing gear pairs, which touch at their teeth.
