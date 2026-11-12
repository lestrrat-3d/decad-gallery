# Hand-crank fan steps

[instructions.md](instructions.md) owns the dimensions and the orientation rules. This file maps them to
Sketch and Decad operations. Every Sketch is solved and then required to be fully constrained
(`sketchcheck.VerifyFullyConstrained`) before its profile is extruded; the build returns every Sketch it used.

## 1. Frames

Every body is sketched on a plane made from an `r3.Frame` and extruded along that frame's normal U×V.
There are no placement transforms before the Booleans.

- `alongX(x)`: origin `(x, 0, 170)`, U = +Y, V = +Z, normal +X. Coordinates are `(y, z-170)`.
- `alongX(x)` is the drive shaft's frame; the crank shaft's frame, `alongCrank(x)`, is the same 40 lower
  (`crankHeight = axisHeight - centerDist`).
- `alongMinusY(y)`: origin `(0, y, 170)`, U = +X, V = +Z, normal -Y. Coordinates are `(x, z-170)`.

- Blade frame: sections on world-XY planes at Z = r (radius) with the sketch origin on the stacking line,
  placed by `r3.FromFrame(NewFrame((0, -86, 170), U = +Z, V = -Y))`, so the local Z becomes +X, local X the
  blade's direction of motion and local Y downstream (-Y). A local point `(u, v)` of a section at angle `beta`
  for chord position `xi` (mm from the leading edge) and offset `eta` toward the suction side is
  `(xiP - xi)(cos beta, -sin beta) + (eta - etaP)(-sin beta, -cos beta)`, with `xiP = c/2` and
  `etaP = c yc(0.5)`.

## 2. Handle

1. Sections: one `sketch.NewWorld()`. For each of the 14 stations of the instructions' table, a sketch on
   `CreateOffsetPlane(XY, z)`. Each holds 25 fit points on the oval of width W and length L centred at
   `(0, (z - 68) tan 15°)`, from angle -90° + 2° to 270° - 2°, each made by `sketchcheck.DimensionPoint`. A tail
   point carries `NewMidpointOf(tail, first, last)`, one `CreateFitSpline` runs through the points, and two lines
   close the section. Solve, require `sketchcheck.VerifyFullyConstrained`, and select the valid profile with three
   outer edges.
2. `decad.LoftSections` through the 14 sections in ascending z, then `decad.Transform` by the translation
   `(7, -23.5 - 60 tan 15°, -26.75)`, where the drop is `boxFloor - tightFit - 127.9`, so the grip's top sits a Tight fit
   under the floor. The sections are drawn about the origin so each sees it along rays, which the loft needs.
3. For each `handle-bolt-*` site, `fastenerCuts(loft, site, false, true, true)` cuts the clearance hole and the nut cavity
   from the loft. Name `handle`, group `handle`, motion `fixed`.
4. `b.hardware(site)` for each site adds `handle-bolt-k` and `handle-nut-k` (below).

## 3. Gearbox

The gearbox is built as one tray and cut into four pieces. The floor's underside is the literal `boxFloor` 101.4 and the
floor is 5 thick, which leaves the crank gear's tip 1.62 above the cavity floor.

1. Block: `roundedPrism` on the frame origin `(-20, 0, 0)`, U = +Y, V = +Z (normal +X): the outline (-41.7,101.4),
   (39.2,101.4), (39.2,190), (-41.7,190) with four corners of radius 3 from `sketch.CreateFillet`, each line placed by
   `NewDistancePointLine` from the origin. Extrude 51.2, then `decad.Fillet` the outer loops of both end caps with
   radius 2.
2. Cavity: `roundedPrism` on the frame origin `(-21, 0, 0)`: the outline (-31.9,106.4), (29.4,106.4), (29.4,191),
   (-31.9,191) with the second corner rounded with radius 3, extruded 42.4, so it runs through the left end. The other
   corners stay sharp, because a fillet where a seam crosses it leaves a thin sliver in the piece beside the seam.
   Sketch's `CreateFillet` cannot round the inside corner of an L section while keeping the sketch fully constrained, so
   the tray is made as a block minus a cavity.
3. Post: a panel on the frame origin `(-22, 0, 0)`, (-32.9..30.4, 105.4..192), extruded 11.8. `Cut` it from the cavity,
   which leaves a post across the block's whole left end up to X -10.2. Then `Cut` the cavity from the block. Building
   the post by `Union` after the cut fails in Decad.
