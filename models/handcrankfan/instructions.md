# Hand-crank fan

## Purpose

Build a handheld fan. The operator holds the grip in the left hand and turns the crank on the fan's right side with
the right hand. The crank shaft sits 40 below the fan's axis, so the crank handle swings clear of the blades. A pair
of equal spur gears inside the gearbox carries the crank's turn up to the drive shaft, a straight-tooth bevel gear pair
turns the fan shaft at twice that speed, and the rotor blows air toward the operator. The rotor is the three-blade
turbine fan of `models/turbinefan`: its blades are shaped like small wind turbine blades and bolt to the hub one by one,
so a damaged blade can be swapped, and its hub grips a flat on the fan shaft with one set screw. The whole fan is meant to be
3D-printed; the only bought parts are M2 bolts and nuts and seven 608-2RS ball bearings. The right-hand crank and the
operator-facing rotor are the two orientation rules of this model.

The bevel pair is the `models/bevelgear` construction, the spur gears are the `models/spurgear` construction and the
hub and blades are the `models/turbinefan` construction (`turbinefan.BuildParts` with `Params608ZZ`), imported and not
copied. The bolts are `models/m2bolt` bodies and the nuts `models/m2nut` bodies.

## Compliance with AGENTS.md

The model follows the Fasteners, Part clearances, Gear train clearances and Corners and roots rules, with two open
exceptions.

- Fits: every pair of separate printed parts keeps a gap of its class, and the test proves the lower bound of each row
  of the table under Fits.
- Fasteners: every head sinks into a pocket and ends flush, and every gearbox, grip and crank nut sits in a cavity its
  part encloses. Those nuts go in during a print pause (see Fasteners). The rotor's nuts sit in slots open to a face,
  and its set screw is the one named exception (see Named exceptions).
- Spur pair: the gears sit at the sum of their pitch radii, 40 apart, with a deeper root and a tooth backlash. The test
  proves the tip-to-root clearance at five angles across one tooth pitch.
- Corners and roots: every printed part except the driving gear's tooth roots and the gearbox's ear roots is checked
  in its print orientation, and the press-fit walls are at least 1.6 mm (see Print orientation and Retained sharp
  corners). The pinion's teeth are checked from the toe to the heel; their roots carry a fillet.
- Open: the bevel mesh has no proven clearance (see Open non-compliance: the bevel mesh).
- Open: the gearbox's ear roots and the driving gear's tooth roots are not yet rounded (see Retained sharp corners).
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

Package `handcrankfan`. Expose `Params{CrankAngle, BladePitch float64}`.
Defaults: `5*pi/4` and `25*pi/180`.

- `CrankAngle` is any finite angle in radians. `Ratio*CrankAngle` must also be finite.
- `BladePitch` lies in `[10°, 40°]`, in radians. It is the blade angle at radius 48 (75 % of the tip radius 64),
  measured from the plane of rotation. It turns the whole blade about its radial line, as a pitch bearing does.
The rotor always has `turbinefan.BladeCount` (3) blades. `Validate` also returns turbinefan's own error for the rotor's
parameters.

Reject an invalid input before the document changes.

