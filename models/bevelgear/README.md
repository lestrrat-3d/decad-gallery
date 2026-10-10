# Bevel gear construction in progress

The image below renders an eight-tooth, two-section loft used to check the
Decad model and renderer. It is a visual check. The construction requirements
come from `fusion360-gear-generator/spec/bevelgear/instructions.md`. Its
"Create the Gear Bodies" section requires a revolved blank for each member,
an apex-to-tooth loft or spiral tooth construction, conical end cuts, a tooth
pattern, a join, a conditional through-bore, and meshing rotation.

![Eight-tooth loft geometry check](images/hero.png)

`Build(ctx, doc, Params)` currently constructs a two-section loft in millimeters.
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
apex stations, cone surfaces, far caps, and verified meshes. The final
tooth-to-cone cut is still an upstream integration gap.

The conical tooth end trims, patterned join, final-body bore, and curved
tooth branches still need integration. The current `Build` uses two full-gear
sections rather than the verified point tooth and blank, and rejects the
unimplemented options. The Decad/Sketch handoff is in
`../decad/.tmp/bevelgear-gallery-handoff.md` from the repository root.
