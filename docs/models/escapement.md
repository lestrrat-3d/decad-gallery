# Graham escapement package

The `models/escapement` package builds the movement that the [escapement gallery page](../../models/escapement/)
shows. Its source is [`instructions.md`](../../models/escapement/instructions.md), and the
[steps](../../models/escapement/steps.md) map that contract to Sketch and Decad operations. Lengths are in millimeters
and angles are in radians.

## Using the package

`Build(ctx, doc, Params)` adds the movement's 32 bodies to a decad document and returns a `*Movement`. The document must
come from `NewDocument()`, which sets `decad.WithChainTolerance` to 0.1 mm; the plates take many cuts and then a chamfer,
and a plain document refuses them. `Build` rejects a nil context or document and invalid parameters before it adds
anything to the document, and it returns every later failure with the part and step that failed.

`Movement` holds one field per body, every Sketch it used in `Sketches`, and the pose it was built at in `State`.
`Named()` lists the bodies with their display names, in the order of the Bodies table of `instructions.md`: the two plates,
the seven 605ZZ bearings, the three arbors, the escape wheel and anchor, the crutch, pendulum, suspension pin and cap,
the great wheel, the pawl carrier, the mainspring, the spring spacer, the spring anchor pin and the winding knob, then four
bolts and four nuts. The bodies sit in the containers Frame, Bearings,
Arbors, Escapement (Escape Wheel, Anchor), Pendulum Assembly (Crutch, Pendulum, Suspension), Drive (Great Wheel, Pawl Carrier,
Mainspring, Spring Spacer, Spring Anchor, Winding Knob), Bolts and Nuts, and `doc.VerifyContainers().Err()` is nil.

`Params` has seven fields:

| Field | Default | Accepted values |
| --- | --- | --- |
| `PendulumLength` | 180 | 120 through 250 mm, from the suspension to the bob's centre |
| `Amplitude` | 5° | `(LiftAngle + LockAngle) / 2 + 1°` through 6°, in radians |
| `LockAngle` | 1.5° | 1.25° through 1.5°, in radians |
| `LiftAngle` | 2.5° | 2° through 3°, in radians |
| `Phase` | 0.25 | 0 up to but not including 1, the fraction of one pendulum period; 0.25 hangs the pendulum at rest |
| `Backlash` | 0.15 | 0 through 0.3 mm of tooth thinning per gear |
| `Placement` | nil | an `r3.Frame` that carries the model frame, or nil for the identity |

`DefaultParams()` returns the defaults and `Validate()` applies the ranges. Zero is not a request for the default. The
error names the field. Every other dimension is a fixed constant of `instructions.md`.

`Layout(Params)` returns a `Geometry` that holds every derived value and point, the mainspring's `spiralspring.Dimensions`
(`Spring`), the spring's turn `SpringTurn`, the carrier's offset `CarrierOffset`, the anchor pin's centre `SpringAnchor` and the
torque estimates `SpringRate`, `FullWindTorque`, `FullWindStrain` and `RunTurns`; it also holds the pallet corners, the anchor outline's
points, the tooth feet, the crutch and pendulum outlines, the gear radii and the axes. `Motion(Params)` returns the
`State` at `Phase` in closed form: the anchor angle ψ, the wheel angle φ, the pendulum angle β, the great wheel's angle
and which pallet holds the wheel. `Geometry.StateAt(t)` gives the state at any phase. The great wheel arbor `G` is directly below the escape arbor `O`, at `(0, -45.75)`, so the suspension, pallet arbor, escape
arbor and great wheel arbor stand on the plumb line `x = 0`. Both plates are open frames, mirror-symmetric about that line,
with two windows each. The model frame has its origin on
the escape arbor in the plane of the back plate's inner face, `+X` to the right, `+Y` up and `+Z` toward the viewer.

`Build` poses every body at `Phase`: the wheel, anchor and pendulum turn about their axes, the great wheel follows the pinion, and the pawl carrier, spring spacer, knob and
their arbor turn with it, offset by a constant that lines the arbor's flat with the fixed mainspring's bore. The mainspring and
its anchor pin stay fixed, and the build backs the wheel off by 0.05° so the tooth and the pallet do not touch.

## Shared tooling

The package uses `internal/gearroot` for the pinion's and the great wheel's deep root, `internal/edgefinish` for every
chamfer, and `internal/m2fastener` for the four bolt sites: the head pockets, clearance holes, nut cavities and the bolts
and nuts themselves. The mainspring is a `models/spiralspring` body, built by `spiralspring.Build` with the constants of `SpringParams()` and not
copied. The gears are `models/spurgear` bodies; the bearings are `models/ballbearing` bodies of the 605ZZ
size, and the bolts and nuts are `models/m2bolt` and `models/m2nut` bodies.

## Checks

Run the checks with:

```sh
go test ./models/escapement -timeout 20m
```

The package tests, about 4.5 minutes in all, build the default movement once and cover:

- `TestValidate`, `TestLayout` and `TestMotion`: the ranges, every default number of `instructions.md` to 1e-4, and 4000
  samples of the motion.
- `TestActionAnalytic` and `TestEscapementAction`: a quasi-static simulation that drives the anchor by ψ = -A cos 2πt
  and advances the wheel as far as the two outlines allow, first on analytic outlines and then on the section outlines of
  the real wheel and anchor, for three cases of lock, lift and amplitude. They require one tooth per period, two drops
  of at least 1.30 mm that land on a locking face, no recoil, and gaps to the pallets of at least 0.5 mm, 1.0 mm and 2.0 mm.
- `TestSketches`, `TestSketchesBranches`: every Sketch is fully constrained, for the default and the two cases that
  change the anchor and pendulum outlines.
- `TestInventory`, `TestPlacement`, `TestBuildValidates`: 29 solid bodies in their containers, a placed build with the
  same volumes and mapped bounds, and the rejection of bad parameters before the document is touched.
- `TestSpringLayout`, `TestMainspring`, `TestPawlCarrier`, `TestFrontPlateSpringAnchor` and `TestSpringFits`: the spring's
  derived numbers and torque, the spring and carrier bodies, the front plate's anchor socket, and the spring's fits.
- `TestCentreLine`, `TestPlateWindows`, `TestPlateMaterial` and `TestKnob`: the four axes on `x = 0`, ψ = 0 and β = 0 at the
  default phase, plates that are symmetric about the line, windows open where the mechanism shows and material in the
  spine, hub rings and rims, and a winding knob under 400 mm².
- `TestInstallability`: each between-plate part's D bore passes a 605ZZ seat and keeps its press fit at both ends on
  the stepped arbor.
- `TestArbors`, `TestEscapeWheel`, `TestAnchor`, `TestPendulumParts`, `TestRatchet`, `TestPlates` and `TestFasteners`: each
  part's shape and each fastener's place.
- `TestGearMesh`, `TestCoupling`, `TestFits`, `TestEdgeFinishing` and `TestPoseNoOverlap`: the gear train clearances at five
  angles, the crutch pin in its slot at four phases, every row of the fit table, the chamfers' volume, and no overlap
  between any two bodies at five phases.

The renderer's `TestEscapementExploded` checks that no two parts overlap in the exploded view.

## Rendering

Render the images with the commands in the [renderer page](../rendering.md).
