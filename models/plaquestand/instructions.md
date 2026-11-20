# Plaque stand

## Purpose

Construct a desk or shelf stand for a flat, heavy rectangular plaque from separate printed parts that use far less
plastic than one prism: two identical brackets, one tie rail, and two M2 x 6 bolts and nuts. The stand holds the plaque
reclined in slots with a front lip, adapts to plaque sizes inside the limits below, and does not tip. Every printed part
fits a 256 x 256 x 256 mm build volume at every allowed size. Everything the stand holds is called a plaque.

| Body | Count | Placement | Print |
| --- | --- | --- | --- |
| Bracket | 2, identical | bracket k spans x from `x_k` to `x_k + w_b`, `x_1` = 0, `x_2` = `S - w_b` | on either side face, profile in the bed plane |
| Rail | 1 | x from 0 to `S`, centred at y = `y_ch`, z from `T_b - h_r + j` to `T_b + j` | flat on its bottom face, head pockets up |
| M2 x 6 bolt | 2 (the M2 preset of `models/fastener`) | one per bracket, head flush with the rail top | bought |
| M2 nut | 2 (the M2 preset of `models/fastener`) | in the hex pocket open to the bracket's bottom face | bought |
| Plaque | 1 reference | centred on the span: `x0 = (S - W)/2` | not printed |

Each bracket is a slice of one profile: a base slab with a rail channel in its rear top, a front lip, a slot reclined at
`Recline` that ends in a full round, and a back rest. The profile is a closed outline of 13 lines and 13 tangent arcs.
One fully constrained sketch draws it, and `decad.Extrude` pulls it along +X by `w_b` (20 mm). Both side faces' outlines
take a 0.5 mm chamfer. A bracket is symmetric about its mid-plane, because the channel runs through its whole width and
the bolt lies on the mid-plane, so one print file makes both. The rail drops into both channels and one M2 x 6 bolt per
bracket, driven down through the rail into a nut in the bracket's bottom face, fixes it. The bolted rail sets the
spacing and keeps the brackets square, so they cannot spread, slide along the plaque's edge or yaw.

`Build` also adds a reference plaque body in its own container, posed in the slots. It is not printed. A user exporting
for print exports only the `Brackets` and `Rail` bodies and prints the bracket twice.

All lengths are in millimeters and all angles in degrees.

## Default plaque dimensions

The default plaque size, 250 x 305 x 12.8 mm, is the measured size of the user's plaque. Only the three
`DefaultPlaque*` constants change if the plaque changes. Changing them changes the geometry, so regenerate every plaque
stand image in the same change.

## Symbols

| Symbol | Meaning | Value or formula |
| --- | --- | --- |
| `W`, `H`, `t` | plaque width (along X), height (along the slot axis), thickness | `PlaqueWidth`, `PlaqueHeight`, `PlaqueThickness` |
| `theta` | recline from vertical (0 stands upright, 15 leans the top 15 degrees back) | `Recline` |
| `c` | slot clearance per side | `SlotClearance` |
| `T_b`, `L_t`, `B_t` | base, lip and back rest thickness | 8, 5, 6 |
| `r_r`, `r_c` | root round (lip and back-rest roots on the base top), base corner round | 4, 2 |
| `F` | least flat run of the base top in front of the lip root | 5 |
| `phi` | least tilt before the assembly tips | 15 |
| `e` | side-face outline chamfer | 0.5 |
| `S` | span, outer face to outer face (the rail's length) | `clamp(0.8 W, 120, 246)` |
| `w_b` | bracket width | 20 |
| `w_r`, `h_r`, `r_q` | rail section width, height, corner round | 10, 4, 1 |
| `j` | rail-to-channel gap (Regular, 0.35 + 0.05 margin) | 0.40 |
| `w_c`, `h_c`, `r_f`, `r_m` | channel width, depth, floor round, mouth round | `w_r + 2j` (10.8), 4, `r_q + j` (1.4), 1 |
| `H_b` | back rest height | `clamp(0.2 H, 30, 60)` |
| `s`, `R` | slot width, slot round radius | `t + 2c`, `s/2` |
| `h_b`, `h_l` | straight back wall and front wall lengths | `H_b - B_t/2`, `LipHeight - L_t/2` |
| `u`, `n` | slot axis, slot normal (points back and down) | `(0, sin theta, cos theta)`, `(0, cos theta, -sin theta)` |
| `C` | seat: the slot round's centre | `(0, C_y, C_z)` with `C_z = T_b + R` |
| `alpha` | play angle: the plaque's lean inside the slot clearance | `atan(2c / h_b)` |
| `D` | bracket depth along Y | `ceil(max(D_geom, D_stab))` |

## Inputs, defaults and limits

| Field | Default | Limit | Meaning |
| --- | ---: | --- | --- |
| `PlaqueWidth` | 250 | 150 to 400 | `W`, along X |
| `PlaqueHeight` | 305 | 120 to 350 | `H`, along the slot axis |
| `PlaqueThickness` | 12.8 | 2 to 15 | `t` |
| `Recline` | 15 | 10 to 25 | `theta` |
| `LipHeight` | 10 | 6 to 20 | the lip top's height above the seat, along the slot axis |
| `SlotClearance` | 0.55 | 0.55 to 1.5 | `c`, per side; the minimum is `LooseFit + FitMargin` (0.5 + 0.05) |

Limit rationale:

- The span is at most 246 mm, so the rail fits the bed lengthwise with 5 mm free at each end.
- At the worst allowed case a bracket reaches 160 mm deep and 74.1 mm high, both under 246.
- Plaque width 150 to 400: below 150 the 120 mm minimum span nears the plaque's width. Above 400 the 246 mm span cap
  puts the brackets under 0.62 of the plaque's width apart.
- Plaque height 120 to 350: above 350 the centre of gravity rises, and the 120 mm span's side tip angle (17.3 degrees
  at 150 x 350) approaches the 15 degree minimum.
