# M5 bolt Decad steps

Follow instructions.md for the API and governing dimensions. Emit `m5bolt.go` and `m5bolt_test.go`. The package
mirrors `models/m2bolt`, but the head comes from `internal/hexnut` and the shank is a separate revolve.

## 1. Constants and API

Package `m5bolt`. Exported constants, each with a doc comment: `ShankDiameter = 5.0`, `AcrossFlats = 8.0`,
`HeadHeight = 3.5`, `BearingDiameter = 6.88`, `ChamferAngle = hexnut.ChamferAngle`, `TipChamfer = 0.5`.
Unexported `shankOverlap = 1.0`: how far the shank reaches into the head.
`AcrossCorners()` returns `AcrossFlats / math.Cos(math.Pi/6)`.
`StandardLengths()` returns a fresh `[]float64{16, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80}`.
`Params{Length float64; Placement *r3.Frame}`. `DefaultParams()` returns `Params{Length: 25}`.
`Params.Validate()`: NaN or Inf Length gives "m5bolt: Length must be finite"; a Length not in `StandardLengths()`
(`slices.Contains`) gives "m5bolt: Length must be one of %v mm"; a non-nil Placement with `!IsValid()` gives
"m5bolt: Placement must be a valid frame".
`Model{Body *decad.Body; Head, Shank *sketch.Sketch}`.

## 2. Validate

`Build(ctx, doc, p)` rejects nil ctx ("m5bolt: context must not be nil"), nil doc ("m5bolt: document must not be
nil") and `p.Validate()`, in that order, then `ctx.Err()` (wrapped "m5bolt: before creating head: %w"), all before
any sketch or body is created.

## 3. Head

Import `internal/hexnut`. Call `hexnut.Build(ctx, doc, hexnut.Size{Thread: ShankDiameter, AcrossFlats: AcrossFlats,
Thickness: HeadHeight, BearingDiameter: BearingDiameter}, hexnut.Options{Bore: false, ChamferLower: true,
ChamferUpper: true}, nil)`. Pass nil placement: the placement is applied last, to the joined body. Thread is only
validated for `Thread < BearingDiameter < AcrossFlats` (5 < 6.88 < 8). The result gives `Body` (the head, lower face
at z = 0, upper at z = 3.5, corners on X), `Hexagon` and the two chamfer sketches. Set `Model.Head = r.Hexagon`.
Wrap an error as "m5bolt: head: %w".

## 4. Shank

Create a fresh `sketch.NewWorld()` and a sketch on `world.XZ()`; set `Model.Shank` to it. Points are (U, V) =
(radius, height), with `r = ShankDiameter/2` (2.5), `l = p.Length`, `c = TipChamfer`:
tip center (0, -l); tip edge (r-c, -l); tip shoulder (r, -l+c); shank top (r, shankOverlap);
top center (0, shankOverlap).
Lines in order: axis = line(tip center, top center); tip = line(tip center, tip edge);
chamfer = line(tip edge, tip shoulder); wall = line(tip shoulder, shank top); top = line(shank top, top center).
Constraints (all `sketch.New...`):

- `Vertical(axis)`, `PointOnLine(origin, axis)`;
- `VerticalDistance(origin, tipCenter, -l)` and `VerticalDistance(origin, topCenter, shankOverlap)`;
- `Horizontal(tip)`, `Distance(tipCenter, tipEdge, r-c)`;
- `HorizontalDistance(origin, tipShoulder, r)`, `VerticalDistance(tipEdge, tipShoulder, c)`;
- `Vertical(wall)`, `Horizontal(top)`.

This pins every point. If `internal/sketchcheck.VerifyFullyConstrained` or `s.Verify` reports a redundant
constraint or a nonzero DOF, drop or swap the one named constraint; do not fix points by coordinates.
Call `s.Solve(ctx)`, then `sketchcheck.VerifyFullyConstrained(ctx, s)` (package
`github.com/lestrrat-3d/decad-gallery/internal/sketchcheck`). Require exactly one profile and `Valid`.
`decad.Revolve(ctx, doc, s, profile, decad.SketchLine{Start: decad.Point2{U: 0, V: 0},
End: decad.Point2{U: 0, V: 1}}, decad.FullRevolution{})`. Wrap errors with their step.

## 5. Join

`body, err := decad.Union(ctx, head, shank)`. The shank's top lies 1 mm inside the head and clears the chamfer
cone (bearing radius 3.44 against shank radius 2.5), so the bodies overlap in a volume and there is no coplanar
contact. Union may consume its inputs;
  after it, require that the build added exactly one body (compare `len(doc.Bodies())` before and after). If the head and shank bodies
remain in doc, stop and report that count with the smallest failing case instead of deleting bodies by hand.

## 6. Place

If `p.Placement != nil`: `r3.FromFrame(*p.Placement)` then `decad.Transform(ctx, body, transform)`.
Require `body.IsSolid()`. Return `&Model{Body, Head, Shank}`. Wrap every error with its step and the "m5bolt:" prefix.

## 7. Tests (`m5bolt_test.go`, package `m5bolt_test`, testify `require` only)

Mirror `models/m2bolt/m2bolt_test.go`. Reuse the fully-constrained helper (`s.Verify(t.Context())`: status
`sketch.FullyConstrained`, DOF 0, no Redundant, no Conflicts, `Check()` nil).

- `TestM5BoltDefault`: one body, solid, both sketches fully constrained. Bounds (`tol = Bound.Base()+1e-8`): X in
  `[-e/2, e/2]`, Y in `[-s/2, s/2]`, Z in `[-25, HeadHeight]`.
  Volume equals the independent integral
  `headVolume + pi*r^2*(l) - tipRing`, where `headVolume = hexArea*k - 2*chamfer` with the same midpoint integral
  as `models/m5nut` (20000 steps, `depth = (e/2 - dw/2)*tan30`, `hexArea = sqrt(3)/2*s^2`, no bore term) and
  `tipRing = 2*pi*(r - c/3)*(c*c/2)` (the removed tip triangle). Tolerance `Volume().Bound.Base()+1e-3`.
  Tessellate with `decad.VerifyAll` at `units.Millimeters(.02)`: `BoundaryVerified()` and `VolumeVerified()`.
  Face kinds: exactly 2 cones from the head chamfers plus 1 cone from the tip chamfer, 1 cylinder (shank), and
  plane faces: six flats, upper bearing face, the underside ring and the tip face. Assert the cone and cylinder
  counts (3 and 1) and that every face has a known kind; do not assert the total.
- `TestM5BoltLengths`: for every `StandardLengths()` value, a new doc builds with Z min `-l` and Z max `HeadHeight`
  within tolerance; also `Params{Length: 22}` is rejected.
- `TestM5BoltPlacement`: a frame like the m5nut test; the placed bolt's bounds follow the frame and the first
  bolt in the same doc still reports bounds.
- `TestM5BoltRejectsInvalidInputs`: table of zero Params, NaN, +Inf, 22, invalid Placement (`&r3.Frame{}`), each
  an error with the doc left at zero bodies; cancelled ctx gives `context.Canceled` and zero bodies; nil doc error
  contains "document"; nil ctx (`//nolint:staticcheck`) contains "context".
- `TestM5BoltStandardLengths`: the list is the eleven values above and mutating the returned slice does not alter
  the next call.
