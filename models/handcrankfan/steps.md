# Hand-crank fan steps

[instructions.md](instructions.md) owns the dimensions and the orientation rules. This file maps them to
Sketch and Decad operations. Every Sketch is solved and then required to be fully constrained
(`sketchcheck.VerifyFullyConstrained`) before its profile is extruded; the build returns every Sketch it used.

## 1. Frames

Every body is sketched on a plane made from an `r3.Frame` and extruded along that frame's normal U×V.
There are no placement transforms before the Booleans.

- `alongX(x)`: origin `(x, 0, 170)`, U = +Y, V = +Z, normal +X. Coordinates are `(y, z-170)`.
- `alongX(x)` is the drive shaft's frame; the crank shaft's frame, `alongCrank(x)`, is the same 40.4 lower.
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
   `(7, -23.5 - 60 tan 15°, 0)`. The sections are drawn about the origin so each sees it along rays, which the loft
   needs.
3. Two pilot cylinders of radius 0.9, from Z 116, length 13, at (X -2, Y -23.5) and (X 16, Y -23.5), each `Cut` from the
   loft. Name `handle`, group `handle`, motion `fixed`.
4. `handle-bolt-1` and `handle-bolt-2`: a screw body each (below) with its head's bearing face at Z 134.1, pointing
   -Z, length 16, shank radius 0.8, no nut.

A screw body is one `Union` of a head (a cylinder of radius 1.9 and height 2.0 behind the bearing face), a shank (a
cylinder from 0.8 behind the bearing face, so it overlaps the head) and, when there is a nut, a hexagon of 4.0 across
flats and 1.6 thick at the nut's near face. The nut is part of the body because a separate nut would touch the
shank and Decad cannot decide a touching pair.

## 3. Gearbox

The gearbox is built as one tray and cut into four pieces. The floor's underside is 1.2 under the crank gear's tip and 5
thick, which puts it at Z 101.4.

1. Block: `roundedPrism` on the frame origin `(-20, 0, 0)`, U = +Y, V = +Z (normal +X): the outline (-41.5,101.4),
   (39,101.4), (39,190), (-41.5,190) with four corners of radius 3 from `sketch.CreateFillet`, each line placed by
   `NewDistancePointLine` from the origin. Extrude 51, then `decad.Fillet` the outer loops of both end caps with
   radius 2.
2. Cavity: `roundedPrism` on the frame origin `(-21, 0, 0)`: the outline (-31.7,106.4), (29.2,106.4), (29.2,191), (-31.7,191)
   with the second corner rounded with radius 3, extruded 42.2, so it runs through the left end. The other corners stay sharp,
   because a fillet where a seam crosses it leaves a thin sliver in the piece beside the seam. Sketch's `CreateFillet`
   cannot round the inside corner of an L section while keeping the sketch fully constrained, so the tray is made as a
   block minus a cavity.
3. Post: a panel on the frame origin `(-22, 0, 0)`, (-32.7..30.2, 105.4..192), extruded 11.8. `Cut` it from the cavity, which
   leaves a post across the block's whole left end up to X -10.2. Then `Cut` the cavity from the block. Building the post
   by `Union` after the cut fails in Decad.
4. Bearing blocks and ears, each starting inside its wall, so the `Union` shares no face. Fan block: `roundedPrism` on the
   frame origin `(0, -40.5, 0)`, U = +X, V = +Z (normal -Y): the outline (-15,101.4), (15,101.4), (15,190), (-15,190) with four
   corners of radius 3, extruded 25, with its end cap rounded with radius 2. Crank block: on the frame origin `(30, 0, 0)`,
   U = +Y, V = +Z (normal +X): the same outline in (Y, Z), extruded 15, with its end cap rounded with radius 2. Each ear is a
   panel on the frame origin at the seam minus 8 along its screw, 16 long, over its range across the screw and its Z range
   (under the floor, Z 93.4 to 102.4, or on the top, Z 189 to 198). `Union` the body, the two blocks and the eight ears.
5. Seats and holes, by `b.seat(host, axis, pocketFrom, pocketTo, reliefFrom, reliefTo)`, which `Cut`s a cylinder of radius
   11.1 between the first two stations along the shaft and one of radius 9.5 between the last two. Stations run along
   the crank and drive shafts by X and along the fan shaft by the distance behind the apex. The post's seat on the drive
   axis and the crank axis is `b.seat` from -21 to -12.6 with its relief from -13 to -9.2. A seat in a wall's inner face is
   `b.funnelSeat(host, axis, floor, mouth)`: a cylinder of radius 11.1 between the mouth (1 before the face) and the floor
   (7.4 behind it), and a cylinder of radius 9.5 and 5.5 long just behind the floor whose far end cap `decad.Chamfer`
   takes by 5.3, which makes a 45° cone down to radius 4.2. The crank-side wall has one on each of the crank axis and the
   drive axis, and the fan-side wall one on the fan axis. On the crank axis a hole of radius 4.2 (`b.hole`) runs from 33.8 to
   35.7 and the block's seat from 37.6 to 46 with its relief from 35.6 to 37.7. On the fan axis a hole of radius 4.2 runs
   from 44.3 to 56.2 and the block's seat from 58.1 to 66.5 with its relief from 56.1 to 58.2. Two cylinders of radius 1.2 on
   the planes Z = 100.4, length 8, at (X -2, Y -23.5) and (X 16, Y -23.5). Each ear gets a through hole of radius 1.3, a
   counterbore of radius 2.15 and a hexagon trap of 4.3 across flats, along its screw, with the trap's corners on Z.
