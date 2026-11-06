# Herringbone gear Decad construction steps

These steps implement [instructions.md](instructions.md). The whole-outline loft, root fillet,
bore, and cap chamfer reuse the helical and spur packages.

1. Reject a `HelixAngle` outside +1°–+45°. Call `helicalgear.BuildFilletedLoft` with the plane offset
   `thickness / 2`. It draws the whole toothed outline on z=0 and the same outline turned by `HelixAngle` on the
   plane at `helix_plane_offset`, anchors, dimensions, solves and verifies both with the spur builder, lofts them
   and rounds the `2N` root seams with `helical_fillet_radius`.
2. Mirror the filleted half across the middle plane at `helix_plane_offset` with `decad.MirroredCopy`, then
   `decad.Union` the halves. Check that the joined gear is one solid with a closed verified mesh. There is
   no root disc, tooth pattern or per-tooth union.
3. When requested, cut the centered spur bore with `spurgear.CutThroughBore` (center and dimension its tool
   circle, solve and verify its sketch, then apply the Cut). Apply the outer-cap chamfer with
   `spurgear.ChamferOuterCaps` when requested. Both go through `helicalgear.Finish`.
4. Build steps 1-3 on the local frame.
5. When `Placement` is set, reject an invalid frame first. Build steps 1–3 on the
   local frame, then call `decad.Transform` with `r3.FromFrame(Placement)` on
   the finished gear. Check the placed gear's volume and bounds against the
   unplaced gear, for normals of both signs.
6. Tessellate the finished gear at 0.1 mm with `VerifyAll`. Check one solid,
   one connected closed shell without a bore, or one through-hole with a bore.
   Check the full height and opposite tooth slopes around the middle plane.

The source-order eight-tooth and default 17-tooth gears pass step 5 without a
bore and with 1 mm and 3 mm bores, respectively. Both bored meshes have one
through-hole. Both also pass with 0.1 mm outer-cap chamfers.
