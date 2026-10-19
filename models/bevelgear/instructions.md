# Bevel gear construction instructions

## Source and implementation status

`fusion360-gear-generator/spec/bevelgear/instructions.md` defines the final
geometry, operation order, and defaults. Images in that repository are visual
checks. The gallery `Build` constructs the source-default 31/31 spiral member
through one Combine-Join and an optional post-join bore. It also constructs
verified straight 4/4, 8/8, and 16/12 members. `BuildPair` builds the pinion first and
then rotates the driving member by half a tooth pitch.
The source enables the bore by default and resolves a zero diameter automatically.
`DefaultParams()` also selects the source's 35° mean spiral angle and both enabled bores.

## Inputs and supported range

`Build(ctx, doc, Params)` builds one member of a meshing bevel pair. The caller
supplies its tooth count and the mate's tooth count; both members use the same
module and shaft angle. Lengths are millimeters and angles are radians. A
`BuildPair` constructs and places both members for the verified straight 4/4,
8/8, and 16/12 cases and the source-default spiral 31/31 case. Its pinion is built first, and both
members share the world origin as their pitch apex.

| Input | Default | Rule |
| --- | ---: | --- |
| `Module` | 1 mm | Finite and positive. |
| `Teeth` | 31 | At least three, and at least the computed cone-angle floor below. |
| `MateTeeth` | 31 | Same tooth-count rule, using the mate's cone angle. |
| `ShaftAngle` | 90° | From 30° through the maximum shaft angle below. |
| `FaceWidth` | 0 | Zero selects the derived width; a positive value is explicit. |
| `PressureAngle` | 20° | Greater than zero and below 90°. |
| `Samples` | 15 | At least three fit points per involute flank. |
| `EnableBore` | true | False skips both cuts and ignores both diameter inputs. |
| `BoreDiameter` | 0 | Zero auto-calculates the driving member's bore when enabled. |
| `MateBoreDiameter` | 0 | Zero auto-calculates the pinion bore when enabled. |
| `SpiralAngle` | 35° | Zero makes straight teeth; a curved tooth uses `(0, 60°)`. |
| `SpiralHand` | right | Right or left when `SpiralAngle` is positive. |
| `CutterRadius` | 0 | Zero selects the mean cone distance; otherwise positive. |
| `ToothSpacing` | 0 | Nonnegative clearance offset. |
| `ToeExtension` | 0 | Percentage in `[0, 100]`. |
| `ToeRadius` | 0 | Zero selects the derived toe radius; otherwise positive. |
| `MateToeRadius` | 0 | Zero selects the mate's derived toe radius; otherwise positive. |
| `Placement` | nil | `*r3.Frame`; see Placement below. Nil keeps the world frame. |
| `BaseHeight` | 0 | Zero selects the bounded default; otherwise within its limits. |
| `MateBaseHeight` | 0 | Zero scales the resolved first member's height by the tooth ratio; otherwise within the mate's limits. |

## Placement

`Placement` anchors the pair with a reference point and a reference plane. The
frame origin is the common pitch apex. The frame's `U` and `V` axes span the plane
that holds both shafts, and the driving shaft runs along `U`. The pinion shaft
turns toward `V` by `ShaftAngle`. With no placement the frame is the world
origin with `U=+X` and `V=+Y`, which is the construction described below.

`Build` and `BuildPair` construct every member about the world origin, then move each
finished body with one `decad.Transform` from the world frame onto `Placement`
(`r3.FromFrame`). The tooth Unions and trims fail on bodies built at an offset, so
the move happens after the last boolean. Reject an invalid frame before adding bodies.
The lower-level helpers (`BuildBlank`, `BuildPointTooth`, the cones, and the spiral
stages) ignore `Placement`.

The first real `Build` case is an eight-tooth member of an equal pair at
Module 1 and a 90° shaft angle, with straight teeth, five flank samples,
and no optional offsets. Both an unbored body and a body with a 1 mm
post-join bore pass geometry checks. `BuildPair` places two bored members
with perpendicular shafts and the source's driving half-tooth-pitch phase.
The source-default spiral pair also passes one 31-tooth Combine-Join and
an auto-calculated post-join bore for each hand. The straight 16/12 pair
passes the same closed-shell and through-bore checks. Other full-gear cases return
an error until their construction passes those checks. The individual blank,
tooth, cone, and bore helpers have separate tested parameter cases.