- Thickness 2 to 15: thinner sheets flex, and thicker plaques exceed typical award plaques and widen the slot beyond
  18 mm.
- Recline 10 to 25 degrees: under 10 a bump tips the plaque forward easily. Over 25 the plaque faces the ceiling and the
  base grows past 160 mm.

Print volume: the bed is 256 x 256 x 256 mm and the stand keeps 5 mm free on each side, so every stand extent is at most
246 mm.

Plaque weight: a bracket is 20 mm wide, so each back rest carries about half the plaque's weight on 20 mm. The README states
a plaque limit of 3 kg (about 9 MPa at the largest geometry and 3 MPa at the defaults). A heavy steel plaque at the
largest size (about 16 kg) would put about 50 MPa on two back rests, which is too much.

## Fixed dimensions

The fixed dimensions are constants, not parameters: `BaseThickness` 8, `LipThickness` 5, `BackThickness` 6, `RootRadius`
4, `CornerRadius` 2, `FrontFlat` 5, `BracketWidth` 20, `SpanRatio` 0.8, `MinSpan` 120, `MaxSpan` 246, `RailWidth` 10,
`RailHeight` 4, `RailRound` 1, `RegularFit` 0.35, `JointFit` 0.40 (`RegularFit + FitMargin`), `ChannelMouthRound` 1,
`ChannelLand` 2, `RearLand` 8, `BoltLength` 6 (in `m2fastener.BoltSize.StandardLengths`), `NutPocketRound` 0.6,
`BackHeightRatio` 0.2, `MinBackHeight` 30, `MaxBackHeight` 60, `MinTipAngle` 15, `ChamferSize` 0.5, `BedSize` 256 and
`BedMargin` 5.

`RearLand` is 8 because a 2 mm rear land leaves the profile sketch's constraint system too close to singular for
sketch's conditioning check (below 4e-5, four of the six representative cases). With 8 mm the least conditioning over
the 729-point grid of the limits is 6.8e-5, and `TestPlaqueStandProfileConditioning` guards it at 6e-5.

## Local frame

- The origin is the front-bottom corner (before rounding) of bracket 1's outer face, on x = 0.
- +X runs along the span (the extrusion direction); the stand spans x from 0 to `S`.
- +Y runs from front to back; the viewer stands at -Y. +Z is up, and the base's bottom face is z = 0.
- A bracket's profile lies in the plane x = `x_k`. The outline runs counter-clockwise in (y, z) as seen from +X.

## Derived values

All points are (y, z) pairs on the plane x = 0. Let `P(a, b) = C + a n + b u`, `k_f = cot(45 + theta/2)` and
`k_b = cot(45 - theta/2)`.

