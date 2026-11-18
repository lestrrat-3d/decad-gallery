# Plaque stand

## Purpose

Construct a one-piece desk or shelf stand for a flat, heavy rectangular plaque. The stand holds the plaque reclined in
a slot with a front lip, adapts to plaque sizes inside the limits below, and does not tip. Every printed part fits a
256 x 256 x 256 mm build volume at every allowed size. Everything the stand holds is called a plaque.

The stand is one extruded prism, narrower than the plaque and centred under it, printed lying on one side face. Its side
profile is a closed outline of 9 lines and 9 tangent arcs: a base slab, a front lip, a slot reclined at `Recline` that
ends in a full round, and a back rest. One fully constrained sketch draws the outline, and `decad.Extrude` pulls it
along +X by the stand width. Both side faces' outlines take a 0.5 mm chamfer. Every corner of the outline is a tangent
arc, so the prism has no edge parallel to X and the side-face chamfers satisfy
[Surface edge finishing](../../docs/model-pipeline.md#surface-edge-finishing) alone.

`Build` also adds a reference plaque body in its own container, posed in the slot. It is not printed. A user exporting
for print exports only the `Stand` body.

All lengths are in millimeters and all angles in degrees.

## Placeholder dimensions

The default plaque size, 300 x 250 x 6 mm, is a placeholder until the real plaque's measurements arrive. Only the three
`DefaultPlaque*` constants change then. Replacing them changes the geometry, so regenerate every plaque stand image and
`docs/images/gallery.png` in the same change.

## Symbols

| Symbol | Meaning | Value or formula |
| --- | --- | --- |
| `W`, `H`, `t` | plaque width (along X), height (along the slot axis), thickness | `PlaqueWidth`, `PlaqueHeight`, `PlaqueThickness` |
| `theta` | recline from vertical (0 stands upright, 15 leans the top 15 degrees back) | `Recline` |
| `c` | slot clearance per side | `SlotClearance` |
| `T_b`, `L_t`, `B_t` | base, lip and back rest thickness | 8, 5, 6 |
| `r_r`, `r_c` | root round (lip and back-rest roots on the base top), base corner round | 4, 2 |
| `F`, `B` | least flat run of the base top in front of the lip root and behind the back-rest root | 5, 5 |
| `phi` | least tilt before the assembly tips | 15 |
| `e` | side-face outline chamfer | 0.5 |
| `W_s` | stand width | `clamp(0.6 W, 120, 200)` |
| `H_b` | back rest height | `clamp(0.2 H, 30, 60)` |
| `s`, `R` | slot width, slot round radius | `t + 2c`, `s/2` |
| `h_b`, `h_l` | straight back wall and front wall lengths | `H_b - B_t/2`, `LipHeight - L_t/2` |
| `u`, `n` | slot axis, slot normal (points back and down) | `(0, sin theta, cos theta)`, `(0, cos theta, -sin theta)` |
| `C` | seat: the slot round's centre | `(0, C_y, C_z)` with `C_z = T_b + R` |
| `alpha` | play angle: the plaque's lean inside the slot clearance | `atan(2c / h_b)` |
| `D` | stand depth along Y | `ceil(max(D_geom, D_stab))` |

## Inputs, defaults and limits

| Field | Default | Limit | Meaning |
| --- | ---: | --- | --- |
| `PlaqueWidth` | 300 | 150 to 400 | `W`, along X |
| `PlaqueHeight` | 250 | 120 to 350 | `H`, along the slot axis |
| `PlaqueThickness` | 6 | 2 to 15 | `t` |
| `Recline` | 15 | 10 to 25 | `theta` |
| `LipHeight` | 10 | 6 to 20 | the lip top's height above the seat, along the slot axis |
| `SlotClearance` | 0.55 | 0.55 to 1.5 | `c`, per side; the minimum is `LooseFit + FitMargin` (0.5 + 0.05) |

Limit rationale:

- Stand width at most 200 mm becomes the print height when the part lies on its side, and 200 is under 246.
- At the worst allowed case the stand reaches 160 mm deep and 74.1 mm high, both under 246.
- Plaque width 150 to 400: below 150 the 120 mm minimum stand nears the plaque's width. Above 400 the 200 mm stand cap
  supports under half the plaque's width.
- Plaque height 120 to 350: above 350 the centre of gravity rises, and the 120 mm stand's side tip angle (17.3 degrees
  at 150 x 350) approaches the 15 degree minimum.
- Thickness 2 to 15: thinner sheets flex, and thicker plaques exceed typical award plaques and widen the slot beyond
  18 mm.
