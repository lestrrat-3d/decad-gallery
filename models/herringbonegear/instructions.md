# Herringbone gear construction instructions

The canonical construction is
`fusion360-gear-generator/spec/herringbonegear/instructions.md`. It inherits
the helical and spur gear parameters and construction. Herringbone changes the
tooth: the twisted section lies at half the body thickness, and the lofted
lower half is mirrored across that plane and joined to make a chevron.

## Inputs and invalid cases

`Params` is the [helical parameter set](../helicalgear/instructions.md). It adds
no herringbone input. The default has Module 1, 17 teeth, 20° pressure angle,
10 mm thickness, 15 fitted points per flank, and a +14.5° helix angle. Lengths
are millimeters and angles are radians. The full angle rotates the middle
section relative to the lower one.

The inherited spur checks reject nonpositive module or thickness, fewer than
three teeth or flank samples, pressure angles outside `(0, π/2)`, negative bore
diameter or cap chamfer, and a bore reaching the root circle. The inherited
gallery helix-angle input accepts only finite angles from +1° through +45°.
The ordinary six-curve tooth section requires its base circle to lie outside
the root circle. The Fusion source sets no helix-angle range.

## Parameters

`Build` runs the helical variant of the spur pipeline with one change: the
parameter `helix_plane_offset` is `thickness / 2`, so the twisted section, the
mirror plane and the loft all read the mid-body height from the table. The
other helical parameters (`helix_angle`, `helical_fillet_radius`) and every
spur parameter are unchanged and appear in `doc.Bindings()` when read.

## Sketch constraints

The lower and rotated middle tooth sections use the
[spur tooth constraints](../spurgear/instructions.md#sketch-constraints).
Anchor each section to its plane origin and constrain its fitted flanks and
shared arc endpoints before lofting. The inherited helical root-disc sketch
uses an origin-centered circle dimensioned to the root radius. The optional
spur bore sketch uses an origin-centered circle dimensioned to the requested
diameter. Do not fix every computed point.

Solve and verify every tooth section, root-disc sketch, and optional bore
sketch before its geometry is consumed. Require `Sketch.Verify(ctx)` to report
`sketch.FullyConstrained`, zero degrees of freedom, no redundant constraints,
and no conflicts. Exercise the optional bore branch in a real herringbone
build. Mirroring the tooth body creates no additional sketch.

## Placement

`Params` carries the spur gear's optional `Placement` (`*r3.Frame`). Its origin is
the reference point on the gear axis, its `U` and `V` axes span the reference plane
that holds the lower cap, and the gear grows along `U × V`. `Build` constructs the
whole gear about the local origin and +Z, then moves the finished body with one
`decad.Transform` from the local frame onto `Placement`. A nil `Placement` leaves
the gear on the world XY plane at the origin. An invalid frame is rejected before
any body is added. The tooth Unions fail on bodies built at an offset, so the move
follows the last boolean. `BuildTooth` works in the local frame and ignores
`Placement`.

## Required build order

1. Draw the whole ordinary spur outline on the z=0 plane. Draw the same outline rotated by `HelixAngle` on the
   plane at `Thickness / 2`.
2. Loft the lower outline to the middle outline and round the `2N` root seams with the spur root-fillet radius
   multiplied by `cos(HelixAngle)` (the helical root fillet). Skip the fillet when the radius is nonpositive.
3. Mirror that half across the middle plane and join the two halves with one union. There is no root disc and
   no tooth pattern.
4. Cut a centered through-bore when `BoreDiameter` is positive. Chamfer only
   the completed gear's outer cap edges when `CapChamfer` is positive; leave
   bore rims sharp.

The first real gallery cases are eight teeth, five flank samples and 2 mm thickness, and the default 17-tooth
gear, both with the default +14.5° angle. Each builds one solid with `2N` fillet roles that passes
`Tessellate(0.1 mm, VerifyAll)`. `BuildTooth` still builds one chevron tooth for callers that pattern teeth
themselves; the joined tooth alone passes the same mesh check for +14.5° and +35° angles.
