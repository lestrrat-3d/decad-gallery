# Helical gear Decad construction steps

These steps implement [instructions.md](instructions.md). Lengths are
millimeters, and `HelixAngle` is a signed angle in radians.

1. Validate the inherited spur inputs and a finite helix angle. Derive the
   spur pitch, base, root, and tip radii. Reject a tooth whose base circle
   lies inside its root circle; this loft uses the ordinary tooth profile.
2. Draw one closed six-curve tooth profile on the z=0 Sketch plane. Use the
   spur tooth sampler's fitted involute flanks, tip and root arcs, and radial
   flank-to-root lines.
3. Draw the same profile on the plane at z=`Thickness`, rotating every point
   by the signed `HelixAngle`. Loft the lower Sketch profile to the upper
   one to create one twisted tooth solid.
4. Draw the root-circle region on the lower plane and extrude it through
   `Thickness`. Copy the twisted tooth around the shaft at tooth-pitch
   angles, then join each copy to the root disc.
5. Compute the spur root radius multiplied by `cos(HelixAngle)` and skip
   the fillet if that result is nonpositive. Otherwise search the root
   cylinder for straight axial edges. Fillet the selected edges with that
   radius, or skip the operation when the selector finds none.
6. When `BoreDiameter` is positive, cut a centered cylinder through the
   joined gear. When `CapChamfer` is positive, chamfer the outer loop of
   each completed-gear cap. Leave bore rims out of the chamfer selection.
7. Tessellate the completed body with `VerifyAll`. Check that it is one
   solid and that a requested bore remains a through-hole. The 8-tooth
   case passes with and without a bore and cap chamfers. The source-default
   17-tooth gear passes with a 1 mm bore and both 0.1 mm cap chamfers. A
   case that selects nonempty root-fillet edges still needs a real check.
