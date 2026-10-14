# Herringbone gear Decad construction steps

These steps implement [instructions.md](instructions.md). The tooth section,
root disc, fillet, bore, and cap chamfer reuse the helical and spur packages.

1. Reject a `HelixAngle` outside +1°–+45°. Create Sketch tooth profiles on
   z=0 and z=`Thickness / 2`. Rotate the middle profile by `HelixAngle` while
   drawing it. Use the spur tooth constraints to anchor, dimension, solve,
   and verify both sections before consuming their profiles.
2. Call `Document.Loft` on the two verified profiles. Use `MirroredCopy` across
   the middle Sketch plane, then `Union` the halves. Check that the joined
   tooth is one solid with a closed verified mesh.
3. Center and dimension the helical root-disc circle. Solve and verify its
   sketch, then extrude it through `Thickness`. Make `Teeth - 1` rigid copies
   of the joined tooth at successive pitch angles. Pass the original and all
   copies to one variadic `Union` with the disc as target.
4. Use the helical root-edge selector and its cosine-scaled fillet radius.
   When requested, center and dimension the spur bore-tool circle, solve and
   verify its sketch, then apply the Cut. Apply the outer-cap Chamfer when
   requested.
5. Tessellate the finished gear at 0.1 mm with `VerifyAll`. Check one solid,
   one connected closed shell without a bore, or one through-hole with a bore.
   Check the full height and opposite tooth slopes around the middle plane.

The source-order eight-tooth and default 17-tooth gears pass step 5 without a
bore and with 1 mm and 3 mm bores, respectively. Both bored meshes have one
through-hole. Both also pass with 0.1 mm outer-cap chamfers.