The source's 16 driving / 12 pinion pair uses Module 4, a 90° shaft angle,
15 flank samples, and no optional offsets. Its first real gate built the
smaller straight pinion without a bore.
The real 12-tooth pinion passes the source-order Combine-Join and 0.1 mm
`VerifyAll` with merged Decad PR #1421 in 128.11 seconds. Merged Decad
PR #1424 also builds the 16-tooth driving member in 176.63 seconds. The
unbored, placed 16/12 pair passes `VerifyAll` for both bodies in 528.89
seconds. The pinion's source-default automatic bore passes with merged
Decad PR #1429 in 156.85 seconds and leaves one through-hole. The driving
member's automatic bore passes with merged Decad PR #1432 in 188.89
seconds. The source 16/12 pair builds and places both bored members;
each passes 0.1 mm `VerifyAll` with one through-hole. That full pair test
takes 491.07 seconds under a 12-minute cap. Other tooth counts, spiral
angles, and toe controls need their own real-body gates.

The straight 4/4 pair uses Module 4, a 90° shaft angle, 15 flank samples,
and no optional offsets. Its virtual tooth count is 5.657, the smallest in
the source's solid table. The unbored member passes the source-order tooth
Combine-Join and 0.1 mm `VerifyAll` with merged Decad PR #1442. The
automatically bored member passes the same geometry checks with merged
Decad PR #1445 and has one through-hole. `BuildPair` places both bored
members; each passes 0.1 mm `VerifyAll` with one through-hole. That full pair
test takes 498.33 seconds locally under a 15-minute cap.

## Source-default pair target

The source-default request has Module 1, a 90° shaft angle, 31 teeth on each
member, a 35° mean spiral angle, and a right-hand driving trace. The pinion
uses the opposite hand. Zero face width, base heights, toe radii, cutter
radius, spacing, and toe extension select the source's derived values. The
source enables bores for both members by default. Each zero bore-diameter
input resolves independently to the smaller of that member's pitch diameter
divided by four and its maximum permitted bore diameter.

`ResolveBoreDiameter` implements that enabled per-member resolution from
the real blank section. For the default 90° pair it resolves 7.75 mm; for a
31/31 pair at a 35° shaft angle the heel bound caps the value. Both resolved
diameters produce a verified through-hole in an isolated real blank and after
the full 31-tooth join. `BuildPair` validates both bore inputs before building
either member and cuts each bore after its own pattern joins. The pinion
places on the second shaft, and the driving member rotates by half a tooth pitch.

## Derived geometry

For this member let `r = Module × Teeth / 2`, and let `rMate` use
`MateTeeth`. The pitch cone angle is
`γ = atan2(sin(ShaftAngle) × r, rMate + cos(ShaftAngle) × r)`.
The mate's angle is `ShaftAngle − γ`. The pitch cone distance is
`R = r / sin(γ)`. The two members use the same `R`.

Reject a shaft angle at or above `acos(-min(Teeth, MateTeeth) /
max(Teeth, MateTeeth))`, since one cone would reach 90°. Cap the permitted
shaft angle at 150°, inclusive when that cap is the limiting one. Reject a
member whose tooth count is less than `5.27 cos(γ)`; apply the corresponding
rule to the mate. Reject nonfinite values and any computed zero or negative
radius or width.

For each member, the base-height minimum is `1.05 × 1.25 × Module × sin(γ)`;
the maximum is `0.95 × (r − 1.25 × Module × cos(γ)) × tan(γ)`. When
`BaseHeight` is zero, clamp `Module × Teeth / 8` to that interval. Reject an
explicit height outside it.

`BuildBlankPair` treats `Teeth` as the driving member and `MateTeeth` as the
pinion. It builds both around the same apex. The pinion is rotated about the
world Z axis by `ShaftAngle` from the driving member's world X shaft. A zero
`MateBaseHeight` uses the driving member's resolved base height multiplied by
`MateTeeth / Teeth`, then clamps it to the mate's own bounds. The pair swaps
the two toe-radius inputs when building the pinion, so both blanks use the
same face width and root-element length.

The current straight-tooth build revolves the six-sided reference blank,
lofts one fitted back-cone tooth from the common apex, cuts that tooth at
the toe and heel cones, places eight rotated copies around the shaft, and
joins each copy to the blank with sequential binary `Union` calls. This is
the verified interim 8/8 construction. The source-default spiral member uses
one Combine-Join with the blank as target and all 31 patterned teeth as tools.
Both hands pass `VerifyAll` at 0.1 mm before and after their auto-calculated
bores. `BuildPair` rotates the pinion about world Z by the shaft angle, then
rotates the driving member by `π / Teeth` about its shaft.