6. Cut: `decad.Copy` the finished solid three times. Each piece is a copy with `Cut` slabs removed (`b.slab`): the fan-side wall
   is the original less everything at Y >= -31.8, the post a copy less Y <= -31.6 and X >= -10.3, the crank-side wall a copy
   less Y <= -31.6 and X <= 21.3, and the base a copy less Y <= -31.6, X <= -10.1 and X >= 21.1. They are `gearbox-fan-wall`,
   `gearbox-post`, `gearbox-crank-wall` and `gearbox-base`, group `housing`, motion `fixed`.
7. `gearbox-bolt-1` .. `gearbox-bolt-8`: a screw body each (below) with its head's bearing face 0.1 short of its counterbore,
   pointing from the head to the nut, nut corners on Z, length 12, shank radius 1.0, at the ears' hole axes.

The walls are 9.8 thick and each shaft crosses a wall in a bearing seat, so nothing round sticks out of the housing. A
boss or a sleeve would be a horizontal cylinder that needs support when printed. Cutting the tray at the walls lets each
piece print with its shaft axes vertical, so no bore lies on its side. Each wall's inner face is 0.2 from its bevel gear's back
face, so the gears need no spacer.

## 4. Bevel pair, spur pair and crank

`bevelgear.BuildPair` with `Module 2`, `Teeth 28`, `MateTeeth 14`, `SpiralAngle 0`, `EnableBore` false and `Placement`
frame origin `(21, 0, 170)`, U = -Y, V = +Z. `doc.MoveBody` then moves both bodies to the document's root container,
because Decad cuts only within one container, and each gets its spline bore by `b.splineCut`.
The pair's driving body is `driving-gear` and its pinion body is `pinion`.

- `driving-gear` (motion `drive`) and `pinion` (motion `fan`) are the pair's bodies with their spline bores.
- Splines, in `bearings.go`: `shaftAxis` has a point, a direction and two perpendicular axes, and `spokeFrame(t, k)` is the
  plane at station t whose U axis points at angle 90° + 120° k. `b.grooves(shaft, axis, from, to)` `Cut`s three boxes
  (U 2.9 to 5.4, V -1 to 1) along the shaft. `b.splineCut(host, axis, from, length)` makes the tool by a cylinder of
  radius 4.0 minus three boxes (U 3.0 to 5.0, V -0.9 to 0.9, each running 1 past both ends) and `Cut`s it from the host.
- `bearing-*` (motion `fixed`, group `bearing`): `models/ballbearing` with bore 8, outside diameter 22 and width 7,
  placed by `spokeFrame` at the start station of each of the seven sites in the instructions' table.
- `alongCrank(x)`: origin `(x, 0, 129.6)`, U = +Y, V = +Z, normal +X. `alongPin(x)`: origin `(x, 0, 184.6)`, U = +Y,
  V = +Z, normal +X. Coordinates are `(y, z - 129.6)` and `(y, z - 184.6)`.
- `drive-gear` (motion `drive`) and `crank-gear` (motion `crank`): `spurgear.Variant{RootFillet: identity}.Build` with
  `Module 2`, `Teeth 20`, `Thickness 6`, no bore and a `Placement` frame origin `(-9.2, 0, z)`, U = +Y, V = +Z,
  with z = 170 for the drive gear and 129.6 for the crank gear. Decad cannot fillet the roots of a module-2
  gear, so the variant leaves them sharp. The drive gear is then turned by half a tooth, pi / 20, about its axis, and each gear gets its spline bore.
  `decad.Intersect` of the two gears is empty, which is the proof that they do not collide.
- `drive-shaft` (motion `drive`): a cylinder of radius 3.9 on `alongX(-22.8)` to X 28.3, `Cut` by the three grooves from
  X -10 to 29.3, then `Union` with the collar, a cylinder of radius 6 on `alongX(-23.8)`, extruded 4. The shaft starts 1
  inside the collar.
