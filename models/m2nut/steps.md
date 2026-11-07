# M2 nut Decad steps

Follow instructions.md for the API and governing dimensions. Emit one construction file.

## 1. Validate

Implement Params, DefaultParams, Validate, AcrossCorners and Model as specified. Build validates ctx, doc and
params, then ctx.Err, before it creates a sketch.

## 2. Extrude the hexagon

Create a fresh sketch.World per call and a sketch on world.XY(). Call sketch.CreatePolygon(0, 0, 6, s/sqrt(3))
and a circle at the origin. Constrain the polygon center coincident with the origin, the second side
horizontal, and the distance from the center to that side to s/2. Add sketch.NewDiameter(d) to the circle.
Solve, then internal/sketchcheck.VerifyFullyConstrained. Select the one valid profile with one hole.
decad.Extrude along +Z by m.

## 3. Cut the chamfers

For the lower face and then the upper face, create a sketch on world.XZ() with a triangle (U radius, V height):
start, foot, corner. The outer radius is e/2 + 0.2. The cone through (dw/2, face height) rises 30° from the
face toward the nut, so the corner is (outer, face height ± (outer - dw/2) tan30°). The start lies 0.3 mm
outside the face on the same cone, at radius dw/2 - 0.3/tan30°, and the foot is directly beside it at the outer
radius. Constrain the foot's radius and height, the start-foot side horizontal, the start's radius, the
foot-corner side vertical, and the 30° angle between the start-foot side and the cone (negated for the upper face).
Verify full constraint, decad.Revolve a full turn about the line (0,0)-(0,1), and decad.Cut it from the body.

## 4. Place

If Placement is set, apply r3.FromFrame(*p.Placement) with decad.Transform. Require the body solid.
Return &Model{Body, Hexagon, LowerChamfer, UpperChamfer}. Wrap every error with its step.

## Boolean choice

Intersecting the hexagon with a revolved ring also gives the shape, but decad then holds the bore and both
chamfers as faceted faces. Cutting the triangles from an extrusion that already holds the bore keeps them exact.
