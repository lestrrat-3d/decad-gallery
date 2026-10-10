# Bevel gear construction instructions

## Inputs and supported range

`Build(ctx, doc, Params)` builds one member of a meshing bevel pair. The caller
supplies its tooth count and the mate's tooth count; both members use the same
module and shaft angle. Lengths are millimeters and angles are radians. A
caller can place the returned body in an assembly after construction.

| Input | Default | Rule |
| --- | ---: | --- |
| `Module` | 1 mm | Finite and positive. |
| `Teeth` | 31 | At least three, and at least the computed cone-angle floor below. |
| `MateTeeth` | 31 | Same tooth-count rule, using the mate's cone angle. |
| `ShaftAngle` | 90° | From 30° through the maximum shaft angle below. |
| `FaceWidth` | 0 | Zero selects the derived width; a positive value is explicit. |
| `PressureAngle` | 20° | Greater than zero and below 90°. |
| `Samples` | 15 | At least three fit points per involute flank. |
| `BoreDiameter` | 0 | Zero omits the bore; a positive value must fit the toe and heel. |
| `SpiralAngle` | 0 | Zero makes straight teeth; a curved tooth uses `(0, 60°)`. |
| `SpiralHand` | right | Right or left when `SpiralAngle` is positive. |
| `CutterRadius` | 0 | Zero selects the mean cone distance; otherwise positive. |
| `ToothSpacing` | 0 | Nonnegative clearance offset. |
| `ToeExtension` | 0 | Percentage in `[0, 100]`. |
| `ToeRadius` | 0 | Zero selects the derived toe radius; otherwise positive. |
| `BaseHeight` | 0 | Zero selects the bounded default; otherwise within its limits. |

The first real build is an eight-tooth member of an equal pair at a 90° shaft
angle, with straight teeth, no bore, and no optional offsets. That build must
pass its geometry checks before more inputs are enabled. A parameter that the
construction has not yet implemented returns an error; it must not silently
change the gear or produce substitute geometry.

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

The initial straight-tooth construction uses two sections on the member's
shaft. The heel section is at the pitch radius `r`. The toe section is nearer
the common apex by the face width, so its pitch radius is
`rToe = r × (1 − FaceWidth / R)`. The section scale is `rToe / r`.
The heel tooth is an involute with base radius `r cos(PressureAngle)`, root
radius `r − 1.25 Module`, and tip radius `r + Module`. Its flanks are sampled
as in the [spur gear instructions](../spurgear/instructions.md). The toe uses
the same angular profile at the section scale. Corresponding Sketch fit
splines, arcs, and radial edges form a ruled Decad loft, so each tooth tapers
toward the shared pitch apex. Both sections must contain one valid region.

When `FaceWidth` is zero, start with
`sqrt((2r)^2 + (2rMate)^2) / 6`. Bound the result below the apex and the
toe-section root radius. Reject an explicit width that reaches either limit.
The full pair construction must use the smaller member's face-width bound,
so the two tooth bands meet at the same stations. The reference Fusion model's
toe extension, toe radius, bore, spacing, and spiral rules are source material
for those later branches; each branch must retain its own geometry check.

## Geometry checks

The first case must produce one solid with `Teeth` tapered teeth, a narrower
toe than heel, and axial bounds equal to the two section stations. Tessellate
the real Decad body with `VerifyAll`. Later cases must also check unequal
tooth counts, non-90° shaft angles, both flank/root routes, bore surfaces,
toe and spacing changes, and right and left spiral traces.
