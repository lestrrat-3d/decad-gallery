# Spur gear construction instructions

## Inputs

Build one cylindrical gear with straight involute teeth. Inputs are module
`m` in millimeters, integer tooth count `N`, pressure angle `α` in radians,
axial thickness `h` in millimeters, and `S` involute samples per flank.
The default values are `m=1`, `N=17`, `α=20°`, `h=10`, and `S=15`.
An optional bore has diameter `b` in millimeters; `b=0` omits it.
An optional equal-distance cap chamfer has setback `c` in millimeters;
`c=0` omits it.

## Derived dimensions

- Pitch radius: `mN/2`.
- Base radius: `(mN/2) cos α`.
- Root radius: `(mN - 2.5m)/2`.
- Tip radius: `(mN + 2m)/2`.
- Root fillet radius: `0.9 Rroot (π/N - 2(tan α - α))/2`.

Reject inputs with nonpositive radii, thickness, or module; fewer than three
teeth or three samples; pressure angle outside `(0, π/2)`; negative bore or
chamfer; a bore reaching the root circle; or nonpositive root fillet radius.

## Tooth flank

Sample `S` radii from base to tip, including both endpoints:
`rᵢ = Rbase + (Rtip - Rbase)i/(S-1)`. For each radius, let
`t = tan(acos(Rbase/rᵢ))`. The unrotated involute point is
`(Rbase(cos t + t sin t), Rbase(sin t - t cos t))`.
Mirror its Y coordinate to form the left flank's inward spiral.

At the pitch radius, compute the mirrored involute point `(px, py)`.
Rotate the left flank by `π/(2N) - atan2(py, px)`, so its pitch crossing is
`π/(2N)` above +X. Mirror the rotated left flank across X to form the
right flank. Every tooth is a rigid copy rotated by `2πk/N`.

## Body

For each tooth, connect the right flank from base to tip, the tip arc from
right to left, and the left flank from tip to base. When the base radius is
outside the root radius, extend both flank starts radially to the root circle.
Connect each tooth's left root contact to the next tooth's right root contact
with a root-circle arc. The continuous closed outline must have smooth
fit-spline flanks, not chord replacements. Extrude it by `h` along +Z.

If the base radius lies inside the root circle, trim the flanks at their
root-circle intersections without changing the fit-spline shape.

Round the two axial root corners of every tooth with the derived fillet
radius. If `b>0`, cut a centered through-bore of that diameter. If `c>0`,
chamfer the outer edge of both end caps; leave bore edges sharp.

## Verification

Check that the sketch has one valid outer region, the result is one solid,
all `N` teeth are present, the bounds match the tip radius and thickness,
and the optional bore and chamfer change the expected surfaces. Exercise
both an ordinary gear and an embedded-flank gear.
