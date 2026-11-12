# Spiral spring

## Purpose and source

Construct a flat spiral (clock or motor) spring for wind-up mechanisms, as one printable solid. The caller
sets the relaxed (unwound) outer diameter, the strip thickness and height, and the placement of the spring
axis. The inner end is a hub with a D bore that fits an arbor. The outer end is a straight tail ending in a
round eye that hooks over a housing pin.

The model is the spring only. The arbor, the barrel or housing, and the anchor pin are not modeled. Torque,
fatigue and creep are not simulated.

All dimensions are in millimeters.

## Symbols

| Symbol | Meaning | Formula |
| --- | --- | --- |
| `R` | hub radius | `HubDiameter/2` |
| `R_c` | coil radius | `OuterDiameter/2` |
| `t`, `h` | strip thickness, strip height | inputs |
| `N_max` | turns that fit | `floor((R_c - R)/(t + MinGap) + 1e-9) - 1` |
| `N` | turns | `Turns` if above 0, else `N_max` |
| `p` | pitch per turn | `(R_c - R)/(N + 1)` |
| `g` | gap between turns | `p - t` |
| `d` | center offset `|AB|` | `p/2` |
| `rho0` | inner-edge radius of half turn 0 | `R + g + d` |
| `r_k` | inner-edge radius of half turn `k`, `k = 0..2N-1` | `rho0 + k*d`; outer edge is `r_k + t` |
| `w` | spoke width | `2t` |
| `r_b` | bore radius | `ArborDiameter/2 + BoreClearance` |
| `f` | bore flat distance from the axis (flat bore only) | `ArborDiameter/2 - ArborFlatDepth + BoreClearance` |
| `D_hole` | eye hole diameter | `AnchorPinDiameter + 2*AnchorClearance` |
| `R_e` | eye outer radius | `D_hole/2 + t` |
| `rho_prev` | outer radius of half turn `2N-2` (about `B`) | `r_{2N-2} + t` |
| `L` | eye center depth below the X axis | `max(2*R_e, sqrt(max(0, (rho_prev + g + R_e)^2 - (r_{2N-1} + t/2 + d)^2)))` |
| `E` | eye center | `(-(r_{2N-1} + t/2), -L)` |
| `r_f` | 3D fillet radius | `min(t/2, g/4)` |

`ActiveLength` is the centerline length of the `2N` arcs: `pi*(2N*(rho0 + t/2) + d*N*(2N - 1))`.
`WindStrain` estimates the surface strain when the first turn wraps the hub:
`(t/2)*(1/(R + t/2) - 1/(R + g + t/2))`.

## API

Package: `models/spiralspring`. Exported surface:

- Constants `ThicknessFloor = 1.5`, `GapFloor = 1.0`, `ClearanceFloor = 0.2` and `MaxTurns = 40`.
- `Params` with the float fields `OuterDiameter`, `HubDiameter`, `Thickness`, `Height`, `MinGap`,
  `ArborDiameter`, `ArborFlatDepth`, `BoreClearance`, `AnchorPinDiameter`, `AnchorClearance`, the integer
  `Turns` (0 selects as many turns as fit) and `Placement *r3.Frame` (nil selects the local frame).
- `DefaultParams() Params`.
- `Params.Validate() error` and `Params.Derive() (Dimensions, error)`.
- `Dimensions` with these fields, each holding the symbol shown: `Turns` (`N`), `Pitch` (`p`), `Gap` (`g`),
  `CenterOffset` (`d`), `HubRadius` (`R`), `CoilRadius` (`R_c`), `FirstInnerRadius` (`rho0`), `SpokeWidth` (`w`),
  `BoreRadius` (`r_b`), `BoreFlatDistance` (`f`, 0 when the bore is round), `EyeCenterX` and `EyeCenterY` (the
  coordinates of `E`, so `EyeCenterY = -L`), `EyeRadius` (`R_e`), `EyeHoleDiameter` (`D_hole`), `TailLength`
  (`L` itself, not the straight part of the tail), `FilletRadius` (`r_f`), `ActiveLength` and `WindStrain`.
  `Dimensions` has no thickness or height field; those stay in `Params`.
- `Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)`. `Model` exposes
  `Body *decad.Body`, `Profile *sketch.Sketch` and `Dimensions Dimensions`. Build adds exactly one body.

