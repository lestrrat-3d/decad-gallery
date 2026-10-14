# Spur gear Decad construction steps

These steps implement [instructions.md](instructions.md). All lengths are
millimeters. `Build(ctx, doc, Params)` returns one final Decad body.

1. Validate inputs and derive pitch, base, root, and tip radii. Compute the
   root fillet radius from the instructions. A nonpositive fillet radius
   selects the no-fillet path.
2. For each tooth, sample the original involute at `S` radii from base to
   tip. Mirror and rotate the left flank into its right mate, then rotate
   the pair by the tooth pitch angle. Preserve fit splines through those
   samples rather than replacing them with line segments. Constrain each
   fit point with signed horizontal and vertical dimensions from the sketch
   origin, except for the arc endpoints in step 3. Share points at curve
   joins instead of fixing coordinates.
3. Build the profile at z=0 and z=`h` in separate Sketch planes. In the
   ordinary branch (`Rbase >= Rroot`), draw radial flank-to-root lines,
   tip arcs, and root arcs into one closed loop. Each arc already relates
   its endpoint radii, so dimension only the tangential coordinate of one
   endpoint. Solve and verify each sketch before requiring exactly one valid
   Sketch profile.
4. In the embedded branch (`Rbase < Rroot`), draw both original fit splines
   and the tip arc for each tooth, plus one full root circle. Dimension
   the root circle to `Rroot`. Solve and verify the sketch, then union all
   valid bounded regions with `UnionProfiles`. Require
   one hole-free profile with `4N` outer edges. For each partial fit spline,
   require a certified trim parameter and its start on the root circle.
   Retain those `2N` exact contact points for root-edge selection.
5. Loft the matching lower and upper profiles with `Document.Loft`. Require
   one solid with cap planes at z=0 and z=`h`.
6. When the derived fillet radius is positive, select the `2N` axial edges
   at the saved root contacts and call `Body.Fillet` once. Require the exact
   edge count. Skip this step for a zero or negative radius.
7. When the bore diameter is positive, extrude a centered circular tool
   from below the lower cap to above the upper cap. Use the sketch origin
   as its center and dimension its diameter to `b`. Solve and verify the
   tool sketch before extrusion. Cut it from the gear
   with `decad.Cut`. Require one solid and a through-hole of the requested
   diameter.
8. When the cap chamfer is positive, select the outer loop of each cap and
   call `Body.Chamfer`. The bore loops remain unselected and sharp. Require
   one cap face at each end and one outer loop, plus a bore loop when present.
9. Verify the ordinary default gear, the embedded 30-tooth/30° no-fillet
   gear, and the embedded 24-tooth/30° filleted gear. Check bore removal
   volume with `CutThroughBore` on a plain extruded box. Exercise chamfer
   alone and bore plus chamfer on real gears. Check dimensions,
   topology, and geometry verification on the complex final bodies. For every
   profile, standalone tooth section, and optional bore sketch, require
   `Sketch.Verify(ctx)` to report `sketch.FullyConstrained`, `DOF == 0`,
   no redundant constraints, and no conflicts before consuming its geometry.
