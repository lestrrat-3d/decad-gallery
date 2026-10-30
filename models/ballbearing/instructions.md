# Purchased ball-bearing mounting envelope

## Purpose and source

Construct the nominal mounting envelope of a purchased ball bearing for fitting shafts and housings.
The first preset is 608-2RS, with bore8, outside diameter22 and width7, all in millimeters.
Use [JTEKT's 608 2RS catalogue](https://www.jtekt.co.jp/products/bearing/en/p/detail/?pno=608+2RS)
as the source for these three dimensions. The catalogue identifies contact seals on both sides.

This model represents the occupied annular envelope. It does not model individual rings, balls,
seals, cage, corner relief, lubrication or manufacturing tolerances. It is not bearing manufacturing geometry
or a load/life calculation. Keep the nominal bore, outside diameter and width exact.
Do not invent vendor-specific internal dimensions or scale the preset to fit an unrelated shaft.

## API

Package: `models/ballbearing`.
Expose `Params` with BoreDiameter, OuterDiameter, Width float64 in millimeters and Placement *r3.Frame.
Nil Placement selects local XY at Origin, extruding along +Z. A supplied Placement maps the lower
cap center to its Origin, the local XY axes to U,V and the bearing axis to N.
Expose `DefaultParams() Params`, returning the 608-2RS dimensions, and `Params.Validate() error`.
Expose `Build(ctx context.Context, doc *decad.Document, p Params) (*Model,error)`.
`Model` exposes `Body *decad.Body` and `Sketch *sketch.Sketch`.
The returned Sketch is the actual local section consumed by the extrusion, even when the body is placed.
Build must add exactly one final bearing body to doc, preserving other existing bodies.

## Validation

Reject nil ctx/doc at the public boundary. Reject nonfinite or nonpositive dimensions.
Reject OuterDiameter<=BoreDiameter and an invalid nonnil placement frame.
Perform validation and check ctx.Err before adding any geometry to doc.
The zero Params value is invalid; callers use DefaultParams or supply all three dimensions.

## Geometry and constraints

1. Draw two circles centered at Sketch.Origin on the local XY plane.
2. Constrain each circle with its diameter; the centers share the anchored origin.
3. Solve and require FullyConstrained, DOF0, no redundancy and no conflicts before consuming a profile.
4. Select exactly one valid annular profile with one hole. Do not select the inner disc.
5. Extrude that profile by Width in +Z, producing an exact cylindrical through-bore.
6. Transform the completed body to Placement if supplied. The transform retires its local input.
7. Require Body.IsSolid before returning the body and its actual Sketch.

Return contextual errors naming the failed creation, solve, verification, selection, extrusion or placement.
Do not fix computed points. No chamfer, boolean cut, raceway or substitute mesh is required.

## Checks and images

Run one real default build and VerifyAll tessellation before expanding coverage.
Then check the default dimensions, through-bore, axial width, exact annulus volume and four analytic faces.
Check every returned sketch fully constrained. Check a translated/rotated placement by bounds and bore axis.
Check custom valid dimensions and reject invalid inputs before mutating the document.
Check repeated builds preserve each previously returned body and leave exactly one body per build.

Register ballbearing/hero in tools/render. Generate a1600x900 hero and480x270 index from the real body.
The README must explicitly identify the image as a mounting envelope, not an internal bearing model.
Add the model to models/README.md and run scripts/check_repository.py.
