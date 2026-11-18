# M5 hex nut

## Purpose and source

Construct the M5 hex nut that pairs with `models/m5bolt`. It follows ISO 4032 (style 1) and JIS B 1181.
All dimensions are in millimeters. The model has the same shape as `models/m2nut` at a larger size.

| Symbol | Meaning | Value |
| --- | --- | --- |
| `d` | bore diameter, the nominal thread diameter | 5 |
| `s` | width across flats | 8 |
| `e` | width across corners | `s / cos 30°` = 9.2376 |
| `m` | thickness | 4.7 |
| `dw` | diameter of the flat bearing circle on each face | 6.9 |
| (30°) | angle of the corner chamfer from the face | 30° |

The standards give `e` as a minimum (8.79) and `m` as a range (4.4 to 4.7). The model uses the exact hexagon corner
and the maximum thickness, so it is a clean envelope for fitting a wrench, a recess or a pocket. The thread is not
modeled. The bore is the nominal 5 mm, equal to the bolt's shank, so a pair needs no clearance added here.

## API

Package: `models/m5nut`.
Expose `Params` with `Placement *r3.Frame`. A nil Placement selects the local frame: the origin is the center
of the lower face, +Z runs to the upper face, and the corners lie on the X axis (flats face Y).
A supplied Placement maps that local frame onto it.
Expose `DefaultParams() Params`, `Params.Validate() error`, the dimensions above as exported constants
(`BoreDiameter`, `AcrossFlats`, `Thickness`, `BearingDiameter`, `ChamferAngle`), and `AcrossCorners() float64`.
Expose `Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)`.
`Model` exposes `Body *decad.Body` and the sketches `Hexagon`, `LowerChamfer` and `UpperChamfer`.
Build adds exactly one body to doc.

## Validation

Reject nil ctx and doc at the public boundary and an invalid Placement. Check ctx.Err before adding geometry.

## Geometry and constraints

`internal/hexnut` builds the whole nut. It is the construction `models/m2nut` uses, so M5 adds no geometry of its own.

1. Sketch a regular hexagon on XY centered on the origin, with a flat parallel to X and `s` across the flats,
   and a circle of diameter `d` at the origin. Verify the sketch is fully constrained.
2. Extrude the profile with one hole along +Z by `m`.
3. For each face, sketch on XZ a triangle revolved about Z. Its slanted side lies on the chamfer cone, which
   meets the face at the circle of diameter `dw` and rises 30° from the face. Verify it is fully constrained.
4. Revolve each triangle a full turn and cut it from the body.
5. Transform the body to Placement if one is supplied. Require the body to be solid.

Every face stays exact: six flats, two bearing faces, two cones and the bore.

## Checks and images

Check the default build's bounds, its volume against an independent integral of the chamfers, its face kinds
(8 planes, 2 cones, 1 cylinder, 11 faces), `VerifyAll` tessellation and the full constraint of all three sketches.
Check that the bore equals the M5 bolt's shank diameter, a placed nut and invalid inputs.
This change adds no image, README or `models/README.md` row for the nut; a later task does.