- `crank-shaft`: a cylinder of radius 3.9 on `alongCrank(-22.8)`, extruded 77.6, `Cut` by the three grooves from X -10 to 56,
  then `Union` with the collar, a cylinder of radius 6 on `alongCrank(-23.8)`, extruded 4. There is no slot or screw
  hole, because the arm is a tight fit on the splines. Cutting after the union with the collar fails in Decad.
- `crank-arm`: on `alongCrank(49)`, a stadium: an arc of radius 10 about the origin and an equal arc about (0, 55),
  `NewEqualRadius`, with four `NewTangent` lines, plus a circle of radius 1.2 at the pin end, extruded 8; the
  profile selection requires one hole. `decad.Fillet` the outer loops of both caps with radius 1.5. Equal end
  radii make the joins exactly tangent, which the loop fillet needs. `Cut` a spline bore on `alongCrank(48)`, extruded 7, and a hexagon of 7.13
  across flats (0.1 larger than the pin's stub on every flat) on `alongPin(51)`, extruded 7.
- `crank-pin`: a hexagon of 6.93 across flats (8 across corners) on `alongPin(51.2)`, extruded 6.3; a shank of radius 4.5 on
  `alongPin(57.1)` to X 104; a head of radius 5.8 on `alongPin(103.6)`, extruded 3. `Union` the three left to right,
  then `Cut` a box on `alongPin(53)`, (-2.15..2.15, -2.5..4.5), extruded 1.9, and a cylinder of radius 1.2 on
  `alongPin(50)`, extruded 10.
- `crank-grip`: one sketch on the frame origin `(57.2, 0, 184.6)`, U = +X, V = +Z. The meridian is the closed polygon
  through (0.5, 4.7), (46.2, 4.7), (46.2, 6.4), (49.5, 6.4), (50, 6.9), (50, 8.3), (48.8, 9.3), (43, 8), (36, 7.25),
  (22, 8), (8, 7.8), (1.2, 7.2), (0, 6) and (0, 5.2), each vertex made by `sketchcheck.DimensionPoint`. `decad.Revolve`
  it a full turn about the pin axis. The sloped sections need every vertex dimensioned from the origin; the
  repository's rule against fixing every computed point has no better answer for a profile whose segments are
  all slanted.
- `crank-pin-bolt`: a screw body with its bearing face centre at (48.9, 0, 184.6), pointing +X, nut corners on Z,
  length 10, nut near face 4.2 from the bearing face, shank radius 1.0. The shaft to the arm has no screw: it is a tight fit
  on the splines.

## 5. Fan

- `fan-shaft`: circle radius 3.9 on `alongMinusY(-26)`, extruded 74, `Cut` by the grooves from 25 to 32 and from 75 to 101
  behind the apex, then `Union` with the collar, a circle of radius 6 on `alongMinusY(-65.3)`, extruded 10.6.
- `onY(y)`: origin `(9.5, y, 170)`, U = +X, V = +Z, normal -Y. Coordinates are measured from blade 1's bolt line.
- Hub: one sketch on the frame origin `(0, -76, 170)`, U = +X, V = +Z: circle radius 14 and `Blades` circles of radius 1.15 at `(9.5 cos th_k, 9.5 sin th_k)`, `th_k = 2 pi k / Blades`, each
  centre dimensioned from the origin by a horizontal and a vertical distance. Extrude 20 toward -Y, round the end caps with radius 1, and `Cut` the spline bore from 75 to 97 behind the apex.
- Hub tools, each made for blade 1, with the others from `decad.PatternCopies` with
  `CircularPattern{Center (0, 0, 170), Axis -Y, Count Blades}`, and every tool running 1 mm past the face it
  opens:
  - socket: a circle of radius 3.15 on the frame origin `(5.6, -86, 170)`, U = +Y, V = +Z (normal
    +X), extruded 9.4, a plain round hole;
  - counterbore: circle radius 2.15 on `onY(-93.8)`, extruded 3.2;
  - nut trap: a regular hexagon of 4.3 across flats on `onY(-75)`, extruded 5. It is a `CreatePolygon` of six
    sides, grounded by `NewCoincident(centre, origin)`, turned by `NewHorizontal(side 1)` so the corners lie on
    U, and sized by `NewDistance(centre, vertex 0, 4.3 / sqrt 3)`.

  `Cut` the hub by each tool in turn, 3 x `Blades` cuts. The result is `fan-hub`.
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
     circle of radius 1.15 on the frame origin `(0, -5, 9.5)`, U = +Z, V = +X, extruded 10 (the bolt hole). The
     cut happens before the union, so the faceted loft never takes part in it.
  4. `Union` the loft and the stud: they overlap in volume and share no face.
  5. `decad.Transform` by `r3.FromFrame(NewFrame((0, -86, 170), U = +Z, V = -Y))`, so the local Z becomes +X,
     local X the direction of motion and local Y downstream (-Y).
  6. `decad.PatternCopies` with the hub's pattern gives blades 2..N. `Transform` retires its input, so blade 1 is
     not turned again.
- Bolts: a head (circle radius 1.9 on `onY(-127.9)`, extruded 2.0), a shank (circle radius 1.0 on `onY(-111.9)`,
  extruded 16.8, so it starts 0.8 inside the head) and a nut (a hexagon of 4.0 across flats on `onY(-112.3)`,
  extruded 1.6). `Union` the three, folded left to right. `PatternCopies` gives bolts 2..N. A separate nut
  would touch the shank, and Decad cannot decide a touching pair, so the nut is part of the bolt body.
  The 0.1 gaps stop `doc.Verify` reporting an undecided contact between the bolt and the hub.
- The document must come from `NewDocument`: without its chain tolerance, Decad refuses the blade-stud union
  because the loft's mesh bound is coarser than the pair's chord tolerance.

## 6. Parts set and exploded view

`Assembly.Set(doc)`: `parts.NewSet(doc)`, then `Add` each part (`Manufactured` PLA, or `Purchased` steel with a `Spec`
for the hardware and bearing groups) and `Join` each joint as a `PinInBore` with `Axis` in posed coordinates: the crank axis
through `State.Crank` along X, the drive axis through the apex along X, the pin axis through
`State.Crank + 55 * ArmDirection` along X, the fan axis along -Y, each shaft's axis for its bearings (each shaft in each bearing turns, each bearing is fixed in the gearbox), each blade's radial line, each blade bolt's line
parallel to the fan axis at 9.5 along the blade, and each grip screw's vertical line at X -2 or X 16, Y -23.5. The pin screw's joint with the arm has `Pull` (-1, 0, 0).

`Assembly.Explode`: `Set.Explode` with `WithExplodeGround("gearbox-base")` and `WithExplodeStages`, each stage `ClearAll` along a
direction. Decad plans from boxes, which makes nested parts jump over each other, so the order and the directions of
the lifts come from a search over the order of the pinion, the grip screws and the three gears and over the directions
of the driving gear and the pinion, scored on no overlap and the shortest largest move. The order is: three stages of ear screws (the fan-side wall's toward -Y, the post's and the crank-side wall's lower one toward -X, the crank-side wall's upper one toward +X), `arm` (+X, the
default clearing), `handle`, `pin` (+X), `pin screw` (-X), `drive shaft` and `crank shaft` (-X), `pinion`
(-X), `grip screws` (+Z), `crank gear`, `drive gear` and `driving gear` (+Z), `crank block bearing` (+X), `wall bearings` and
`post bearings` (-X), `fan wall bearing` (+Y), `fan block bearing` (-Y), `post` (-X), `crank-side wall` (+X), `fan-side wall` (-X) and `grip` (-Z).

## 7. Pose

Each part is built at crank angle 0. Each `crank` part is then rotated about the line through `State.Crank` along -X
by `CrankAngle`, each `drive` part about the line through `(0, 0, 170)` along +X by `CrankAngle`, and each `fan` part
about the line along -Y by `-2*CrankAngle`.

## 8. Checks

`go test ./models/handcrankfan -timeout 15m` runs, for the default, a flat-pitch three-blade variant and a 40° three-blade variant:

- the parameter and `Motion` checks (no build);
- part names, solids, and no leftover tool bodies;
- every Sketch fully constrained;
- `Tessellate` with `VerifyAll` for the handle, gearbox, crank shaft (at 0.2 mm), crank arm, pin, handle, hub, blade 1 and bolt 1;
- the orientation rules by bounds;
- the blade pitch from the blade mesh: leading edge upstream and chord angle within 0.5° of `beta(r)` at four
  radii, and the thickness ratio against the table (`LocatePoint` leaves too many points inside lofted bodies
  undecided to check a mirror point);
- the hub empty at a socket and a nut trap;
- the crank's sweep against the blades: every vertex of the arm, pin and handle turned through a full turn about the
  crank axis in 3° steps stays 15 clear of the blades' disc;
- the grip's mesh rings at Z -0.6, 58.4 and 87.4 against the table (26.6 lower than drawn), and the 15° rake;
- the gear back faces at X = 21 and Y = -31.5 after undoing the pose, which retires the gears;
- the parts set and its explosion at the default pose: the arm, handle and pin in order along X with gaps, the main parts
  moving, the shafts and the pinion sliding to the left and the gears lifting, no two parts sharing space, and the bill of
  materials counting the screws and the seven bearings;
- no overlap or undecided pair among the finished parts from `doc.Verify` (the rotor, the handle and its screws,
  the gearbox, the bearings, and the crank's shaft, arm, pin, handle and screws, and the drive shaft), and none between
  a shaft or a bearing and any other part, which proves each groove lines up with its tooth;
- blades 2 to N and bolts 2 to N as copies of the first, which retires the bodies it turns and so runs last.
