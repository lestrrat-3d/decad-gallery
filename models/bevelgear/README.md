# Bevel gear

![Eight-tooth bevel gear rendered from its Decad body](images/hero.png)

`Build(ctx, doc, Params)` constructs one member of a bevel pair in millimeters.
The first verified case has eight straight involute teeth, an eight-tooth
mate, a 90° shaft angle, and no bore. Its two Sketch sections use the
[spur gear profile builder](../spurgear/README.md), and Decad lofts them
into a tapered solid. The [instructions](instructions.md) define the
parameters and invalid cases; the [steps](steps.md) map the first construction
to Sketch and Decad calls.

Run `go test ./models/bevelgear -run TestEightToothStraightBevelGear` to
check the real body. The test checks all eight tip and root sectors, the toe
and heel cap areas, the axial bounds, and `Tessellate(VerifyAll)`. The
[renderer](../../tools/render/README.md) builds the same body for the hero
image and checks the PNG with its own integration test.

`BuildVirtualToothProfile` also draws one back-cone tooth for the planned
apex loft. Its Sketch tests cover fractional virtual tooth counts for equal
and unequal pairs and an embedded 60-tooth flank whose root trim is exact.
`BuildBlank` revolves the reference's six-sided axial section. Its focused
test checks two planar ends, three conical surfaces, and a verified mesh.
`BuildBoredBlank` cuts the real revolved blank. Its test checks the small
eight-tooth case and a 31-tooth pair at Module 4. Both produce a verified
solid with a through-hole in the mesh.

The exact apex loft, conical tooth end trims, patterned join, final-body
bore, and curved tooth branches still need integration. The current `Build`
rejects those options. The Decad/Sketch handoff is in
`../decad/.tmp/bevelgear-gallery-handoff.md` from the repository root.
