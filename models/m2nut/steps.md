# M2 nut Decad steps

Follow instructions.md for the API and governing dimensions. Emit one construction file.

## 1. Validate

Implement Params, DefaultParams, Validate, AcrossCorners and Model as specified. Build validates ctx, doc and
params, then ctx.Err, before it creates a sketch.

## 2. Delegate to internal/hexnut

Build calls hexnut.Build with Size{Thread: d, AcrossFlats: s, Thickness: m, BearingDiameter: dw}, Options{Bore,
ChamferLower, ChamferUpper all true} and p.Placement. internal/hexnut holds the whole construction: the hexagon
extrusion (sketch on world.XY(), polygon of radius s/sqrt(3), centered on the origin, second side horizontal, center
to that side s/2, bore circle with a diameter constraint, one valid profile with one hole, extrusion along +Z by m),
the two 30 degree chamfer cuts, the placement and the solid check. Do not copy that geometry here.

## 3. Return

Wrap a hexnut error with the m2nut prefix. Return &Model{Body, Hexagon, LowerChamfer, UpperChamfer} from the
hexnut Result.

## Boolean choice

Intersecting the hexagon with a revolved ring also gives the shape, but decad then holds the bore and both
chamfers as faceted faces. Cutting the triangles from an extrusion that already holds the bore keeps them exact.