The blade, its socket tool and the hub and blade-joint constants belong to [`models/turbinefan`](../turbinefan/), which
this model imports: `turbinefan.BuildBlade`, `BladeSocketTool`, `Beta` and the `Blade...` constants.

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
the fan is `turbinefan.RightHand`, whose blades push air forward when the shaft turns clockwise seen from its front,
which is the operator's side. The blade is turbinefan's `BuildBlade`: at radius r its chord, from trailing edge to
leading edge, runs along `(0, sin beta(r), -cos beta(r))` in this model's frame, with `beta(r) = turbinefan.Beta(r,
BladePitch)`. The leading edge therefore leads the motion and sits upstream (+Y), and the concave pressure face leads the
motion and faces -Y, so it pushes air toward the operator. A blade at the other angles is the same blade turned about the
fan axis. The blade's shape, station table and formulas are in [`models/turbinefan`](../turbinefan/instructions.md).

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

A gear and the crank arm carry the torque on a spline: three grooves 2 wide, from radius 2.9 outward,
cut into the shaft along its length at 90°, 210° and 330° (measured from the shaft's first axis toward its second:
Y toward Z for the crank and drive shafts, X toward Z for the fan shaft), and a spline bore in the part, a round bore of
radius 4.05 with three teeth 1.9 wide that reach in to radius 2.95. Each tooth sits 0.05 off its groove's floor and
walls, a press fit, because the part turns with its shaft. The bore of a bevel gear is larger by the bevel gears' held
mesh bound of 0.1: radius 4.15, with teeth 1.7 wide that reach in to radius 3.05. That puts six faces under load, and the shaft still slides through every
bore, because the round shaft's grooves pass the teeth. The grooves run only where a bore has to slide over them: from
X -10 to the shaft's end on the crank shaft and the drive shaft, and on the fan shaft from Y -25 to -32.2 for the pinion
only: the fan hub sits on a D flat instead (see Fan parts). The shaft is round inside every bearing except where a groove has to pass
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
- `fan-shaft`: a round shaft of radius 4.0 on the fan axis, Y [-95, -26], with the pinion's three grooves from Y -25 to
  -32.2, a D flat and a collar of radius 6. It slides in from the front through the two bearings and into the pinion, and
  stops with the collar against the front bearing. Its tip is clear of the drive gear's teeth, and its front end is 1.0
  inside the hub's front face, past the bore's 0.5 chamfer, so 19 mm of it lies in the hub (turbinefan asks for 6 or more).
  - The D flat removes everything with X below -(4.0 - 0.75) = -3.25 from Y -74 to Y -96, where 0.75 is turbinefan's
    `Params608ZZ().FlatDepth`, read and not copied. The flat faces -X at crank angle 0 and turns with the fan. The collar
    refills Y -74 to -75.95, so the flat runs from Y -75.95 to its end.
  - The collar is Y [-75.95, -65.8], 10.15 long: 0.25 (Tight) from the front bearing's face (row T5) and 0.05 from the hub's
    back face, the press-fit gap that Arbor collars asks of a collar that stops a pressed part.
- `fan-hub`, group `rotor`: turbinefan's hub for the 608ZZ shaft (`turbinefan.Params608ZZ()`, with the pitch, the right hand
  and the placement below), a disc of radius 14 on the fan axis, Y [-96, -76], mid plane Y -86. The hub's back face is
  10.3 in front of the bearing block's end (Y -65.7). The crank is lower than the fan axis, so the blades stay clear of
  its reach without a long shaft. It has a D bore (a 0.05 press fit on the round and on the flat), three sockets of radius
  3.4, three blade bolt sites with a head pocket in the front face and a hex nut slot open to the round face, and the set
  screw's counterbore, hole and nut slot. A guard test requires the parameters to equal `Params608ZZ()` apart from the
  pitch, the hand and the placement.
- `fan-blade-1` .. `fan-blade-3`, group `blade`: turbinefan's blades. The first lies along +X at fan angle 0, the others
  at 120° and 240° toward +Z. Each blade's stud engages its socket from r 6.1 to r 14 with a 0.4 clearance (a Regular fit).
- `fan-bolt-1` .. `fan-bolt-3` and `fan-nut-1` .. `fan-nut-3`, group `hardware`: one M2 x 16 bolt and one nut for each
  blade, at turbinefan's blade sites. The bolt runs from the front face through the hub, the blade's stud and the hub
  again to the nut slot. Undoing the bolt from the front frees the blade; the nut stays in its slot.
- `fan-set-screw` and `fan-set-screw-nut`, group `hardware`: one M2 x 6 bolt and its nut at turbinefan's set screw site,
  drawn backed off (see Named exceptions). The screw's head floats in a counterbore in the hub's round face, 60° from
  blades 2 and 3, which is where a 1.5 mm key reaches it once the crank turns it to the side.

`fanLocalFrame` maps turbinefan's local frame onto the fan at crank angle 0: its origin to the fan axis in the hub's back
face, its +Z (forward) to -Y, its +Y (blade 1) to +X and its +X to -Z. The set screw head is then at X -11.2, Y -80
(its Dir is +X), so it points up at the default pose, and the bore's flat faces -X. `fanPlacement` is the same frame moved
to the hub's front face, Y -96, which is where turbinefan's `Placement` takes it.

The hub is 3D-printed with its front face down and needs no print pause. The sockets and the slots are sized for a printed
fit and may need a test print first.

Assembly order: the shaft goes in from the front through both bearings into the pinion, with the collar against the front
bearing. Drop the set screw's nut and the screw into the hub (two turns, the tip out of the bore), fit the blades, press
the hub onto the shaft flat to flat with the set screw toward the flat until it stops on the collar, and tighten the set screw.

### Arbor collars

[Arbor collars](../../docs/model-pipeline.md#arbor-collars) applies to the fan shaft, which carries the hub. The hub is
pressed on the shaft end outside the frame, and its flush side is the front bearing's outer face (Y -65.55).

| Collar | Shaft | Carried part | Flush side | Y | Len | Dia | Option |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Fan collar | `fan-shaft` | `fan-hub`, pressed on the D flat | `bearing-fan-block`'s face, Y -65.55 | [-75.95, -65.8] | 10.15 | 12 | 1: the 10.45 gap less 0.25 at the bearing and 0.05 at the hub |

The collar stops the pressed hub, so it takes the press-fit gap to the hub's back face (0.05), not the Tight 0.25 it had
before. The crank and drive shaft collars and the pinion's side of the fan shaft are older and are not listed or checked
here.

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
| T6c | `fan-shaft` and `fan-hub` | D bore on the round and the flat, collar face and hub back face | Press, proven at least 0.02 | 0.040 |
| S2 | `fan-set-screw` and `fan-shaft` | set screw tip to the shaft's flat | None: a proven gap above 0 | 0.047 |
| T6d | `drive-shaft` and `driving-gear`, `fan-shaft` and `pinion` | bevel spline | Press, proven at least 0.02 | 0.043 |
| T7 | `crank-pin` and `crank-arm` | stub in the pocket, stub end, shoulder | Tight | 0.247 |
| T8 | `drive-gear` and `driving-gear` | two gears on one shaft | Tight | 12.9 |
| R1 | each shaft and the pieces it crosses | shaft through the housing hole | Regular | 0.385 |
| R2 | `driving-gear` and the crank wall, `pinion` and the fan wall | gear back face to the wall | Regular | 0.424 |
| R3 | `driving-gear` and `gearbox-base` | gear to the back wall, at the heel | Regular | 4.75 |
| R4 | `bearing-drive-wall` and `driving-gear`, `bearing-fan-wall` and `pinion` | bearing face to the gear back | Regular | 0.427 |
| R5 | each `fan-blade-k` and `fan-hub` | stud in the socket, stud end, root to the rim | Regular | 0.377 |
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
and the arm's axial gap stay at 0.25; T5 measures the collar faces. The hub stops 0.05 from the collar, and the axial
row of T6c measures that gap with the round and the flat.

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
  nut and it cannot fall out. The nut's corners lie on the direction named below. The rotor is the exception: its nuts
  sit in hex slots open to a face, cut by turbinefan, so no nut is embedded and the hub needs no pause. The bolt holds
  each nut there, and the slot's flats stop it turning.
- Every pocket and cavity keeps 1 mm or more of material round it, and under the head pocket.
- A bolt tip ends inside a hole beyond its nut, never outside a face.

| Bolts | Nuts | Size | Head in | Nut in | Nut corners on |
| --- | --- | --- | --- | --- | --- |
| `fan-bolt-1` .. `3` | `fan-nut-1` .. `3` | M2 x 16 | `fan-hub` front face | `fan-hub`, a slot open to the round face | the radial line |
| `fan-set-screw` | `fan-set-screw-nut` | M2 x 6 | `fan-hub` counterbore in the round face | `fan-hub`, a slot open to the back face | Z at crank angle 0 (posed with the fan) |
| `gearbox-bolt-1` .. `4` | `gearbox-nut-1` .. `4` | M2 x 10 | `gearbox-fan-wall` ear | `gearbox-post` (1, 3), `gearbox-crank-wall` (2, 4) | Z |
| `gearbox-screw-1`, `2` | `gearbox-screw-nut-1`, `2` | M2 x 16 | `gearbox-post` outer face | `gearbox-base` back wall | Z |
| `gearbox-screw-3`, `4` | `gearbox-screw-nut-3`, `4` | M2 x 16 | `gearbox-crank-wall` outer face | `gearbox-base` back wall | Z |
| `handle-bolt-1`, `2` | `handle-nut-1`, `2` | M2 x 10 | `gearbox-base` floor, top face | `handle` | Y |
| `crank-pin-bolt` | `crank-pin-nut` | M2 x 10 | `crank-arm` inner face | `crank-pin` stub | Y |

A fan has 15 bolts and 15 nuts. Seven of the bolts are M2 x 10, seven are M2 x 16 (four for the walls and three for the
blades) and one is the M2 x 6 set screw. The parts list holds each nut right after its bolt. The bolt lengths come from
`m2bolt.StandardLengths()`.

#### Print pause

Each gearbox, grip and crank nut cavity is enclosed, so its piece pauses during the print. Pause when the printed layers reach the cavity's
roof, drop the nut in, and resume. Each cavity has a flat floor and a flat roof in the orientation below, so the nut
lies flat and the printer bridges the roof.

| Piece | Prints with | Nuts it holds |
| --- | --- | --- |
| `gearbox-post` | inner face down, X vertical | `gearbox-nut-1`, `gearbox-nut-3` |
| `gearbox-crank-wall` | the face that meets the base down, X vertical | `gearbox-nut-2`, `gearbox-nut-4` |
| `gearbox-base` | back wall down, Y vertical | `gearbox-screw-nut-1` .. `4` |
| `handle` | upright, the base down | `handle-nut-1`, `handle-nut-2` |
| `crank-pin` | upright, the head down | `crank-pin-nut` |

The tests check that each head ends flush, that each nut sits in its cavity with the part enclosing it, that the bolt
tip stays inside the part, and that no bolt or nut overlaps a part it passes through. For the rotor's open slots the
tests check the blade bolts' flush heads and the hub's material round their pockets, the head seat, that no bolt or nut
overlaps or is undecided against the hub, and the four section gaps (head in its pocket, shank in its hole, nut in its
slot, shank past the nut), each 0.05 or more. The slots' geometry, the set screw's head and tip, and the hub's material
are proven by turbinefan's `TestTurbineFanSetScrew`, `TestTurbineFanBladeBolts` and `TestTurbineFanAccess` for the 608ZZ
case.

### Print orientation

Corners and roots reads each part in its print orientation. The first six pieces are in the table above.

| Part | Prints with | Print axis in the posed assembly |
| --- | --- | --- |
| `gearbox-post` | inner face down | X |
| `gearbox-crank-wall` | the face that meets the base down | X |
| `gearbox-fan-wall` | the face that meets the base down, block up | Y |
| `gearbox-base` | back wall down | Y |
| `handle` (grip) | upright, base down | Z |
| `fan-hub` | front face down | Y |
| `crank-pin` | upright, head down | X |
| `crank-arm` | either face down | X |
| `crank-gear`, `drive-gear` | either face down | X |
| `driving-gear`, `pinion` | back face down | X and Y |
| `crank-shaft`, `drive-shaft` | standing on the collar end | X |
| `fan-shaft` | standing on its front tip, so the collar's hub-side face is the overhang and its bearing-side face stays clean | Y |
| `crank-grip` (handle barrel) | standing on its inner end | X |
| `fan-blade-k` | standing on the stud end, radial line vertical | the blade's radial line |

Every separate printed part pressed on a shaft prints with its axis vertical: the arm, the four gears and the hub.

### Retained sharp corners

The corner test (`TestHandCrankFanCorners`) cuts each part at the middle of each run of layers and requires no finding
outside this list. The shaft's spline and the mating corners carry the torque.

| Part | Corner | Reason |
| --- | --- | --- |
| Crank Gear, Drive Gear, Crank Arm, Driving Gear | the six corners of the spline bore, where its three teeth meet the round bore (turn about 108°), annulus r 2.9 to 4.2 about the shaft axis | mating corners: the teeth sit in the shaft's grooves at a 0.05 press gap, and the grooves stay sharp too |
| Crank Shaft, Drive Shaft, Fan Shaft | the groove floor corners (90°), annulus r 2.8 to 4.1 about the shaft axis | mating corners with the spline teeth |
| Crank Arm | the six corners of the pin's hexagonal pocket (turn 60°) | mating corners with the pin's hexagonal stub, 0.25 Tight gap on every flat |
| Fan Blade | the four corners where the bolt hole crosses the stud, at r 9.5 and 10.2 | the material angle is 64° to 90°, which meets the rule; the checker adds the stud's curvature and reads 125° to 130° |
| Gearbox Post, Gearbox Crank-side Wall, Gearbox Base | the two flat ends of each nut cavity whose axis lies in the layer plane (90°) | mating corners with the nut: a 0.5 round would hit the nut's corners |
| Crank Shaft, Drive Shaft, Fan Shaft; Crank Pin; Crank Handle | the collar steps; the stub root and the head step; the counterbore floor | corners between layers, out of the checker's sections (as in the escapement's arbors) |
| Driving Gear | the tooth roots: 56 corners per section (about 83°, radius 22.6 to 26.7 about the shaft, x 13.64 to 16.14) | the driving gear's flank is an involute that starts below the root, and a fillet tangent to it needs a sketch constraint and a decad join that do not exist yet (see `models/bevelgear/instructions.md`). The corner test does not read them: the one height `MidHeights` finds on it (x 17.5) cuts the hub behind the teeth. `TestBevelToothRootCorners` records them. They stay until `bevelgear` rounds involute roots |
| Gearbox Post, Gearbox Crank-side Wall, Gearbox Fan-side Wall | the ear roots: the ends of each ear against the floor's underside and the wall top (90°), and the fan block's corners against the wall's edge round (57.5°) | pending decad: a side root in the tray's end round is refused (the tangent chain cannot be decided at the junction of the end round and the wall's outer round, SX2), and so are several ears that share one floor face (a vertex of the wall lies on an end face's plane) |

The fan hub is turbinefan's, so its corners and its retained sharp corners are `models/turbinefan`'s: the corner test of
`TestHandCrankFanCorners` leaves it out, because `TestTurbineFanCorners` reads the same hub for its 608ZZ bore case, and
`TestFanHubIsTurbinePreset` proves this model builds that preset. The fan shaft's D flat meets the round at a turn of
35.7 degrees, under the checker's limit, so it needs no row.

The pinion's roots are rounded and are not in the list. Its teeth carry a root fillet of radius 0.76 mm (0.38 Module)
that ends in a 10° crease where its run-out crosses the root cone (see `models/bevelgear/instructions.md`, Root
fillet). The corner test cuts the pinion at 8 stations from the toe to the heel (`internal/rootcheck`) and requires no
finding there but the spline bore's six corners, which the bore row above exempts. It also requires that each section
cuts every tooth twice and that every root turns as a fillet, so an empty section or a sharp root cannot pass. The
join's mesh bound is 0.041 mm, bored or not (the corner check needs 0.05 mm or less).

The model stays on `CORNER_PENDING` in `scripts/check_repository.py` (and in `CORNER_PARTIAL`, which says its corner
test runs) until the ear-root row and the driving gear's tooth-root row are gone. What remains is the gearbox's ear
roots (three gearbox parts) and the driving gear's 56 root corners per section. The test lists the ear-root corners by
name in `earRootCorners`. The driving gear's hub and spline bore are checked, with the held mesh under the checker's
0.05 mm after the spline cut (0.028 mm). Its teeth are not.

Rounds below half their section: none in this model (the hub's socket floor round is turbinefan's).

The post's funnel seat for the fan shaft ends 0.05 past the fan wall's face, so it no longer cuts a scallop into
the post's corner. The blade's stud has a 0.3 chamfer at its inner end.

### Named exceptions

- The set screw is drawn 0.1 short of tight: `fan-set-screw` sits 0.1 back along its own line from turbinefan's site, with
  its nut unmoved. turbinefan places the tip against a shaft pushed 0.05 toward the screw, which the bearing-centred
  shaft is not, so at turbinefan's site the tip would overlap the modeled shaft by 0.05. Drawn backed off, the tip is level
  with the bore's flat (3.30) and 0.05 off the shaft's flat (3.25), and no part overlaps. Tightening the screw closes that
  gap. Row S2 proves it.

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
seven `M2 x 16 socket head cap screw` (four for the walls and one for each blade), one `M2 x 6 socket head cap screw` (the
set screw), 15 `M2 hex nut` and seven `608-2RS ball bearing`. The build has 56 parts: 45 fixed ones (including
`fan-shaft`, `fan-hub` and `pinion`), `fan-blade-1` .. `3`, `fan-bolt-1` .. `3`, `fan-nut-1` .. `3`, `fan-set-screw` and
`fan-set-screw-nut`. Every
joint is a `PinInBore`, with its axis stated in the posed coordinates, because the faceted bores cannot state it:

- each shaft in each of its bearings (turns) and each bearing in the gearbox;
- the crank shaft in the crank gear and in the arm, the pin and the pin bolt in the arm, and the handle on the pin (turns);
- the drive shaft in the drive gear and in the driving gear;
- the fan shaft in the pinion and in the hub, each blade's stud in the hub, each blade bolt in the hub and in its blade,
  and the set screw in the hub;
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
7. the blade bolts, the blades, the hub with its nuts and the set screw (which stays in its nut in the hub) and the fan
   shaft, which come off along the fan axis (-Y);
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
  larger Y than the trailing edge, and the chord angle `atan2(dY, dZ)` lies within 0.5° of `pi - beta(r)`
  (`turbinefan.Beta`). The blade's thickness ratio is checked in `models/turbinefan`.
- The fan's rotor is turbinefan's 608ZZ preset: `TestFanHubIsTurbinePreset` requires the parameters to equal
  `turbinefan.Params608ZZ()` apart from the pitch, the hand and the placement, and `TestTurbineFanCorners` reads that hub.
- The hub is empty at a socket, at a head pocket and at a nut slot. No two of the finished parts overlap or are
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
- Blades 2 and 3, bolts 2 and 3 and nuts 2 and 3 match the first, turned by `2 pi (k-1) / 3`, within 0.05.
- The exploded view, checked at the default pose, puts the arm, the handle and the pin in that order along the crank
  axis, slides the shafts and the pinion to the left, lifts the three gears, moves each joined nut with its piece, keeps
  the base's nuts in place, and leaves no two parts sharing space. The set screw leaves with the hub. The bill of materials
  counts 15 bolts, 15 nuts and seven bearings.
- The crank and fan parts turn by the angles above.
- The bevel gears mesh at the default pose: the driving gear's back face lies at X = 21 and the pinion's at Y = -31.5.
- No two parts overlap except the meshing gear pairs, which touch at their teeth.
- The `fasteners` subtest proves, for each of the 15 sites, that the bolt's head ends flush with its part's
  face, that the part is solid beside the pocket and keeps 1 mm round the pocket and the cavity, that the nut sits
  centred in its cavity with the part closed round it, that the bolt tip is held by the part, and that no bolt or nut
  overlaps a part it passes through. Each gap between a fastener and its pocket, hole or cavity has a proven lower bound
  of 0.05 or more.
- The `clearances` subtest proves every row of the fit table (see Fits). The `mesh` subtest proves the spur pair (see
  Spur pair). Both run for the default parameters only: `BladePitch` turns each blade about its radial line and `CrankAngle` turns each
  part about its own axis, and neither changes a fit against the housing or within one motion group.
- The corner test checks every printed part in its print orientation, as Retained sharp corners lists, and requires each
  listed corner to match a finding. `TestPressFitWalls` requires at least 1.6 mm of wall round the bore of the hub
  (through its socket floors, 1.65), the arm and the two spur gears, from the section outline.
- Sketch coverage: the fan's own sketches, the bearing sketches and every `m2bolt` and `m2nut` sketch are in
  `Assembly.Sketches`, which the test verifies for each case. `spurgear` and `bevelgear` verify each of their sketches
  inside `Build` and `BuildPair`, so a build that passes covers them. `BladePitch` and `CrankAngle` change no sketch.