4. Bearing blocks and ears, each starting inside its wall, so the `Union` shares no face. Fan block: `roundedPrism` on the
   frame origin `(0, -40.7, 0)`, U = +X, V = +Z (normal -Y): the outline (-15,101.4), (15,101.4), (15,190), (-15,190) with
   four corners of radius 3, extruded 25, with its end cap rounded with radius 2. Crank block: on the frame origin
   `(30.2, 0, 0)`, U = +Y, V = +Z (normal +X): the same outline in (Y, Z), extruded 14.8, with its end cap rounded with
   radius 2. Each ear is a panel on the frame origin at the seam minus 8 along its bolt, 16 long, over its range across
   the bolt and its Z range (under the floor, Z 93.4 to 102.4, or on the top, Z 189 to 198), from `gearboxEars`.
   `Union` the body, the two blocks and the four ears.
5. Seats and holes, by `b.seat(host, axis, pocketFrom, pocketTo, reliefFrom, reliefTo)`, which `Cut`s a cylinder of
   radius 11.25 (`seatRadius`, 11 + `tightFit`) between the first two stations along the shaft and one of radius 9.5
   between the last two. Stations run along the crank and drive shafts by X and along the fan shaft by the distance
   behind the apex. The post's seat on the drive axis and the crank axis is `b.seat` from -21 to -12.6 with its relief
   from -13 to -9.2. A seat in a wall's inner face is `b.funnelSeat(host, axis, floor, mouth)`: a cylinder of radius
   11.25 between the mouth (1 before the face) and the floor (7.4 behind it), and a cylinder of radius 9.5 and 5.5 long
   just behind the floor whose far end cap `decad.Chamfer` takes by 5.3, which makes a 45° cone down to radius 4.2. The
   crank-side wall has one on each of the crank axis and the drive axis, and the fan-side wall one on the fan axis. On
   the crank axis a hole of radius 4.2 (`b.hole`) joins the wall's seat to the block's seat, and the block's seat has its
   relief behind its floor. The fan axis has the same pair.
6. Bolt sites: `fastenerCuts` for every site of `fastenerSites()` whose bolt name starts with `gearbox-` cuts its head
   pocket, its hole and its nut cavity from the whole tray, so the cavity lands in the piece that holds the nut after the
   cut. The `handle-bolt-*` sites cut only the pocket and the hole here.
7. Cut: `decad.Copy` the finished solid three times. Each piece is a copy with `Cut` slabs removed (`b.slab`), and
   `splitGap` is `tightFit / 2`: the fan-side wall is the original less everything at Y >= -32.025, the post a copy less
   Y <= -31.775 and X >= -10.325, the crank-side wall a copy less Y <= -31.775 and X <= 21.525, and the base a copy less
   Y <= -31.775, X <= -10.075 and X >= 21.275. They are `gearbox-fan-wall`, `gearbox-post`, `gearbox-crank-wall` and
   `gearbox-base`, group `housing`, motion `fixed`.
8. `b.hardware(site)` for each `gearbox-` site adds the four ear bolts and nuts (`gearbox-bolt-k`, `gearbox-nut-k`) and
   the four wall bolts and nuts (`gearbox-screw-k`, `gearbox-screw-nut-k`).

The walls are 9.8 thick and each shaft crosses a wall in a bearing seat, so nothing round sticks out of the housing. A
boss or a sleeve would be a horizontal cylinder that needs support when printed. Cutting the tray at the walls lets each
piece print with its shaft axes vertical, so no bore lies on its side. Each wall's inner face is 0.4 from its bevel
gear's back face, so the gears need no spacer.

## 4. Bevel pair, spur pair and crank