- Recline 10 to 25 degrees: under 10 a bump tips the plaque forward easily. Over 25 the plaque faces the ceiling and the
  base grows past 160 mm.

Print volume: the bed is 256 x 256 x 256 mm and the stand keeps 5 mm free on each side, so every stand extent is at most
246 mm.

## Fixed dimensions

The fixed wall dimensions are constants, not parameters: `BaseThickness` 8, `LipThickness` 5, `BackThickness` 6,
`RootRadius` 4, `CornerRadius` 2, `FrontFlat` 5, `BackFlat` 5, `StandWidthRatio` 0.6, `MinStandWidth` 120,
`MaxStandWidth` 200, `BackHeightRatio` 0.2, `MinBackHeight` 30, `MaxBackHeight` 60, `MinTipAngle` 15, `ChamferSize`
0.5, `BedSize` 256 and `BedMargin` 5.

## Local frame

- The origin is the stand's front-bottom corner (before rounding) on the side face x = 0.
- +X runs along the stand's width (the extrusion direction); the stand spans x from 0 to `W_s`.
- +Y runs from front to back; the viewer stands at -Y. +Z is up, and the base's bottom face is z = 0.
- The profile lies in the plane x = 0. The outline runs counter-clockwise in (y, z) as seen from +X.

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
| `D_geom` | `Q3_y + r_r k_b + B + r_c` |
| `D_stab` | `r_c + G_b.y + G_b.z tan phi` |
| `TipFront` | `atan((G_f.y - r_c) / G_f.z)` |
| `TipBack` | `atan((D - r_c - G_b.y) / G_b.z)` |
| `TipSide` | `atan((W_s/2 - e) / G_f.z)` |
| `C4`, `C6` | top round centres: `C + (R + B_t/2) n + h_b u` and `C - (R + L_t/2) n + h_l u` |
| `Height` | `max(C4.z + B_t/2, C6.z + L_t/2)` (the lip is the top at H = 120, theta = 25, `LipHeight` = 20) |

`Derive` makes no geometry calls. It checks the six inputs in the order of the table above (a value `v` fails when
`!(v >= min && v <= max)`, which also rejects NaN and infinities), then the two derived rules:

| # | Check | Message |
| ---: | --- | --- |
| 1 to 6 | each field, in table order | `plaquestand: <Field> must be from <min> to <max> <unit>, got <v>` |
| 7 | `TipSide < MinTipAngle` | `plaquestand: side tip angle <%.2f> degrees is under 15` |
| 8 | stand width, depth or height over 246 | `plaquestand: stand <width/depth/height> extent <%.1f> mm exceeds the 246 mm print limit` |

Rules 7 and 8 cannot fail inside the limits. They guard later edits of the constants, and the tests prove the margins.
Every message starts with `plaquestand: `.

## Outline

The 18 segments, in traversal order, are the stand's profile. Each End equals the next segment's Start, segment 17 ends
at segment 0's Start, and every join is tangent.

| i | Name | Kind | Start to End | Centre | Radius | Convex |
| ---: | --- | --- | --- | --- | --- | --- |
| 0 | L1 bottom | line | `(r_c, 0)` to `(D-r_c, 0)` | | | |
| 1 | A1 | arc | `(D-r_c, 0)` to `(D, r_c)` | `(D-r_c, r_c)` | `r_c` | yes |
| 2 | L2 back face | line | `(D, r_c)` to `(D, T_b-r_c)` | | | |
| 3 | A2 | arc | `(D, T_b-r_c)` to `(D-r_c, T_b)` | `(D-r_c, T_b-r_c)` | `r_c` | yes |
| 4 | L3 rear base top | line | `(D-r_c, T_b)` to `(Q3_y + r_r k_b, T_b)` | | | |
| 5 | A3 back-rest root | arc | `(Q3_y + r_r k_b, T_b)` to `(Q3_y, T_b) + r_r k_b u` | `(Q3_y + r_r k_b, T_b + r_r)` | `r_r` | no |
| 6 | L4 back-rest back | line | `(Q3_y, T_b) + r_r k_b u` to `P(R+B_t, h_b)` | | | |
| 7 | A4 back-rest top | arc | `P(R+B_t, h_b)` to `P(R, h_b)` | `P(R+B_t/2, h_b)` | `B_t/2` | yes |
| 8 | L5 slot back wall | line | `P(R, h_b)` to `P(R, 0)` | | | |
| 9 | A5 slot round | arc | `P(R, 0)` to `P(-R, 0)` | `C` | `R` | no |
| 10 | L6 slot front wall | line | `P(-R, 0)` to `P(-R, h_l)` | | | |
| 11 | A6 lip top | arc | `P(-R, h_l)` to `P(-R-L_t, h_l)` | `P(-R-L_t/2, h_l)` | `L_t/2` | yes |
| 12 | L7 lip front | line | `P(-R-L_t, h_l)` to `(Q7_y, T_b) + r_r k_f u` | | | |
| 13 | A7 lip root | arc | `(Q7_y, T_b) + r_r k_f u` to `(Q7_y - r_r k_f, T_b)` | `(Q7_y - r_r k_f, T_b + r_r)` | `r_r` | no |
| 14 | L8 front base top | line | `(Q7_y - r_r k_f, T_b)` to `(r_c, T_b)` | | | |
| 15 | A8 | arc | `(r_c, T_b)` to `(0, T_b-r_c)` | `(r_c, T_b-r_c)` | `r_c` | yes |
| 16 | L9 front face | line | `(0, T_b-r_c)` to `(0, r_c)` | | | |
| 17 | A9 | arc | `(0, r_c)` to `(r_c, 0)` | `(r_c, r_c)` | `r_c` | yes |

