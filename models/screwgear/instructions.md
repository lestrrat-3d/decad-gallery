# Screw gear construction instructions

Use this contract to derive [steps.md](steps.md), then generate the Go package and tests. This file owns
geometry, dimensions, input rules, and case inputs. The steps own interfaces, Sketch/Decad calls, and
acceptance assertions.

The contract describes the geometry the Go package builds. It follows the Fusion add-in's screw gear
specification for the shape and the numbers. It leaves out that specification's print history, Fusion API
calls, dialog layout, anchor-sketch and construction-plane mechanics, and bore identification marks. The
Fusion add-in's print diagnostics and the marks it cuts into the sleeve are out of scope for this model.

## What the mechanism is

A screw gear pair has three bodies:

- Gear A and gear B. Each is a flat ribbon with a cosine-toothed edge, twisted about its own centre line into a
  helix. The two are the same part.
- The cage. It is one thick-walled tube, called the sleeve, with four twisted bores cut through its wall and up
  to two windows cut through its wall between the bores.

Each ribbon passes through the sleeve's wall twice, once on each side of the tube, through a bore. A bore is the
ribbon's crest rectangle plus a clearance, twisted at the ribbon's own lead. A ribbon can advance through its bores
only by turning at the rate the bore twists, so each gear moves by a screw motion: a rotation about its axis
combined with a translation along that axis. The two toothed edges meet in the hollow of the sleeve. Pushing one
gear along its axis moves the other by the same tooth pitch, a 1:1 ratio.

## Units and frame

Use millimeters for lengths and radians for angles. The mechanism is anchored by one `r3.Frame` called the
placement:

| Symbol | Placement part | Meaning |
| --- | --- | --- |
| `C` | origin | Centre of the mechanism |
| `ê` | `U` axis | Reference direction in the mechanism's mid plane |
| `k̂` | `V` axis | `n̂ × ê`, level and square to `ê` |
| `n̂` | normal `U × V` | Common perpendicular of the two gear axes; the sleeve's axis |

All construction below is stated in the pair frame: `C` is the origin, `x` runs along `ê`, `y` along `k̂` and
`z` along `n̂`. With no placement (nil), the pair frame is the world XY frame at the origin. A non-nil placement
moves every body onto the placement after the last boolean operation, and builds nothing differently.

## Inputs

| Symbol | Parameter | Type | Default | Meaning |
| --- | --- | --- | --- | --- |
| `W` | `Width` | float64 | 15 | Ribbon width across the plate |
| `N` | `ToothCount` | int | 68 | Teeth along one ribbon |
| — | `TwistLead` | float64 | 49.5 | Advance along the axis per full turn |
| `T` | `Thickness` | float64 | 3.75 | Ribbon thickness |
| `P` | `ToothPitch` | float64 | 2.625 | Tooth pitch along the axis |
| `H` | `ToothHeight` | float64 | 2.625 | Crest to root depth of the toothed edge |
| `Slant` | `ToothSlant` | float64 | `25.8π/180` | Lean of the tooth ridges across the thickness (rad) |
| `Bow` | `ToothBow` | float64 | 0.048 | Drop of the toothed edge by `Bow·v²`, in mm per mm² |
| — | `CageRadius` | float64 | 15.5 | Where each bore sits on its gear's axis; middle of the sleeve wall |
| — | `CageRise` | float64 | 18.75 | Half the sleeve's height, to its flat end faces |
| — | `Clearance` | float64 | 0.20 | Added all round every bore |
| `a` | `RoofAllowance` | float64 | 0.60 | Added to the clearance on the roof face of every bore |
| — | `CollarHalf` | float64 | 3 | Half the sleeve wall's thickness |
| — | `CollarWall` | float64 | 3 | Least material left round a bore or window |
| `Σ` | `CrossAngle` | float64 | `80π/180` | Angle between the two gear axes (rad) |
| — | `Engagement` | float64 | 1.15 | How far the crests overlap |
| `Φ_A` | `MountAngleA` | float64 | 0 | Gear A's cross-section angle where the axes cross (rad) |
| `Φ_B` | `MountAngleB` | float64 | 0 | Gear B's cross-section angle where the axes cross (rad) |
| — | `AssemblyPhase` | float64 | −1.31 | Tooth phase of gear B when gear A is at zero |
| — | `Placement` | `*r3.Frame` | nil | Pair frame; nil selects the world XY frame at the origin |