`bevelgear.BuildPair` with `Module 2`, `Teeth 28`, `MateTeeth 14`, `SpiralAngle 0`, `EnableBore` false and `Placement`
frame origin `(21, 0, 170)`, U = -Y, V = +Z. `doc.Root().AddBody` then moves both bodies to the document's root container,
because Decad cuts only within one container, and each gets its spline bore by `b.splineCut`.
`moveBodies` (`containers.go`) puts them back in the pair's own containers at the end of `Build`.
The pair's driving body is `driving-gear` and its pinion body is `pinion`. The pair has no `Backlash` parameter and its
mesh is not measured (see the instructions' Open non-compliance: the bevel mesh).

- `driving-gear` (motion `drive`) and `pinion` (motion `fan`) are the pair's bodies with their spline bores, cut with
  clearance `tightFit + bevelBound`.
- Splines, in `bearings.go`: `shaftAxis` has a point, a direction and two perpendicular axes, and `spokeFrame(t, k)` is the
  plane at station t whose U axis points at angle 90° + 120° k. `b.grooves(shaft, axis, from, to)` `Cut`s three boxes
  (U 2.9 to `shaftRadius + 1.5`, V -1 to 1) along the shaft. `b.splineCut(host, axis, from, length, clearance)` makes the
  tool by a cylinder of radius `shaftRadius + clearance` minus three boxes (U `2.9 + clearance` to the bore plus 1,
  V `-(1 - clearance)` to `1 - clearance`, each running 1 past both ends) and `Cut`s it from the host. The spur gears, the
  arm and the hub pass `tightFit` (bore 4.0, teeth from 3.15, 1.5 wide).
- `bearing-*` (motion `fixed`, group `bearing`): `models/ballbearing` with bore 8, outside diameter 22 and width 7,
  placed by `spokeFrame` at the start station of each of the seven sites in the instructions' table.
- `alongCrank(x)`: origin `(x, 0, 130)`, U = +Y, V = +Z, normal +X. `alongPin(x)`: origin `(x, 0, 185)`, U = +Y,
  V = +Z, normal +X. Coordinates are `(y, z - 130)` and `(y, z - 185)`.
- `drive-gear` (motion `drive`) and `crank-gear` (motion `crank`): `spurgear.Variant{Define: spurRootClearance}.Build`
  with `Module 2`, `Teeth 20`, `Thickness 6`, `Backlash 0.15`, no bore and a `Placement` frame origin `(-9.2, 0, z)`,
  U = +Y, V = +Z, with z = 170 for the drive gear and 130 for the crank gear. `spurRootClearance` sets
  `gear_root_clearance` 0.5 mm and `gear_root_margin` 0.02 mm with `SetValue` (a literal length inside `SetExpr`
  fails), then `root_radius = pitch_radius - module - gear_root_clearance - gear_root_margin` (17.48) and
  `tip_radius = pitch_radius + module - gear_root_margin` (21.98) with `SetExpr`. The variant keeps the spur gear's root
  fillet. The drive gear is then turned by half a tooth, pi / 20, about its axis, and each gear gets its spline bore
  with `tightFit`. The `mesh` test proves the tip-to-root clearance of the pair (see Checks).
- `drive-shaft` (motion `drive`): a cylinder of radius 3.75 on `alongX(-23.1)` to X 28.5, `Cut` by the three grooves from
  X -10 to 29.5, then `Union` with the collar, a cylinder of radius 6 on `alongX(-24.1)`, extruded 4. The shaft starts 1
  inside the collar.
- `crank-shaft`: a cylinder of radius 3.75 on `alongCrank(-23.1)` to X 54.75, `Cut` by the three grooves from X -10 to
  56, then `Union` with the collar, a cylinder of radius 6 on `alongCrank(-24.1)`, extruded 4. There is no slot or bolt
  hole, because the arm is a tight fit on the splines. Cutting after the union with the collar fails in Decad.
- `crank-arm`: on `alongCrank(49)`, a stadium: an arc of radius 10 about the origin and an equal arc about (0, 55),
  `NewEqualRadius`, with four `NewTangent` lines, plus a circle of radius 1.3 (`boltHole`) at the pin end, extruded 8;
  the profile selection requires one hole. `decad.Fillet` the outer loops of both caps with radius 1.5. Equal end
  radii make the joins exactly tangent, which the loop fillet needs. `Cut` a spline bore (`tightFit`) on `alongCrank(48)`,
  extruded 7, and a hexagon of 7.43 across flats (`stubFlats + 2 * tightFit`) on `alongPin(52)`, extruded 6. Then
  `fastenerCuts` cuts the pin site's head pocket from the inner face.
- `crank-pin`: a hexagon of 6.93 across flats (8 across corners) on `alongPin(52.25)`, extruded 5.3; a shank of radius
  4.5 on `alongPin(57.25)` to X 104.4; a head of radius 5.8 on `alongPin(104)`, extruded 3. `Union` the three left to
  right, then `fastenerCuts` cuts the pin site's hole and nut cavity.
- `crank-grip`: one sketch on the frame origin `(57.4, 0, 185)`, U = +X, V = +Z. The meridian is the closed polygon
  through (0.5, 4.9), (46.2, 4.9), (46.2, 6.4), (49.5, 6.4), (50, 6.9), (50, 8.3), (48.8, 9.3), (43, 8), (36, 7.25),
  (22, 8), (8, 7.8), (1.2, 7.2), (0, 6) and (0, 5.4), each vertex made by `sketchcheck.DimensionPoint`. `decad.Revolve`
  it a full turn about the pin axis. The sloped sections need every vertex dimensioned from the origin; the
  repository's rule against fixing every computed point has no better answer for a profile whose segments are
  all slanted.
- `crank-pin-bolt` and `crank-pin-nut`: `b.hardware(site)` (below) with the head's top face centre at (49, 0, 185),
  pointing +X, nut corners on Y, length 10.

## 5. Fan

- `fan-shaft`: circle radius 3.75 on `alongMinusY(-26)`, extruded 74, `Cut` by the grooves from 25 to 32.2 and from 75 to
  101 behind the apex, then `Union` with the collar, a circle of radius 6 on `alongMinusY(-65.8)`, extruded 9.95.
- Hub: one sketch on the frame origin `(0, -76, 170)`, U = +X, V = +Z: circle radius 14 and `Blades` circles of radius 1.3
  at `(9.5 cos th_k, 9.5 sin th_k)`, `th_k = 2 pi k / Blades`, each centre dimensioned from the origin by a horizontal
  and a vertical distance. Extrude 20 toward -Y, round the end caps with radius 1, and `Cut` the spline bore (`tightFit`)
  from 75 to 97 behind the apex.
- Hub tools, each made for blade 1, with the others from `decad.PatternCopies` with
  `CircularPattern{Center (0, 0, 170), Axis -Y, Count Blades}` through `b.ring`, and every tool running 1 mm past the
  face it opens:
  - socket: a circle of radius 3.4 (`studRadius + regularFit`) on the frame origin `(5.6, -86, 170)`, U = +Y, V = +Z
    (normal +X), extruded 9.4, a plain round hole.
  - head pocket and nut cavity: `fastenerCuts(hub, site, true, false, true)` for each of the `Blades` sites of
    `bladeSites`. The bolt holes are the sketch's circles.

  `Cut` the hub by each tool in turn. The result is `fan-hub`.
- Blade sections: one `sketch.NewWorld()`. For each of the ten stations of the instructions' table, a sketch on
  `CreateOffsetPlane(XY, r)`. It holds 21 fit points, each made by `sketchcheck.DimensionPoint`, in the order
  suction surface at x_k for k = 10..1, the leading edge, then the pressure surface for k = 1..10, with
  `x_k = (1 - cos(pi k / 10)) / 2`. A tail point carries `NewMidpointOf(tail, first, last)`. There is one
  `CreateFitSpline` through the 21 points and the lines `last -> tail` and `tail -> first`. Solve, require
  `sketchcheck.VerifyFullyConstrained`, and select the valid profile with no holes and three outer edges.
  The sections are always drawn at the 25° blade angle. Every section sketch is returned in `Assembly.Sketches`.
- Blade 1:
  1. `decad.LoftSections` through the ten sections in ascending r.
  2. When `BladePitch` is not 25°, `decad.Transform` the loft by `r3.RotationAround(origin, +Z,
     -(BladePitch - 25°))` in the blade's local frame.
  3. Root stud: a circle of radius 3 on the frame origin `(0, 0, 6)`, U = +X, V = +Y, extruded 10. `Cut` it by a
     circle of radius 1.3 on the frame origin `(0, -5, 9.5)`, U = +Z, V = +X, extruded 10 (the bolt hole). The
     cut happens before the union, so the faceted loft never takes part in it.
  4. `Union` the loft and the stud: they overlap in volume and share no face.
  5. `decad.Transform` by `r3.FromFrame(NewFrame((0, -86, 170), U = +Z, V = -Y))`, so the local Z becomes +X,
     local X the direction of motion and local Y downstream (-Y).
  6. `decad.PatternCopies` with the hub's pattern gives blades 2..N. `Transform` retires its input, so blade 1 is
     not turned again.
- Bolts and nuts: `placeBolt` and `placeNut` of blade 1's site (below), then `PatternCopies` gives bolts 2..N and
  nuts 2..N. Each nut follows its bolt in the parts list.
- The document must come from `NewDocument`: without its chain tolerance, Decad refuses the blade-stud union
  because the loft's mesh bound is coarser than the pair's chord tolerance.

### Fasteners

`fasteners.go` owns every bolt and nut. `fastenerSites()` returns the site table of the instructions' Fasteners section
at crank angle 0: a `fastenerSite` holds the bolt and nut names, the head host and nut host, the motion, the head's top
face centre `head`, the direction `dir` from the head to the tip, the unit vector `corner` that the nut's corners lie
on, the bolt `length` and the `cavity` offset from the head. `bladeSites(n)` turns blade 1's site about the fan axis.

- `placeBolt(site)`: `m2bolt.Build` with `Length` and a `Placement` frame whose origin is `head + dir * 2.0` (the head
  pocket depth), U = `corner` and V = `(-dir) x corner`, so the bolt's local +Z (underside to head) points back to the
  head. It appends the bolt's `Profile` and `Socket` sketches to `Assembly.Sketches`.
- `placeNut(site)`: `m2nut.Build` with a `Placement` frame whose origin lies at `head + dir * (cavity + 0.1)`, U = `corner`
  and V = `dir x corner`. It appends the nut's `Hexagon`, `LowerChamfer` and `UpperChamfer` sketches.
- `fastenerCuts(host, site, pocket, hole, cavity)` cuts the features whose flag is true. The pocket is a cylinder of
  radius 2.0 from 1 outside the face to 2.0 inside. The hole is a cylinder of radius 1.3 from 1 outside the face to 1
  beyond the bolt's tip. The cavity is a hexagon (`b.hexagon`) 4.2 across flats and 1.8 long, starting at `cavity`,
  corners on `corner`. The caller passes the flags for the body it holds.
- `hardware(site)` adds the bolt and then the nut as parts of group `hardware` with the site's motion.
- `fastenerSpec(name)` maps a part name to its bill of materials spec (`M2 x 10 socket head cap screw`,
  `M2 x 16 socket head cap screw` or `M2 hex nut`). Blade sites match by prefix.

## 6. Parts set and exploded view

`Assembly.Set(doc)`: `parts.NewSet(doc)`, then `Add` each part (`Manufactured` PLA, or `Purchased` steel with a `Spec`
for the hardware and bearing groups) and `Join` each joint as a `PinInBore` with `Axis` in posed coordinates: the crank
axis through `State.Crank` along X, the drive axis through the apex along X, the pin axis through
`State.Crank + 55 * ArmDirection` along X, the fan axis along -Y, each shaft's axis for its bearings (each shaft in each
bearing turns, each bearing is fixed in the gearbox), each blade's radial line, each blade bolt's line parallel to the
fan axis at 9.5 along the blade, each grip bolt's vertical line at X -2 or X 16, Y -23.5, each ear bolt's line through
its ear's centre along Y and each wall bolt's line along X at Y 34.3. Every nut joins its host and its bolt along the
bolt's line (`nutJoints`), except `gearbox-screw-nut-*` in the ground piece, which has no joint. The head-side joints of
the pin bolt, the ear bolts and the wall bolts have `Pull` set to the direction the head comes out.