Sweeps: A1, A2, A8 and A9 are 90 degrees; A4, A5 and A6 are 180; A3 is 90 + theta; A7 is 90 - theta. The convex total
less the concave total is 360 degrees. Every line has positive length in the allowed range (checked over all 64 corner
combinations): L7 at least 2.87, L4 at least 18.9, L8 at least 5.0 (equal to `F` when `C_y,geom` governs) and L3 at
least 15.3.

`Outline.Area` is half the sum of `Start.Y End.Z - End.Y Start.Z` over every segment, plus `(r^2/2)(phi - sin phi)` for
each convex arc and minus it for each concave arc, with `phi` the arc's sweep. `Outline.Length` is the sum of the lines'
lengths and each arc's `Radius` times `Sweep`.

## Plaque pose

The reference plaque is a box `W` wide, `H` high and `t` thick. It is centred on the stand in X, from `x0 = (W_s - W)/2`
to `x0 + W`. Its bottom face's centre line passes through the seat `C`, and the plaque lies along `u` away from the seat,
with its front face toward -`n`. The plaque's back-bottom edge starts at `(x0, C_y + (t/2) cos theta, C_z - (t/2) sin
theta)`.

## Construction and feature order

1. Derive the dimensions and the outline.
2. Sketch the outline on the plane x = 0 and require it fully constrained.
3. Extrude it along +X by `W_s`.
4. Chamfer both side faces' outer loops by `e` (0.5), one chamfer call over the planes x = 0 and x = `W_s`.
5. Sketch the plaque rectangle, require it fully constrained, and extrude it by `t` toward the front.
6. File each body in its container.

There is no Boolean, so a plain `decad.New()` document suffices. A Sketch or Decad refusal returns the wrapped error
and stops. There is no fallback geometry.

## Bodies and containers

`Build` adds exactly two bodies. `Stand` is one body, the one member of its kind, and `Plaque` is one body. Each is a
child of the document root, created with `ChildNamed` so several stands can share one document (the renderer's `sizes`
image). See [Body containers](../../docs/model-pipeline.md#body-containers).

## Part fits

The plaque sits in the slot with the Loose fit of [Part clearances](../../docs/model-pipeline.md#part-clearances):
`SlotClearance` per side at least 0.55 (0.5 + 0.05 margin). The wall gap equals `c`. The plaque's bottom corners sit
`t/2` from `C`, so their gap to the round is `R - t/2 = c`. The plaque leans on the slot's back wall under gravity, and
a looser slot tolerates plaque thickness variation and over-extrusion. The plaque is a nominal part, so all the
clearance is in the stand. The model is Mixed: the slot clearance is adjustable and the plaque is a fixed nominal part.

## Stability

The plaque is a rigid rectangular sheet with square edges and uniform density, and its weight dominates the stand's.
The stability rule ignores the stand's mass, which only adds margin.

The tipping edges are the ends of the bottom flat (y = `r_c` and y = `D - r_c`) and the side faces less the chamfer (x =
`e` and x = `W_s - e`). The assembly tips only after a tilt of `MinTipAngle` (15 degrees) or more forward, backward or
sideways: the plaque's centre of gravity, at its leaned pose, stays at least 15 degrees inside each tipping edge. The
plaque leans inside the slot clearance by up to `alpha` (contact at the back wall's top and the slot's front edge). Front
tipping uses `theta - alpha`, and back tipping uses `theta + alpha`. The plaque's modeled pose sits higher than its real
rest (it drops `sqrt(c t + c^2)` onto the round), which only lowers the real centre of gravity.

## Printing and bed

Lie the stand on its x = 0 face. The profile is in the bed plane, no surface needs support, and the print height is
`W_s`, at most 200 mm. The bed footprint is `D` by `Height`, at most 160 by 74.1 mm.

Strength: the model checks no strength. A steel plaque at the maximum size (400 x 350 x 15 mm, about 16 kg) at 25 degrees
puts roughly 10 MPa of bending on the back rest's root, a sustained load on PLA. At that size prefer PETG or a lighter
plaque.

## Surface edge finishing

The side faces are large (at least 632 mm^2 of profile area). Both side faces' outer loops take a 0.5 mm chamfer. Every
other face meets its neighbours on a tangent seam or on the chamfered loops, so no other edge qualifies. There is no
smaller chamfer and no retained sharp edge.

## Corners and roots

Print axis +X; one section at x = `W_s/2` reads the whole profile.

- The roots A3 and A7 have radius 4, at least half the thinner section (lip 5 gives 2.5, back rest 6 gives 3, base 8).
- The slot ends in a full round of radius `s/2`, as the rule asks for a slot too narrow for two roots. It covers the lip
  and back-rest roots in the slot at every plaque thickness.
- The base corners have radius 2, the lip top 2.5 and the back-rest top 3.
- The lip and the back rest are at least 1.2 mm thick (they bend in use), and their root widths equal their sections.
- No retained sharp corners.

## Not applicable

Fasteners, arbor collars, split shafts, gear train clearances and look-alike marks do not apply. The model has no
`Look-alike parts` section.

## Representative cases

The cases the tests pin to 1e-6 (width, height, thickness, recline, lip, clearance):

| Case | Params | `W_s` | `H_b` | `C_y` | `D` | Height | TipFront | TipBack | TipSide | Area | Length |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| default | 300, 250, 6, 15, 10, 0.55 | 180 | 50 | 19.872139 | 93 | 58.253249 | 19.610016 | 15.293816 | 33.934962 | 1136.317882 | 320.343749 |
| min | 150, 120, 2, 10, 6, 0.55 | 120 | 30 | 17.280750 | 51 | 38.349710 | 18.644768 | 15.499808 | 40.766229 | 632.557129 | 182.688124 |
| max | 400, 350, 15, 25, 20, 1.5 | 200 | 60 | 29.192341 | 160 | 66.588125 | 27.346216 | 15.206476 | 29.031289 | 1894.182043 | 509.428323 |
| forward | 300, 350, 2, 10, 10, 0.55 | 180 | 60 | 23.828646 | 109 | 67.893943 | 15.000000 | 15.262544 | 26.130591 | 1296.557130 | 366.688124 |
| narrow-tall | 150, 350, 15, 10, 6, 1.5 | 120 | 60 | 31.809631 | 124 | 74.050264 | 15.000000 | 15.154925 | 17.328331 | 1514.617026 | 412.092989 |
| short-steep | 150, 120, 15, 25, 20, 1.5 | 120 | 30 | 29.192341 | 81 | 40.220496 | 32.136555 | 15.245825 | 38.859428 | 1082.182043 | 291.428323 |

Other default values: `s` 7.1, `R` 3.55, `h_b` 47, `h_l` 7.5, `C_z` 11.55, `alpha` 1.340720 degrees, `C_y,geom`
19.872139, `C_y,stab` 8.122718, `D_stab` 92.276241 and `D_geom` 41.020708.

Branch coverage: `forward` and `narrow-tall` take `C_y` from `C_y,stab`, and the others from `C_y,geom`. `min` and
`short-steep` clamp `W_s` and `H_b` low, `max` clamps them high, and `short-steep` takes the height from the lip.

## Checks and images

Check every case above: every sketch fully constrained (`Sketch.Verify` reports `FullyConstrained`, DOF 0, no
redundant constraints, no conflicts), both bodies solid in their containers, the stand's and plaque's bounds, the stand's
volume against `W_s A - e^2 (P - 2 pi e / 3)`, sections at the middle and 0.1 mm inside each side face, `VerifyAll`
meshes, the print fit, the Loose clearance between plaque and stand (`internal/meshgap`), the three tip angles read off
the built bodies, and `internal/cornercheck` on the stand with print axis +X.

Images come from `tools/render` (`plaquestand` readme shot): hero, stand, side, back, exploded, print, sizes, and the
two part images.
