# Hand-crank fan steps

[instructions.md](instructions.md) owns the dimensions and the orientation rules. This file maps them to
Sketch and Decad operations. Every Sketch is solved and then required to be fully constrained
(`sketchcheck.VerifyFullyConstrained`) before its profile is extruded; the build returns every Sketch it used.

## 1. Frames

Every body is sketched on a plane made from an `r3.Frame` and extruded along that frame's normal U×V.
There are no placement transforms before the Booleans.

- `alongX(x)`: origin `(x, 0, 170)`, U = +Y, V = +Z, normal +X. Coordinates are `(y, z-170)`.
- `alongX(x)` is the drive shaft's frame; the crank shaft's frame, `alongCrank(x)`, is the same 70.4 lower.
- `alongMinusY(y)`: origin `(0, y, 170)`, U = +X, V = +Z, normal -Y. Coordinates are `(x, z-170)`.

- Blade frame: sections on world-XY planes at Z = r (radius) with the sketch origin on the stacking line,
  placed by `r3.FromFrame(NewFrame((0, -76, 170), U = +Z, V = -Y))`, so the local Z becomes +X, local X the
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
   `(7, -28 - 60 tan 15°, 0)`. The sections are drawn about the origin so each sees it along rays, which the loft
   needs.
3. Two pilot cylinders of radius 0.9, from Z 116, length 13, at (X -2, Y -28) and (X 16, Y -28), each `Cut` from the
   loft. Name `handle`, group `handle`, motion `fixed`.
4. `handle-bolt-1` and `handle-bolt-2`: a screw body each (below) with its head's bearing face at Z 134.1, pointing
   -Z, length 16, shank radius 0.8, no nut.

A screw body is one `Union` of a head (a cylinder of radius 1.9 and height 2.0 behind the bearing face), a shank (a
cylinder from 0.8 behind the bearing face, so it overlaps the head) and, when there is a nut, a hexagon of 4.0 across
flats and 1.6 thick at the nut's near face. The nut is part of the body because a separate nut would touch the
shank and Decad cannot decide a touching pair.

## 3. Gearbox, gear case and cover

Gearbox, the L-shaped tray:

1. Block: `roundedPrism` on the frame origin `(-20, 0, 0)`, U = +Y, V = +Z (normal +X): the outline (-46,128),
   (11,128), (11,181), (-46,181) with four corners of radius 3 from `sketch.CreateFillet`, each line placed by
   `NewDistancePointLine` from the origin. Extrude 44, then `decad.Fillet` the outer loops of both end caps with
   radius 2.
2. Cavity: `roundedPrism` on the frame origin `(-21, 0, 0)`: the outline (-36.2,134), (12,134), (12,182), (-36.2,182)
   with the first corner rounded with radius 3, extruded 46, so it runs through both ends. `Cut` it from the block.
   Sketch's `CreateFillet` cannot round the inside corner of an L section while keeping the sketch fully constrained,
   so the L is made as a block minus a cavity.
3. Bearing block: `roundedPrism` on the frame origin `(0, -45, 0)`, U = +X, V = +Z (normal -Y): the outline
   (-11,128), (11,128), (11,181), (-11,181) with four corners of radius 3, extruded 15, with its end cap rounded with
   radius 2. It starts 1 inside the wall, so the `Union` shares no face.
4. `Cut` a cylinder of radius 3.2 (`alongMinusY(-30)`, length 74), two of radius 1.2 on the planes Z = 127, length 8, at
   (X -7, Y -28) and (X 11, Y -28), and five of radius 0.9, length 10, from X 15 toward +X at the five case screws'
   (Y, Z) positions. Name `gearbox`, group `housing`, motion `fixed`.

Gear case:

1. One block `roundedPrism` on the frame origin `(42, 0, 0)`, U = +Z, V = +Y (normal -X): the outline (Z, Y)
   (56.4,-46), (181,-46), (181,46), (56.4,46) with four corners of radius 3, extruded 17.8 toward -X, with its end cap
   (the inner face at X 24.2) rounded with radius 1.5.
2. Chamber: `roundedPrism` on the frame origin `(34, 0, 0)`, U = +Y, V = +Z: the outline (-38.2,61.4), (38.2,61.4),
   (38.2,182), (-38.2,182) with the two lower corners rounded with radius 3, extruded 10. `Cut` it from the block.
   A union of a plate and a rim shares their rounded corners, which Decad refuses, so the case is one block.
