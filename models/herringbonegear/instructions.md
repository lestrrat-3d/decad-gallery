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

## Required build order

1. Draw one ordinary spur involute tooth section on the z=0 plane. Draw the
   same section rotated by `HelixAngle` on the plane at `Thickness / 2`.
2. Loft the lower section to the middle section. Mirror that solid across the
   middle plane and join the two halves into one tooth.
3. Extrude the root-circle region across the full thickness. Circular-pattern
   the completed tooth around the shaft, including the original tooth. Join
   the pattern to the disc in one Combine-Join.
4. Apply the inherited helical root-fillet selection. Its radius is the spur
   root-fillet radius multiplied by `cos(HelixAngle)`. Skip it when the radius
   is nonpositive or no straight axial root edges match.
5. Cut a centered through-bore when `BoreDiameter` is positive. Chamfer only
   the completed gear's outer cap edges when `CapChamfer` is positive; leave
   bore rims sharp.

The first real gallery case uses eight teeth, five flank samples, 2 mm
thickness, and the default +14.5° angle. It and the source-default 17-tooth
case pass `Tessellate(0.1 mm, VerifyAll)` after one variadic `Union` on merged
Decad `96cdbfe9610e`.
The joined tooth alone passes the same mesh check for +14.5° and +35° angles.
Its bounds span the full thickness and its centroid lies at the mirror plane.
The same small gear with a 1 mm through-bore and the default gear with a 3 mm
through-bore pass `VerifyAll`. Their mesh Euler characteristic is zero. Both
gears also pass `VerifyAll` and retain Euler characteristic zero with 0.1 mm
outer-cap chamfers on merged Decad `9dee32fbe599`.
