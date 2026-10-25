# Bevel gear Decad construction steps

These steps implement [instructions.md](instructions.md). The canonical
operation order comes from `fusion360-gear-generator/spec/bevelgear/instructions.md`,
under "Create the Gear Bodies". Lengths are millimeters. `Build(ctx, doc,
Params)` returns one straight 8/8 member or one spiral
member, each with an optional bore. `BuildPair` places both members of either
supported pair.

1. Validate both tooth counts, the shaft angle, module, face width, and
   optional inputs before adding any body. The full build accepts the tested
   Module 1 straight 8/8 case with five flank samples, or the source-default
   Module 1 spiral 31/31 case with fifteen samples. Enabled zero bore inputs
   select each member's bounded automatic diameter.
2. Resolve the shared pitch cone distance, both cone angles, and the face
   width. Draw the reference's six-sided axial blank profile. Dimension its
   stations and radii from the sketch origin, then solve and verify it before
   revolving about the shaft. Check its two planar ends, three conical surfaces,
   and `VerifyAll` mesh.
3. Draw one fitted involute tooth on the tilted back-cone plane. Keep
   `Teeth / cos(γ)` as a real number. Anchor the tooth center to the sketch
   origin; dimension its fit points and arcs for both root routes. Solve and
   verify the Sketch profile before lofting
   it from the exact common apex with `decad.LoftFromPoint`.
4. Dimension, solve, and verify the triangular axial sections before
   revolving the cutting cones from the blank's toe-dish and heel-cone lines.
   Cut the point tooth with the toe cone first. Intersect that keeper with
   the heel cone for the verified 8/8 case. The tooth retains fitted flank
   and conical end faces, and its mesh passes `VerifyAll`.
5. Make all seven `PlacedCopy` instances before consuming the original
   tooth in a Boolean. Rotate them around the shaft at successive 45°
   increments. Join the original and each copy to the revolved blank with
   sequential Decad `Union` calls. This is the verified interim 8/8 path.
6. Check that the joined body is one solid and its unbored mesh has one
   closed outer shell. `Tessellate(0.1 mm, VerifyAll)` verifies its
   boundary and volume.
7. When requested, cut the bounded through-bore **after** the tooth pattern
   and tooth join. Center the bore circle on its shaft-normal sketch origin,
   dimension its resolved diameter, then solve and verify it before the Cut.
   `BuildBoredBlank` separately checks the blank-only
   operation. The final joined body with a 1 mm bore passes `VerifyAll` and
   has one through-hole. Use this build for both members of the interim pair.
8. Build the pinion first. Rotate it about world Z by the shaft angle, then
   build the driving member and rotate it by half a tooth pitch about its
   own shaft. Check both bored bodies with `VerifyAll` in the shared
   document. The spiral branch uses opposite hands. Both placed members pass
   `VerifyAll` at 0.1 mm.
   When `Placement` is set, `prepareBuild` rejects an invalid frame first. After the
   last boolean, `Build` and `BuildPair` call `decad.Transform` with
   `r3.FromFrame` of the apex frame on each returned body. The apex frame has the apex at the origin plus the
   normal times the driving heel station, the build X axis on the negated normal, and the build Y axis on U. Check a placed member's volume
   and bounds against the unplaced member, and a placed pair's two bodies in one
   document.
9. For a spiral case, slice the untrimmed point tooth
   with eight planes parallel to its parent tooth plane. Make each sheet two cone
   distances long, or four if decad reports the sheet ends inside the section. Anchor and dimension
   each slicing-sheet line on its sketch plane, then solve and verify before
   extruding. Space the planes
   by one sixth of the root-edge toe-to-heel span toward the apex. Retry the
   opposite offset sign if none cuts. Sort the pieces by their solid
   centroids, remove the apex scrap, and verify the eight remaining slabs.
10. Intersect the cutter circle with circles just beyond the toe and heel.
    Choose the intersections nearest the mean root point. Divide their
    crown-plane azimuth difference by the sine of the pitch-cone angle for
    the twist magnitude. Mirror the cutter arc for the pinion.
11. Rotate each slab from its post-slice heel-face centroid. Scale the seven
    inner slabs from their post-twist root edges, leaving the outer heel
    slab full size. Loft the toe-most toe face and eight ordered heel faces.
    Both real curved tooth bodies pass `VerifyAll` at 0.1 mm.
12. Cut the apex-side tip with the toe cone, then intersect the keeper with
    the heel cone. Both members' trimmed teeth pass `VerifyAll` at 0.1 mm
    against the reviewed Decad heel branch. Place 31 copies around each
    shaft; every copy passes the same check.
13. Combine all 31 tooth pieces with each source-default blank in one
    operation, using the blank as target, then resolve and cut each bore.
    Both real joins and post-join bores pass `VerifyAll` at 0.1 mm. The
    source-default pair passes the same check after placement. The renderer
    builds that pair and produces its 1600 × 900 hero PNG.
