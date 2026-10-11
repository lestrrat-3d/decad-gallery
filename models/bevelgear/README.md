# Bevel gear construction in progress

The image renders the verified straight 8/8 pair. Each member uses one
revolved blank, an apex-to-tooth loft, conical toe and heel cuts, eight
patterned teeth joined into one solid, and a 1 mm through-bore cut after
the join. The pinion and driving member share a pitch apex and follow the
source's meshing rotation. It is an interim visual check. The rules come from
`fusion360-gear-generator/spec/bevelgear/instructions.md`; its default pair
has 31 teeth per member and spiral teeth, which this image does not yet show.

![Bored straight eight-tooth bevel gear pair construction check](images/hero.png)

`Build(ctx, doc, Params)` currently constructs that straight member in
millimeters for Module 1, 8/8 teeth, a 90° shaft angle, and the tested profile
sampling. It supports no bore or a bounded positive through-bore; the 1 mm
case passes the full geometry check. `BuildPair` builds and places both
members for this verified case. Both functions reject other full-gear cases
until their construction is verified. The [instructions](instructions.md)
define their parameters and invalid cases; the [steps](steps.md) name the
Sketch and Decad operations.

Run the real-body checks separately:

```sh
go test ./models/bevelgear -run '^TestRealEightToothFullPatternJoin$'
go test ./models/bevelgear -run '^TestJoinedEightToothPairThroughBore$'
```

The tests check a closed outer shell before the bore, two placed solids with
through-holes afterward, and `VerifyAll` for each member. The
[renderer](../../tools/render/README.md) builds that pair for the hero image
and checks the PNG in its integration test.

`BuildVirtualToothProfile` also draws one back-cone tooth for the planned
apex loft. Its Sketch tests cover fractional virtual tooth counts for equal
and unequal pairs and an embedded 60-tooth flank whose root trim is exact.
`BuildBackConeToothProfile` places it on the tilted plane and tests the
half-millimeter tooth-spacing offset without changing the plane's distance
from the apex.
`BuildPointTooth` lofts that fitted profile from the exact common apex.
Its tests build 8/8, spaced and oblique 16/12, and embedded-root 60/60
teeth; all four have the origin apex and a verified Decad mesh.
`BuildBlank` revolves the reference's six-sided axial section. Its focused
test checks two planar ends, three conical surfaces, and a verified mesh.
`BuildBoredBlank` cuts the real revolved blank. Its test checks the small
eight-tooth case and a 31-tooth pair at Module 4. Both produce a verified
solid with a through-hole in the mesh.
`BuildBlankPair` revolves both members around one apex. Its unequal 16/12
test checks the two shaft directions, each heel station, and verified meshes.
`BuildToeCone` and `BuildHeelCone` revolve the exact conical cutting tools
from the blank's toe-root corner and heel-cone line. Their tests check the
apex stations, cone surfaces, far caps, and verified meshes. The ordered toe
cut and heel trim now pass `VerifyAll` for 8/8 and spaced 16/12 teeth, retain
cone and fitted flank faces, and select the larger heel piece.
The real 8/8 blank and one trimmed tooth join into a verified solid. The
first join test confirms that the tooth seats inside the revolved blank and
that the joined volume exceeds the blank's certified upper bound. All eight
trimmed teeth, copied around the shaft in 45° steps, join the blank into one
closed solid and pass `VerifyAll`. A 1 mm through-bore then cuts that joined
body, and its mesh has Euler characteristic zero.

The default 31/31 spiral tooth now passes its first real stage: eight cuts
parallel to the parent tooth plane yield eight `VerifyAll` slabs after the
apex scrap is removed. Their root-cone frame feeds the source's cutter arc
and pitch-cone twist law; the two hands produce mirrored traces. Slab twist,
crown, loft, trim, pattern, and the final spiral pair remain to be built.
The Decad/Sketch handoff is in
`../decad/.tmp/bevelgear-gallery-handoff.md` from the repository root.