Defaults: `OuterDiameter` 60, `HubDiameter` 12, `Thickness` 1.6, `Height` 8, `MinGap` 2.0, `Turns` 0,
`ArborDiameter` 5, `ArborFlatDepth` 0.5, `BoreClearance` 0.2, `AnchorPinDiameter` 3, `AnchorClearance` 0.35,
nil `Placement`.

Default derived values: `N = 5`, `p = 4.0`, `g = 2.4`, `d = 2.0`, `rho0 = 10.4`, `r_9 = 28.4`, `w = 3.2`,
`r_b = 2.7`, `f = 2.2`, `D_hole = 3.7`, `R_e = 3.45`, `L = 13.129`, `E = (-29.2, -13.129)`, `r_f = 0.6`,
`ActiveLength` 634.6, `WindStrain` 0.0307.

## Validation

Every message starts with `spiralspring: `. Validate checks in this order and returns the first failure:

1. Every float field is finite: `<Field> must be finite`.
2. `Thickness >= ThicknessFloor`: `Thickness must be at least 1.5 mm`.
3. `MinGap >= GapFloor`.
4. `Height >= 2*Thickness`, which keeps turns from sliding axially past each other.
5. `ArborDiameter > 0` and `AnchorPinDiameter > 0`.
6. `0 <= ArborFlatDepth <= ArborDiameter/4`.
7. `BoreClearance >= ClearanceFloor` and `AnchorClearance >= ClearanceFloor`.
8. `R - r_b >= ThicknessFloor`, the hub wall.
9. `R >= 2.5*Thickness`, so the spoke and its fillet fit on the hub.
10. `OuterDiameter > HubDiameter`, and `Turns >= 0`.
11. `N_max >= 1`: `OuterDiameter leaves no room for one turn`.
12. `Turns > 0` and `Turns > N_max`: `Turns %d needs a gap below MinGap; at most %d turns fit`.
13. `N <= MaxTurns`, the limit of the sketch's numerical conditioning.
14. `Placement != nil` and not a valid frame: `Placement must be a valid frame`.

Build rejects a nil context (`context must not be nil`) and a nil document (`document must not be nil`), then
derives the dimensions and checks `ctx.Err()` before it creates any geometry. A rejected build leaves the
document without bodies.

## Geometry

The spring is one extrusion of one fully constrained sketch in the XY plane that holds only lines and
circular arcs. Three 3D features follow: fillets on five vertical concave edges, a chamfer on the cap edges,
and the placement transform.

1. **Coil.** The coil is a two-center arc spiral. Half-turn arcs are centered alternately at `A`, the axis
   (the origin), and `B = (d, 0)`. Each half turn grows the radius by `d`, so the pitch per turn is the
   constant `p`. The strip's two edges are concentric arcs, so the normal gap between turns is exactly
   `p - t` everywhere. Junctions lie on the X axis where both centers are collinear, so the strip is tangent
   continuous without a tangent constraint. Half turn `k` is centered at `B` for even `k` and at `A` for
   odd `k`; its inner edge has radius `r_k` and its outer edge `r_k + t`.
2. **Turn count.** The hub counts as turn 0, so the coil radius is `R + (N + 1)*p`. Hence `N` and `p` follow
   from the outer diameter and the minimum gap, or from `Turns`.
3. **Root.** A short radial spoke of width `w = 2t` joins the hub to the first half turn, which starts `g`
   outside the hub. Every corner of the root is a real corner of about 90 degrees, so fillets are well posed.
   The hub is an arc of radius `R` about the axis. The spoke runs along -X.
4. **Tail and eye.** The last half turn ends on the X axis. A straight vertical tail of width `t` runs down
   from there to the eye. The eye is a round end of radius `R_e` about `E`, with a round hole of diameter
   `D_hole` through it. The tail length `L` keeps the eye `g` clear of the previous turn and at least
   `2*R_e` long.
5. **Bore.** The hub's bore is a D shape when `ArborFlatDepth > 0`: an arc of radius `r_b` closed by a flat at
   distance `f` from the axis, on the +Y side. It is a circle of radius `r_b` when `ArborFlatDepth = 0`.
