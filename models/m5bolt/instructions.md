# M5 hex head bolt

## Purpose and source

Construct an M5 hex head bolt envelope whose length a caller chooses. It pairs with `models/m5nut`.
The shape follows the dimension drawing of ISO 4017 / JIS B 1180: a hexagonal head with both faces chamfered,
a plain shank and a 45 degree chamfer at the tip.

All dimensions are in millimeters.

| Symbol | Meaning | Value |
| --- | --- | --- |
| `ds` | shank diameter | 5 |
| `s` | head width across flats | 8 |
| `e` | head width across corners | `s / cos 30°` = 9.2376 |
| `k` | head height | 3.5 |
| `dw` | diameter of the flat bearing circle on each head face | 6.88 |
| (30°) | angle of the head's corner chamfer from the face | 30° |
| `u` | radial and axial size of the tip chamfer | 0.5 |
| `l` | length, underside of the head to the tip | 16, 20, 25, 30, 35, 40, 45, 50, 60, 70 or 80; default 25 |

The thread is not modeled. The shank is a plain cylinder of the nominal diameter `ds`. There is no underhead
fillet. The model gives the bolt's envelope for fitting parts and holes; it is not manufacturing geometry, and it
does not describe the thread length `b`, the thread tolerance or the strength class.

## API

Package: `models/m5bolt`.
Expose `Params` with `Length float64` in millimeters and `Placement *r3.Frame`.
A nil Placement selects the local frame: the origin is the center of the head's underside, the head lies
along +Z, the shank along -Z, and the head's corners lie on the X axis (flats face Y).
A supplied Placement maps that local frame onto it.
Expose `DefaultParams() Params`, returning Length 25 and nil Placement.
Expose `Params.Validate() error`.
Expose the dimensions above as exported constants `ShankDiameter`, `AcrossFlats`, `HeadHeight`,
`BearingDiameter`, `ChamferAngle` and `TipChamfer`, plus `AcrossCorners() float64` and `StandardLengths()`.
Expose `Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)`.
`Model` exposes `Body *decad.Body`, `Head *sketch.Sketch` (the head hexagon) and `Shank *sketch.Sketch` (the
axial section of the shank).
Build adds exactly one body to doc.

## Validation

Reject nil ctx and doc at the public boundary. Reject a nonfinite Length, a Length that is not one of
`StandardLengths()` and an invalid Placement. Validate and check ctx.Err before adding geometry.
The zero Params value is invalid.

## Geometry and constraints

1. Build the head with `internal/hexnut` and no bore: a hexagonal prism of `s` across flats and height `k` along
   +Z from the underside, with a 30° chamfer from the circle of diameter `dw` on both faces.
2. Sketch the shank's axial section on the XZ plane, U as radius and V as height, from the tip center up the axis:
   tip face, 45° tip chamfer, cylinder wall, and a top that reaches 1 mm into the head so the two bodies overlap.
   Verify it is fully constrained, then revolve its one valid profile a full turn about the Z axis.
3. Join the head and the shank into one solid.
4. Transform the body to Placement if one is supplied. Require the body to be solid.

The head sketch is anchored to its origin by `internal/hexnut`. The shank sketch is anchored to the origin; only the
axis and the underside are dimensioned from it, and the rest is held by horizontal, vertical and dimension
constraints between neighboring points.

## Checks and images

Check the default build's bounds, its volume against an independent integral (head, shank, tip chamfer),
`VerifyAll` tessellation and the full constraint of both sketches.
Check every standard length, a length between them, a placed bolt, invalid inputs and the unchanged document.