| Value | Formula |
| --- | --- |
| `lambda_7` | `-(R + (R + L_t) sin theta) / cos theta` |
| `lambda_3` | `(-R + (R + B_t) sin theta) / cos theta` |
| `theta_f`, `theta_b` | `theta - alpha`, `theta + alpha` |
| `C_y,geom` | `r_c + F + (R + L_t) cos theta - lambda_7 sin theta + r_r k_f` |
| `C_y,stab` | `r_c + (C_z + (H/2) cos theta_f) tan phi - (H/2) sin theta_f` |
| `C_y` | `max(C_y,geom, C_y,stab)` |
| `Q7_y` | `C_y - (R + L_t) cos theta + lambda_7 sin theta` (the lip's front line meets z = `T_b`) |
| `Q3_y` | `C_y + (R + B_t) cos theta + lambda_3 sin theta` (the back rest's back line meets z = `T_b`) |
| `G_f` | `(C_y + (H/2) sin theta_f, C_z + (H/2) cos theta_f)`: the plaque's centre of gravity leaned forward by `alpha` |
| `G_b` | `(C_y + (H/2) sin theta_b, C_z + (H/2) cos theta_b)`: leaned back by `alpha` |
| `D_geom` | `Q3_y + r_r k_b + ChannelLand + 2 r_m + w_c + RearLand + r_c` |
| `ChannelBack` `y_cb` | `D - r_c - RearLand - r_m` (= `D - 11`) |
| `ChannelFront` `y_cf` | `y_cb - w_c` (= `D - 21.8`) |
| `ChannelFloor` `z_f` | `T_b - h_c` (= 4) |
| `RailCenter` `y_ch` | `(y_cb + y_cf)/2` (= `D - 16.4`) |
| `Sites` | per bracket: `Head (x_k + w_b/2, y_ch, T_b + j)`, `Dir (0, 0, -1)`, `Corner (1, 0, 0)`, `Length` `BoltLength`, `Cavity` 6.4 |
| `D_stab` | `r_c + G_b.y + G_b.z tan phi` |
| `TipFront` | `atan((G_f.y - r_c) / G_f.z)` |
| `TipBack` | `atan((D - r_c - G_b.y) / G_b.z)` |
| `TipSide` | `atan((S/2 - e) / G_f.z)` |
| `C4`, `C6` | top round centres: `C + (R + B_t/2) n + h_b u` and `C - (R + L_t/2) n + h_l u` |
| `Height` | `max(C4.z + B_t/2, C6.z + L_t/2)` (the lip is the top at H = 120, theta = 25, `LipHeight` = 20) |

`Derive` makes no geometry calls. It checks the six inputs in the order of the table above (a value `v` fails when
`!(v >= min && v <= max)`, which also rejects NaN and infinities), then the two derived rules:

| # | Check | Message |
| ---: | --- | --- |
| 1 to 6 | each field, in table order | `plaquestand: <Field> must be from <min> to <max> <unit>, got <v>` |
| 7 | `TipSide < MinTipAngle` | `plaquestand: side tip angle <%.2f> degrees is under 15` |
| 8 | span, depth or height over 246 | `plaquestand: rail length <%.1f> mm exceeds the 246 mm print limit` for the span; `plaquestand: stand <depth/height> extent <%.1f> mm exceeds the 246 mm print limit` |

Rules 7 and 8 cannot fail inside the limits. They guard later edits of the constants, and the tests prove the margins.
Every message starts with `plaquestand: `.

## Outline

The 26 segments, in traversal order, are a bracket's profile (13 lines alternating with 13 arcs, starting with a line).
Each End equals the next segment's Start, segment 25 ends at segment 0's Start, and every join is tangent. The channel
is segments 4 to 12 and runs through the bracket's whole width.

| i | Name | Kind | Start to End | Centre | Radius | Convex |
| ---: | --- | --- | --- | --- | --- | --- |
| 0 | L1 bottom | line | `(r_c, 0)` to `(D-r_c, 0)` | | | |
| 1 | A1 | arc | `(D-r_c, 0)` to `(D, r_c)` | `(D-r_c, r_c)` | `r_c` | yes |
| 2 | L2 back face | line | `(D, r_c)` to `(D, T_b-r_c)` | | | |
| 3 | A2 | arc | `(D, T_b-r_c)` to `(D-r_c, T_b)` | `(D-r_c, T_b-r_c)` | `r_c` | yes |
| 4 | L3a rear land | line | `(D-r_c, T_b)` to `(y_cb + r_m, T_b)` | | | |
| 5 | M1 back mouth | arc | `(y_cb + r_m, T_b)` to `(y_cb, T_b - r_m)` | `(y_cb + r_m, T_b - r_m)` | `r_m` | yes |
| 6 | W1 back wall | line | `(y_cb, T_b - r_m)` to `(y_cb, z_f + r_f)` | | | |
| 7 | F1 back floor round | arc | `(y_cb, z_f + r_f)` to `(y_cb - r_f, z_f)` | `(y_cb - r_f, z_f + r_f)` | `r_f` | no |
| 8 | Fl floor | line | `(y_cb - r_f, z_f)` to `(y_cf + r_f, z_f)` | | | |
| 9 | F2 front floor round | arc | `(y_cf + r_f, z_f)` to `(y_cf, z_f + r_f)` | `(y_cf + r_f, z_f + r_f)` | `r_f` | no |
| 10 | W2 front wall | line | `(y_cf, z_f + r_f)` to `(y_cf, T_b - r_m)` | | | |
| 11 | M2 front mouth | arc | `(y_cf, T_b - r_m)` to `(y_cf - r_m, T_b)` | `(y_cf - r_m, T_b - r_m)` | `r_m` | yes |
| 12 | L3b front land | line | `(y_cf - r_m, T_b)` to `(Q3_y + r_r k_b, T_b)` | | | |
| 13 | A3 back-rest root | arc | `(Q3_y + r_r k_b, T_b)` to `(Q3_y, T_b) + r_r k_b u` | `(Q3_y + r_r k_b, T_b + r_r)` | `r_r` | no |
| 14 | L4 back-rest back | line | `(Q3_y, T_b) + r_r k_b u` to `P(R+B_t, h_b)` | | | |
| 15 | A4 back-rest top | arc | `P(R+B_t, h_b)` to `P(R, h_b)` | `P(R+B_t/2, h_b)` | `B_t/2` | yes |
| 16 | L5 slot back wall | line | `P(R, h_b)` to `P(R, 0)` | | | |
| 17 | A5 slot round | arc | `P(R, 0)` to `P(-R, 0)` | `C` | `R` | no |
| 18 | L6 slot front wall | line | `P(-R, 0)` to `P(-R, h_l)` | | | |
| 19 | A6 lip top | arc | `P(-R, h_l)` to `P(-R-L_t, h_l)` | `P(-R-L_t/2, h_l)` | `L_t/2` | yes |
| 20 | L7 lip front | line | `P(-R-L_t, h_l)` to `(Q7_y, T_b) + r_r k_f u` | | | |
| 21 | A7 lip root | arc | `(Q7_y, T_b) + r_r k_f u` to `(Q7_y - r_r k_f, T_b)` | `(Q7_y - r_r k_f, T_b + r_r)` | `r_r` | no |
| 22 | L8 front base top | line | `(Q7_y - r_r k_f, T_b)` to `(r_c, T_b)` | | | |
| 23 | A8 | arc | `(r_c, T_b)` to `(0, T_b-r_c)` | `(r_c, T_b-r_c)` | `r_c` | yes |
| 24 | L9 front face | line | `(0, T_b-r_c)` to `(0, r_c)` | | | |
| 25 | A9 | arc | `(0, r_c)` to `(r_c, 0)` | `(r_c, r_c)` | `r_c` | yes |

Sweeps: A1, A2, A8, A9 and the four channel arcs (M1, F1, F2, M2) are 90 degrees; A4, A5 and A6 are 180; A3 is 90 +
theta; A7 is 90 - theta. The convex total less the concave total is 360 degrees. Line lengths: Fl is 8, L3a is 8 and W1
and W2 are 1.6; over all 64 corners of the limits L3b is at least 2.56, L7 at least 2.87 and L4 at least 18.9.

`Outline.Area` is half the sum of `Start.Y End.Z - End.Y Start.Z` over every segment, plus `(r^2/2)(phi - sin phi)` for
each convex arc and minus it for each concave arc, with `phi` the arc's sweep. `Outline.Length` is the sum of the lines'
lengths and each arc's `Radius` times `Sweep`. The channel changes them in closed form: `Area = Area_old(D) - 42.787965` (that is `w_c h_c - 2(1 - pi/4) r_f^2 + 2(1 - pi/4)
r_m^2`) and `Length = Length_old(D) + 5.939823`, where `Area_old` and `Length_old` are the outline without the channel at
the same `D`.

## Rail

The rail is a rounded rectangle `w_r` x `h_r` (10 x 4) with corner rounds `r_q` (1), extruded along +X by `S`. It sits in
the channel with the Regular gap `j` (0.40) on both walls and on the floor, and its corners are concentric with the
channel's floor rounds (`r_f = r_q + j`). Its ends are flush with the brackets' outer faces. The model shows it
`j` proud of the base top (z from 4.4 to 8.4): tightening the bolts pulls it down onto the channel floor, its top then
flush with the base top and each bolt's tip flush with the bottom face. Modeling it flush would leave the clamped tip
0.4 mm below the desk face, and M2 x 6 is the shortest standard bolt.

## Plaque pose

The reference plaque is a box `W` wide, `H` high and `t` thick. It is centred on the span in X, from `x0 = (S - W)/2`
to `x0 + W`. Its bottom face's centre line passes through the seat `C`, and the plaque lies along `u` away from the seat,
with its front face toward -`n`. The plaque's back-bottom edge starts at `(x0, C_y + (t/2) cos theta, C_z - (t/2) sin
theta)`.

## Joinery and the fastener stack

Each bracket holds one M2 x 6 bolt (`BoltLength`, in `m2fastener.BoltSize.StandardLengths`) and one M2 nut. The bolt runs down
through a head pocket and hole in the rail and a hole in the bracket's channel floor, into a hex nut pocket open to the
bracket's bottom face. The pocket's flats stop the nut turning, and the tightened bolt holds the nut there.

- Modeled: head top z = 8.4, flush with the rail top. Head pocket floor z = 6.4, with 2 mm of rail under it down to 4.4.
- Modeled tip = 2 + 6 = 8 below the head, so z = 0.4, inside the nut pocket and 0.4 mm above the bottom face.
- Clamped: everything above the floor drops 0.4 mm; the head top is at z = 8 (flush with the base top) and the tip at
  z = 0 (flush).
- Nut cavity z from 0 to 2 (`Cavity` 6.4 to `CavityFar` 8.4 below the head), open to the bottom face. The nut sits at z
  from 0.2 to 1.8. The bracket keeps 4 - 2 = 2 mm between the pocket and the channel floor.
- Side walls: rail around the head pocket 5 - 2.2 = 2.8; rail end to the head pocket 10 - 2.2 = 7.8; bracket face to the
  nut pocket corner 10 - 2.51 = 7.49. With `Corner` = X the hexagon has flats normal to +-Y (half-width 2.17) and a
  corner pointing up in print, so the pocket bridges itself without support.
- The hex pocket's floor outline is rounded by `NutPocketRound` (0.6) with `m2fastener.RoundedHexPocketTool`, so a layer
  section across a flat's edge reads a round, not a corner. The crank handle keeps its own private copy of that cut.

## Construction and feature order

1. Derive the dimensions and the outline.
2. For each bracket k = 1, 2: sketch the outline on the plane x = `x_k` and require it fully constrained, extrude it along
   +X by `w_b`, chamfer both side faces' outer loops by `e` (0.5, one `edgefinish.Chamfer` call over the planes x = `x_k`
   and x = `x_k + w_b`), then cut `m2fastener.RoundedHexPocketTool` (depth `NutCavityHeight + Stock`, round
   `NutPocketRound`) and `m2fastener.HoleTool` for the site. Require a solid.
3. Sketch the rail's section, require it fully constrained, extrude it by `S` along +X, and cut `PocketTool` and
   `HoleTool` for both sites. Require a solid. The rail takes no chamfer (see below).
4. Sketch the plaque rectangle, require it fully constrained, and extrude it by `t` toward the front.
5. `m2fastener.PlaceBolt` and `PlaceNut` for each site.
6. File each body in its container.

There is no Boolean between parts, and plain `decad.New()` suffices. A Sketch or Decad refusal returns the wrapped error
and stops. There is no fallback geometry.

## Bodies and containers

`Build` adds eight bodies: `Brackets` (2), `Rail` (1), `Bolts` (2), `Nuts` (2) and `Plaque` (1). Each container is a child
of the document root, created with `ChildNamed` so several stands can share one document (the renderer's `sizes` image).
See [Body containers](../../docs/model-pipeline.md#body-containers). `Model.Sketches` holds the 12 sketches: 2 profiles,
2 hex pockets, 2 bracket holes, the rail, 2 head pockets, 2 rail holes and the plaque rectangle.

## Part fits

| Fit | Class | Modeled | Measure |
| --- | --- | --- | --- |
| Plaque in the brackets | Loose | `c` at least 0.55 | `internal/meshgap` |
| Rail in the channels | Regular | 0.40 on both walls, the floor and the concentric corners | `internal/meshgap` |
| Bolts and nuts | Fasteners rules | head 0.3, hole 0.3, nut 0.2 | bounds and sections; the head on the pocket floor and the bolt in the nut touch by design (a nominal pair) |

The plaque's bottom corners sit `t/2` from `C`, so their gap to the slot round is `R - t/2 = c`. The plaque leans on the
slot's back wall under gravity, and a looser slot tolerates plaque thickness variation and over-extrusion. The plaque is
a nominal part, so all the clearance is in the brackets. The rail is custom, so it too takes its clearance in the
channels. The model is Mixed: the slot clearance is adjustable and the plaque and the hardware are fixed nominal parts.

## Stability

The plaque is a rigid rectangular sheet with square edges and uniform density, and its weight dominates the stand's.
The stability rule ignores the stand's mass, which only adds margin.

The tipping edges are the ends of the bottom flat (y = `r_c` and y = `D - r_c`) and the brackets' outer faces less the
chamfer (x = `e` and x = `S - e`). The assembly tips only after a tilt of `MinTipAngle` (15 degrees) or more forward,
backward or sideways: the plaque's centre of gravity, at its leaned pose, stays at least 15 degrees inside each tipping
edge. The plaque leans inside the slot clearance by up to `alpha` (contact at the back wall's top and the slot's front
edge). Front tipping uses `theta - alpha`, and back tipping uses `theta + alpha`. The plaque's modeled pose sits higher
than its real rest (it drops `sqrt(c t + c^2)` onto the round), which only lowers the real centre of gravity. The rail
never touches the desk: its bottom is at z = 4.4.

## Printing and bed

Print each bracket on a side face, so the profile is in the bed plane, no surface needs support, and the print height is
`w_b` (20 mm). The bed footprint is `D` by `Height`, at most 160 by 74.1 mm. Print the rail flat on its bottom face with
the head pockets up, at most 246 mm long. Print the bracket twice and the rail once, with no supports and no pause. The
nuts go into their pockets at assembly.

Strength: the model checks no strength. The README states a plaque limit of 3 kg.

## Surface edge finishing

The brackets' side faces are large (at least 585 mm^2). Both side faces' outer loops take a 0.5 mm chamfer. The new rims
(hex pocket 15.1 mm, holes 8.2 mm, head pockets 13.8 mm) are under 20 mm.

The rail takes no chamfer. Its long edges are tangent seams of its rounded section. Its top, bottom and side faces meet
the end faces in 8 mm or 2 mm runs, under 20 mm, and each end face (39 mm^2) is under 400 mm^2. The end face's own outline
is one tangent run of 26.3 mm, but it bounds a small face, and `edgefinish` skips faces under 400 mm^2.

## Corners and roots

Brackets print on a side face (axis +X). One section at `x_k + w_b/2` reads the profile, including the channel (floor
rounds 1.4, mouth rounds 1.0), the bolt hole and the hex pocket roof with its 0.6 round. The rail prints flat (axis +Z);
sections at z = 5.4 and 7.4 show a rectangle with round holes.

- The roots A3 and A7 have radius 4, at least half the thinner section (lip 5 gives 2.5, back rest 6 gives 3, base 8).
- The slot ends in a full round of radius `s/2`, as the rule asks for a slot too narrow for two roots.
- The base corners have radius 2, the lip top 2.5 and the back-rest top 3.
- The lip and the back rest are at least 1.2 mm thick (they bend in use), and their root widths equal their sections.

Retained sharp corners and edges, none read by `cornercheck`:

| Part | Corner or edge | Reason |
| --- | --- | --- |
| Bracket | Each hex pocket's six corners | Mating corners of the M2 nut. Their edges run along Z, parallel to the bed in print, so no layer section reads them. |
| Rail | Each head pocket's floor rim | The 0.3 mm head clearance leaves no room for a round, and the rim lies parallel to the bed (the crank handle's precedent). |
| Rail | The two end outlines | Each bounds an end face of 39 mm^2, under the 400 mm^2 large-face limit of "Surface edge finishing". |

## Not applicable

Arbor collars, split shafts, gear train clearances and look-alike marks do not apply: both brackets are the same part and
the rail looks nothing like them, so the model has no `Look-alike parts` section.

## Representative cases

The cases the tests pin to 1e-6 (width, height, thickness, recline, lip, clearance). `Height`, `C_y` and `TipFront` do not
depend on the span or the channel; `D` and `TipBack` change only where the channel decides `D` (the `min` case, 51 to 55).

| Case | Params | `S` | `H_b` | `C_y` | `D` | Height | TipFront | TipBack | TipSide | Area | Length |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| default | 250, 305, 12.8, 15, 10, 0.55 | 200 | 60 | 24.303105 | 112 | 70.432523 | 19.875977 | 15.042312 | 31.403024 | 1361.179509 | 394.964986 |
| min | 150, 120, 2, 10, 6, 0.55 | 120 | 30 | 17.280750 | 55 | 38.349710 | 18.644768 | 18.572439 | 40.766229 | 621.769165 | 196.627946 |
| max | 400, 350, 15, 25, 20, 1.5 | 246 | 60 | 29.192341 | 160 | 66.588125 | 27.346216 | 15.206476 | 34.345575 | 1851.394079 | 515.368145 |
| forward | 300, 350, 2, 10, 10, 0.55 | 240 | 60 | 23.828646 | 109 | 67.893943 | 15.000000 | 15.262544 | 33.224387 | 1253.769166 | 372.627946 |
| narrow-tall | 150, 350, 15, 10, 6, 1.5 | 120 | 60 | 31.809631 | 124 | 74.050264 | 15.000000 | 15.154925 | 17.328331 | 1471.829062 | 418.032811 |
| short-steep | 150, 120, 15, 25, 20, 1.5 | 120 | 30 | 29.192341 | 81 | 40.220496 | 32.136555 | 15.245825 | 38.859428 | 1039.394079 | 297.368145 |

Printed volume (two brackets and the rail, from `Body.Volume()`), against the one-piece stand it replaced:

| Case | Bracket mm^3 | Rail mm^3 | Total cm^3 | One-piece cm^3 |
| --- | ---: | ---: | ---: | ---: |
| default | 27082.87 | 7746.18 | 61.91 | 210.50 |
| min | 12344.22 | 4614.89 | 29.30 | 75.86 |
| max | 36857.07 | 9546.67 | 83.26 | 378.71 |
| forward | 24940.22 | 9311.82 | 59.19 | |
| narrow-tall | 29290.11 | 4614.89 | 63.19 | |
| short-steep | 20671.57 | 4614.89 | 45.96 | |

Branch coverage: `forward` and `narrow-tall` take `C_y` from `C_y,stab`, and the others from `C_y,geom`. `min` and
`short-steep` clamp `H_b` low and `max` clamps it high; `min`, `narrow-tall` and `short-steep` clamp the span at its
minimum and `max` at its maximum. `short-steep` takes the height from the lip. `min` takes `D` from `D_geom` (the channel).

## Checks and images

Check every case above: all 12 sketches fully constrained (`Sketch.Verify` reports `FullyConstrained`, DOF 0, no
redundant constraints, no conflicts), all eight bodies solid in their containers, each bracket's and the rail's bounds,
the brackets' and the rail's volume, sections at the middle and 0.1 mm inside each bracket face, `VerifyAll` meshes, the
Loose clearance between the plaque and every other body and the Regular clearance of the rail in the channels
(`internal/meshgap`), the three tip angles read off the built bodies, each bolt's head, tip and nut against its pockets
with no overlap, and `internal/cornercheck` on each bracket (axis +X) and the rail (axis +Z). The sketch's conditioning is
checked over the 729-point grid of the limits (`TestPlaqueStandProfileConditioning`, kept off the PR list).

Images come from `tools/render` (`plaquestand` readme shot): hero, frame, joint, side, back, exploded, print, sizes, and
the five part images.