6. **Fillets.** Five concave vertical edges are rounded with a 3D fillet of radius `r_f`: the root corners
   `C1`, `C3`, `C4` and the tail-to-eye junctions `P_i`, `P_o`. This is a native CAD operation, not a stand-in
   for a sketch fillet.
7. **Cap chamfer.** Surface edge finishing applies to each cap whose area is at least 400 mm^2: chamfer every
   edge of its outer loop with 0.5 mm. Chamfer the bore loop and the eye-hole loop too when their edge run is
   at least 20 mm. The outer loop has one corner after the fillets, so it is one run of hundreds of
   millimeters. A spring whose cap is smaller than 400 mm^2 is not chamfered.
8. **Placement.** A supplied `Placement` maps the local frame onto it, after the last operation.

The sketch is anchored to its origin on the axis. The centers, junctions and ends are placed by horizontal,
vertical and radius dimensions and by the shared junction points, not by fixing computed points.

## Local frame and winding direction

The origin is on the spring axis in the bottom face. +Z runs along the axis to the top face. The spoke runs
along -X, the bore flat faces +Y, and the eye lies in the -X/-Y quadrant. The spring winds counter-clockwise
outward seen from +Z. To wind (tighten) it, turn the arbor clockwise seen from +Z while the eye is held.
The opposite hand comes from a placement frame that reverses local +Z, which makes the spring wind clockwise
seen from the new +Z; there is no hand parameter.

Body bounds in the local frame: `MaxX = MaxY = R_c`, `MinX = E.x - R_e`,
`MinY = min(-(R_c - d), -L - R_e)`, Z from 0 to `h`. For the defaults X is `[-32.65, 30]`, Y is `[-28, 30]`
and Z is `[0, 8]`.

## Valid range

- `Thickness` at least 1.5 mm, so every strip, tail and eye wall is at least `Thickness` thick.
- `MinGap` at least 1.0 mm. Every air gap between facing walls when relaxed is at least `Gap`, which is at
  least `MinGap`.
- At most 40 turns, so an outer diameter of 320 mm with the defaults is rejected.
- `BoreClearance` and `AnchorClearance` at least 0.2 mm.

## Part fits

The hub-to-arbor fit is Tight: 0.2 mm radial clearance by default, and the part stays on its arbor. The D flat
carries the torque, as on the planetary gearbox shafts. The eye-to-pin fit is Regular: 0.35 mm radial
clearance by default, so the eye turns on the pin. Both clearances are parameters with a 0.2 mm floor, so
the model is Adjustable.

## Representative cases

- Default: `OuterDiameter` 60 gives 5 turns, a flat bore, a chamfered cap.
- `one-turn`: `OuterDiameter` 28 gives one turn; the cap is under 400 mm^2, so there is no chamfer.
- `round-bore`: `ArborFlatDepth` 0.
- `turns-override`: `Turns` 3 gives `p = 6.0`, `g = 4.4`.
- `large-arbor`: `OuterDiameter` 60, `HubDiameter` 16, `ArborDiameter` 8; the bore loop is chamfered.
- `thickness-floor`: `OuterDiameter` 30, `HubDiameter` 12, `Thickness` 1.5, `MinGap` 1, `Height` 4 gives 2
  turns and no chamfer.
- `max-turns`: `OuterDiameter` 310 gives 40 turns.
- Placed: the frame with origin `(10, 5, 0)`, U `(0, 1, 0)` and V `(0, 0, 1)` maps local +Z to world +X, local X
  to world Y and local Y to world Z.

## Printing and material

Print the spring flat with the axis vertical. Every layer holds the whole spiral and the strip bends in the
layer plane along its perimeters, so layer adhesion carries no bending load. The default `Thickness` of 1.6 mm is 4
extrusion lines of 0.4 mm. No supports are needed. Use PETG or nylon. PLA creeps under
constant load and cracks at a lower strain.

## Checks and images

Check the default build's bounds, face count, volume, and the full constraint of the profile sketch, for every case
above. Mesh every case but `max-turns` with `VerifyAll`; the 40-turn mesh takes minutes. Check the bore and eye-hole clearances on the built faces. Slice the
mid-height section to check that every air gap is at least `Gap` and every wall at least `Thickness`.
Register `spiralspring/hero` in `tools/render`, add the model to `models/README.md` and run
`scripts/check_repository.py`.
