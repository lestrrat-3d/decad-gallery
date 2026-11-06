# Hand-crank fan

## Purpose

Build a handheld fan. The operator holds the grip in the left hand and turns the crank on the fan's right side with
the right hand. The crank shaft sits 40.4 below the fan's axis, so the crank handle swings clear of the blades. A pair
of equal spur gears inside the gearbox carries the crank's turn up to the drive shaft, a straight-tooth bevel gear pair turns the fan
shaft at twice that speed, and the rotor blows air toward the operator. The blades are shaped like small wind turbine blades and bolt to the hub one by one, so a damaged blade
can be swapped. The whole fan is meant to be 3D-printed; the only bought parts are M2 bolts and nuts and seven 608-2RS ball bearings. The right-hand crank and the operator-facing rotor are the two orientation rules of this model.

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

- The crank is on the right. The crank shaft lies on a line along X at Y 0, Z 129.6, 40.4 below the drive shaft, and
  leaves the gearbox toward +X. It ends beyond the crank bearing block at X 57.8, and the crank pin and grip have larger X than every
  fixed part.
- The fan faces the operator. The fan shaft lies on the line through the apex along Y and runs toward -Y. The
  rotor has smaller Y than every other part except the fan shaft and the grip's raked base, and its blades blow air
  toward -Y.
- The crank handle never meets a blade. The arm, the pin and the handle reach 65 from the crank axis. Turned through a
  full turn about the crank axis, none of their vertices comes within 15 of the disc the blades sweep.

## Crank turn and fan turn

The crank shaft turns 40.4 below the drive shaft, and a pair of equal spur gears joins them. The pair turns the drive
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

The apex is the point where the drive shaft's axis meets the fan axis: `(0, 0, 170)`. The crank axis is 40.4 below it.

### Bevel pair

`bevelgear.BuildPair` with Module 2, 28 driving teeth, 14 pinion teeth, 90° shaft angle, straight teeth and
default pressure angle and samples. Both gears are built without a bore, and each gets a spline bore afterwards (see
Shafts, bearings and splines). Decad refuses those cuts in a document whose chain tolerance is below the bevel gears'
mesh bound of about 0.09 mm, so the document's chain tolerance is 0.1. The placement frame has origin
`(21, 0, 170)`, U = -Y and V = +Z, so the driving gear's back face lies in the plane X = 21 and grows toward -X.
The pinion's back face lies in the plane Y = -31.5.

The heel distances 21 and 31.5 follow from the pair: the driving heel is `Module*14/2 + Module*28/8` and the
pinion heel is `Module*28/2 + Module*14/8`.

### Fixed parts

- Handle, `handle`: an oval grip held in the left hand, a smooth loft through 14 oval sections, raked 15° like a
  pistol grip with its base toward the operator (-Y). It spans Z [-18.6, 101.3], so its top is 0.1 below the
  gearbox floor (Z 101.4), and the centre of its top is at X 7, Y -23.5. The table's `z` below is the height the
  loft is drawn at; the finished grip is 26.6 lower. Section `z`, width `W` (along X) and length `L` (along Y), in
  mm, centred on `X 0, Y (z - 68) tan 15°` before the loft is placed:

  | z | 8 | 9 | 11 | 14 | 18 | 26 | 45 | 65 | 85 | 100 | 114 | 122 | 126 | 127.9 |
  | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
  | W | 28 | 33 | 37 | 36 | 32 | 31 | 32 | 33.5 | 34 | 33 | 31 | 31.5 | 33 | 35 |
  | L | 38 | 43 | 47 | 46 | 41 | 39 | 40.5 | 42 | 42.5 | 41 | 38 | 36.5 | 35.5 | 35.5 |

  The base flares to stop the little finger sliding off, the palm swell is at Z 85, and the neck under the
  gearbox is the narrowest part. The grip is 35 to 38 mm across when measured round, inside the 30 to 50 mm range
  for a power grip, and the hand zone is about 108 mm long. There are no finger grooves, which make pressure
  points for other hand sizes. Each section is a fit spline through 25 points that leaves a 2° gap at the back,
  closed by two short lines, as the blade sections are. Two pilot holes of diameter 1.8 run from the table's Z 116 (89.4 on the
  finished grip) up through the top, at X -2 and X 16 on Y -23.5.