3. `Cut` a cylinder of radius 4.2 on `alongX(20)`, length 20 (the drive shaft), one of radius 4.2 on
   `alongCrank(28)`, length 7 (the crank shaft's pocket), for each case screw a cylinder of radius 1.15 from X 24 and a
   counterbore of radius 2.15 from X 31.7, and for each cover screw a pilot of radius 0.9 from X 30.5, length 13.5.
   Name `gear-case`, group `housing`, motion `fixed`.

Gear cover:

1. Plate: `roundedPrism` on the frame origin `(42.2, 0, 0)`, U = +Y, V = +Z: the case's outline, extruded 7.
2. Bearing block: `roundedPrism` on `(48.2, 0, 0)`: a 22 square centred on the crank axis, corners of radius 3, extruded 9,
   with its end cap rounded with radius 2. `Union` it with the plate (it starts 1 inside).
3. `Cut` a cylinder of radius 4.2 on `alongCrank(42.1)` through the plate and the block, one of radius 4.2 on
   `alongX(42.1)`, 5.2 long, for the drive shaft, and for each cover screw a hole of radius 1.15 and a counterbore of
   radius 2.15 from X 46.9. Name `gear-cover`, group `housing`, motion `fixed`.

The case and the cover are drawn from the plane at X 24.2, 34 and 42.2, so the plate is 9.8 thick and each shaft
turns in a long flat-walled bore. A boss or a sleeve would be a horizontal cylinder that needs support when printed.
The plate's inner face is 0.2 from the driving gear's back face, so the gear needs no spacer. The `gear-case` is bolted
to the tray by five screws and the cover to the rim by four.

## 4. Bevel pair, spur pair and crank

`bevelgear.BuildPair` with `Module 2`, `Teeth 32`, `MateTeeth 16`, `SpiralAngle 0`, `BoreDiameter 8`,
`MateBoreDiameter 6` and `Placement` frame origin `(24, 0, 170)`, U = -Y, V = +Z.
The pair's driving body is `driving-gear` and its pinion body is `pinion`.

- `driving-gear` (motion `drive`) and `pinion` (motion `fan`) are the pair's bodies.
- `alongCrank(x)`: origin `(x, 0, 99.6)`, U = +Y, V = +Z, normal +X. `alongPin(x)`: origin `(x, 0, 154.6)`, U = +Y,
  V = +Z, normal +X. Coordinates are `(y, z - 99.6)` and `(y, z - 154.6)`.
- `drive-gear` (motion `drive`) and `crank-gear` (motion `crank`): `spurgear.Variant{RootFillet: identity}.Build` with
  `Module 2`, `Teeth 35`, `Thickness 6`, `BoreDiameter 8` and a `Placement` frame origin `(35, 0, z)`, U = +Y, V = +Z,
  with z = 170 for the drive gear and 99.6 for the crank gear. Decad cannot fillet the roots of a 35-tooth module-2
  gear, so the variant leaves them sharp. The drive gear is then turned by half a tooth, pi / 35, about its axis.
  `decad.Intersect` of the two gears is empty, which is the proof that they do not collide.
- `drive-shaft` (motion `drive`): a cylinder of radius 4 on `alongX(14)`, extruded 33.
- `crank-shaft`: a cylinder of radius 4 on `alongCrank(28.2)`, extruded 38.8. `Cut` a box on `alongCrank(60.7)`,
  (-5..5, -5..-3), extruded 7 (the flat), a box on `alongCrank(63.4)`, (-2.15..2.15, -4.5..2.5), extruded 1.9 (the nut
  slot), and a cylinder of radius 1.2 on `alongCrank(58.2)`, extruded 10. Cutting the flat after the round shaft is
  built avoids a union of two coincident cylinders.
- `crank-arm`: on `alongCrank(61.2)`, a stadium: an arc of radius 10 about the origin and an equal arc about (0, 55),
  `NewEqualRadius`, with four `NewTangent` lines, plus circles of radius 1.2 at both centres, extruded 8; the
  profile selection requires two holes. `decad.Fillet` the outer loops of both caps with radius 1.5. Equal end
  radii make the joins exactly tangent, which the loop fillet needs. `Cut` a D pocket of radius 4.15 and flat 3.15
  (toward -Z) on `alongCrank(60.2)`, extruded 7, and one (toward +Z) on `alongPin(63.2)`, extruded 7.
- `crank-pin`: a D prism of radius 4 and flat 3 (toward +Z) on `alongPin(63.4)`, extruded 6.3; a shank of radius 4.5 on
  `alongPin(69.3)` to X 116.2; a head of radius 5.8 on `alongPin(115.8)`, extruded 3. `Union` the three left to right,
  then `Cut` a box on `alongPin(65.2)`, (-2.15..2.15, -2.5..4.5), extruded 1.9, and a cylinder of radius 1.2 on
  `alongPin(62.2)`, extruded 10.
- `crank-grip`: one sketch on the frame origin `(69.4, 0, 154.6)`, U = +X, V = +Z. The meridian is the closed polygon
  through (0.5, 4.7), (46.2, 4.7), (46.2, 6.4), (49.5, 6.4), (50, 6.9), (50, 8.3), (48.8, 9.3), (43, 8), (36, 7.25),
  (22, 8), (8, 7.8), (1.2, 7.2), (0, 6) and (0, 5.2), each vertex made by `sketchcheck.DimensionPoint`. `decad.Revolve`
  it a full turn about the pin axis. The sloped sections need every vertex dimensioned from the origin; the
  repository's rule against fixing every computed point has no better answer for a profile whose segments are
  all slanted.
- `crank-arm-bolt`: a screw body with its bearing face centre at (69.3, 0, 99.6), pointing -X, nut corners on Z,
  length 10, nut near face 4.1 from the bearing face, shank radius 1.0. `crank-pin-bolt`: bearing face centre at
  (61.1, 0, 154.6), pointing +X, length 10, nut near face 4.2 from the bearing face.

## 5. Fan

- `fan-shaft`: circle radius 3 on `alongMinusY(-22)`, extruded 68.
- `onY(y)`: origin `(9.5, y, 170)`, U = +X, V = +Z, normal -Y. Coordinates are measured from blade 1's bolt line.
- Hub: one sketch on the frame origin `(0, -66, 170)`, U = +X, V = +Z: circle radius 14, a concentric circle
  radius 3, and `Blades` circles of radius 1.15 at `(9.5 cos th_k, 9.5 sin th_k)`, `th_k = 2 pi k / Blades`, each
  centre dimensioned from the origin by a horizontal and a vertical distance. Extrude 20 toward -Y.
- Hub tools, each made for blade 1, with the others from `decad.PatternCopies` with
  `CircularPattern{Center (0, 0, 170), Axis -Y, Count Blades}`, and every tool running 1 mm past the face it
  opens:
  - socket: circle radius 3.15 on the frame origin `(5.6, -76, 170)`, U = +Y, V = +Z (normal +X), extruded 9.4;
  - counterbore: circle radius 2.15 on `onY(-83.8)`, extruded 3.2;
  - nut trap: a regular hexagon of 4.3 across flats on `onY(-65)`, extruded 5. It is a `CreatePolygon` of six
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
  5. `decad.Transform` by `r3.FromFrame(NewFrame((0, -76, 170), U = +Z, V = -Y))`, so the local Z becomes +X,
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
for the hardware group) and `Join` each joint as a `PinInBore` with `Axis` in posed coordinates: the crank axis
through `State.Crank` along X, the drive axis through the apex along X, the pin axis through
`State.Crank + 55 * ArmDirection` along X, the fan axis along -Y, each blade's radial line, each blade bolt's line
parallel to the fan axis at 9.5 along the blade, each grip screw's vertical line at X -7 or X 11, Y -28, and each case
and cover screw's line along X. The pin screw's joint with the arm has `Pull` (-1, 0, 0).

