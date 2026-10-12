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
are millimeters and angles are radians. A negative angle reverses the chevron
hand; the full angle rotates the middle section relative to the lower one.

The inherited spur checks reject nonpositive module or thickness, fewer than
three teeth or flank samples, pressure angles outside `(0, π/2)`, negative bore
diameter or cap chamfer, and a bore reaching the root circle. A nonfinite helix
angle is invalid. The ordinary six-curve tooth section requires its base circle
to lie outside the root circle. The source sets no helix-angle magnitude limit.

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
The joined tooth alone passes the same mesh check for signed 14.5°, 35°, and
90° angles. Its bounds span the full thickness and its centroid lies at the
mirror plane.
The same small gear with a 1 mm through-bore and the default gear with a 3 mm
through-bore pass `VerifyAll` on merged Decad `6ab45eb70643`. Their mesh Euler
characteristic is zero. The separate approved outer-cap Chamfer repair has not
passed `VerifyAll` on the bored gear yet.