- `handle-bolt-1`, `handle-bolt-2`, group `hardware`: two M2 x 16 screws, one at each pilot. Each stands on
  Y -23.5 with its head 2.0 high on the floor's top face (0.1 above it) and a shank of radius 0.8, which is the
  thread root, down into the pilot. They cut their own thread in the printed pilot. Each is one plain body.
- Gearbox, one part, `gearbox`: one tray with rounded edges, a back wall, open at the top, and three square blocks.
  Its floor sits low enough for the crank gear to turn above it, so the spur pair lives in the tray and the grip hangs
  from the lowered floor. A bearing sits wherever a shaft crosses a wall (see Shafts, bearings and splines), so it has
  no boss or sleeve. Printed floor down, every wall, the post and the blocks are vertical, so it needs no support.
  - Block: the outline Y [-41.5, 39], Z [101.4, 190] with four corners of radius 3, on the plane X -20, extruded
    51 toward +X (to X 31), with both end faces rounded with radius 2. The floor is 5 thick: its underside is 1.2 under
    the crank gear's tip (Z 107.6 minus 1.2, minus 5).
  - Cavity: the outline Y [-31.7, 29.2], Z [106.4, 187] with the two floor corners rounded with radius 3, on the
    plane X -21, extruded 42.2 toward +X, so it runs through the left end, less a notch (Y [-15, 30.2]) at its left end
    up to X -10.2. It leaves the floor, the fan-side wall (X [-20, 31], Y [-41.5, -31.7]), the crank-side wall (X [21.2, 31],
    Y [-41.5, 39]), the back wall (X [-20, 21.2], Y [29.2, 39]) and the left post (X [-20, -10.2], Y [-15, 29.2]), all
    Z [101.4, 190] except the floor. The post joins the back wall, so that wall is tied at both ends. Y -31.7 is 0.2 from
    the pinion's back face, X 21.2 is 0.2 from the driving gear's back face, and Y 29.2 is 0.3 beyond the driving gear's
    reach (Y 28.9).
  - Fan bearing block: a rounded square 30 wide (X [-15, 15]) and as tall as the tray (Z [101.4, 190]), corners of radius
    3, from 1 inside the fan-side wall to Y -65.5, with its end face rounded with radius 2.
  - Crank bearing block: the same square (Y [-15, 15], Z [101.4, 190]) from 1 inside the crank-side wall to X 45, 14
    long beyond the wall, with its end face rounded with radius 2.
  - Seats: a bearing seat is a pocket of radius 11.1 and 7.4 deep, with a relief bore of radius 9.5 and 2.1 deep
    behind its floor, so the floor holds only the bearing's outer ring. There is one in the post's outer face (X -20) on
    each of the drive axis (Z 170) and the crank axis (Z 129.6), one in the crank-side wall's inner face (X 21.2) on
    each of them, one in the crank block's end (X 45) on the crank axis, and one in the fan-side wall's inner face
    (Y -31.7) and one in the fan block's end (Y -65.5) on the fan axis. The crank axis and the fan axis each have a hole of
    radius 4.2 between their two seats. The drive shaft's seat in the crank-side wall is blind. Every seat, relief
    and hole is a teardrop: the round bore plus a pointed cap of two lines tangent to it at 45° from the vertical, with
    the point `r * sqrt(2)` above the axis, so the point of the drive axis's seats is at Z 185.7. The tray prints floor
    down, so these bores lie on their sides, and the 45° roof needs no support. A bearing sits in the seat's round
    part, held by the lower three quarters of its wall.
  - Two holes of diameter 2.4 through the floor at X -2 and X 16, Y -23.5, for the grip screws.

### Shafts, bearings and splines

Every shaft is 7.8 across: an 8 mm shaft with the model's usual 0.1 mm clearance, so it turns in a 608-2RS bearing
(bore 8, outside diameter 22, 7 wide). The bearings are the `models/ballbearing` envelope, group `bearing`, one purchased
part each, fixed, and each sits 0.1 off its seat's floor and 0.3 inside the seat's mouth:

| Part | Shaft | Position along the shaft |
| --- | --- | --- |
| `bearing-crank-post`, `bearing-drive-post` | crank, drive | X [-19.7, -12.7] |
| `bearing-crank-wall`, `bearing-drive-wall` | crank, drive | X [21.5, 28.5] |
| `bearing-crank-block` | crank | X [37.7, 44.7] |
| `bearing-fan-wall` | fan | Y [-39, -32] |
| `bearing-fan-block` | fan | Y [-65.2, -58.2] |

The bearings' envelopes carry no fit allowance, as `docs/models/ballbearing.md` says, so a real fit is the printer's to
tune.

A gear, the crank arm and the fan hub carry the torque on a spline: three grooves 2 wide, from radius 2.9 outward,
cut into the shaft along its length at 90°, 210° and 330° (measured from the shaft's first axis toward its second:
Y toward Z for the crank and drive shafts, X toward Z for the fan shaft), and a spline bore in the part, a round bore of
radius 4.0 with three teeth 1.8 wide that reach in to radius 3.0. Each tooth sits 0.1 off its groove's floor and walls.
That puts six faces under load, and the shaft still slides through every bore, because the round shaft's grooves pass
the teeth. The grooves run only where a bore has to slide over them: from X -10 to the shaft's end on the crank shaft
and the drive shaft, and on the fan shaft from Y -25 to -32 for the pinion and from Y -75 to its end for the hub. The
shaft is round inside every bearing except where a groove has to pass through a bearing's bore, which leaves the
bearing resting on three arcs of 90°. The spline bores are cut after the gears are built: the spur gears are built
without a bore, and the bevel gears with `EnableBore` off, which moves them to the document's root container for the cut.
The drive gear's bore is cut after its half-tooth turn, so that its teeth line up with the drive shaft's grooves.

### Crank parts, motion `crank`

The crank parts turn about the crank axis, which lies at Y 0, Z 129.6, along X.

