# Seven-segment display API

`github.com/lestrrat-3d/decad-gallery/models/sevensegment` builds the display. The construction contract is
[`instructions.md`](../../models/sevensegment/instructions.md) and the Decad steps are
[`steps.md`](../../models/sevensegment/steps.md).

## Build

```go
doc := sevensegment.NewDocument() // the face teeth need its chain tolerance
m, err := sevensegment.Build(ctx, doc, sevensegment.Params{Digit: 2})
```

`Params` has two fields: `Digit` (0 to 9, default 2) and `Placement` (a frame; nil keeps the world frame in which +Z is
up, +Y runs toward segment a and the button is toward -Y; the box stands on its back wall, button end up). `Build` adds 61 bodies in named containers: `Frame` (`Walls`,
`Plates` with the guide, cradle and top plates), `Camshaft` (`Shaft`, `Cams`), `Click Indexer` (`Hub`, `Rotor`,
`Housing`, `Button`, `Spring`), `Segment Sliders`, `Flags` (one `Flag` unit per segment, each with a `Yellow Face` and a
`Black Body`), `Bearings`, `Bolts` and `Nuts`. `BuildParts` builds the printed bodies only.

`Derive` returns `Dimensions`, which holds the ramps of each cam, with methods for the lift
(`Lift`), the rotor angle (`FlagAngle`), the groove radius and the pressure angle. `PressTurn`, `TimingMargin`, `PressPose` and `ReleasePose` give the click indexer's motion. They need no geometry, so a test or a renderer can
pose the display by angle.

## Checks

`go test ./models/sevensegment` calculates the truth table, the cam profile and the click indexer's timing margin (from the built dimensions;
it is not a measurement of a print) and checks that every press turns the shaft far enough for the cams. The geometry tests read each part's corners, edge finishing, fits,
look-alike marks and split shaft on the built bodies.

## Uses

`models/ballbearing` (605ZZ), `models/compressionspring` (Lesjofors Cat.no 2353), the M2 bolt and nut from `models/fastener` through `internal/m2fastener`.
