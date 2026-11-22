# Purchased compression spring Decad steps

Follow `instructions.md`. Emit one Go file `compressionspring.go` with package
`compressionspring`. Define `Spec` with `Name string`, `OutsideDiameter`,
`WireDiameter`, `FreeLength`, `TotalTurns`, `ActiveTurns`, `Rate`, `SolidHeight`
and `LoadAtSolid` as float64. Define `LeeLC024C14M() Spec`, `LeeLC026D16S() Spec` and `LeeLC022D03S() Spec` with the values in
instructions.md. Define `LesjoforsStock2353() Spec` the same way; its `ActiveTurns` is zero because the catalogue
does not publish it, and its outside diameter, solid height and load at solid are computed (see instructions.md). Define `KSSC1087() Spec` the same way; its `ActiveTurns` is zero because the supplier page does not publish it, and
its load at solid is computed (see instructions.md). Define `Params` with `Spec Spec` and `InstalledLength float64`.
`DefaultParams` selects the Lee part at 43.8 mm. Define `Model` with `Body
*decad.Body`, `Profile *sketch.Sketch`, and `Pitch` and `Force` float64.

`Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)`
checks nil context and document and validates all numeric inputs before
creating geometry. Check all spec fields except `ActiveTurns` are finite and positive. Check `ActiveTurns` is
finite, not negative and no greater than total turns; zero means not published. Check the inequalities in
instructions.md.
Return contextual errors with the prefix `compressionspring:`. Compute pitch
and force using instructions.md.

Create `sketch.NewWorld()` and an XY sketch. Create the wire center at
`((OutsideDiameter-WireDiameter)/2, 0)` and a circle of radius
`WireDiameter/2`. Constrain the center horizontal to `s.Origin()`, its
horizontal distance from the origin, and the circle's diameter. Solve and
call `internal/sketchcheck.VerifyFullyConstrained` before using the sole valid
profile. Use the pinned Sketch profile’s exported boolean field `Valid` to check it. Call `decad.Coil` with `decad.SketchLine` from `(0,0)` to `(0,1)`,
pitch in millimeters and total turns as a dimensionless scalar. Require a
solid. Return its body, sketch, pitch and nominal force.