- `crank-gear`: a spur gear of Module 2, 20 teeth, 6 thick, with a spline bore, on the crank axis at X [-9.2, -3.2], 1 clear of
  the post. Its tip radius is 22. Its roots are sharp, because Decad cannot fillet them ("loft fillets consume section
  segment").
- `crank-shaft`: a round shaft of radius 3.9 on the crank axis, X [-23.8, 54.8], with a collar of radius 6 at its left end
  (X [-23.8, -19.8], 0.1 clear of the post's bearing). It slides in from the left through the post's bearing, the crank
  gear, the crank-side wall's bearing, the crank block's bearing and the arm, and the collar on the post bearing
  and the arm on the block bearing hold it with about 4 mm of play. The splines' grooves run from X -10 to the end,
  and the arm sits on them. The grooves are cut from the bare shaft before the collar joins it, because Decad refuses
  to cut after the union.
- `crank-arm`: a stadium 20 wide, X [49, 57], with the axis and the pin 55 apart (end radius 10, so the arm
  is 75 long), both faces rounded with radius 1.5, and a hole of diameter 2.4 at the pin end. A spline bore at X [48, 55]
  opens on the inner face for the shaft, and a hexagonal pocket (7.13 across flats) at X [51, 58] opens on the outer
  face for the pin stub. The bores are 0.1 larger than the parts in them, the gap every joint of the model keeps before
  it is tightened; the printed bore is made to grip the shaft, so the arm is pressed on and the splines carry the
  torque. The pin pocket leaves a 2 mm floor with the screw
  hole.
- `crank-pin`: a hexagonal stub (8 across corners, 6.93 across flats) at X [51.2, 57.5] with a nut
  slot at X [53, 54.9] and a screw hole of diameter 2.4 from X 50, a round shank of radius 4.5 from X 57.1 (the
  shoulder, 0.1 clear of the arm) to X 104, and a head of radius 5.8 at X [103.6, 106.6]. The pin is 55 from the
  crank axis. The handle slides on over the stub, so nothing on the pin is wider than the handle's bore except
  the head.
- `crank-grip`: the crank handle, a small revolved barrel X [57.2, 107.2], 50 long, for the thumb, index and middle
  fingers. Its outer diameter is 14.4 at the inner end, 15.6 at 8 from it, 16 at 22, 14.5 at 36 (the waist), 16 at
  43, 18.6 at 48.8 (the flare) and 16.6 at the outer end. This is inside MIL-STD-1472G's light-load handle (10 to 16
  mm thick, 25 to 75 mm long). It has a bore of diameter 9.4 (0.4 clearance on the shank, so it spins freely and tilts
  at most 0.5°) and a counterbore of diameter 12.8, 3.8 deep, at the outer end for the pin head. It is made of cones
  and cylinders, with chamfers of 0.5 to 1.5 on its corners: Decad cannot decide a handle with rounded corners
  against its pin.
- `crank-pin-bolt`, group `hardware`: an M2 x 10 screw with a nut, one plain body. Its head is on the arm's inner face
  and it runs up the pin axis through the pin stub; the nut sits in the pin's slot. Together with the hexagonal stub it holds the
  pin to the arm, so nothing is loose, and the handle spins on the pin, held by the pin head. The shaft to the arm needs
  no screw: it is a tight fit on the splines.

Each joint just before it is tightened keeps a gap of 0.1 to 0.2 mm, because Decad cannot decide a pair of
parts whose faces touch.

### Drive parts, motion `drive`

The drive parts turn about the drive shaft's axis, along X through the apex, the other way round from the crank.

- `driving-gear`: the driving bevel gear, with a spline bore.
- `drive-gear`: a spur gear like the crank gear, with a spline bore, on the drive axis at X [-9.2, -3.2], turned half a tooth about its axis so
  that its tooth meets the crank gear's gap on the line between the two axes. The centre distance is 40.4, so the
  teeth have 0.4 of backlash, and the two gears' solids do not intersect. Its tip radius of 22 leaves 9.7 to the
  fan-side wall's inner face in Y.
- `drive-shaft`: a round shaft of radius 3.9 on the drive axis, X [-23.8, 28.3], with a collar of radius 6 at its left end
  like the crank shaft's. It slides in from the left through the post's bearing, the drive gear and the bevel gear, and
  ends in the crank-side wall's bearing. The grooves run from X -10 to the end.

### Fan parts, motion `fan`

- `pinion`: the pinion bevel gear, with a spline bore.
- `fan-shaft`: a round shaft of radius 3.9 on the fan axis, Y [-100, -26], with a collar of radius 6 at Y [-75.9, -65.3],
  between the front bearing and the hub. It slides in from the front through the two bearings and into the pinion,
  and stops with the collar against the front bearing. The hub slides on from the front over its grooves, 0.1 from
  the collar. Its tip is clear of the drive gear's teeth.
- `fan-hub`, group `rotor`: a disc of radius 14 on the fan axis, Y [-96, -76], mid plane Y -86. The hub's back face is 10.5 in front of the bearing block's end (Y -65.5). The crank is
  lower than the fan axis, so the blades stay clear of its reach without a long shaft. It has a spline bore
  and, for each blade k at angle `2 pi (k-1) / Blades` about the fan axis (the first along +X, turning
  toward +Z):
  - a radial socket of radius 3.15 on the hub's mid plane, from r 5.6 outward through the hub surface, a teardrop
    whose point is toward the back face (+Y), which is up when the hub prints front face down, so it needs no support;
  - a bolt hole of radius 1.15 parallel to the fan axis at r 9.5, through the hub;
  - a head counterbore of radius 2.15 in the front face, 2.2 deep (floor at Y -93.8);
  - a hex nut trap of 4.3 across flats in the back face, 4.0 deep (floor at Y -80), with its corners on the
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
are `Manufactured` in PLA, and the screws and nuts (`handle-bolt-*`, `crank-pin-bolt`, `fan-bolt-*`) and the
bearings are `Purchased` steel, grouped by `Spec` for the bill of materials: M2 x 16 self-tapping screws for the grip,
M2 x 10 screws with nuts for the crank, M2 x 16 screws with nuts for the blades and seven 608-2RS ball bearings. Every joint is a `PinInBore`, with its axis
stated in the posed coordinates, because the faceted bores cannot state it:

- each shaft in each of its bearings (turns) and each bearing in the gearbox;
- the crank shaft in the crank gear and in the arm, the pin and the pin screw in the arm and in each other, and the handle on the pin (turns);
- the drive shaft in the drive gear and in the driving gear;
- the fan shaft in the pinion and in the hub, each blade's stud in the hub, and each blade bolt
  in the hub and in its blade;
- each grip screw in the floor and in the grip.

`Assembly.Explode(ctx, set)` runs `Set.Explode` with the gearbox as the ground. The pin screw's joint declares a pull
toward the shaft, because its head is on the arm's inner face. Decad plans the explosion from boxes, which makes nested
parts jump over each other, so the order and the directions of the lifts are the result of a search that leaves no two
parts sharing space at the default pose and keeps the largest move shortest. The stages run in this order:

1. the arm (+X), the handle and the pin (+X): the handle slides over the pin's stub end, so along the
   crank axis the arm, the handle and the pin follow in that order with gaps between them; the pin screw goes toward -X;
2. the drive shaft and the crank shaft, which slide out to the left through the post;
3. the pinion, which slides off to the left;
4. the two grip screws, the crank gear, the drive gear and the driving gear, which lift out of the open top in that order;
5. the bearings: the crank block's goes toward +X, the two in the crank-side wall and the two in the post toward -X,
   the fan wall's toward +Y and the fan block's toward -Y;
6. the grip, which drops.

The fan shaft, the hub, the blades and the blade bolts come off along the fan axis in the last stage. The explosion is
checked at the default pose only: Decad's boxes depend on the crank angle.

## Pose

Build every part at `CrankAngle = 0`. Then turn each `crank` part about the line along -X through `State.Crank` by
`CrankAngle`, each `drive` part about the line through the apex along +X by `CrankAngle`, and each `fan` part about
the line through the apex along -Y by `-Ratio*CrankAngle`. The gearbox and the handle do not move.

## Verification

- `Build` rejects an invalid `Params` before the document changes.
- Every returned part is a native solid, every part name is unique, and the document holds no other body.
- Every sketch passes `Sketch.Verify` as fully constrained: DOF 0, no redundant constraint, no conflict.
- The crank is on the right and the fan faces the operator, by the bounds rules above.
- The grip hangs under the gearbox floor, is longer than 100, and no part reaches below its lower end. Its rings at
  Z -0.6, 58.4 and 87.4 (the table's 26, 85 and 114, 26.6 lower) have the widths and lengths of the table, and their
  centres follow the 15° rake.
- The blade pitch pushes air toward -Y. In the mesh of `fan-blade-1` with the pose undone, at radii near 30, 40,
  50 and 60, the chord runs from the midpoint of the trailing edge to the leading edge. The leading edge has a
  larger Y than the trailing edge, and the chord angle `atan2(dY, dZ)` lies within 0.5° of `beta(r)`.
- The blade is thick at the root. The extent of a ring across the chord over its extent along it lies within 0.03
  of the thickness ratio of the table at those radii, and is at least 0.9 near r 15.
- The hub is empty at a socket and at a nut trap. No two of the finished parts overlap or are undecided in
  `doc.Verify`. The finished parts are the hub, the blades and the bolts of the rotor, the handle and its screws,
  the gearbox, the seven bearings, and the crank's shaft, arm, pin, handle and screws, and the drive shaft. A shaft or a
  bearing is also checked against every other part, the gears included. That proves each stud fits its socket and each
  bolt its holes and pockets, that every bearing sits in its seat and on its shaft, and that every groove lines up with
  its tooth, because a shaft that met a gear's tooth would share material with it. The meshing gears are checked only
  against the shafts and the bearings.
- A hand on the crank handle cannot meet a blade. Every vertex of the arm, the pin and the handle is turned through a
  full turn about the crank axis in steps of 3°, and none comes within 15 of the blades' disc, taken as the Y range of
  the blades and the largest distance of a blade vertex from the fan axis.
- The bolts lie inside the hub's front and back faces. Blades 2 to N and bolts 2 to N are copies of the first,
  turned by `2 pi (k-1) / Blades`.
- The exploded view, checked at the default pose, puts the arm, the handle and the pin in that order along the crank
  axis, slides the shafts and the pinion to the left, lifts the three gears, and leaves no two parts sharing space. The
  bill of materials counts 2 + 1 screws and one for each blade, and seven bearings.
- The crank and fan parts turn by the angles above.
- The bevel gears mesh at the default pose: the driving gear's back face lies at X = 21 and the pinion's at Y = -31.5.
- No two parts overlap except the meshing gear pairs, which touch at their teeth.