For the reference's back-cone construction, one tooth is drawn on a plane
through the back-cone line. Its virtual pitch radius is `r / cos(γ)` and its
virtual tooth count is the real number `Teeth / cos(γ)`. Do not round that
count: it sets the involute flank's half-thickness angle
`π / (2 × virtual tooth count)`. The virtual tooth's base and tip radii are
`virtual pitch radius × cos(PressureAngle)` and
`virtual pitch radius + Module`. Its root radius is
`virtual pitch radius − 1.25 Module − 0.1125 Module`; the last term seats
the tooth root inside the gear blank for the later join. Use the existing
spur involute sampler for its two fitted flanks. Connect them with a tip arc,
a root arc, and radial lines when the base circle lies above the root.
When the base circle lies below the root, retain the fitted curves and trim
them at the root circle with Sketch rather than replacing them with chords.
Center the single tooth at 180° in its local plane, as the reference does;
this keeps the root-circle seam away from the tooth's trimmed root arc.
Place that plane through the point
`(R / cos γ + ToothSpacing sin γ, −ToothSpacing cos γ, 0)` relative to the
apex, with local horizontal direction `(sin γ, −cos γ, 0)` and local vertical
direction `(0, 0, 1)`. Its normal stays at perpendicular distance `R` from
the apex when spacing changes. The center moves past the shaft axis by the
specified offset; the virtual tooth count and radii do not change.

`BuildPointTooth` uses that fitted back-cone profile as the far section of
`decad.LoftFromPoint`, with the exact common apex at world `(0, 0, 0)`.
It returns one untrimmed tooth solid. The blank-only base-height and toe
controls do not change this intermediate tooth; they set the later blank
and conical trims. The point loft itself does not cut a bore or curve the
tooth. It must pass `VerifyAll` and include the exact
origin vertex. Its current faceted faces do not establish the final
analytic flank or cone surfaces.

When `FaceWidth` is zero, start with
`sqrt((2r)^2 + (2rMate)^2) / 6`. Cap that default at
`0.95 R × min(sin²γ, sin²γMate)` and reject an explicit width above that
limit. The bound comes from both members, so their tooth bands meet at the
same stations. Reject a width that collapses either toe section's root.
The reference Fusion model's toe extension, toe radius, bore, spacing, and
spiral rules are source material for those later branches; each branch must
retain its own geometry check.

The first default-spiral stage uses the same untrimmed point tooth with
31/31 teeth, a 35° spiral angle, and right hand selected for the driving
member. Eight sheets parallel to the parent tooth plane split it at
successive `span / 6` offsets toward the apex. The apex scrap is removed;
the eight remaining solids pass `VerifyAll`. Their root-cone distances feed
the source's cutter-circle near intersections. The developed azimuth is
divided by `sin γ` using the **pitch** cone angle, and the pinion trace is
mirrored. The private spiral-tooth construction rejects any non-default
module, tooth counts, shaft angle, face width, pressure angle, sample count,
spacing, toe controls, base heights, cutter radius, spiral angle, or driving
hand. The bore diameter does not change this intermediate tooth.

For this source-default case, step G rotates each slab about its shaft by the
trace angle keyed to the post-slice heel-face centroid. Step H scales seven
slabs about the root edge of each post-twist heel face; it leaves the
outermost heel slab full size. Step I reorders the post-crown heel faces and
lofts the toe-most toe face followed by all eight heel faces. The resulting
driving and pinion tooth solids each pass `VerifyAll` at 0.1 mm. The real toe
cone then cuts off each tooth's apex-side tip, and each result passes the
same verification against the Decad toe-cut branch. The heel `Intersect`
also passes `VerifyAll` at 0.1 mm for both members against the reviewed
Decad heel branch. Each trimmed tooth produces 31 placed copies, and every
copy passes the same check. The first tooth-to-blank `Union` for each member
passes `VerifyAll` at 0.1 mm. The full patterned Combine-Join and post-join
bores pass `VerifyAll` for both members. The placed source-default pair passes
the same check, and the renderer produces its 1600 × 900 PNG.

## Reference gear blank

The reference's gear blank is a six-sided axial profile revolved around the
shaft, with the common pitch apex at station zero. Write `d=1.25 Module`.
The heel pitch point is `(R cos γ, r)` in `(shaft station, radius)` coordinates.
The dedendum corner is
`C=(R cos γ + d sin γ, r − d cos γ)`. The heel back face is at station
`R cos γ + BaseHeight`, with radius `r − BaseHeight / tan γ`. Resolve a zero
base height by clamping `Module × Teeth / 8` to the minimum and maximum
above; reject an explicit value outside those bounds.

