# M5 nut Decad steps

Follow instructions.md for the API and governing dimensions. Emit one construction file, `m5nut.go`, and one
test file, `m5nut_test.go`. The package mirrors `models/m2nut`.

## 1. Constants and API

Package `m5nut`. Exported constants, each with a doc comment:
`BoreDiameter = 5.0`, `AcrossFlats = 8.0`, `Thickness = 4.7`, `BearingDiameter = 6.9`,
`ChamferAngle = hexnut.ChamferAngle`.
`AcrossCorners() float64` returns `AcrossFlats / math.Cos(math.Pi/6)`.
`Params{Placement *r3.Frame}`. `DefaultParams()` returns `Params{}`.
`Params.Validate()` returns an error "m5nut: Placement must be a valid frame" when Placement is non-nil and
`!Placement.IsValid()`, else nil.
`Model{Body *decad.Body; Hexagon, LowerChamfer, UpperChamfer *sketch.Sketch}`.

## 2. Validate

`Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)` rejects nil ctx ("m5nut: context must
not be nil"), nil doc ("m5nut: document must not be nil") and `p.Validate()` errors, in that order, before any
other work. The ctx.Err check runs inside `hexnut.Build` before it creates a sketch, and its error wraps
`context.Canceled`.

## 3. Delegate to internal/hexnut

Import `github.com/lestrrat-3d/decad-gallery/internal/hexnut`. Call
`hexnut.Build(ctx, doc, hexnut.Size{Thread: BoreDiameter, AcrossFlats: AcrossFlats, Thickness: Thickness,
BearingDiameter: BearingDiameter}, hexnut.Options{Bore: true, ChamferLower: true, ChamferUpper: true}, p.Placement)`.
It returns `(*hexnut.Result, error)` with `Body`, `Hexagon`, `LowerChamfer`, `UpperChamfer`.
`hexnut.Size.Validate` needs `Thread < BearingDiameter < AcrossFlats`: 5 < 6.9 < 8 holds.
`hexnut` already does: the hexagon extrusion with a diameter-constrained bore on XY, the two 30 degree cone cuts,
the placement transform and the solid check. Do not copy any of it.

## 4. Return

Wrap a hexnut error as `fmt.Errorf("m5nut: %w", err)`. Return
`&Model{Body: r.Body, Hexagon: r.Hexagon, LowerChamfer: r.LowerChamfer, UpperChamfer: r.UpperChamfer}`.
Build adds exactly one body to doc.

## 5. Tests (`m5nut_test.go`, package `m5nut_test`, testify `require` only)

Mirror `models/m2nut/m2nut_test.go`, with `m5nut` constants:

- Helper `requireFullyConstrained(t, s)`: `report := s.Verify(t.Context())`; require
  `report.Status == sketch.FullyConstrained`, `report.DOF == 0`, no `Redundant`, no `Conflicts`, `report.Check()`
  returns nil.
- `TestM5NutDefault`: one body in doc, solid; all three sketches fully constrained. Bounds with
  `tol := box.Bound.Base() + 1e-8`: X in `[-e/2, e/2]`, Y in `[-s/2, s/2]`, Z in `[0, Thickness]`.
  Volume (`m.Body.Volume()`, compare `.Value.Base()` within `.Bound.Base()+1e-3`) equals an independent integral:
  `hexArea*Thickness - pi*d^2/4*Thickness - 2*chamfer`, where `hexArea = sqrt(3)/2*s^2`,
  `slope = tan(30 deg)`, `bearingR = dw/2`, `depth = (e/2 - bearingR)*slope`, and `chamfer` is the midpoint sum over
  20000 steps of `(hexArea - clippedHexArea(bearingR + z/slope)) * depth/steps` with `z = (i+.5)/steps*depth`.
  `clippedHexArea(rho)`: apothem `s/2`; `rho <= apothem` gives `pi*rho^2`; `rho >= e/2` gives `hexArea`; otherwise
  `half = acos(apothem/rho)` and `pi*rho^2 - 6*rho^2*(half - sin(half)*cos(half))`.
  Face kinds: 8 `decad.KindPlane`, 2 `decad.KindCone`, 1 `decad.KindCylinder`, 11 faces in all.
  `Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))` has
  `BoundaryVerified()` and `VolumeVerified()` true.
- `TestM5NutFitsM5Bolt`: `require.Equal(t, m5bolt.ShankDiameter, m5nut.BoreDiameter)`. The m5bolt package is built
  by a later task. Until it exists, emit the test with this one check removed, and add it when `models/m5bolt`
  lands.
- `TestM5NutPlacement`: build a default nut, then one placed on the frame from
  `r3.NewFrame(r3.NewVec(10, 5, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))` (origin, then two direction vectors,
  as in the m2nut test, so +Z maps to world +X). The placed nut spans X in `[10, 10+Thickness]` and Y in
  `[5-e/2, 5+e/2]`. Doc holds two bodies; the first body still reports `Bounds()` without error; the placed
  `Hexagon` sketch stays fully constrained.
- `TestM5NutRejectsInvalidInputs`: `Params{Placement: &r3.Frame{}}` errors and leaves `len(doc.Bodies()) == 0`;
  a cancelled context returns `errors.Is(err, context.Canceled)` with no body added; nil doc error contains
  "document"; nil ctx (with `//nolint:staticcheck`) error contains "context".
