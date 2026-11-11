# Pin ring Decad steps

Follow instructions.md for the API and governing dimensions. Emit one construction file.

## 1. Validate

Implement Params, DefaultParams, InnerRadius, OuterRadius, Validate and Model as specified. Build validates ctx,
doc and params, then ctx.Err, before it creates a sketch.

## 2. Extrude the annulus

Create a fresh sketch.World per call and a sketch on world.XY(). Create a circle of the outer radius and a
circle of the inner radius, both centered on s.Origin(). Add sketch.NewDiameter to each circle. Solve, then
internal/sketchcheck.VerifyFullyConstrained. Select the one valid profile with one hole.
decad.Extrude along +Z by Thickness.

## 3. Place

If Placement is set, apply r3.FromFrame(*p.Placement) with decad.Transform. Require the body solid.
Return &Model{Body, Profile}. Wrap every error with its step.
