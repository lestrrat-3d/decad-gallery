# M2 hex nut

## Purpose and source

Construct the M2 hex nut that pairs with `models/m2bolt`. It follows ISO 4032 (style 1) and JIS B 1181,
which give the same M2 envelope. All dimensions are in millimeters.

| Symbol | Meaning | Value |
| --- | --- | --- |
| `d` | bore diameter, the nominal thread diameter | 2 |
| `s` | width across flats | 4 |
| `e` | width across corners | `s / cos 30°` = 4.619 |
| `m` | thickness | 1.6 |
| `dw` | diameter of the flat bearing circle on each face | 3.48 |
| (30°) | angle of the corner chamfer from the face | 30° |

The standards give `e` as a minimum (4.32) and `m` as a range (1.35 to 1.6). The model uses the exact
hexagon corner and the maximum thickness, so it is a clean envelope for fitting a wrench, a recess or a pocket.
The thread is not modeled. The bore is the nominal 2 mm, equal to the bolt's shank, so the two share a cylinder
and a pair needs no clearance added here.

## API

Package: `models/m2nut`.
Expose `Params` with `Placement *r3.Frame`. A nil Placement selects the local frame: the origin is the center
of the lower face, +Z runs to the upper face, and the corners lie on the X axis (flats face Y).
A supplied Placement maps that local frame onto it.
Expose `DefaultParams() Params`, `Params.Validate() error`, the dimensions above as exported constants,
and `AcrossCorners() float64`.
Expose `Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)`.
`Model` exposes `Body *decad.Body` and the sketches `Hexagon`, `LowerChamfer` and `UpperChamfer`.
Build adds exactly one body to doc.

## Validation

Reject nil ctx and doc at the public boundary and an invalid Placement. Check ctx.Err before adding geometry.

## Geometry and constraints

1. Sketch a regular hexagon on XY centered on the origin, with a flat parallel to X and `s` across the flats,
   and a circle of diameter `d` at the origin. Verify the sketch is fully constrained.
2. Extrude the profile with one hole along +Z by `m`.
3. For each face, sketch on XZ a triangle revolved about Z. Its slanted side lies on the chamfer cone, which
   meets the face at the circle of diameter `dw` and rises 30° from the face. The triangle reaches past the face
   and its outer side clears the hexagon's corners, so nothing touches tangentially. Verify it is fully constrained.
4. Revolve each triangle a full turn and cut it from the body.
5. Transform the body to Placement if one is supplied. Require the body to be solid.

Every face stays exact: six flats, two bearing faces, two cones and the bore.

## Checks and images

Check the default build's bounds, its volume against an independent integral of the chamfers, its face kinds,
`VerifyAll` tessellation and the full constraint of all three sketches.
Check that the bore equals the M2 bolt's shank diameter, a placed nut and invalid inputs.
Register `m2nut/hero`, `top`, `side`, `low` and `withbolt` in `tools/render`.
Add the model to `models/README.md` and run `scripts/check_repository.py`.