`Assembly.Explode`: `Set.Explode` with `WithExplodeGround("gearbox")` and `WithExplodeStages`. Decad plans from boxes, and
the nested parts along X make it jump parts over each other, so the order comes from a search: all 1152 combinations of
the order of the four groups (case, shafts and gears, cover, crank), the order of the arm, handle, pin and arm screw,
and the arm's clearing, scored on no overlap and the arm, handle and pin in order. The order is: `case screws` and
`case`; `drive shaft`, `crank shaft`, `drive gear`, `crank gear`; `cover screws` and `cover`; `arm` (the default
clearing, direction +X); `handle`, `pin`, `arm screw`; `pin screw` (-X); `driving gear` (-X); `pinion` (+Z); `grip
screws` (+Z); `grip` (-Z), each other stage `ClearAll` along +X.

## 7. Pose

Each part is built at crank angle 0. Each `crank` part is then rotated about the line through `State.Crank` along +X
by `CrankAngle`, each `drive` part about the line through `(0, 0, 170)` along -X by `CrankAngle`, and each `fan` part
about the line along -Y by `2*CrankAngle`.

## 8. Checks

`go test ./models/handcrankfan -timeout 15m` runs, for the default, a flat-pitch three-blade variant and a 40° three-blade variant:

- the parameter and `Motion` checks (no build);
- part names, solids, and no leftover tool bodies;
- every Sketch fully constrained;
- `Tessellate` with `VerifyAll` for the handle, gearbox, crank shaft (at 0.2 mm, because Decad's minimum mesh bound
  refuses its D stub at 0.1 mm), crank arm, pin, handle, hub, blade 1 and bolt 1;
- the orientation rules by bounds;
- the blade pitch from the blade mesh: leading edge upstream and chord angle within 0.5° of `beta(r)` at four
  radii, and the thickness ratio against the table (`LocatePoint` leaves too many points inside lofted bodies
  undecided to check a mirror point);
- the hub empty at a socket and a nut trap;
- the crank's sweep against the blades: every vertex of the arm, pin and handle turned through a full turn about the
  crank axis in 3° steps stays 15 clear of the blades' disc;
- the grip's mesh rings at Z 26, 85 and 114 against the table, and the 15° rake;
- the gear back faces at X = 24 and Y = -36 after undoing the pose, which retires the gears;
- the parts set and its explosion at the default pose: the arm, handle and pin in order along X with gaps, the main parts
  moving, the driving gear sliding off to the left and the pinion lifting, no two parts sharing space, and the bill of
  materials counting the screws;
- no overlap or undecided pair among the finished parts from `doc.Verify` (the rotor, the handle and its screws,
  the gearbox, the gear case and the cover with their screws, and the crank's shaft, arm, pin, handle and screws), after
  removing the fan shaft;
- blades 2 to N and bolts 2 to N as copies of the first, which retires the bodies it turns and so runs last.
