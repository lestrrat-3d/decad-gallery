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

The exact apex loft, conical end trims, patterned join, bore, and curved
tooth branches still require Decad work. The current `Build` rejects those
options. The Decad/Sketch handoff is in `../decad/.tmp/bevelgear-gallery-handoff.md`
from the repository root.
