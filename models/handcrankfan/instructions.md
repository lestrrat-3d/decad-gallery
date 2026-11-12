# Hand-crank fan

## Purpose

Build a handheld fan. The operator holds the grip in the left hand and turns the crank on the fan's right side with
the right hand. The crank shaft sits 40 below the fan's axis, so the crank handle swings clear of the blades. A pair
of equal spur gears inside the gearbox carries the crank's turn up to the drive shaft, a straight-tooth bevel gear pair
turns the fan shaft at twice that speed, and the rotor blows air toward the operator. The blades are shaped like small
wind turbine blades and bolt to the hub one by one, so a damaged blade can be swapped. The whole fan is meant to be
3D-printed; the only bought parts are M2 bolts and nuts and seven 608-2RS ball bearings. The right-hand crank and the
operator-facing rotor are the two orientation rules of this model.

The bevel pair is the `models/bevelgear` construction and the spur gears are the `models/spurgear` construction,
imported and not copied. The bolts are `models/m2bolt` bodies and the nuts `models/m2nut` bodies.

## Compliance with AGENTS.md

The model follows the Fasteners, Part clearances and Gear train clearances rules of `AGENTS.md`, with one open
exception.

- Fits: every pair of separate printed parts keeps a gap of its class, and the test proves the lower bound of each row
  of the table under Fits.
- Fasteners: every head sinks into a pocket and ends flush, and every nut sits in a cavity its part encloses. The nuts
  go in during a print pause (see Fasteners).
- Spur pair: the gears sit at the sum of their pitch radii, 40 apart, with a deeper root and a tooth backlash. The test
  proves the tip-to-root clearance at five angles across one tooth pitch.
- Open: the bevel mesh has no proven clearance (see Open non-compliance: the bevel mesh).
- `models/README.md` marks the model **Mixed**: its custom fits carry the class clearances, and the bolts, nuts and
  bearings keep their nominal sizes.

## Body containers

`Build` ends by `moveBodies` (`containers.go`), after every cut, because Decad cuts only within one container. The
top-level containers are `Housing` (the four gearbox parts), `Grip` (the handle), `Crank` (arm, pin and crank grip),
`Hub`, `Blades`, `Shafts`, `Bearings`, `Bolts`, `Nuts`, `Spur Pair` (crank gear and drive gear) and `Pair`, the bevel pair that
`bevelgear.BuildPair` made, with `Driving Gear` and `Pinion Gear` inside. The bevel bodies go back into those
containers because Decad cannot remove the emptied ones. The test checks each body's
container and `doc.VerifyContainers().Err()`.

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
`NewDocument` returns `decad.New` with `decad.WithChainTolerance(0.1 mm)` applied before the caller's options.
`Build` needs a document made by it: the joints between the hub and the lofted blades, and the cuts in the bevel gears,
need that tolerance.

`State` holds `CrankAngle`, `FanAngle`, `Apex` (r3.Vec), `Crank` (r3.Vec) and `ArmDirection` (r3.Vec). `Ratio` is the constant 2.
`Assembly` holds `Parts []Part`, `Sketches []*sketch.Sketch` and `State`.
`Part` holds `Name`, `Group`, `Motion` and `Body`. `Motion` is `fixed`, `crank`, `drive` or `fan`.

## Orientation rules

- The crank is on the right. The crank shaft lies on a line along X at Y 0, Z 130, 40 below the drive shaft, and
  leaves the gearbox toward +X. It ends beyond the crank bearing block at X 54.75, and the crank pin and grip have
  larger X than every fixed part.
- The fan faces the operator. The fan shaft lies on the line through the apex along Y and runs toward -Y. The
  rotor has smaller Y than every other part except the fan shaft and the grip's raked base, and its blades blow air
  toward -Y.
- The crank handle never meets a blade. The arm, the pin and the handle reach 65 from the crank axis. Turned through a
  full turn about the crank axis, none of their vertices comes within 15 of the disc the blades sweep.

## Crank turn and fan turn

The crank shaft turns 40 below the drive shaft, and a pair of equal spur gears joins them. The pair turns the drive
shaft the opposite way at the same speed, and the bevel pair turns the fan shaft at twice that speed.

A forward crank turn is clockwise as seen from the right, a rotation about the -X axis. `CrankAngle` is
the forward turn, measured from the arm pointing straight up: the arm points along
`(0, sin CrankAngle, cos CrankAngle)`. The default is 5 pi / 4, the arm pointing down and toward the operator. The
drive shaft and the bevel gear on it turn about +X by `CrankAngle`, the other way round.

The bevel pair has a 90° shaft angle. With the drive shaft's outward direction +X and the fan's outward direction -Y,
the two angular velocities satisfy ω_fan = Ratio · ω_drive along -Y. The forward crank turn therefore turns the
fan about -Y by `-Ratio * CrankAngle`: clockwise as the operator sees it. The spur pair has reversed the turn, so
the blades are mirrored. The text below describes the unmirrored blade, which moves toward +Z at +X while the fan
turns about -Y. The built blade has every section's X coordinate negated and its pitch turn reversed, so it moves toward
-Z at +X while the fan turns forward and pushes air the same way. At radius r the unmirrored chord, from trailing edge
to leading edge, runs along `(0, sin beta(r), cos beta(r))` and the built blade's along `(0, sin beta(r), -cos beta(r))`,
with `beta(r) = alphaD + atan(tan(p0 - alphaD) * 48 / max(r, 21.5)) + (BladePitch - p0)`, `alphaD = 5°` and
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

