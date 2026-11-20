# Swiss lever escapement package

The `models/swisslever` package builds the movement that the [Swiss lever escapement gallery page](../../models/swisslever/)
shows. Its source is [`instructions.md`](../../models/swisslever/instructions.md), and the
[steps](../../models/swisslever/steps.md) map that contract to Sketch and Decad operations. Lengths are in millimeters
and angles are in radians.

## Using the package

`Build(ctx, doc, Params)` adds the movement's 30 bodies to a decad document and returns a `*Movement`. The document must
come from `NewDocument()`, which sets `decad.WithChainTolerance` to 0.1 mm; the plates take many cuts and then a chamfer,
and a plain document refuses them. `Build` rejects a nil context or document and invalid parameters before it adds
anything to the document, and it returns every later failure with the part and step that failed.

`Movement` holds one field per body, every Sketch it used in `Sketches`, and the pose it was built at in `State`.
`Named()` lists the bodies with their display names, in the order of the Bodies table of `instructions.md`: the two plates,
the six 605ZZ bearings, the three arbors, the escape wheel, lever and two stones, the safety roller, roller table,
balance wheel, hairspring and hairspring retainer, the winding knob and the knob collar, then four bolts and four nuts.
The bodies sit in the containers Frame, Bearings, Arbors, Escapement (Escape Wheel, Lever, Pallet Stones), Balance
(Rollers, Balance Wheel, Hairspring, Hairspring Retainer), Drive (Winding Knob), Collars, Bolts and Nuts, and
`doc.VerifyContainers().Err()` is nil.

`Params` has three fields:

| Field | Default | Accepted values |
| --- | --- | --- |
| `Amplitude` | 180° | 45° through 270°, in radians: the balance's swing each way |
| `Phase` | 0.25 | 0 up to but not including 1, the fraction of one balance period; 0.25 puts the balance at rest |
| `Placement` | nil | an `r3.Frame` that carries the model frame, or nil for the identity |

`DefaultParams()` returns the defaults and `Validate()` applies the ranges. Zero is not a request for the default. The
error names the field. Every other dimension is a fixed constant of `instructions.md`, so one parameter branch builds
every sketch.

`Layout(Params)` returns a `Geometry` that holds every derived value and point: the wheel's tooth points, the pallet
stone corners, the lever and fork points, the guard and crescent values, the balance's mass and inertia, the hairspring's
`spiralspring.Dimensions` and rate, the plate, pillar and fastener positions, and the axes `O`, `P` and `B`.
`Motion(Params)` returns the `State` at `Phase` in closed form: the balance angle `θ`, the lever angle `ψ`, the wheel
angle `φ`, the stage of each and which pallet locks the wheel. `Geometry.StateAt(t)` gives the state at any phase, and
`LeverAt` and `WheelAt` map a balance angle to a lever angle and a lever angle to a wheel angle. The three axes stand on
the line `x = 0`. The model frame has its origin on the escape arbor in the plane of the back plate's inner face, `+X` to
the right, `+Y` up and `+Z` toward the viewer.

`Build` poses every body at `Phase`. The wheel, the escape arbor, the knob and the knob collar turn about `O`; the lever,
the stones and the pallet arbor turn about `P`; the balance staff, the safety roller, the roller table, the balance wheel
and the retainer turn about `B`. The hairspring stays fixed, drawn relaxed, because a rigid body cannot wind. The build
backs the wheel and the lever off their last contact by 0.05° so no two bodies touch.

## Known limits

Two operations are not yet accepted by the decad release this repository uses. `Build` calls both and returns their error:

- The balance wheel's hub boss root fillet fails with "the boss's root circle is not an edge of the result" where the
  root circle crosses the four window arcs.
- The lever's dart root fillet fails with "the selected edge ends at the tangent junction".

The tests switch both calls off while decad refuses them (`movementSwitches` in `shared_internal_test.go`), so the
assembly checks read the rest of the movement, and `TestLever` skips on that error text only. See `steps.md` for the cases.

## Shared tooling

The package uses `internal/edgefinish` for every chamfer, `internal/m2fastener` for the four bolt sites, and
`internal/sketchcheck`, `internal/cornercheck`, `internal/splitcheck` and `internal/meshgap` in its tests. The hairspring
is a `models/spiralspring` body built by `spiralspring.Build` with the constants of `SpringParams()`, and not copied.
The bearings are `models/ballbearing` bodies of the 605ZZ size, and the bolts and nuts are the M2 preset of
`models/fastener` bodies.

## Checks

Run the checks with:

```sh
go test ./models/swisslever -timeout 30m
```

The package tests take about 13 minutes with `TestForkAction` skipped, and about 390 seconds more with it. Run one group
while you work, for example:

```sh
go test ./models/swisslever -run 'TestLayout|TestMotion|TestBalance' -timeout 10m
```

The tests cover:

- `TestValidate`, `TestLayout`, `TestMotion`, `TestWheelAt` and `TestGuard`: the ranges, every default number of
  `instructions.md` to 1e-4, 4000 samples of the motion, and the guard freedom of 2.0°.
- `TestActionAnalytic`, `TestErrorStudy` and `TestEscapementAction`: a quasi-static simulation that drives the lever and
  advances the wheel as far as the outlines allow, on analytic outlines, for the 13 print-error cases, and on the section
  outlines of the real wheel and stones. They require one tooth per period and two drops that land on a locking face.
- `TestForkAction` and `TestForkHelpers`: the impulse pin and the fork, the dart and the crescent on sections of the real
  lever, roller table and safety roller.
- `TestSketches` and the per-part sketch tests: every Sketch is fully constrained.
- `TestInventory`, `TestPlacement` and `TestBuildValidates`: 30 solid bodies in their containers, a placed build with the
  same volumes, and the rejection of bad parameters before the document is touched.
- `TestStonesKeyed`: each stone fits its own slot and no other pose.
- `TestCorners` and the per-part corner tests: `internal/cornercheck` on every printed body.
- `TestSplitShafts` and `TestSplitShaftsBuilt`: the three arbors split cleanly at their seams.
- `TestArborCollars`, `TestOutsideCollar`, `TestPressFitWalls` and `TestInstallability`: the collars, the 1.6 mm walls
  round press-fit bores, and each between-plate part's passage over a bearing seat.
- `TestFits`, `TestFitsByMesh` and `TestPoseNoOverlap`: every row of the fit table, and no overlap between any two bodies
  at eight phases.
- `TestFasteners`, `TestBearings`, `TestPlates`, `TestCentreLine` and `TestEdgeFinishing`: the bolts and nuts in their
  pockets and cavities, the plates, the line of centres and the chamfers.
- `TestHairspring`, `TestBalanceInertia` and `TestKnob`: the spring's derived numbers, the balance's mass and inertia
  within 2 % of `Layout`, and the knob.

## Rendering

Render the images with the commands in the [renderer page](../rendering.md).