Do not interpret zero fields as default requests. Provide defaults through `DefaultParams`. A nil `Placement` is
the one exception: it means the world frame.

`ToothSlant` and `ToothBow` are fitted to the default `CrossAngle` and `TwistLead`. At other values the checks
below accept the input, but the tooth contact between the two gears has not been measured.

## Derived values

Compute without rounding. Values at the defaults are in the last column.

| Value | Formula | At defaults |
| --- | --- | --- |
| Screw parameter `Λ` | `TwistLead / 2π` | 7.878 mm per rad |
| Helix angle `β` | `atan(πW / TwistLead)` | 43.6° |
| Axes distance `A` | `W − Engagement` | 13.85 |
| Ribbon length `L` | `N·P` | 178.5 |
| Travel | `L − 2(CageRadius + CollarHalf)` | 141.5 |
| Twist per tooth | `P / Λ` | 19.1° |
| Inner radius `Ri` | `CageRadius − CollarHalf` | 12.5 |
| Outer radius `Ro` | `CageRadius + CollarHalf` | 18.5 |
| Bore half width `hw` | `W/2 + Clearance` | 7.7 |
| Bore half thickness `ht` | `T/2 + Clearance` | 2.075 |
| Corner radius `c` | `hypot(hw, ht + a)` | 8.151 |
| Cut start `sIn` | `sqrt(Ri² − c²) − 1` | 8.477 |
| Cut end `sOut` | `Ro + 1` | 19.5 |
| Axial window | `1.5·sqrt(W² − A²) / sin Σ`, or `W` when `A ≥ W` | 8.77 |
| Mesh footprint radius | `hypot(axial window, hypot(W/2, T/2))` | 11.69 |

`tan(Slant)` is 0.4834 and the toothed edge drops `Bow·(T/2)²` = 0.169 mm at the faces at the defaults.

## Ribbon geometry

Each gear has its own frame. `s` runs along its axis, `u` across the plate's width and `v` across its thickness.
Before the twist the ribbon occupies

```
-W/2 <= u <= Utooth(v, s),    |v| <= T/2
Utooth(v, s) = W/2 - H/2 + (H/2)·cos(2π (s + tan(Slant)·v − Z) / P) − Bow·v²
```

`Z` is the tooth phase. The toothed edge is a plain cosine with no flat crest and no flat root. On the mid plane
`v = 0` its crest is at `u = W/2` and its root at `u = W/2 − H`. The back edge stays at `u = −W/2` and the faces
stay at `v = ±T/2` at every station and every phase. The ridges are the lines of constant `s + tan(Slant)·v`:
across the thickness a ridge moves along the axis by `−tan(Slant)` per unit of `v`. The bow lowers the edge by
`Bow·v²` so the ridge runs straight after the twist. The sign of `Slant` is the pair frame's.

The twist rotates the cross-section at station `s` about the axis by `θ(s) = s/Λ + Φ`. Both gears have the same
hand. The ribbon runs `L` long, centred on station `Z` of its own axis: from `Z − L/2` to `Z + L/2`.

The screw motion is a shift of `Z` and nothing else. Advancing the gear by `dz` along its axis translates it by
`dz` and rotates it by `dz/Λ`. The twisted ribbon is unchanged by exactly that motion, so only the tooth phase
changes in the ribbon's own frame.

The assembly phase is the tooth phase gear B is built at while gear A is at zero. Gear B's ribbon is gear A's
advanced by `AssemblyPhase` along gear B's axis, so at the defaults gear B runs from station −90.56 to +87.94
while gear A runs ±89.25.

The crest rectangle, `W` by `T`, holds every point of the ribbon at every station and every phase. The bores are
cut to that rectangle and never to a tooth.

## Pair geometry

The gear axes lie in two parallel planes `A` apart, crossing at angle `Σ` when seen along `n̂`:

| Item | Gear A | Gear B |
| --- | --- | --- |
| Axis origin | `C − (A/2)·n̂`, pair frame `(0, 0, −A/2)` | `C + (A/2)·n̂`, pair frame `(0, 0, +A/2)` |
| Axis direction `ẑ_g` | `(cos Σ/2, sin Σ/2, 0)` | `(cos Σ/2, −sin Σ/2, 0)` |
| Section `û_g` | `+n̂` | `−n̂` |
| Section `v̂_g` | `ẑ_g × û_g` | `ẑ_g × û_g` |
| Mounting angle `Φ` | `MountAngleA` | `MountAngleB` |
| Tooth phase `Z` | 0 | `AssemblyPhase` |

`û_g` points at the other gear. That makes `Φ` mean the same thing for both members, so the two gears are the
same part. A point of gear `g` at station `s` with section coordinates `(u, v)` is

```
origin_g + s·ẑ_g + (u cos θ − v sin θ)·û_g + (u sin θ + v cos θ)·v̂_g,    θ = s/Λ + Φ_g
```

The pair is not built at the crossed-helical rule `Σ = 2β` (87.2° at the defaults). `Σ` comes from `CrossAngle`
and nothing derives from `β`.

## The sleeve

The sleeve is a tube about `n̂` through `C`. It runs from radius `Ri` to radius `Ro` and from `−CageRise` to
`+CageRise` along `n̂`: 37.5 mm tall with a 584.3 mm² ring at each end at the defaults. Four bores and up to two
windows are cut from it.

### The bores

Each gear has two bores. The `+R` bore is where the gear's axis crosses the circle of radius `CageRadius` at
station `+CageRadius`, and the `−R` bore is at station `−CageRadius`. Each bore is the crest rectangle plus the
clearance all round, with the roof allowance added to one long face, turned at every station to the ribbon's own
angle `θ(s)`.

The bore's opening in the gear's own section runs `u` from `−hw` to `hw` and `v` from `vLo` to `vHi`:

- When the roof face is the `+v` face, `vLo = −ht` and `vHi = ht + a`.
- Otherwise `vLo = −ht − a` and `vHi = ht`.

The opening is 15.4 by 4.75 mm at the defaults. The roof face is the long face that is up when the sleeve stands
on its `−n̂` end at the bore's crossing station `sc = sign·CageRadius`. It is the `+v` face when
`−sin θ(sc)·(û_g·n̂) + cos θ(sc)·(v̂_g·n̂)` is positive, and the `−v` face otherwise. `v̂_g·n̂` is zero for these
gears. At the default mounting angles the roof is the `+v` face for gear A's `−R` bore and gear B's `+R` bore,
and the `−v` face for gear A's `+R` bore and gear B's `−R` bore.

The cut spans station `sIn` to `sOut` of the gear's axis for the `+R` bore and `−sOut` to `−sIn` for the `−R`
bore. The inner face of the sleeve is a cylinder, so the opening's corner reaches into the wall before its centre
line does. A corner first touches the inner face at station `sqrt(Ri² − c²)`. The cut starts a millimeter before
that, where the whole section is in the hollow, and ends a millimeter past `Ro`, where the whole section is
outside. The cut turns through `(sOut − sIn)/Λ`, about 80° at the defaults.

The bores are cut in this order: gear A `−R`, gear A `+R`, gear B `−R`, gear B `+R`. The bore must twist at the
ribbon's lead: a straight hole would bind, since over the wall's `2·CollarHalf` the ribbon's corner sweeps
`(W/2)·(2·CollarHalf)/Λ` = 5.7 mm across the opening against 0.20 mm of clearance.

### The wall between the bores

Round the tube the bores alternate between the gears. The four gaps between neighbours face `+k̂` (gear A `+R`
and gear B `−R`), `−ê` (gear A `−R` and gear B `−R`), `−k̂` (gear A `−R` and gear B `+R`) and `+ê` (gear A `+R`
and gear B `+R`). The separation of a gap is measured from the bores' outlines:

1. For each bore, take stations every 0.1 mm from `sign·sIn` outward while `|s| ≤ sOut`. At each station take
   seventeen points on each side of the opening rectangle, `(hw, v)` and `(−hw, v)` for
   `v = vLo + (vHi − vLo)·i/16`, and `(u, vHi)` and `(u, vLo)` for `u = −hw + 2hw·i/16`, with `i = 0…16`. Turn
   them to the station's angle and place them in the pair frame. Keep a point when its distance from the sleeve
   axis lies within `Ri − 0.5` to `Ro + 0.5`.
