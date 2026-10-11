# Bevel gear Decad construction steps

These steps implement [instructions.md](instructions.md). The canonical
operation order comes from `fusion360-gear-generator/spec/bevelgear/instructions.md`,
under "Create the Gear Bodies". Lengths are millimeters. `Build(ctx, doc,
Params)` returns one straight 8/8 member with an optional bore. `BuildPair`
returns both members placed around their common apex.

1. Validate both tooth counts, the shaft angle, module, face width, and
   optional inputs before adding any body. The current full build accepts
   Module 1, 8/8 teeth, a 90° shaft angle, five flank samples, and the
   default straight controls and an optional bounded bore.
2. Resolve the shared pitch cone distance, both cone angles, and the face
   width. Draw the reference's six-sided axial blank profile and revolve
   it about the shaft. Check its two planar ends, three conical surfaces,
   and `VerifyAll` mesh.
3. Draw one fitted involute tooth on the tilted back-cone plane. Keep
   `Teeth / cos(γ)` as a real number, solve the Sketch profile, and loft
   it from the exact common apex with `Document.LoftFromPoint`.
4. Revolve cutting cones from the blank's toe-dish and heel-cone lines.
   Cut the point tooth with the toe cone first. Intersect that keeper with
   the heel cone for the verified 8/8 case. The tooth retains fitted flank
   and conical end faces, and its mesh passes `VerifyAll`.
5. Make all seven `PlacedCopy` instances before consuming the original
   tooth in a Boolean. Rotate them around the shaft at successive 45°
   increments. Join the original and each copy to the revolved blank with
   sequential Decad `Union` calls. These calls implement the source's
   patterned Combine-Join.
6. Check that the joined body is one solid and its unbored mesh has one
   closed outer shell. `Tessellate(0.1 mm, VerifyAll)` verifies its
   boundary and volume.
7. When requested, cut the bounded through-bore **after** the tooth pattern
   and Combine-Join. `BuildBoredBlank` separately checks the blank-only
   operation. The final joined body with a 1 mm bore passes `VerifyAll` and
   has one through-hole. Use this build for both members of the interim pair.
8. Build the pinion first. Rotate it about world Z by the shaft angle, then
   build the driving member and rotate it by half a tooth pitch about its
   own shaft. Check both bored bodies with `VerifyAll` in the shared
   document. The spiral branch still needs changing tooth sections and
   opposite hands; verify those real bodies before enabling it in `BuildPair`.
9. For the source-default 31/31 spiral case, slice the untrimmed point tooth
   with eight planes parallel to its parent tooth plane. Space the planes
   by one sixth of the root-edge toe-to-heel span toward the apex. Retry the
   opposite offset sign if none cuts. Sort the pieces by their solid
   centroids, remove the apex scrap, and verify the eight remaining slabs.
10. Intersect the cutter circle with circles just beyond the toe and heel.
    Choose the intersections nearest the mean root point. Divide their
    crown-plane azimuth difference by the sine of the pitch-cone angle for
    the twist magnitude. Mirror the cutter arc for the pinion. The real
    slab rotations, anchored crown, loft, and conical trim follow this stage.
