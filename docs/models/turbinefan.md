# Turbine fan

The model in `models/turbinefan` constructs a fan of three replaceable blades on one printed hub that clamps a D-flat
shaft. The hub is a straight prism: a disc of radius 14 with a clamp tab, cut by a blind D bore, a 1 mm slit and an M2
clamp bolt across the tab. Each blade is the hand-crank fan's blade with its round stud in a radial socket in the hub,
held by one M2 bolt and a captive nut.

## Construction and API

`Params` holds the shaft diameter and flat depth, the bore clearance, the blade pitch, the `Hand` and an optional
`Placement` frame. `Params605ZZ()` and `Params608ZZ()` read the shaft diameter from `ballbearing` and the bore clearance
from `ballbearing.PressFitClearance`. `DefaultParams()` returns `Params605ZZ()`. `Params.Derive()` returns the first
failure in the order `instructions.md` lists, or `Dimensions`.

Build in a document from `NewDocument()`, which carries a chain tolerance of 0.1 mm; the blade's stud union refuses a
plain document. `BuildParts(ctx, doc, params)` adds the hub and the three blades and files them in no container.
`Build(ctx, doc, params)` adds those, the four bolts and the four nuts, in the `Hub`, `Blades`, `Bolts` and `Nuts`
containers. A rejected build adds no body.

The model reuses two items by import. `crankhandle` exports `BoreTool`, `SlitTool`, `NutPocketTool`, `TailCenters`,
`TailReach` and `SketchTail`; `handcrankfan` exports `BuildBlade`, `BuildBladeMirrored`, `BladeSocketTool` and the hub and
blade-joint constants. Both exports are moves of code the two models ran before: their bodies, sketch order and images do
not change.

The hub's outline is the disc and the tab joined by two concave roots of radius 1.0, every join tangent, with three bolt
holes. It is extruded 20 and both faces' outlines are chamfered, over the concave roots too. Then it is cut by the slit,
the bore, the three sockets, the blade head pockets and nut cavities, and the clamp's nut pocket, head pocket and hole.

## Build and images

```sh
go test ./models/turbinefan -timeout 20m
go -C tools/render run . -model turbinefan -shot readme -out ../../models/turbinefan/images
```

The readme shot writes `hero.png`, `index.png`, `front.png`, `back.png`, `side.png`, `clamp.png`, `joint.png`,
`exploded.png`, `exploded-back.png`, `sizes.png` and `parts/hub.png`, `blade.png`, `bolt.png` and `nut.png`. The presets
`turbinefan/hero`, `front`, `back`, `side`, `clamp`, `joint`, `exploded`, `exploded-back` and `sizes` render one view each.

## Verification

Tests build six cases: both presets, a 0.35 mm bore clearance, a 40 degree and a 10 degree pitch, and a placed fan. Bore
tests run on the three bore cases and blade tests on the three pitch cases. For each they check the sketches, the twelve
bodies and their containers, the bounds, the blade envelope, the bore, the slit, the edge chamfers, the clamp and blade
bolts, the Regular fits, the blades' pitch and hand, the copies, the replacement paths and `cornercheck` on the hub and
each blade. One extra test builds the left-hand fan and proves its blades are the right-hand blades reflected in x = 0;
the other checks run for the right hand only, because the hub, bolts and nuts do not depend on the hand.

## Known decad limit

The 608ZZ bore's rim in the hub's back face is not chamfered: Decad refuses the chamfer. `models/crankhandle/steps.md`
quotes the error.