2. For a gap facing `d`, let `across = n̂ × d`. Project the kept points of the gap's two bores to
   `((P − C)·across, (P − C)·n̂)` and take the convex hull of each bore's projections.
3. For the unit normal `m` of every edge of either hull, the separation along `m` is the larger of
   `min(m·q) − max(m·p)` and `min(m·p) − max(m·q)`, with `p` over the first hull's corners and `q` over the
   second's. The gap's separation is the largest over all edges. It is negative when the hulls overlap.

The sleeve's separation is the least of the four gaps. It is 5.310 mm at the defaults, across the `−ê` gap
between the two `−R` bores. The input checks require it to be at least `CollarWall`.

### The windows

A window is a prism: a hexagon drawn on the plane through the sleeve axis square to the direction it faces,
pushed straight out through the wall on that side for `Ro + 1` mm. On the plane, `t` runs across the window
(along `across = n̂ × d`) and `z` runs up `n̂`. A point `P` has `t = (P − C)·across`, `z = (P − C)·n̂` and depth
`a = (P − C)·d`.

There are two windows, one per facing. They face `+k̂` then `−k̂` when `Σ ≤ 90°`, and `+ê` then `−ê` when
`Σ > 90°`. These are the two wider gaps between neighbouring bores, `180° − Σ` wide against `Σ` across `±ê`.

The hexagon is the band `lo ≤ z + lean·t ≤ hi`, cut off by the trims `bottom ≤ z − lean·t ≤ top` and by the
upright ends `t = left` and `t = right`. Every edge is upright or at 45°. The search finds each number from the
sleeve's own geometry:

1. The two bores whose crossings have `(crossing − C)·d > 0` flank the window. The other two are the far bores.
   The low flanking bore has the lower crossing along `n̂`. `lean` is `+1` when the high bore's crossing has the
   larger `t`, and `−1` otherwise.