The apex is the point where the drive shaft's axis meets the fan axis: `(0, 0, 170)`. The crank axis is 40 below it.

### Bevel pair

`bevelgear.BuildPair` with Module 2, 28 driving teeth, 14 pinion teeth, 90° shaft angle, straight teeth and
default pressure angle and samples. Both gears are built without a bore, and each gets a spline bore afterwards (see
Shafts, bearings and splines). Decad refuses those cuts in a document whose chain tolerance is below the bevel gears'
mesh bound of about 0.09 mm, so the document's chain tolerance is 0.1. The placement frame has origin
`(21, 0, 170)`, U = -Y and V = +Z, so the driving gear's back face lies in the plane X = 21 and grows toward -X.
The pinion's back face lies in the plane Y = -31.5. The mesh of the two gears has no proven clearance (see Open
non-compliance: the bevel mesh).

The heel distances 21 and 31.5 follow from the pair: the driving heel is `Module*14/2 + Module*28/8` and the
pinion heel is `Module*28/2 + Module*14/8`.

### Fixed parts

- Handle, `handle`: an oval grip held in the left hand, a smooth loft through 14 oval sections, raked 15° like a
  pistol grip with its base toward the operator (-Y). It spans Z [-18.75, 101.15], so its top is 0.25 below the
  gearbox floor (Z 101.4, a Tight fit), and the centre of its top is at X 7, Y -23.5. The table's `z` below is the
  height the loft is drawn at; the finished grip is 26.75 lower. Section `z`, width `W` (along X) and length `L` (along Y), in
  mm, centred on `X 0, Y (z - 68) tan 15°` before the loft is placed:

  | z | 8 | 9 | 11 | 14 | 18 | 26 | 45 | 65 | 85 | 100 | 114 | 122 | 126 | 127.9 |
  | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
  | W | 28 | 33 | 37 | 36 | 32 | 31 | 32 | 33.5 | 34 | 33 | 31 | 31.5 | 33 | 35 |
  | L | 38 | 43 | 47 | 46 | 41 | 39 | 40.5 | 42 | 42.5 | 41 | 38 | 36.5 | 35.5 | 35.5 |

  The base flares to stop the little finger sliding off, the palm swell is at Z 85, and the neck under the
  gearbox is the narrowest part. The grip is 35 to 38 mm across when measured round, inside the 30 to 50 mm range
  for a power grip, and the hand zone is about 108 mm long. There are no finger grooves, which make pressure
  points for other hand sizes. Each section is a fit spline through 25 points that leaves a 2° gap at the back,
  closed by two short lines, as the blade sections are. Two bolt sites (see Fasteners) at X -2 and X 16 on Y -23.5
  cut a hole of radius 1.3 down from the top of the grip to Z 93.4 and, in the grip, a nut cavity.
- `handle-bolt-1`, `handle-bolt-2` and `handle-nut-1`, `handle-nut-2`, group `hardware`: two M2 x 10 bolts, one at each
  site, each with its nut. The head sinks into the gearbox floor and ends flush with the floor's top face (Z 106.4),
  and the nut sits in the cavity in the grip.
