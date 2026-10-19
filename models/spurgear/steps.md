# Spur gear Decad construction steps

These steps implement [instructions.md](instructions.md). All lengths are
millimeters. `Build(ctx, doc, Params)` returns one final Decad body.

1. Validate inputs and derive pitch, base, root, and tip radii. Compute the
   root fillet radius from the instructions. A nonpositive fillet radius
   selects the no-fillet path.
2. For each tooth, sample the original involute at `S` radii from base to
   tip. Mirror and rotate the left flank into its right mate, then rotate
   the pair by the tooth pitch angle. Preserve fit splines through those
   samples rather than replacing them with line segments.
3. Build the profile at z=0 and z=`h` in separate Sketch planes. In the
   ordinary branch (`Rbase >= Rroot`), draw radial flank-to-root lines,
   tip arcs, and root arcs into one closed loop. Require exactly one valid
   Sketch profile.
4. In the embedded branch (`Rbase < Rroot`), draw both original fit splines
   and the tip arc for each tooth, plus one full root circle. Solve the
   sketch and union all valid bounded regions with `UnionProfiles`. Require
   one hole-free profile with `4N` outer edges. For each partial fit spline,
   require a certified trim parameter and its start on the root circle.
   Retain those `2N` exact contact points for root-edge selection.
5. Loft the matching lower and upper profiles with `Document.Loft`. Require
   one solid with cap planes at z=0 and z=`h`.
6. When the derived fillet radius is positive, select the `2N` axial edges
   at the saved root contacts and call `Body.Fillet` once. Require the exact
   edge count. Skip this step for a zero or negative radius.
7. When the bore diameter is positive, extrude a centered circular tool
   from below the lower cap to above the upper cap. Cut it from the gear
   with `decad.Cut`. Require one solid and a through-hole of the requested
   diameter.
8. When the cap chamfer is positive, select the outer loop of each cap and
   call `Body.Chamfer`. The bore loops remain unselected and sharp. Require
   one cap face at each end and one outer loop, plus a bore loop when present.
9. Verify the ordinary default gear, the embedded 30-tooth/30° no-fillet
   gear, and the embedded 60-tooth/20° filleted gear. Exercise bore and
   chamfer independently and together. Check dimensions, topology, and
   geometry verification on the complex final body.

## Sketch constraints

Every sketch is verified by `modelsketch.Verify` before its profile is read.
It must report `FullyConstrained`, `DOF == 0`, and no redundant or
conflicting constraints. `sketch_constraints_test.go` checks each one.

- The center is fixed at the sketch origin. A fixed construction line along
  +X is the angle reference.
- Each flank, tip, and root point has a construction spoke to the center, an
  angle dimension from the reference, and a radius dimension. The tip arc
  and the root arc tie the radius of their second end point, so that end
  point has no radius dimension.
- The embedded branch dimensions the root circle radius.
- The bore sketch fixes its center and dimensions the circle diameter.