2. For each flanking bore, walk its cut span at stations every 0.001 mm from the span's lower end. Take every
   corner of both pieces of its section in the wall. The section in the wall is the opening rectangle turned to
   the station's angle, clipped to the heights within `±CageRise` and to the strips of `y` between `near =
   sqrt(max(0, Ri² − s²))` and `far = sqrt(Ro² − s²)` on each side of the gear's axis. Let `lowReach` be the
   largest `z + lean·t` of the low bore's corners and `highReach` the least of the high bore's. Then
   `lo = lowReach + √2·CollarWall` and `hi = highReach − √2·CollarWall`.
3. `zLimit` is the largest distance from the mid plane reached by any channel in the wall, over all four bores,
   at stations every 0.01 mm from `sign·sIn` outward while `|s| ≤ sOut`, where the section reaches the wall. Then
   `top = min(2·zLimit − hi, hi + √2·Ri)` and `bottom = max(−2·zLimit − lo, lo − √2·Ri)`.
4. Each end stands as far out as the far bores allow. For `right`, bisect `te` 24 times between 0 and
   `min(Ri, Ro/√2)·(1 − 10⁻⁹)`. Keep the middle as the lower end when it is clear and as the upper end otherwise.
   Do the same on the other side for `left` and negate. A candidate `te` at `t = ±te` is clear when `zLow ≤ zHigh`
   (with `zLow = max(lo − lean·t, bottom + lean·t)` and `zHigh = min(hi − lean·t, top + lean·t)`) and every
   sampled point keeps at least `need = CollarWall + 0.1/√2 + 0.005` mm (3.076 mm at the defaults) from both far
   bores' channels in the wall. The sampled points are the end's two corner lines through the wall at
   `z = zLow` and `z = zHigh`, every 0.1 mm from the inner face `a0 = sqrt(max(0, Ri² − t²))` to the outer face
   `a1 = sqrt(max(0, Ro² − t²))`, and the end's upright edge on each face at `n` equal steps with
   `n = ceil((zHigh − zLow)/0.1)` and at heights `−zLimit + j·0.1` that lie strictly between `zLow` and `zHigh`.
5. The hexagon is the square `−2Ro ≤ t, z ≤ 2Ro` clipped in this order to `lean·t + z ≤ hi`,
   `−lean·t − z ≤ −lo`, `−lean·t + z ≤ top`, `lean·t − z ≤ −bottom`, `t ≤ right` and `−t ≤ −left`. A corner within
   0.001 mm of the one before it is dropped, and so is the last when it is that close to the first.

The distance from a point to a bore's channel in the wall is the least, over the bore's stations, of the
distance along the axis to the station's plane combined with the distance within that plane to the section's
pieces. The stations are every 0.002 mm across the span, and every 0.0001 mm between two neighbours where a piece
appears or vanishes. A point farther than `c` plus the search reach from the axis is not walked.

A window is cut only when it has room. It has none when `hi ≤ lo` (the flanking bores leave no band) or when
`right ≤ left`, the hexagon has fewer than three corners, or its area is not positive (the far bores leave the
band no length). A window without room is left out of the sleeve, and the build does not report it.

## Input checks and limits

`Validate` applies these checks in this order and returns the first failure. The error names the offending input
by its dialog id and its bound.

1. Reject a non-nil `Placement` that is not a valid frame.
2. Reject a non-finite value in any numeric field, checked in this order: `ribbonWidth`, `ribbonThickness`,
   `toothPitch`, `twistLead`, `collarHalf`, `collarWall`, `clearance`, `roofAllowance`, `toothHeight`,
   `toothSlant`, `toothBow`, `engagement`, `crossAngle`, `assemblyPhase`, `cageRadius`, `cageRise`,
   `mountAngleA`, `mountAngleB`.
3. Reject `W`, `T`, `P`, `TwistLead`, `CollarHalf`, `CollarWall` or `Clearance` that is not greater than 0, in
   that order.
4. Reject `RoofAllowance < 0`. Zero is allowed and cuts every bore to the clearance alone.
5. Reject `ToothCount < 4`.
6. Reject `ToothHeight <= 0` or `ToothHeight >= W/2`.
7. Reject `ToothSlant` outside the open interval (−π/2, π/2).
8. Reject `ToothBow < 0`.
9. Reject `ToothHeight + ToothBow·(T/2)² >= W/2`. The error names `toothBow`.
10. Reject `Engagement <= 0` or `Engagement > ToothHeight`.
11. Reject `CrossAngle` outside the open interval (0, π).
12. Reject `|AssemblyPhase| >= ToothPitch`.
13. Reject `CageRadius + CollarHalf + 1 >= L/2`, so both bores' cuts lie within the ribbon. The error names
    `cageRadius`.
14. Reject `hypot(c, 1) >= Ri`, so each cut starts in the hollow (`sIn > 0`). The error names `cageRadius`.
15. Reject `meshFootprintRadius + Clearance > Ri`, so the engaged zone lies inside the hollow. The error names
    `cageRadius`.
16. Reject `CageRise < A/2 + c + CollarWall`, so the channels keep `CollarWall` from the end faces. The error
    names `cageRise`.
17. Reject a wall separation (see "The wall between the bores") below `CollarWall`. The error names `collarWall`,
    the two bores and the separation.

No range is enforced on `MountAngleA`, `MountAngleB`, `TwistLead` above 0, or `CrossAngle` away from the open
interval. Nothing is asserted about tooth contact away from the fitted defaults.

## Sketch constraints

Every sketch the build draws is solved and verified before its geometry is consumed. Verification requires
`sketch.FullyConstrained`, `DOF == 0`, no redundant constraints, no conflicts, and a passing report check.

| Sketch | Plane | Anchor | Constraints |
| --- | --- | --- | --- |
| Ribbon section, one per station | Gear's own XY plane offset by `s` along Z | Sketch origin, on the twist axis | Horizontal and vertical distance from the origin on each of the 13 points |
| Tube | World XY plane | Sketch origin at `C` | Radius `Ri` on one circle and `Ro` on the other, both centred on the origin |
| Bore section, one per bore | World XY plane | Sketch origin on the gear axis | Horizontal and vertical distance from the origin on each of the 4 corners |
| Window, one per window with room | Plane with `U = across`, `V = n̂` through `C` | Sketch origin at `C` | Horizontal and vertical distance from the origin on each corner |

Test this requirement for every parameter branch: a successful `Build` has verified each sketch, and the branch
cases below each build.

## Feature order

1. Loft one ribbon for gear A and one for gear B through `10·N + 1` sections each, in the gear's own frame, then
   move each onto its gear frame in the pair frame.
2. Extrude the tube ring and move it to span `−CageRise` to `+CageRise`.
3. Cut the four bore tools from the tube in the order gear A `−R`, gear A `+R`, gear B `−R`, gear B `+R`. Each
   tool is the opening rectangle at the cut's first station, swept along the gear's axis for `sOut − sIn` with a
   twist of `(sOut − sIn)/Λ`, then moved onto the gear's axis.
4. Cut each window that has room, the one facing `d` before the one facing `−d`.
5. Move gear A, gear B and the cage onto `Placement`.

The ribbons are never operands of a boolean. They pass through the channels and the cuts leave them whole. Every
cut leaves exactly one cage body.

## Acceptance cases

| Case | Input | What it must show |
| --- | --- | --- |
| Defaults | `DefaultParams()` | Three solid bodies. Derived values as in the table above. Cage volume 17,088 mm³ within the reported bound, which is under 1% of it. Each ribbon's volume within the reported bound (under 3% of it) of the integral of the section area over the ribbon length. Cage bounds `±CageRise` along Z and `Ro` along X |
| No roof allowance | `RoofAllowance = 0` | Both windows have room. Cage volume between 15,500 and 19,500 mm³ |
| Windows face ê | `CrossAngle = 100°` | First window faces `+ê`, both have room. Same volume range |
| Mounted | `MountAngleA = 5°`, `MountAngleB = −5°` | First window faces `+k̂`, both have room. Same volume range |
| Placed | Frame at `(10, −5, 7)` with `U = +Y`, `V = +Z`, normal `+X` | Cage bounds `10 ± CageRise` along X, `−5 ± Ro` along Y and `7 ± Ro` along Z. Cage volume 17,126 mm³ within the reported bound plus 50 |
| Bore clearance | Default pair, 16 teeth | Distance from each bore wall to the ribbon at the crossing station is 0.19 ± 0.03 mm at the crest, the back edge and the non-roof thickness face, and 0.48 ± 0.05 mm at the roof face |
| Invalid | `Clearance = 0` | Error naming `clearance`. A nil context also fails |

The window search at the defaults, both windows: `lo` −4.537, `hi` 5.130, `bottom` −22.215, `top` 22.808,
`left` −12.121, `right` 12.172, `zLimit` 14.23, area 213.0 mm², `lean` +1. The six corners in `(t, z)`, in the
order the clipping produces them, are (12.17, −7.04), (−8.84, 13.97), (−12.12, 10.69), (−12.12, 7.58),
(8.84, −13.38) and (12.17, −10.04). The long sides are 6.84 mm apart across the band, the ends are 24.29 mm
apart, and the corners reach 13.97 mm from the mid plane against the channels' 14.23 mm. With `CollarWall = 3.5` the
band `hi − lo` narrows by `√2` mm. With `CollarWall = 7` the flanking bores leave no band and neither window is cut.

The pair frame at the defaults: gear A's origin is (0, 0, −6.925) and gear B's is (0, 0, +6.925). The axes are
(cos 40°, ±sin 40°, 0). The angle between them is `Σ`. Gear A's `û` is +Z and gear B's is −Z.

## Verification

- `Params.Validate` rejects each case in the input checks and reports the earlier check when an input fails two.
- `Build` verifies every sketch it draws.
- The cage is one solid and its volume is within the bound of the Fusion proof's 17,088 mm³ voxel count.
- The distance from the cage to points on each ribbon at the bores' crossing stations is measured with
  `DistanceToPoint`. The ribbons are not compared with the cage by a boolean or by `decad.Verify` clearance
  checks (see [steps.md](steps.md)).
- A placed mechanism's cage volume agrees with the unplaced one to within the reported bound plus 50 mm³, and its
  bounds are the unplaced bounds moved onto the frame.

### Section check

Cut both ribbons with planes parallel to the mid-plane between the gear axes, at −0.4, 0 and +0.4 mm from it along
`n̂`. On each plane the two section outlines do not overlap, and the proven least gap between them is at least
0.55 mm for the default inputs. A plane 1 mm toward gear B cuts only gear B, because gear A's crests reach 0.575 mm
across the mid-plane.