`Assembly.Explode`: `Set.Explode` with `WithExplodeGround("gearbox-base")` and `WithExplodeStages`, each stage `ClearAll`
along a direction, with `Together` set on the stages that carry nuts. Decad plans from boxes, which makes nested parts
jump over each other, so the order and the directions of the lifts come from a search over the order of the pinion, the
grip bolts and the three gears and over the directions of the driving gear and the pinion, scored on no overlap and the
shortest largest move. The order is: `ear bolts` (-Y), `post bolts` (-X), `crank-side wall bolts` (+X), `arm` (+X, the
default clearing), `handle` (+X), `pin` with its nut (+X), `pin bolt` (-X), `drive shaft` and `crank shaft` (-X),
`pinion` (-X), `grip bolts` (+Z), `crank gear`, `drive gear` and `driving gear` (+Z), `crank block bearing` (+X),
`wall bearings` and `post bearings` (-X), `fan wall bearing` (+Y), `fan block bearing` (-Y), `blade bolts` (-Y),
`blades`, `hub` with its nuts (-Y), `fan shaft` (-Y), `post` with nuts 1 and 3 (-X), `crank-side wall` with nuts 2 and
4 (+X), `fan-side wall` (-X) and `grip` with its nuts (-Z).

## 7. Pose

Each part is built at crank angle 0. Each `crank` part is then rotated about the line through `State.Crank` along -X
by `CrankAngle`, each `drive` part about the line through `(0, 0, 170)` along +X by `CrankAngle`, and each `fan` part
about the line along -Y by `-2*CrankAngle`.

