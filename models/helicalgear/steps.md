# Helical gear Decad construction steps

These steps implement [instructions.md](instructions.md). Lengths are
millimeters, and `HelixAngle` is a positive angle in radians.

1. Validate the inherited spur inputs and a finite 1°–45° helix angle. Define the
   spur parameters with `spurgear.DefineParameters` plus `helix_angle`, `helix_plane_offset` and
   `helical_fillet_radius` in `doc.Params()` and the sketch world's table. Reject a tooth whose base circle
   lies inside its root circle; the whole-outline fillet uses the ordinary profile.
2. Draw the whole toothed outline on the z=0 Sketch plane with `spurgear.BuildTwistedProfile` at twist 0. Keep
   its sketch, its profile and its `2N` root contacts.
3. Draw the same outline on the plane at `helix_plane_offset` with twist `helix_angle`, so every flank is
   turned in sketch coordinates and the plane is not turned. Keep its profile and root contacts. Both
   sketches are solved and verified by the spur builder. Call `decad.Loft` on the two verified profiles.
4. Select the root seams with one clause per root contact,
   `decad.Edges(decad.EndpointAt(lower contact at z=0), decad.EndpointAt(turned contact at helix_plane_offset))`,
   joined with `Or`, and `.Exactly(2N)`. Read `helical_fillet_radius`, skip the fillet if it is nonpositive,
   and otherwise `decad.Fillet` the selection with it.
5. When `BoreDiameter` is positive, cut a centered cylinder through the
   filleted gear with `spurgear.CutThroughBore` on the local frame. When `CapChamfer`
   is positive, call `spurgear.ChamferOuterCaps`, which chamfers the outer loop of
   each cap and leaves bore rims out of the chamfer selection.
6. Build steps 1-5 on the local frame; the placement step below comes last.
7. When `Placement` is set, validate it before adding bodies. Build steps 1–5
   on the local frame, then call `decad.Transform` with `r3.FromFrame(Placement)`
   on the finished gear. Do not build sketches or run booleans on offset planes.
   Check the placed gear's volume against the unplaced gear, and its bounds
   along the frame normal and about the reference point, for normals of both signs.
8. Tessellate the completed body with `VerifyAll`. Check that it is one
   solid and that a requested bore remains a through-hole. The 8-tooth
   case passes with and without a bore and cap chamfers. The source-default
   17-tooth gear passes with a 1 mm bore and both 0.1 mm cap chamfers. Count the
   distinct face origin roles that start with `fillet(`: 16 for the 8-tooth case and 34 for the default.