- Gearbox, four parts: a tray with rounded edges, a back wall, a post across its whole left end, open at the top, three
  bearing blocks and four ears, cut at the walls. Its floor sits low enough for the crank gear to turn above it, so
  the spur pair lives in the tray and the grip hangs from the lowered floor. A bearing sits wherever a shaft crosses a
  wall (see Shafts, bearings and splines), so it has no boss or sleeve. The solid is built whole, then copied and cut
  into `gearbox-base`, `gearbox-post`, `gearbox-crank-wall` and `gearbox-fan-wall`, each 0.125 short of its seams, so two
  pieces are 0.25 apart (a Tight fit):
  - `gearbox-fan-wall` keeps Y <= -32.025: the fan-side wall with the fan block, which holds the fan shaft;
  - `gearbox-post` keeps X <= -10.325 and Y >= -31.775: the post, with the back wall's left end;
  - `gearbox-crank-wall` keeps X >= 21.525 and Y >= -31.775: the crank-side wall with the crank block;
  - `gearbox-base` keeps X in [-10.075, 21.275] and Y >= -31.775: the floor and the back wall between the others.

  Every bearing seat is a vertical hole when the piece that holds it prints. The post prints on its inner face, with its
  seats opening upward. The crank-side wall and the fan-side wall print on the face that meets the base, with their
  blocks pointing up, so their blocks' seats open upward and their walls' seats open onto the bed, where a 45° cone is
  the ceiling. The base prints on its back wall.
  - Block: the outline Y [-41.7, 39.2], Z [101.4, 190] with four corners of radius 3, on the plane X -20, extruded
    51.2 toward +X (to X 31.2), with both end faces rounded with radius 2. The floor is 5 thick, with its underside at
    Z 101.4, so the crank gear's tip (Z 108.02) stays 1.62 above the cavity floor (Z 106.4).
  - Cavity: the outline Y [-31.9, 29.4], Z [106.4, 191] with its floor-back corner rounded with radius 3, on the
    plane X -21, extruded 42.4 toward +X, less a notch (Y [-32.9, 30.4]) at its left end up to X -10.2. Its other
    corners stay sharp, because a fillet at a seam would leave a thin sliver in the piece beside it. It leaves the
    floor, the fan-side wall (X [-20, 31.2], Y [-41.7, -31.9]), the crank-side wall (X [21.4, 31.2], Y [-41.7, 39.2]),
    the back wall (X [-20, 21.4], Y [29.4, 39.2]) and the post (X [-20, -10.2], Y [-41.7, 39.2]), all Z [101.4, 190]
    except the floor. Y -31.9 is 0.4 from the pinion's back face, X 21.4 is 0.4 from the driving gear's back face, and
    Y 29.4 is 0.5 beyond the driving gear's reach (Y 28.9).
  - Fan bearing block: a rounded square 30 wide (X [-15, 15]) and as tall as the tray (Z [101.4, 190]), corners of radius
    3, from 1 inside the fan-side wall to Y -65.7, with its end face rounded with radius 2.
  - Crank bearing block: the same square (Y [-15, 15], Z [101.4, 190]) from 1 inside the crank-side wall to X 45, 13.8
    long beyond the wall, with its end face rounded with radius 2.
  - Ears: four boxes 16 long along Y and 8 high that stick out 8 from the tray, each straddling the fan-side wall's seam
    (Y -31.9): at X [-19, -11] and X [25, 32.2] under the floor (Z [93.4, 102.4], the outer one clear of the grip and 1 mm wider
    than the others, so the round nut cavity keeps 1 mm of material beside it) and at
    X [-19, -11] and X [22, 30] on the top of the walls (Z [189, 198]). The fan block leaves no room for a bolt head on
    the wall's outer face, so these seams need ears. Each ear carries one bolt site (see Fasteners) along Y: the head
    on the fan-side wall's half and the nut cavity in the post's or the crank-side wall's half.
  - Wall bolts: the post and the crank-side wall need no ears. Each has two bolt sites (see Fasteners) along X in its
    outer face (X -20, and X 31.2), at Y 34.3 (the middle of the back wall) and Z 112 and 185. The head sits in the
    wall, and the nut cavity in the end of the base's back wall.
  - Seats: a bearing seat is a pocket of radius 11.05 and 7.4 deep (a 0.05 press fit around the bearing's radius 11), with a
    relief behind its floor, so the floor holds only the bearing's outer ring. There is one in the post's outer face
    (X -20) on each of the drive axis (Z 170) and the crank axis (Z 130), one in the crank-side wall's inner face
    (X 21.4) on each of them, one in the crank block's end (X 45) on the crank axis, and one in the fan-side wall's inner
    face (Y -31.9) and one in the fan block's end (Y -65.7) on the fan axis. The crank axis and the fan axis each have a
    hole of radius 4.4 between their two seats (a Regular fit around the shaft). The drive shaft's seat in the crank-side wall is blind. Every seat,
    relief and hole is round. The relief is a cylinder of radius 9.5 and 2.1 deep for the post's seats and the blocks'
    seats, and a 45° cone from radius 9.5 down to radius 4.4 for the seats in the walls' inner faces, which open onto the
    bed: their ledge for the bearing is 1.55 wide.
  - Bolt sites in the floor: the two grip bolts (see Fasteners) cut a head pocket and a hole into the floor at X -2 and
    X 16, Y -23.5.

### Shafts, bearings and splines

Every shaft is 8 across (radius 4.0): the 8 mm bore of a 608-2RS bearing (outside diameter 22, 7 wide), with no
clearance. The bearings are the `models/ballbearing` envelope at its nominal size, group `bearing`, one
purchased part each, fixed. Each seat is a press fit, 0.05 larger than the bearing's radius, and each bearing sits 0.25
off its seat's floor and 0.15 inside the seat's mouth:

| Part | Shaft | Position along the shaft |
| --- | --- | --- |
| `bearing-crank-post`, `bearing-drive-post` | crank, drive | X [-19.85, -12.85] |
| `bearing-crank-wall`, `bearing-drive-wall` | crank, drive | X [21.55, 28.55] |
| `bearing-crank-block` | crank | X [37.85, 44.85] |
| `bearing-fan-wall` | fan | Y [-39.05, -32.05] |
| `bearing-fan-block` | fan | Y [-65.55, -58.55] |

The bearings' envelopes carry no fit allowance, as `docs/models/ballbearing.md` says. The fit row S1 measures the
printed seat, row Z1 the shaft in the bearing's bore, and row T5 the collar to the bearing face.

A gear, the crank arm and the fan hub carry the torque on a spline: three grooves 2 wide, from radius 2.9 outward,
cut into the shaft along its length at 90°, 210° and 330° (measured from the shaft's first axis toward its second:
Y toward Z for the crank and drive shafts, X toward Z for the fan shaft), and a spline bore in the part, a round bore of
radius 4.05 with three teeth 1.9 wide that reach in to radius 2.95. Each tooth sits 0.05 off its groove's floor and
walls, a press fit, because the part turns with its shaft. The bore of a bevel gear is larger by the bevel gears' held
mesh bound of 0.1: radius 4.15, with teeth 1.7 wide that reach in to radius 3.05. That puts six faces under load, and the shaft still slides through every
bore, because the round shaft's grooves pass the teeth. The grooves run only where a bore has to slide over them: from
X -10 to the shaft's end on the crank shaft and the drive shaft, and on the fan shaft from Y -25 to -32.2 for the pinion
and from Y -75 to its end for the hub. The shaft is round inside every bearing except where a groove has to pass
through a bearing's bore, which leaves the bearing resting on three arcs of 90°. The spline bores are cut after the
gears are built: the spur gears are built without a bore, and the bevel gears with `EnableBore` off, which moves them
to the document's root container for the cut. The drive gear's bore is cut after its half-tooth turn, so that its
teeth line up with the drive shaft's grooves.

### Crank parts, motion `crank`

The crank parts turn about the crank axis, which lies at Y 0, Z 130, along X.

- `crank-gear`: a spur gear of Module 2, 20 teeth, 6 thick, with a spline bore, on the crank axis at X [-9.2, -3.2],
  1.125 from the post's piece. See Spur pair for its tip, root and backlash. It keeps the spur gear's root fillet.
- `crank-shaft`: a round shaft of radius 4.0 on the crank axis, X [-24.1, 54.75], with a collar of radius 6 at its left
  end (X [-24.1, -20.1], 0.25 clear of the post bearing's face). It slides in from the left through the post's bearing,
  the crank gear, the crank-side wall's bearing, the crank block's bearing and the arm, and the collar on the post
  bearing and the arm on the block bearing hold it with about 4 mm of play. The splines' grooves run from X -10 to
  X 56, and the arm sits on them. The grooves are cut from the bare shaft before the collar joins it, because Decad
  refuses to cut after the union.
- `crank-arm`: a stadium 20 wide, X [49, 57], with the axis and the pin 55 apart (end radius 10, so the arm
  is 75 long), both faces rounded with radius 1.5, and a hole of diameter 2.6 at the pin end. A spline bore at X [48, 55]
  opens on the inner face for the shaft, and a hexagonal pocket (7.43 across flats) at X [52, 58] opens on the outer
  face for the pin stub. The bore is a 0.05 press fit on the shaft and its floor stands 0.25 from the shaft's end, and the
  pocket leaves 0.25 on every flat of the stub. The printed bore grips the shaft, so the arm is pressed on
  and the splines carry the torque. The pin's bolt site (see Fasteners) sinks the bolt head into the inner face.
- `crank-pin`: a hexagonal stub (8 across corners, 6.93 across flats) at X [52.25, 57.55], a round shank of radius 4.5
  from X 57.25 (the shoulder, 0.25 clear of the arm) to X 104.4, and a head of radius 5.8 at X [104.0, 107.0]. The pin
  is 55 from the crank axis. The handle slides on over the stub, so nothing on the pin is wider than the handle's bore
  except the head. The pin's bolt site cuts a hole of radius 1.3 along the pin axis and a nut cavity in the stub.
- `crank-grip`: the crank handle, a small revolved barrel X [57.4, 107.4], 50 long, for the thumb, index and middle
  fingers. Its outer diameter is 14.4 at the inner end, 15.6 at 8 from it, 16 at 22, 14.5 at 36 (the waist), 16 at
  43, 18.6 at 48.8 (the flare) and 16.6 at the outer end. This is inside MIL-STD-1472G's light-load handle (10 to 16
  mm thick, 25 to 75 mm long). It has a bore of diameter 9.8 (0.4 clearance on the shank, a Regular fit, so it spins
  freely) and a counterbore of diameter 12.8, 3.8 deep, at the outer end for the pin head. The handle starts 0.4 from
  the arm. It is made of cones and cylinders, with chamfers of 0.5 to 1.5 on its corners: Decad cannot decide a handle
  with rounded corners against its pin.
- `crank-pin-bolt` and `crank-pin-nut`, group `hardware`: an M2 x 10 bolt and its nut. The head sinks into the arm's
  inner face and the bolt runs up the pin axis through the pin stub; the nut sits in a cavity in the stub. Together
  with the hexagonal stub it holds the pin to the arm, so nothing is loose, and the handle spins on the pin, held by
  the pin head. The shaft to the arm needs no bolt: it is a tight fit on the splines.

Every pair of separate parts keeps a gap of its class (see Fits). Decad can decide such a pair, which it cannot when
the faces touch. A bolt head on its pocket floor is the one stated contact (see Fasteners).

### Drive parts, motion `drive`

The drive parts turn about the drive shaft's axis, along X through the apex, the other way round from the crank.

- `driving-gear`: the driving bevel gear, with a spline bore.
- `drive-gear`: a spur gear like the crank gear, with a spline bore, on the drive axis at X [-9.2, -3.2], turned half a
  tooth about its axis so that its tooth meets the crank gear's gap on the line between the two axes. See Spur pair
  for the mesh. Its tip radius of 21.98 leaves 9.92 to the fan-side wall's inner face in Y.
- `drive-shaft`: a round shaft of radius 4.0 on the drive axis, X [-24.1, 28.5], with a collar of radius 6 at its left
  end like the crank shaft's. It slides in from the left through the post's bearing, the drive gear and the bevel gear,
  and ends in the crank-side wall's bearing, 0.05 short of the bearing's far face. The grooves run from X -10 to
  X 29.5.

### Spur pair

The crank gear and the drive gear are `spurgear` gears of Module 2 and 20 teeth, 6 thick, with `Backlash` 0.15 and no
bore. Their axes are 40 apart, half the sum of their pitch diameters. The gear train rule asks for a tip-to-root
clearance of at least max(0.25 m, 0.5 mm) = 0.5 mm, so the variant `spurRootClearance` sets the root radius to
17.48 (pitch radius 20 less the module 2, less the 0.5 clearance, less a 0.02 margin) and the tip radius to 21.98 (pitch
radius plus the module, less the margin). Each tip clears the other gear's root by 40 - 21.98 - 17.48 = 0.54. The
0.15 backlash thins each tooth at the pitch circle and gives the pair about 0.3 of play. It does not count toward the
tip-to-root clearance. The test turns the pair through one tooth pitch in four steps (five samples) and proves the
clearance, from the gears' own section outlines, at each sample.

### Fan parts, motion `fan`

- `pinion`: the pinion bevel gear, with a spline bore.
- `fan-shaft`: a round shaft of radius 4.0 on the fan axis, Y [-100, -26], with a collar of radius 6 at
  Y [-75.75, -65.8], between the front bearing and the hub (0.25 from the bearing's face and 0.25 from the hub). It
  slides in from the front through the two bearings and into the pinion, and stops with the collar against the front
  bearing. The hub slides on from the front over its grooves. Its tip is clear of the drive gear's teeth.
- `fan-hub`, group `rotor`: a disc of radius 14 on the fan axis, Y [-96, -76], mid plane Y -86. The hub's back face is
  10.3 in front of the bearing block's end (Y -65.7). The crank is lower than the fan axis, so the blades stay clear of
  its reach without a long shaft. It has a spline bore (a 0.05 press fit) and, for each blade k at angle
  `2 pi (k-1) / Blades` about the fan axis (the first along +X, turning toward +Z):
  - a radial socket of radius 3.4 on the hub's mid plane, from r 5.6 outward through the hub surface, a round hole,
    6.8 across, small enough to print on its side when the hub prints front face down;
  - a bolt site (see Fasteners) parallel to the fan axis at r 9.5: a hole of radius 1.3 through the hub, a head pocket
    in the front face and a nut cavity near the back face.
- `fan-blade-1` .. `fan-blade-N`, group `blade`: one blade each, with N = `Blades`. A blade is a smooth loft
  through the ten sections of the table below, stacked on its radial line through the hub's mid plane. The loft
  starts at r 14.4, 0.4 outside the hub rim. A root stud of radius 3, from r 6 to r 16 on the radial line, is joined
  to it, with a bolt hole of radius 1.3 parallel to the fan axis at r 9.5. The stud engages the socket from r 6
  to r 14 with a 0.4 clearance (a Regular fit). The first blade lies along +X at fan angle 0.

  | r | 14.4 | 16.3 | 19 | 21.5 | 25 | 30 | 38 | 46 | 55 | 64 |
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
- `fan-bolt-1` .. `fan-bolt-N` and `fan-nut-1` .. `fan-nut-N`, group `hardware`: one M2 x 16 bolt and one nut for each
  blade, at its bolt site (see Fasteners). The bolt runs from the front face through the hub, the blade's stud and the
  hub again to the nut cavity. Undoing the bolt from the front frees the blade; the nut stays in its cavity.

The hub is 3D-printed with its front face down. The sockets and the cavities are sized for a printed fit and may need
a test print first.

### Fits

Each row is one distinct fit of separate parts. The class sets the minimum surface-to-surface clearance of the Part
clearances rule: Tight 0.2, Regular 0.35, Loose 0.5. Each built gap is the class minimum plus a 0.05 margin, so the
proven lower bound clears the class, and a fit that involves a bevel gear adds the 0.1 held mesh bound of the bevel
bodies. `requireClearances` in `clearance_test.go` measures each row on the placed parts at the default fan and
requires its proven lower bound to meet the class. The last column is the smallest lower bound of the row at the last
full run, in mm. It is a dated illustration, and the test owns the number.

| ID | Parts | Fit | Class | Lower bound |
| --- | --- | --- | --- | --- |
| T1 | each pair of gearbox pieces | the seams | Tight | 0.217 |
| T2 | `handle` and `gearbox-base` | grip top under the floor | Tight | 0.214 |
| T3 | `handle` and the fan wall, the post, the crank wall | grip beside the lower ears | Tight | 0.552 |
| S1 | each `bearing-*` and the piece that holds it | outside diameter of the seat | Press, proven at least 0.015 | 0.0173 |
| Z1 | each shaft and each of its bearings | shaft in the bore, no clearance | Zero | 0 within the section bound |
| T5 | the post bearings, the front fan bearing and the shaft collars | collar to the bearing face | Tight | 0.250 |
| T6 | `crank-shaft` and `crank-gear`, `drive-shaft` and `drive-gear` | spline | Press, proven at least 0.02 | 0.045 |
| T6b | `crank-shaft` and `crank-arm` | spline | Press, proven at least 0.02 | 0.044 |
| T6c | `fan-shaft` and `fan-hub` | spline | Press, proven at least 0.02 | 0.044 |
| T6d | `drive-shaft` and `driving-gear`, `fan-shaft` and `pinion` | bevel spline | Press, proven at least 0.02 | 0.043 |
| T7 | `crank-pin` and `crank-arm` | stub in the pocket, stub end, shoulder | Tight | 0.247 |
| T8 | `drive-gear` and `driving-gear` | two gears on one shaft | Tight | 12.9 |
| R1 | each shaft and the pieces it crosses | shaft through the housing hole | Regular | 0.385 |
| R2 | `driving-gear` and the crank wall, `pinion` and the fan wall | gear back face to the wall | Regular | 0.424 |
| R3 | `driving-gear` and `gearbox-base` | gear to the back wall, at the heel | Regular | 4.75 |
| R4 | `bearing-drive-wall` and `driving-gear`, `bearing-fan-wall` and `pinion` | bearing face to the gear back | Regular | 0.427 |
| R5 | each `fan-blade-k` and `fan-hub` | stud in the socket, stud end, root to the rim | Regular | 0.379 |
| R6 | `crank-grip` and `crank-pin` | handle bore, counterbore and head face | Regular | 0.390 |
| R7 | `crank-arm` and `crank-grip` | handle end to the arm | Regular | 0.395 |
| L1 | `crank-gear` and `drive-gear` with `gearbox-post` | gear to the post | Loose | 1.08 |
| L2 | `crank-gear` and `drive-gear` with `gearbox-base`, `drive-gear` with `gearbox-fan-wall` | gear to floor and walls | Loose | 0.50 or more at all 5 mesh angles |
| L3 | `crank-gear` with `driving-gear`, and both spur gears with `pinion` | gears that do not mesh | Loose | 12.9 axial, 0.50 or more at all 5 mesh angles |
| L4 | `crank-arm` and `gearbox-crank-wall` | arm to the block end | Loose | 3.97 |
| L5 | `fan-hub` and `gearbox-fan-wall` | hub to the fan block | Loose | 10.2 |

A press fit is 0.05 on the radius, for a bearing seat and for a bore that turns with its shaft. The 0.2 Tight minimum does
not apply to it, and the test requires only that the section proves a positive gap, because the proof subtracts the
mesh bound. A shaft in a purchased bearing's bore has no clearance by rule (row Z1). Decad's contact report calls that
pair undecided, so the overlap test skips the shaft and bearing pairs and relies on Z1. The collar to the bearing face
and the arm and hub axial gaps stay at 0.25; T5 measures the collar faces, and the others are not measured separately
from the spline rows.

Rows R2, R4, R3 and T6d involve a bevel gear, whose section is taken at 0.11 and whose bound includes its mesh bound.
Rows that depend on the crank angle (L2 and part of L3) are sampled at five angles across one tooth pitch, as the spur
mesh is (see Spur pair).

Two interfaces carry no class. A bolt and its own nut are two nominal parts, a bolt shank in a nut bore: this is a
nominal exception, and neither part is changed or measured. A bolt head rests on the floor of its pocket, which is the
one stated zero-clearance interface, because the Fasteners rule asks for a pocket as deep as the head is tall. The
test proves the head seat with a contact query and requires its gap to be at most 0.05.

### Fasteners

Every bolt is a `models/m2bolt` body and every nut a `models/m2nut` body at nominal size. Each bolt site is a
head pocket, a clearance hole and a nut cavity cut into the parts, and the clearances around a head, a shank and a nut
are the Fasteners rule's (`internal/m2fastener`), not a fit class.

- Head pocket: radius 2.2 (head radius 1.9 plus 0.3) and 2.0 deep, as deep as the head is tall, so the
  head's top face lies flush with the part's face. Nothing stands proud.
- Clearance hole: radius 1.3 (2.6 across), above the 2.4 mm that an M2 shank needs. It runs through the part and
  past the bolt's tip.
- Nut cavity: a circle of radius 2.51 (through the nut's corners, radius 2.31, plus 0.2) and 2.0 high (nut 1.6 plus
  0.2 each side), so the nut floats 0.2 from both faces. The part encloses it on every side except the bolt hole, so the part keeps the
  nut and it cannot fall out. The nut's corners lie on the direction named below.
- Every pocket and cavity keeps 1 mm or more of material round it, and under the head pocket.
- A bolt tip ends inside a hole beyond its nut, never outside a face.

| Bolts | Nuts | Size | Head in | Nut in | Nut corners on |
| --- | --- | --- | --- | --- | --- |
| `fan-bolt-1` .. `N` | `fan-nut-1` .. `N` | M2 x 16 | `fan-hub` front face | `fan-hub` | the radial line |
| `gearbox-bolt-1` .. `4` | `gearbox-nut-1` .. `4` | M2 x 10 | `gearbox-fan-wall` ear | `gearbox-post` (1, 3), `gearbox-crank-wall` (2, 4) | Z |
| `gearbox-screw-1`, `2` | `gearbox-screw-nut-1`, `2` | M2 x 16 | `gearbox-post` outer face | `gearbox-base` back wall | Z |
| `gearbox-screw-3`, `4` | `gearbox-screw-nut-3`, `4` | M2 x 16 | `gearbox-crank-wall` outer face | `gearbox-base` back wall | Z |
| `handle-bolt-1`, `2` | `handle-nut-1`, `2` | M2 x 10 | `gearbox-base` floor, top face | `handle` | Y |
| `crank-pin-bolt` | `crank-pin-nut` | M2 x 10 | `crank-arm` inner face | `crank-pin` stub | Y |

A fan has 11 + `Blades` bolts and as many nuts: 15 and 15 for four blades. Seven of the bolts are M2 x 10 and
4 + `Blades` are M2 x 16. The parts list holds each nut right after its bolt. The bolt lengths come from
`m2bolt.StandardLengths()`.

#### Print pause

Each nut cavity is enclosed, so its piece pauses during the print. Pause when the printed layers reach the cavity's
roof, drop the nut in, and resume. Each cavity has a flat floor and a flat roof in the orientation below, so the nut
lies flat and the printer bridges the roof.

| Piece | Prints with | Nuts it holds |
| --- | --- | --- |
| `fan-hub` | front face down | one for each blade |
| `gearbox-post` | inner face down, X vertical | `gearbox-nut-1`, `gearbox-nut-3` |
| `gearbox-crank-wall` | the face that meets the base down, X vertical | `gearbox-nut-2`, `gearbox-nut-4` |
| `gearbox-base` | back wall down, Y vertical | `gearbox-screw-nut-1` .. `4` |
| `handle` | upright, the base down | `handle-nut-1`, `handle-nut-2` |
| `crank-pin` | upright, the head down | `crank-pin-nut` |

The tests check that each head ends flush, that each nut sits in its cavity with the part enclosing it, that the bolt
tip stays inside the part, and that no bolt or nut overlaps a part it passes through.

### Open non-compliance: the bevel mesh

The bevel pair (`driving-gear` and `pinion`) has no proven tooth clearance, so the Gear train clearances rule is not
met for it. `bevelgear` has no backlash parameter, and Decad cannot prove a gap between two faceted bodies: with
`ToothSpacing` 0 and 0.2, `doc.Verify` reports `unsupported_pair_contact` for the pair, and a mesh-gap query proves a
lower bound of 0 against mesh bounds of 0.087 and 0.091. The model builds the pair as `bevelgear` does today and
leaves it out of the fit table, the overlap check and the clearance tests. Closing it needs two changes: a `Backlash`
parameter in `models/bevelgear`, and Decad proof of gaps between faceted bodies. The fan's images and tests change with
the first.

## Parts set and exploded view

`Assembly.Set(doc)` returns a `parts.Set` (Decad's `parts` package) over the build's document. The printed parts
are `Manufactured` in PLA, and the bolts, the nuts and the bearings are `Purchased` steel, grouped by `Spec` for the
bill of materials: seven `M2 x 10 socket head cap screw` (two for the grip, four for the ears, one for the pin),
`4 + Blades` `M2 x 16 socket head cap screw` (four for the walls and one for each blade), `11 + Blades` `M2 hex nut`
and seven `608-2RS ball bearing`. The build has `45 + 3 x Blades` parts: 57 for four blades and 54 for three. Every
joint is a `PinInBore`, with its axis stated in the posed coordinates, because the faceted bores cannot state it:

- each shaft in each of its bearings (turns) and each bearing in the gearbox;
- the crank shaft in the crank gear and in the arm, the pin and the pin bolt in the arm, and the handle on the pin (turns);
- the drive shaft in the drive gear and in the driving gear;
- the fan shaft in the pinion and in the hub, each blade's stud in the hub, and each blade bolt in the hub and in its blade;
- each grip bolt in the floor and in the grip, and each ear bolt and each wall bolt in the two pieces it joins;
- each nut in the piece that holds it and on its bolt, except the nuts in `gearbox-base`, which have no joint: the base
  is the ground, and a captive nut does not leave its piece.

`Assembly.Explode(ctx, set)` runs `Set.Explode` with the gearbox's base as the ground. The pin bolt's joint declares a
pull toward the shaft, because its head is on the arm's inner face. A stage marked together carries the nuts held by its
part. Decad plans the explosion from boxes, which makes nested parts jump over each other, so the order and the
directions of the lifts are the result of a search that leaves no two parts sharing space at the default pose and keeps
the largest move shortest. The stages run in this order:

1. the four ear bolts (-Y), then the two wall bolts of the post (-X) and the two of the crank-side wall (+X);
2. the arm (+X), the handle (+X), the pin with its nut (+X): the handle slides over the pin's stub end, so along the
   crank axis the arm, the handle and the pin follow in that order with gaps between them; the pin bolt goes toward -X;
3. the drive shaft and the crank shaft, which slide out to the left through the post;
4. the pinion, which slides off to the left;
5. the two grip bolts, the crank gear, the drive gear and the driving gear, which lift out of the open top in that order;
6. the bearings: the crank block's goes toward +X, the two in the crank-side wall and the two in the post toward -X,
   the fan wall's toward +Y and the fan block's toward -Y;
7. the blade bolts, the blades, the hub with its nuts and the fan shaft, which come off along the fan axis (-Y);
8. the post with its two nuts, which goes toward -X, the crank-side wall with its two nuts, which goes toward +X, and the
   fan-side wall, which goes toward -X;
9. the grip with its two nuts, which drops.

The explosion is checked at the default pose only: Decad's boxes depend on the crank angle.

## Pose

Build every part at `CrankAngle = 0`. Then turn each `crank` part about the line along -X through `State.Crank` by
`CrankAngle`, each `drive` part about the line through the apex along +X by `CrankAngle`, and each `fan` part about
the line through the apex along -Y by `-Ratio*CrankAngle`. The gearbox pieces, their bolts and nuts and the handle do not move.

## Verification

- `Build` rejects an invalid `Params` before the document changes.
- Every returned part is a native solid, every part name is unique, and the document holds no other body.
- Every sketch passes `Sketch.Verify` as fully constrained: DOF 0, no redundant constraint, no conflict.
- The crank is on the right and the fan faces the operator, by the bounds rules above.
- The grip hangs under the gearbox floor, is longer than 100, and no part reaches below its lower end. Its rings at
  Z -0.75, 58.25 and 87.25 (the table's 26, 85 and 114, 26.75 lower) have the widths and lengths of the table, and their
  centres follow the 15° rake.
- The blade pitch pushes air toward -Y. In the mesh of `fan-blade-1` with the pose undone, at radii near 30, 40,
  50 and 60, the chord runs from the midpoint of the trailing edge to the leading edge. The leading edge has a
  larger Y than the trailing edge, and the chord angle `atan2(dY, dZ)` lies within 0.5° of `beta(r)`.
- The blade is thick at the root. The extent of a ring across the chord over its extent along it lies within 0.03
  of the thickness ratio of the table at those radii, and is at least 0.9 near r 15.
- The hub is empty at a socket, at a head pocket and at a nut cavity. No two of the finished parts overlap or are
  undecided in `doc.Verify`. The finished parts are the hub, the blades, the bolts and nuts, the handle, the gearbox,
  the seven bearings, and the crank's shaft, arm, pin and handle, and the drive shaft. A shaft or a bearing is also
  checked against every other part, the gears included. That proves each stud fits its socket and each bolt its holes
  and pockets, that every bearing sits in its seat and on its shaft, and that every groove lines up with its tooth,
  because a shaft that met a gear's tooth would share material with it. The pairs the `fasteners` subtest proves (a bolt
  with its head host and with its own nut) and the meshing gears are left out. The meshing gears are checked only
  against the shafts and the bearings.
- A hand on the crank handle cannot meet a blade. Every vertex of the arm, the pin and the handle is turned through a
  full turn about the crank axis in steps of 3°, and none comes within 15 of the blades' disc, taken as the Y range of
  the blades and the largest distance of a blade vertex from the fan axis.
- Blades 2 to N, bolts 2 to N and nuts 2 to N are copies of the first, turned by `2 pi (k-1) / Blades`.
- The exploded view, checked at the default pose, puts the arm, the handle and the pin in that order along the crank
  axis, slides the shafts and the pinion to the left, lifts the three gears, moves each joined nut with its piece, keeps
  the base's nuts in place, and leaves no two parts sharing space. The bill of materials counts `11 + Blades` bolts,
  `11 + Blades` nuts and seven bearings.
- The crank and fan parts turn by the angles above.
- The bevel gears mesh at the default pose: the driving gear's back face lies at X = 21 and the pinion's at Y = -31.5.
- No two parts overlap except the meshing gear pairs, which touch at their teeth.
- The `fasteners` subtest proves, for each of the `11 + Blades` sites, that the bolt's head ends flush with its part's
  face, that the part is solid beside the pocket and keeps 1 mm round the pocket and the cavity, that the nut sits
  centred in its cavity with the part closed round it, that the bolt tip is held by the part, and that no bolt or nut
  overlaps a part it passes through. Each gap between a fastener and its pocket, hole or cavity has a proven lower bound
  of 0.05 or more.
- The `clearances` subtest proves every row of the fit table (see Fits). The `mesh` subtest proves the spur pair (see
  Spur pair). Both run for the default parameters only: `BladePitch` turns each blade about its radial line, `Blades`
  repeats identical sockets and `CrankAngle` turns each part about its own axis, and none of the three changes a fit
  against the housing or within one motion group.
- Sketch coverage: the fan's own sketches, the bearing sketches and every `m2bolt` and `m2nut` sketch are in
  `Assembly.Sketches`, which the test verifies for each case. `spurgear` and `bevelgear` verify each of their sketches
  inside `Build` and `BuildPair`, so a build that passes covers them. `Blades` changes the hub sketch, and the default
  and both variants run it. `BladePitch` and `CrankAngle` change no sketch.
