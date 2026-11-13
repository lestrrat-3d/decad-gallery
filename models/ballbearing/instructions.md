# Purchased ball-bearing mounting envelope

## Purpose and source

Construct the nominal mounting envelope of a purchased ball bearing for fitting shafts and housings.
The first preset is 608-2RS, with bore8, outside diameter22 and width7, all in millimeters.
Use [JTEKT's 608 2RS catalogue](https://www.jtekt.co.jp/products/bearing/en/p/detail/?pno=608+2RS)
as the source for these three dimensions. The catalogue identifies contact seals on both sides.

The second purchased-part preset is 605ZZ, with bore 5, outside diameter 14 and width 5 mm.
Use [EZO's 605ZZ catalogue](https://www.ezo-brg.co.jp/english/product/spec.php?eid=00529)
as its dimension source. Both presets use the same exact annulus construction.

This model represents the occupied annular envelope. It does not model individual rings, balls,
seals, cage, corner relief, lubrication or manufacturing tolerances. It is not bearing manufacturing geometry
or a load/life calculation. Keep the nominal bore, outside diameter and width exact.
Do not invent vendor-specific internal dimensions or scale the preset to fit an unrelated shaft.

## API

Package: `models/ballbearing`.
Expose `Params` with BoreDiameter, OuterDiameter, Width float64 in millimeters and Placement *r3.Frame.
Nil Placement selects local XY at Origin, extruding along +Z. A supplied Placement maps the lower
cap center to its Origin, the local XY axes to U,V and the bearing axis to N.
Expose an optional `Lift *r3.Transform`. A non-nil Lift reads Placement in Lift's local coordinates:
the section is sketched on the Placement plane and Lift is the body's only transform.
Bodies sharing one Lift then share one exact placement for interference proofs.
Expose `DefaultParams() Params`, returning the 608-2RS dimensions, and `Params.Validate() error`.
Expose `Params608ZZ() Params`, returning BoreDiameter 8, OuterDiameter 22, Width 7 and nil Placement, the
default 608-2RS envelope under the name of the shielded 608ZZ, which has the same outline. Return a fresh
Params value on each call. It needs no image of its own.

Expose `Params605ZZ() Params`, returning BoreDiameter 5, OuterDiameter 14, Width 5 and nil Placement.
Return a fresh Params value on each call so callers can place bearings independently.
Expose `Build(ctx context.Context, doc *decad.Document, p Params) (*Model,error)`.
`Model` exposes `Body *decad.Body` and `Sketch *sketch.Sketch`.
The returned Sketch is the actual local section consumed by the extrusion, even when the body is placed.
Build must add exactly one final bearing body to doc, preserving other existing bodies.

### Catalog

Expose `Spec`, one purchased bearing, with fields `Name string`, `BoreDiameter`, `OuterDiameter`, `Width` and
`InnerRaceDiameter`, all float64 in millimeters. `InnerRaceDiameter` is the outside diameter of the inner race's
face, so a shoulder or collar that touches the face must be narrower. It is a repository assumption, not a
catalogue value: 8.0 for 605ZZ and 12.0 for 608ZZ. The EZO 605ZZ catalogue lists a shoulder `da max` of 6.8 mm,
which is narrower than the assumed 8.0.
Expose `Catalog() []Spec`, a fresh copy in catalog order, `Lookup(name string) (Spec, bool)`, an exact
case-sensitive match on `Name` that returns `Spec{}, false` for an unknown name, and `Spec.Params() Params` with the
three dimensions and nil Placement and Lift. The catalog holds two entries in this order:

| Name | Bore | Outside diameter | Width | Inner race diameter |
|---|---|---|---|---|
| 605ZZ | 5 | 14 | 5 | 8 |
| 608ZZ | 8 | 22 | 7 | 12 |

`Params605ZZ()` and `Params608ZZ()` return the `Params()` of the matching catalog entry.

The fit rules of [Part clearances](../../docs/model-pipeline.md#part-clearances) are exported as constants and
`Spec` methods, so every model sizes a bearing fit the same way:

| Name | Value |
|---|---|
| `SeatClearance` | 0.05, radial clearance between an outer race and the bore that holds it fixed |
| `PressFitClearance` | 0.05, radial clearance between a shaft and a printed bore it is pressed into |
| `Spec.ShaftDiameter()` | `BoreDiameter`; a shaft through the bore takes no clearance |
| `Spec.SeatDiameter()` | `OuterDiameter + 2*SeatClearance` |
| `Spec.PressBoreDiameter()` | `BoreDiameter + 2*PressFitClearance`, the bore a `ShaftDiameter` shaft presses into |

## Validation

Reject nil ctx/doc at the public boundary. Reject nonfinite or nonpositive dimensions.
Reject OuterDiameter<=BoreDiameter, an invalid nonnil placement frame and an invalid nonnil Lift.
Perform validation and check ctx.Err before adding any geometry to doc.
The zero Params value is invalid; callers use DefaultParams or supply all three dimensions.

## Geometry and constraints

1. Draw two circles centered at Sketch.Origin on the local XY plane.
2. Constrain each circle with its diameter; the centers share the anchored origin.
3. Solve and require FullyConstrained, DOF0, no redundancy and no conflicts before consuming a profile.
4. Select exactly one valid annular profile with one hole. Do not select the inner disc.
5. Extrude that profile by Width in +Z, producing an exact cylindrical through-bore.
6. Transform the completed body to Placement if supplied, or by Lift alone when Lift is set.
   The transform retires its local input.
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
Register ballbearing/605zz and produce images/605zz.png at 1600x900 from Params605ZZ through Build.
Check the 605ZZ catalogue dimensions, annular volume, bore and real rendered output separately.
Add the model to models/README.md and run scripts/check_repository.py.
