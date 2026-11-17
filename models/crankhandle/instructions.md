# Crank handle

## Purpose and source

Construct a printed hand crank for a D-flat shaft. It is the hand-crank siren's crank, moved here so that the siren and
any other mechanism build it from one construction. The parts are:

- an arm, 20 mm wide, with a 17 mm hub, a boss under the bore that joins the 8 mm arm through a 30° ramp. The hub's
  bore pinches the shaft: a slit through the hub and two M2 x 16 bolts across it close the bore. The 8 mm shaft of the
  `Params6082RS` crank has two opposite flats, and the bore has two;
- a pin, whose hexagonal stub sits in a pocket in the arm and whose head holds the handle on. It prints as two halves
  split lengthwise ([Split shafts](../../docs/model-pipeline.md#split-shafts));
- a handle, a barrel that turns freely on the pin's shank;
- an M2 x 10 bolt from the arm's back face and an M2 nut in the pin's stub, which clamp the pin in the arm;
- two M2 x 6 cross bolts and M2 nuts, which clamp the pin's two halves together;
- two M2 x 16 clamp bolts and M2 nuts across the arm's hub, stacked along the shaft, which close the slit.

The shaft is not modeled. It belongs to the mechanism, and its diameter equals the bore of the purchased bearing it runs
in ([Part clearances](../../docs/model-pipeline.md#part-clearances)). The presets read that bore from
[`models/ballbearing`](../ballbearing/). The bolt and nut are [`models/m2bolt`](../m2bolt/) and
[`models/m2nut`](../m2nut/), placed with `internal/m2fastener`. [`models/handcranksiren`](../handcranksiren/) builds its
crank with `BuildParts(Params6082RS())`.

All dimensions are in millimeters.

## Symbols

| Symbol | Meaning | Formula |
| --- | --- | --- |
| `D`, `d`, `c` | shaft diameter, flat depth, bore clearance | `ShaftDiameter`, `FlatDepth`, `BoreClearance` |
| `r_b` | bore radius | `D/2 + c` |
| `f_b` | distance from the axis to the bore's flat, toward the pin | `D/2 - d + c` |
| `b` | distance from the axis to the bore's wall on -Y, where the slit enters | `f_b` with `DoubleFlat`, else `r_b` |
| `L_b` | length of the bore's outline | `2 r_b (pi - acos(f_b / r_b)) + 2 sqrt(r_b^2 - f_b^2)`; with `DoubleFlat` `2 r_b (pi - 2 acos(f_b / r_b)) + 4 sqrt(r_b^2 - f_b^2)` |
| `H` | the hub's length, the back face to the boss top | `HubLength` |
| `y_f` | where the ramp meets the arm's outer face | `ArmRadius + (HubLength - ArmThickness) / tan(RampAngle)` |
| `L` | throw, axis to pin axis | `Throw` |
| `y_c` | the clamp bolts' axis | `-(r_b + 1.3 + ClampHoleWall)` |
| `y_s` | where the arm's sides stop being straight | `y_c - ClampSideRun` |
| `c_R`, `c_E` | right corner arc's centre, end arc's centre | `(ArmRadius - ClampCornerR, y_s)`, `(0, y_s + sqrt((ClampEndR - ClampCornerR)^2 - (ArmRadius - ClampCornerR)^2))` |
| `R_e` | the end arc's reach behind the axis | `ClampEndR - c_E.y` |

## API

Package: `models/crankhandle`. Exported surface:

- The fixed dimensions as constants: `FitMargin`, `TightFit` (0.25), `RegularFit` (0.40), `ArmRadius` (10),
  `ArmThickness` (8), `HubLength` (17), `RampAngle` (30), `BoreFloor` (1), `PinHoleR` (1.3), `PocketFlats` (7.53), `PocketFrom` (3.25), `PocketLength`
  (5.75), `StubFlats` (7.03), `StubFrom` (3.5), `StubLength` (5.8), `ShankR` (5.15), `ShankFrom` (8.25), `RootR`
  (3.5), `ShankTo` (52.75), `HeadR` (6.45), `HeadFrom` (55), `HeadLength` (3), `HandleFrom` (7.4), `PinBolt` (10),
  `PinCavity` (4.75), `PressWall` (1.6), `ChamferSize` (0.5), `CrossBolt` (6), `CrossFlat` (4.0), `CrossFlatHalf`
  (3.25), `CrossCavity` (6.0), `ConeAngle` (30), `ClampBolt` (16), `ClampHoleWall` (1.05), `ClampSideRun` (3.6),
  `ClampCornerR` (4.5), `ClampEndR` (22), `SlitWidth` (1.5), `SlitInBore` (1), `NutPocketRound` (0.6), the cross
  bolts' positions `CrossZ` (17, 49), the clamp bolts' positions `ClampZ` (4, 12), and `HandleProfile`, the handle's meridian.
- The container names `ArmContainerName` (`Arm`), `PinContainerName` (`Pin`), `HandleContainerName` (`Handle`),
  `BoltsContainerName` (`Bolts`) and `NutsContainerName` (`Nuts`).
- `Params` with `ShaftDiameter`, `FlatDepth`, `BoreClearance`, `Throw`, `DoubleFlat` (a second flat, `FlatDepth` deep,
  opposite the first) and `Placement *r3.Frame` (nil selects the local frame).
- `Params605ZZ()`, `Params6082RS()` and `DefaultParams()`, which returns `Params605ZZ()`.
- `Params.Validate() error` and `Params.Derive() (Dimensions, error)`. `Dimensions` holds `BoreRadius`, `BoreFlat`,
  `BoreBack` (`b`), `BoreLoop`, `RampFoot` (`y_f`), `Site`, the pin bolt's `m2fastener.Site`, `CrossSites`, the two
  cross bolts' sites, `ClampY` (`y_c`), `SideEnd` (`y_s`), `Reach` (`R_e`) and `ClampSites`, the two clamp bolts' sites,
  in the local frame, with the methods
  `CornerCenter()` (`c_R`) and `EndCenter()` (`c_E`).
- `NewDocument(opts ...decad.DocumentOption)` returns `decad.New(WithChainTolerance(0.01 mm), opts...)`. The pin's
  cuts after its union need a chain tolerance of 0.005 mm or more; build in such a document.
- `BuildParts(ctx, doc, p) (*Parts, error)` adds the arm, the pin and the handle and puts them in no container. `Parts`
  holds `Arm`, `Pin`, `Handle`, every `Sketches`, `Dimensions`, and in the world frame the pin bolt's `Site`, the cross
  bolts' `CrossSites`, the clamp bolts' `ClampSites` and `SplitNormal`, the unit normal of the plane that splits the
  pin.
- `Build(ctx, doc, p) (*Model, error)` adds the parts, the bolts and the nuts, each in its container. `Model` embeds
  `Parts` and holds `Bolt *m2bolt.Model`, `Nut *m2nut.Model`, `CrossBolts [2]*m2bolt.Model`,
  `CrossNuts [2]*m2nut.Model`, `ClampBolts [2]*m2bolt.Model` and `ClampNuts [2]*m2nut.Model`.

## Presets and defaults

| Field | `Params605ZZ` (default) | `Params6082RS` (the siren's) |
| --- | --- | --- |
| `ShaftDiameter` | 5, `ballbearing.Params605ZZ().BoreDiameter` | 8, `ballbearing.Params6082RS().BoreDiameter` |
| `FlatDepth` | 0.5, the planetary gearbox shafts' flat | 0.75, the siren shafts' flat |
| `BoreClearance` | 0.05, `ballbearing.PressFitClearance` | 0.05 |
| `Throw` | 55 | 55 |
| `DoubleFlat` | false | true, the shaft's second flat |

Derived: `r_b` 2.55 and 4.05, `f_b` 2.05 and 3.30, `b` 2.55 and 3.30, `L_b` 15.8 and 24.8, `y_f` 25.59 (both), `y_c`
-4.90 and -6.40, `y_s` -8.50 and -10.00, `R_e` 13.89 and 15.39. A 608ZZ has the same bore as a 608-2RS.

## Validation

Every message starts with `crankhandle: `. Derive checks in this order and returns the first failure:

1. Every float field is finite and positive: `<Field> must be finite and positive`.
2. `d <= D/4`.
3. `ballbearing.PressFitClearance <= c <= 0.5`.
4. `ArmRadius - r_b >= PressWall`, so the arm keeps a 1.6 mm wall round the bore (`D` at most 16.7).
5. `L >= y_f + h + RegularFit`, where `h` is the handle's largest radius (9.3), so the handle and the pin pocket clear
   the ramp: `Throw must be at least 35.29 mm`.
6. `Placement` is nil or a valid frame.

BuildParts and Build reject a nil context and a nil document, derive the dimensions, and check `ctx.Err()` before any
geometry. A rejected build adds no body.

## Local frame

The origin is on the shaft axis in the arm's back face, the face toward the mechanism. +Z runs along the shaft axis away
from the mechanism, the arm runs along +Y to the pin axis at `(0, L)`, and the bore's flat faces +Y (with `DoubleFlat`, a second flat faces -Y). A `Placement` maps
this frame onto the given one after each part's last operation.

## Geometry

1. **Arm.** A sketch on z = 0: an arc of radius `ArmRadius` about `(0, L)`, the two sides x = ±`ArmRadius` tangent to
   it down to y = `y_s`, a corner arc of radius `ClampCornerR` about `c_R` and its mirror tangent to each side, and an
   end arc of radius `ClampEndR` about `c_E` tangent to both corners from inside, with a circle of radius `PinHoleR`
   about `(0, L)`. Every join is tangent, so the outline has no inside corner and no step. Extrude it `HubLength`
   along +Z, then chamfer the outer loop of both faces by `ChamferSize`. Then:
   - cut the relief (`reliefTool`): a prism across the arm whose side profile, in the plane x = -(`ArmRadius` + 1) with
     sketch (u, v) the local (y, z), has the points (`y_f`, 8), (`L + ArmRadius + 1`, 8), (`L + ArmRadius + 1`, 18) and
     (`ArmRadius - 1/tan(RampAngle)`, 18), extruded `2 ArmRadius + 2`. It leaves the 17 mm boss for y up to `ArmRadius`,
     a 30° ramp falling to z = 8 at y = `y_f`, and the 8 mm arm beyond;
   - chamfer the new outer face's outline by `ChamferSize` (`chamferOuterFace`): its two sides and the pin end's arc,
     without the ramp's foot and the pin hole's rim, and the boss top's edge with the ramp. Each edge is named by its
     two stored end points. The outline is a tangent chain and the ramp's top edge is not, so each is chamfered by its
     own `decad.Chamfer` call;
   - cut, in this order:
     - the slit: x within ±`SlitWidth`/2 from 2 mm past the end arc up to y = `-b + SlitInBore`, closed there by a half
       circle of radius `SlitWidth`/2, from z = -1 through the hub (extruded `H + 2`). Its round end lies inside the bore's
       outline, so it shows only in the bore's floor;
     - the blind bore, a sketch on z = -1 extruded `1 + H - BoreFloor` (to z = 16): a D, an arc of radius `r_b` about the
       axis the long way round through -Y, closed by its flat at `y = f_b`; with `DoubleFlat` two arcs of radius `r_b`
       closed by two flats at `y = ±f_b`;
     - the hexagonal pocket, a hexagon `PocketFlats` across flats about `(0, L)` with its corners on X, from z =
       `PocketFrom` for `PocketLength`, out through the 8 mm outer face;
     - the pin bolt's head pocket;
     - each clamp bolt's nut pocket, head pocket and hole ("Clamp" below), bolt 1 then bolt 2.

   The slit goes first. The bore cut on the chamfered arm leaves a mesh bound round its rim of about 0.0101 mm, over
   the document's 0.01 mm chain tolerance for the slit's cut, while the bore's cut admits the slit's finer rim. The
   relief goes before the cuts because its rims are far from every later cut and the stub pocket opens through the
   face it creates. After the last cut, chamfer the bore's rim in the back face by 0.5 mm: the two arcs and the flat the
   slit leaves, or with `DoubleFlat` the two arcs and three lines, because the slit splits the -Y flat.
2. **Pin.** About `(0, L)`:
   - Join a hexagon `StubFlats` across flats, corners on X, from z = `StubFrom` for `StubLength` (to z 9.3), and a
     revolved meridian: a cone of half angle `ConeAngle` from radius `RootR` at z = `ShankFrom` out to `ShankR` at
     z 11.11, the shank to z = `ShankTo`, a cone out to `HeadR` at z = `HeadFrom`, and the head to z 58. The root cone
     starts inside the stub's flats and grows past its corners (4.06) by z 9.22, inside the stub.
   - Cut each cross bolt's double flat: everything beyond `CrossFlat` (4.0) from the axis on both the +Y and the -Y
     side, over z = `CrossZ[i]` ± `CrossFlatHalf`, so 13.75 to 20.25 and 45.75 to 52.25.
   - Cut the pin bolt's hole and nut cavity with `m2fastener.RoundedHoleTool` and `RoundedCavityTool`: radius 1.3 to
     z 13 and radius 2.51 from z 4.75 to 6.75, each end corner rounded by 0.55 mm.
   - Cut each cross bolt's clearance hole and head pocket with `m2fastener.HoleTool` and `PocketTool`, and its nut
     pocket with the hexagonal pocket tool ("Clamp" below), rounded by `NutPocketRound`.
3. **Handle.** Revolve `HandleProfile` about the pin axis. The sketch plane holds the axis, with its origin at
   `(0, L, HandleFrom)`, U along +Z and V the radius. The handle runs from z = 8.4 to 58.4: a bore of radius 5.55, which
   ends in a 30° cone 0.4 mm off the pin's head cone, a counterbore of radius 7.05 for the pin's head, a waist and a
   flared outer end. Every corner of the meridian is cut by a small chamfer.
4. **Fasteners.**
   - The pin bolt's head lies flush with the arm's back face at `(0, L)` and the bolt runs along +Z: head pocket 2 deep
     in the arm, hole 2.6 across through the arm and into the pin, nut cavity from 4.75 to 6.75 in the stub, and the
     tip at z = 12, inside the shank. `m2fastener.PlaceBolt` and `PlaceNut` place an M2 x 10 bolt and an M2 nut.
   - Each cross bolt's site has its head at `(0, L + CrossFlat, CrossZ[i])`, `Dir` (0, -1, 0), `Corner` (1, 0, 0),
     `Length` 6 and `Cavity` 6. Its head lies flush with the +Y flat in a pocket 2 deep (y from L + 4 to L + 2), its
     hole runs through both halves, its nut sits in a hexagonal pocket open to the -Y flat, from y = L - 2 to L - 4, and
     its tip ends at the -Y flat (y = L - 4). `Derive` rejects a cross site whose tip is not flush with the flat.
     `PlaceBolt` and `PlaceNut` place an M2 x 6 bolt and an M2 nut at each.
5. **Containers.** `Build` files the arm, pin and handle in their containers, the five bolts in `Bolts` and the five
   nuts in `Nuts`, each container a child of the document root.

## Clamp

The slit through the hub closes the bore on the shaft when the clamp bolts are tightened. Two bolts cross the whole hub
along -X, on y = `y_c`, at the heights `ClampZ` (4 and 12), 8 mm apart along the shaft: the first 4 mm above the back
face and the second 4 mm below the bore's floor. The hole between each bolt and the bore keeps `ClampHoleWall` (1.05
mm) where it crosses the slit, with a single D bore, and 1.80 mm with the double flat, whose wall on that side is the
flat at `b = f_b`. Each site is `m2fastener.Site{Head: (ArmRadius, y_c, ClampZ[i]), Dir: (-1, 0, 0), Corner: (0, 0, 1),
Length: 16, Cavity: HeadPocketDepth + ClampBolt - NutCavityHeight}`, so `Cavity` is 16 and both `CavityFar` and the bolt's
tip lie 18 from the head, 2 mm inside the -X side face. Derive rejects a site whose tip is not at `CavityFar` or not
inside that face (message `clamp bolt %d: ...`). M2 x 16 is in `m2bolt.StandardLengths()`. M2 x 18 would end flush with
the face, and M2 x 20 would stand 2 mm proud.

- The head pocket (`m2fastener.PocketTool`, radius 2.2, 2 deep) sinks each head flush in the +X side face.
- Each nut sits at the floor of a hexagonal pocket open to the -X side face. The pocket runs from x = -6 to -10, so
  the nut occupies x = -6.2 to -7.8 and 2 mm of empty pocket lies between it and the face. A pinch
  clamp's nut may sit in a pocket open to a face, because the clamp bolt holds it there (AGENTS.md "Fasteners"). The
  pin's cross-bolt nuts use the same pocket tool, `m2fastener.HexPocketTool` for a site and a depth, with the pocket's
  corners on the site's `Corner`. The clamp's `Corner` is +Z. The hexagon is centred on the bolt's axis with its
  corners on the circle of radius 2.51, one diagonal along Z, so a corner points up and one down and the flats are the
  +Y and -Y sides, 4.35 across. The nut's corners (radius 2.31) cannot pass the flats (2.17 from the axis), so the nut
  cannot turn. The corners stay sharp, 0.2 mm past the nut's corners, and the floor's outline is rounded by
  `NutPocketRound` (0.6) on the tool. The nut's 30° face chamfer clears the floor round, so the nut ends at least 0.2
  mm below the face.
- The hole (`m2fastener.HoleTool`, 2.6 across) runs from the head pocket across the slit and out through the nut
  pocket. The bolt's tip ends at x = -8, the nut cavity's far face, 2 mm inside the -X side face.
- The sides stay straight `ClampSideRun` (3.6) past the bolts' axis, so each side face keeps 1.4 mm of flat beyond the
  head pocket's ring and 1.09 mm beyond the nut pocket's corner before the corner arc begins.

The pocket's roof is two slopes that meet in the upper corner, each 30° above horizontal, or 60° from vertical. That is
steeper than the 45° overhang limit, so each slope counts as a bridge: the two slopes are 2.0 mm along the bolt and
2.17 mm across, well under the 6 mm a bridge may span. The arm prints without supports or a pause. The nuts go in
from the side at assembly.

| Wall round the clamp | 605ZZ | 608-2RS |
| --- | --- | --- |
| Hole to the bore, at the slit | 1.05 | 1.80 |
| Nut pocket to the bore | 4.04 | 3.29 |
| Nut pocket's floor to the outline | 4.00 | 4.00 |
| Nut pocket 1's corners to the back face | 1.49 | 1.49 |
| Between the two nut pockets' corners | 2.98 | 2.98 |
| Nut pocket 2's corners to the boss top | 2.49 | 2.49 |
| -X side face beyond the nut pocket's flats | 1.43 | 1.43 |
| Side face beyond the head pocket | 1.4 | 1.4 |
| Head pocket 1 to the back face / head pocket 2 to the boss top | 1.80 / 2.80 | 1.80 / 2.80 |
| Between the two head pockets | 3.60 | 3.60 |

The press-fit wall rule asks for 1.6 mm round the bore. The arm keeps it everywhere but the 1.05 mm web over the clamp
bolts' holes, at the slit, with the single D bore: the slit makes the hub a clamp, so the bore no longer carries a
pressed ring's hoop stress there. With the double flat the web is 1.80 mm, so the 608-2RS crank keeps 1.6 mm everywhere.
At the widest shaft the arm takes (16.7 mm), the nut pocket still keeps 3.34 mm to the bore.

### Clamp capacity

The arm grips the 8 mm printed PLA crank shaft over 15.75 mm (the bore is 16 mm deep, less its 0.5 mm rim chamfer
and the floor), with two opposite flats (`Params6082RS`) and two clamp bolts. The 3 N.m design torque needs 4.5 N.m
for a 1.5 times margin. The estimate does not reach that: it reaches about 2.4 N.m (1.4 to 3.2).

Symbols: shaft radius `r` = 4, flat depth `d`, flat half-width `w` with `w^2 = r^2 - (r - d)^2`, engagement less the rim
chamfer `L` (15.25 here, 6.25 for the old single bolt and single flat), bearing pressure `p`, friction `mu`, preload
`F` per bolt, `n` bolts, shaft shear strength `tau` and torsional section modulus `Z`.

- Form fit, rigid rotation, pressure rising linearly from the flat's centre to its corner on one half of each flat:
  `T_form = k p L w^2 / 3`, `k` = 2 for two flats, 1 for one. The slit removes the central 1.5 mm of the -Y flat,
  under 3% of its moment.
- Friction from the clamp, each bolt pressing both jaws on the shaft: `T_fric = mu * 2 n F * r`.
- Shaft torsion: `T_shaft = tau Z`, from a finite-difference solve of each section scaled so the 8 mm round reads its
  exact 100.5 mm^3: D section (0.75 flat) 87.4 mm^3, double flat 0.75: 79.2 mm^3, double flat 1.5: 52.6 mm^3. The
  double-flat neck where the shaft enters the hub carries the whole torque, and so does the D section between the hub
  and the crank gear.

| Case | `T_form` | `T_fric` | Joint | Neck `Z tau` | D body `Z tau` | System (min) | Range (low to high inputs) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Old: 1 flat 0.75, L 6.25, 1 bolt | 0.34 | 0.48 | 0.82 | n/a | 2.62 | 0.82 | 0.4 to 1.4 |
| 2 flats 1.5, L 15.25, 2 bolts | 2.97 | 0.96 | 3.93 | 1.58 | 2.62 | 1.58 | 1.05 to 2.10 |
| This design: 2 flats 0.75, L 15.25, 2 bolts | 1.66 | 0.96 | 2.62 | 2.38 | 2.62 | 2.38 | 1.43 to 3.17 |

All values in N.m. Inputs: `p` 30 MPa, `mu` 0.3, `F` 200 N, `tau` 30 MPa. Low: `p` 20, `mu` 0.2, `F` 100, `tau` 20.
High: `p` 40, `mu` 0.4, `F` 300, `tau` 40.

- Neither geometry reaches 4.5 N.m. The 0.75 mm flats reach about 2.4 N.m, about three times the old joint and near the
  3 N.m design torque at the high end. The 8 mm printed shaft limits it, not the hub.
- 1.5 mm flats make the hub stronger (3.9 N.m) but cut the shaft's neck to 1.6 N.m, so the system is weaker than with
  0.75 mm flats.
- 4.5 N.m needs a torque path from the hub to the crank gear of about 150 mm^3 section modulus in printed PLA, more than
  an 8 mm shaft has. A purchased 8 mm steel D shaft with a two-flat end at 1.5 mm and a hub of about 20 mm engagement
  would reach about 4.8 N.m, or a 10 mm printed shaft through 10 mm-bore bearings with a re-bored gearbox.
- The crank gear's D bore (6 mm wide, one 0.75 flat) carries the same torque and estimates about 0.33 N.m before
  press-fit friction. Once the hub holds, slip may move there.

Bench test that settles `p`, `mu`, `F` and `tau` together: lock the shaft stub (a printed crank shaft end in a printed
608 seat), fit the arm with both bolts tightened, and hang masses from the handle. 3 N.m is 5.6 kg at the 55 mm throw
and 4.5 N.m is 8.3 kg. Record the mass at slip or twist and which part gave. `TestCrankHandleGrip` pins the inputs that
come from the geometry: `L`, `w^2`, `k` and `n`.

## Split pin

The pin prints as two halves split by the plane through its axis and local X ([Split
shafts](../../docs/model-pipeline.md#split-shafts)). X is the direction of the hand's driving force, so that force
loads the halves side by side and not across the seam. The plane passes through two corners of the stub's hexagon.
`Parts.SplitNormal` is its normal, local +Y, along the arm toward the pin. The model keeps the pin whole; the export to
Fusion cuts it.

- The +Y half (away from the shaft) holds the cross bolts' head pockets. The -Y half holds their nuts in hexagonal
  pockets open to the -Y flat, each 2 mm deep. A diagonal of each hexagon runs across the pin: its corners point to
  ±X and its flats lie along the pin's axis. That leaves 0.73 mm of pin between each corner and the shank's round at
  the flat, under the 1 mm the fastener rule asks for ("Walls under 1 mm" below). The bolt holds each nut there.
- The two M2 x 6 cross bolts clamp the halves. One sits past the pin bolt's hole and one short of the head cone, so
  the halves act as one section over the span the hand bends.
- The seam cuts the pin bolt's hole and nut cavity in half. The clamped halves enclose its nut, and the stub's place in
  the arm's pocket closes the halves round it too.
- The pin bolt is still needed: it alone holds the pin in the arm, whose pocket opens through the arm's front face.
- Apart from the pockets and cavities round each cross bolt, the halves are mirror images. Each stands 6.45 mm above
  its cut face.

| Wall round a cross bolt | mm |
| --- | --- |
| Pin round the nut pocket's corners at the -Y flat, across the pin | 0.73 (exception, below) |
| Pin round the nut pocket's floor | 2.24 |
| Flat round the head pocket | 1.04 across, 1.05 along the pin |
| Pin under the head pocket to the seam | 2.0 |
| Pin between the nut pocket's floor and the seam | 2.0 |

### Walls under 1 mm

Confirm or override: the pin's wall round each cross nut pocket's corners at the -Y flat is 0.7344 mm, under the 1 mm
`AGENTS.md` asks for round any pocket. The maintainer chose the orientation with the corners across the pin, which
leaves the flats along the axis. With the corners along the axis the wall was 1.07 mm. The shank (radius 5.15) and the
flat (4.0 from the axis) set the wall at `sqrt(5.15^2 - 4^2) - 2.51`, and no dimension changes to raise it. The
shank's round leaves more pin deeper in the pocket, so the wall is least at the flat's face, and the bolt and the
nut hold the corner region closed. The test pins the value (`TestCrankHandleFasteners`).

## Part fits

| Fit | Class | Clearance |
| --- | --- | --- |
| Bore round the shaft | Press fit, the arm turns with the shaft | `c` = 0.05 radial, on the round and on the flat (both flats with `DoubleFlat`, the 608-2RS crank's bore) |
| Pin stub in the arm's pocket | Tight | 0.25 on each flat |
| Handle on the pin's shank | Regular | 0.40 radial (bore 5.55, shank 5.15) |
| Handle's inner end to the arm's front face | Regular | 0.40 |
| Handle's bore cone to the pin's head cone | Regular | 0.40 normal to the cones |
| Handle's counterbore round the pin's head | | 0.6 radial, 0.4 above the head |
| Bolts, nuts | Fastener rules | head pocket 0.3, hole 0.3, cavity 0.2 |
| Clamp nuts and cross nuts in their hexagonal pockets | Fastener rules | 0.17 at each flat, 0.2 past each corner, 0.2 to the floor |

The pin bolt's nut cavity has 0.55 mm rounds at its end corners. At the nut's faces they leave 0.07 mm of the 0.2 mm
radial clearance past the nut's corners.

The handle is a solid of revolution about the pin axis, so it turns freely when every gap in a cut through that axis
clears its class. The pin's head (radius 5.8) is wider than the handle's bore (radius 4.9), so the handle cannot slide
off the pin. The model is **Mixed**: the bore fit is adjustable and the bolt and nut keep their nominal size.

The arm presses onto the shaft, and the clamp bolts close the slit to pinch the bore on it. Nothing holds it along the
shaft but that grip ("Clamp capacity").

## Valid range

- `ShaftDiameter` up to 16.7 at the default clearance; `FlatDepth` up to `D/4`; `BoreClearance` 0.05 to 0.5.
- `Throw` from 35.29 up, so the handle and the pin pocket clear the ramp.

## Representative cases

- `605ZZ`: the default.
- `608-2RS`: the siren's crank.
- `regular-bore`: `Params605ZZ` with `BoreClearance` 0.35.
- `long-throw`: `Params6082RS` with `Throw` 80.
- Placed: the frame with origin `(10, -5, 7)`, U `(0, 1, 0)` and V `(0, 0, 1)`.

## Retained sharp corners and edges

The rule is "Corners and roots" in `docs/model-pipeline.md`. The arm prints on its back face, each half of the pin on
its cut face, and the handle upright on its inner end. In the pin's halves the 30° cones join the stub, the shank and
the head without a sharp step.

| Part | Corner or edge | Reason |
| --- | --- | --- |
| Arm | The six corners of the hexagonal pocket | Joinery: they match the sharp corners of the pin's stub. |
| Arm | The corners of the bore: two (605ZZ) or four, a flat's two ends on each side (608-2RS) | Mating corners of the shaft's flats. They turn 36.5 degrees (605ZZ) and 35.4 (608-2RS), so they need no round. |
| Arm | Each clamp bolt's head pocket floor rim, 2 mm into the +X side face | The 0.3 mm head clearance leaves no room to round it: a round would sit under the head. |
| Arm | The slit's outline in the back face and in the boss top, a run of two walls and the end round | Not chamfered: the slit is the 1.5 mm clamp gap, and a 0.5 mm chamfer each side would open its mouth to 2.5 mm. |
| Arm | The ramp foot, where the ramp meets the 8 mm outer face | A concave edge: an inside turn of 150°, not a sharp outside edge. |
| Arm | The ramp's two edges with the side faces, 18 mm runs | The ramp face is 360 mm², under the 400 mm² large-face limit of "Surface edge finishing". |
| Arm | The six corners of each clamp nut's hexagonal pocket, edges along the clamp bolt in the -X side face | Mating corners: they match the sharp corners of the M2 nut, as the stub pocket's corners match the stub. They run parallel to the bed, so `cornercheck`'s layer sections do not read them. |
| Arm | The 0.6 mm round of clamp pocket 1's floor outline, where a layer cuts it at a slant | The pocket's slopes lie 30° off the bed, so a layer section chords the round into turns of 51.6° to 54.4° at the slopes. The round is the one "Corners and roots" asks for; `TestCrankHandleCorners` exempts only that floor band. No layer reads pocket 2's round. |
| Arm | The 1.05 mm web between the 605ZZ bore and the clamp bolts' holes, at the slit | Under the 1.6 mm press-fit wall: the slit makes the hub a clamp, so the bore carries no pressed ring's hoop stress there ("Clamp" above). The 608-2RS web is 1.80 mm. |
| Pin | The six corners of each cross nut's hex pocket, edges along the cross bolt | Mating corners: they match the M2 nut's sharp corners. Their direction is across the pin (30°, 90°, ... from local Z). |

The pin carries the handle beside the arm's front face without a collar, as on the siren. [Arbor
collars](../../docs/model-pipeline.md#arbor-collars) asks for a collar there; the siren's crank predates the rule.

## Printing

Print the arm on its back face and the handle upright on its inner end, with no supports and no pause. The boss rises
from the outer face and its ramp faces upward, so it needs no support, and the bore's floor (z 16 to 17) bridges the
bore. The clamp nuts go into their pockets at assembly. Print the pin as its two halves, each lying on its cut face, with no supports and no
pause. The half grooves of the pin bolt's hole and nut cavity on the cut face are short bridges, at most 3.6 mm across.
The -Y half prints with the pockets opening upward, and the cross nuts go in at assembly.

To assemble:

1. Drop the pin bolt's M2 nut into its half pocket in one half, and lay the other half on it.
2. Set an M2 nut in each hexagonal pocket of the -Y half, push each M2 x 6 cross bolt through its head pocket in the
   +Y half into its nut, and tighten it.
3. Slide the handle onto the pin's shank from the stub end.
4. Push the stub into the arm's pocket and drive the M2 x 10 bolt in from the arm's back face.
5. Press the arm onto the shaft, set an M2 nut in each hexagonal pocket in the arm's -X side, push an M2 x 16 clamp
   bolt in from the +X side into each, and tighten both.

## Checks and images

Check every case above: every sketch fully constrained, the thirteen bodies and their containers, the bore's clearance
round the nominal shaft, its floor and wall, the slit joining the bore to the end arc and ending in the floor, every
fit between the printed parts with the handle's gaps in a cut through the pin axis, each bolt's head, tip and nut
against its pockets with no overlap, the walls round the cross bolts and round the clamp, the face and ramp chamfers round the
whole outline, the clamp's grip (`TestCrankHandleGrip`), `internal/splitcheck` on the pin with its split normal and the cross bolts' lines exempt, and
`internal/cornercheck` on the arm and handle (axis +Z) and on the pin (axis `SplitNormal`). Check the default's bounds
and `VerifyAll` meshes, and the placed crank against the local one. The siren's own tests check its crank through the assembly.