At Toe Extension zero, the root-element length from the dedendum corner to
the toe is `FaceWidth × hypot(R, d) / R`. Scale `C` about the apex by
`1 − root length / hypot(R, d)` to locate the toe root corner. A zero Toe
Radius resolves to `r − FaceWidth / sin γ`; a nonzero value is explicit.
The mate resolves its toe radius the same way. For a positive Toe Extension,
each radius must be below its ceiling
`(r − d cos γ) × (1 − FaceWidth / R)`. The root-length limit for each
member is `hypot(R, d) − ToeRadius / sin(γ − atan(d / R))`.
The pair uses the smaller limit. Reject a positive extension when that limit
does not exceed the zero-extension length. Otherwise add
`ToeExtension / 100 × 0.99` of the available length to the zero-extension
length. Reject a percentage outside `[0, 100]`.
The toe's inner point lies on the line from the toe root corner in direction
`(sin γ, −cos γ)` at the requested toe radius. Its shaft station locates
the toe axis foot. Revolve the closed sequence: toe axis foot, heel axis
foot, heel back corner, dedendum corner, toe root corner, toe inner point.
The result has two planar annular ends and three conical faces: heel cone,
root cone, and toe dish. A through-bore and teeth are later operations on
this real blank.

For a positive through-bore, the heel radius is
`r − resolved BaseHeight / tan γ`. The toe root radius is
`(hypot(R, d) − resolved root length) × sin(γ − atan(d / R))`.
Reject a bore diameter above `1.9 × min(heel radius, toe root radius)`.
The blank-only `BuildBoredBlank` helper cuts a cylinder along the shaft
from before the toe to beyond the heel. That isolated test must leave one
solid with a verified through-hole. `Build` cuts the resolved bore from
the **joined** patterned body, after Combine-Join.

## Conical end cutting tools

`BuildToeCone` accepts the same straight-blank inputs as `BuildBlank`. It
uses the resolved toe-root corner `(s, q)` from that blank. Extend the toe
dish line to the shaft axis at station `a = s + q tan γ`. Revolve the
triangle with vertices `(a, 0)`, `(-0.1 R, 0)`, and
`(-0.1 R, (a + 0.1 R) / tan γ)` around the shaft. This makes the solid
inside the exact toe cone, with half-angle `90° − γ`. The far cap lies
past the shared pitch apex so that a later tooth cut has no extra cap
inside the tooth band. Reject any blank input that cannot resolve a
positive cone radius or an apex beyond that cap. The cone is a tool; the
tooth still needs a successful cut and keeper selection before assembly.

`BuildHeelCone` revolves the same solid-cone section with its apex at
`R / cos γ`, where the blank's heel-cone edge extended meets the shaft.
Both tools open toward the common pitch apex and end at station `-0.1 R`.
Trim the point tooth with the blank's toe cone face first, then its heel
cone face. The toe face must split the tooth. A heel face that does not
intersect the toe keeper leaves it intact. After each split, remove pieces
that contain the pitch apex and keep the largest remaining piece, as the
reference does. A solid-cone `Cut` result alone does not identify that
keeper. Check both sides of a real heel split before choosing one: the
spaced 16/12 tooth at 90° has material on both sides of its heel cone.

## Sketch constraints

Anchor every axial section to its sketch origin and dimension its shaft
stations and radii. Use the same rule for the six-sided blank and triangular
toe and heel cone tools. On the tilted back-cone plane, use that plane's
origin as the virtual tooth's center. Dimension fitted involute points from
it and share endpoints between adjoining curves. Each tip or root arc
relates its endpoint radii, so dimension only the needed tangential
coordinate of one endpoint. For the embedded-flank route, dimension the
separate root circle to the derived root radius. Do not fix every computed
point.

For each spiral slicing sheet, anchor and dimension the two line endpoints
on its sketch plane before extruding the sheet. When a bore is enabled,
center its circle on the shaft-normal sketch origin and dimension its
resolved diameter. Solve and verify every blank, cone, tooth, slicing-sheet,
and optional bore sketch before using its geometry or returning it. Require
`Sketch.Verify(ctx)` to report `sketch.FullyConstrained`, zero degrees of
freedom, no redundant constraints, and no conflicts. Check the ordinary
and embedded tooth routes, straight and spiral construction, and both bore
states with their real builders.

## Geometry checks

The first full-gear case must produce one solid with eight tapered teeth
joined to the reference blank, one closed outer shell before the bore,
one through-hole after a 1 mm bore, and a `VerifyAll` mesh. Later cases must
also check unequal
tooth counts, non-90° shaft angles, both flank/root routes, bore surfaces,
toe and spacing changes, and right and left spiral traces.
