# Pin ring

## Purpose

Construct a plain flat ring that sits on a pin to hold the pin in place. All dimensions are in millimeters.

| Input | Meaning | Default |
| --- | --- | --- |
| `RodRadius` | radius of the rod the ring sits on | 2.5 |
| `Clearance` | radial gap between the rod and the bore | 0.5 |
| `Width` | radial width of the ring, from the bore outward | 5 |
| `Thickness` | height of the ring along its axis | 5 |

Derived values: the inner radius is `RodRadius + Clearance` (3), and the outer radius is the inner radius plus
`Width` (8).

The ring is a separate part. The bore takes the clearance, so the ring slides on the rod. The ring has no chamfer. Each flat face is 172.8 mm², below the 400 mm² that makes a planar surface large,
so the edge finishing rule does not apply.

## API

Package: `models/pinring`.
Expose `Params` with `RodRadius`, `Clearance`, `Width`, `Thickness` and `Placement *r3.Frame`. A nil Placement
selects the local frame: the axis is Z and the ring spans z from 0 to `Thickness`.
Expose `DefaultParams() Params`, `Params.Validate() error`, `Params.InnerRadius()`, `Params.OuterRadius()` and
the defaults as exported constants.
Expose `Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)`.
`Model` exposes `Body *decad.Body` and the sketch `Profile`. Build adds exactly one body to doc.

## Validation

Reject nil ctx and doc. Reject a non-positive `RodRadius`, `Width` or `Thickness`, a negative `Clearance`,
and an invalid Placement. Check ctx.Err before adding geometry.

## Geometry and constraints

1. Sketch two circles on XY centered on the origin, with diameters `2 * outer` and `2 * inner`. Verify the sketch
   is fully constrained.
2. Extrude the annulus profile along +Z by `Thickness`.
3. Transform the body to Placement if one is supplied. Require the body to be solid.

## Checks and images

Check, for the default and for each parameter branch, the bounds, the volume `pi (outer² - inner²) thickness`,
the four faces (two planes, two cylinders), `VerifyAll` tessellation and the full constraint of the sketch.
Check placement and invalid inputs.
Register `pinring/hero`, `top`, `side` and `low` in `tools/render`.
Add the model to `models/README.md` and run `scripts/check_repository.py`.