## 8. Checks

`go test ./models/handcrankfan -timeout 30m` runs, for the default, a flat-pitch three-blade variant and a 40°
three-blade variant:

- the parameter and `Motion` checks (no build);
- part names, solids, and no leftover tool bodies;
- every Sketch fully constrained, the `m2bolt` and `m2nut` sketches included;
- `Tessellate` with `VerifyAll` for the handle, the four gearbox pieces, crank shaft (at 0.2 mm), crank arm, pin, crank
  gear, handle, hub, blade 1, bolt 1 and the nuts `gearbox-nut-1` and `handle-nut-1`;
- the orientation rules by bounds;
- the blade pitch from the blade mesh: leading edge upstream and chord angle within 0.5° of `beta(r)` at four
  radii, and the thickness ratio against the table (`LocatePoint` leaves too many points inside lofted bodies
  undecided to check a mirror point);
- the hub empty at a socket, at the head pocket and at the nut cavity;
- the crank's sweep against the blades: every vertex of the arm, pin and handle turned through a full turn about the
  crank axis in 3° steps stays 15 clear of the blades' disc;
- the grip's mesh rings at Z -0.75, 58.25 and 87.25 against the table (26.75 lower than drawn), and the 15° rake;
- for the default fan only, three subtests:
  - `fasteners` (`requireFasteners`): for each of the `11 + Blades` sites, the head is flush, the part is solid beside the
    pocket and 1 mm round it and the cavity, the nut sits centred in the part-enclosed cavity, the bolt tip is held by
    the part, the head seat is a contact with a gap of at most 0.05, no bolt or nut overlaps a part it passes through,
    and each gap between a fastener and its pocket, hole or cavity has a proven lower bound of at least 0.05;
  - `clearances` (`requireClearances`): every row of the instructions' Fits table, as the lower bound of a section
    outline gap or an axial bounds gap, against its class minimum, with every failing row listed together;
  - `mesh` (`requireSpurMesh`): the spur pair turned through one tooth pitch in four steps. At each of the five samples
    the outer loops of the two section outlines at X -6.2 give the root of each gear and the reach of its mate's tip,
    and the reach less the root must be 0.5 or more; the root loop must lie within 0.05 of 17.48; the two outlines must
    not overlap; and the sample's angle-dependent fits (gear to floor, back wall and fan wall, gear to pinion) must be
    at least Loose. The centre distance must be 40;
- the parts set and its explosion at the default pose: the arm, handle and pin in order along X with gaps, the main parts
  moving, the shafts and the pinion sliding to the left and the gears lifting, each joined nut moving with its host and
  the base's nuts staying, no two parts sharing space, and the bill of materials counting `11 + Blades` bolts,
  `11 + Blades` nuts and the seven bearings;
- the gear back faces at X = 21 and Y = -31.5 after undoing the pose, which retires the gears;
- no overlap or undecided pair among the finished parts from `doc.Verify` (the rotor, the handle, the gearbox, the
  bearings, the bolts and nuts, and the crank's shaft, arm, pin and handle, and the drive shaft), and none between a
  shaft or a bearing and any other part, which proves each groove lines up with its tooth. A bolt with its head host and
  with its own nut is left to the `fasteners` subtest, and the meshing gears are left out;
- blades 2 to N, bolts 2 to N and nuts 2 to N as copies of the first, which retires the bodies it turns and so runs last.
