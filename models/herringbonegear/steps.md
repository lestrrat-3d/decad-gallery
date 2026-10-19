# Herringbone gear Decad construction steps

These steps implement [instructions.md](instructions.md). The tooth section,
root disc, fillet, bore, and cap chamfer reuse the helical and spur packages.

1. Reject a `HelixAngle` outside +1°–+45°. Create Sketch tooth profiles on
   z=0 and z=`Thickness / 2`. Rotate the middle profile by `HelixAngle` while
   drawing it.
2. Call `Document.Loft` on the two solved profiles. Use `MirroredCopy` across
   the middle Sketch plane, then `Union` the halves. Check that the joined
   tooth is one solid with a closed verified mesh.
3. Extrude the root disc through `Thickness`. Make `Teeth - 1` rigid copies
   of the joined tooth at successive pitch angles. Pass the original and all
   copies to one variadic `Union` with the disc as target.
4. Use the helical root-edge selector and its cosine-scaled fillet radius.
   Apply the spur through-bore Cut and outer-cap Chamfer in that order when
   their dimensions are positive.
5. Tessellate the finished gear at 0.1 mm with `VerifyAll`. Check one solid,
   one connected closed shell without a bore, or one through-hole with a bore.
   Check the full height and opposite tooth slopes around the middle plane.

The source-order eight-tooth and default 17-tooth gears pass step 5 without a
bore and with 1 mm and 3 mm bores, respectively. Both bored meshes have one
through-hole. Both also pass with 0.1 mm outer-cap chamfers.

## Sketch constraints

The lower and middle tooth sections come from `spurgear.BuildToothSection`.
It verifies them as fully constrained, as described in the spur steps. The
root disc and bore sketches are the helical and spur ones, which are also
verified.
