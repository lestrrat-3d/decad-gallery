# M2 socket head cap screw

## Purpose and source

Construct an M2 socket head cap screw whose length a caller chooses.
The shape follows the dimension drawing of ISO 4762 / JIS B 1176: a cylindrical head with a chamfered
top edge and a hexagonal socket, an underhead fillet, a plain shank and a 45 degree chamfer at the tip.

All dimensions are in millimeters.

| Symbol | Meaning | Value |
| --- | --- | --- |
| `ds` | shank diameter | 2 |
| `dk` | head diameter | 3.8 |
| `k` | head height | 2 |
| `s` | socket size across flats | 1.5 |
| `t` | socket depth | 1 |
| `v` | radial width of the head's top chamfer | 0.2 |
| (30°) | angle of that chamfer from the top face | 30° |
| `r` | underhead fillet radius | 0.1 |
| `u` | radial and axial size of the tip chamfer | 0.25 |
| (118°) | included angle of the drill point under the socket | 118° |
| `l` | length, underside of the head to the tip | 6, 10, 16, 18, 20 or 25; default 10 |

The thread is not modeled. The shank is a plain cylinder of the nominal diameter `ds`. The model gives
the screw's envelope for fitting parts and holes; it is not manufacturing geometry, and it does not
describe the thread length `b`, the thread tolerance or the strength class.

## API

Package: `models/m2bolt`.
Expose `Params` with `Length float64` in millimeters and `Placement *r3.Frame`.
A nil Placement selects the local frame: the origin is the center of the head's underside, the head lies
along +Z and the shank along -Z. A supplied Placement maps that local frame onto it.
Expose `DefaultParams() Params`, returning Length 10 and nil Placement.
Expose `Params.Validate() error`.
Expose the dimensions above as exported constants, plus `StandardLengths()`,
`PilotDepth` (0.1) and `DrillPointBaseRadius` (0.7), documented in the code.
Expose `Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)`.
`Model` exposes `Body *decad.Body` and the sketches `Profile` and `Socket`.
Build adds exactly one body to doc.

## Validation

Reject nil ctx and doc at the public boundary. Reject a nonfinite Length, a Length that is not one of
`StandardLengths()` and an invalid Placement. Validate and check ctx.Err before adding geometry.
The zero Params value is invalid.

## Geometry and constraints

1. Sketch the axial section on the XZ plane, U as radius and V as height, from the tip center up the axis,
   around the shank, the head and the pilot well, and back down to the axis.
   The section holds the tip chamfer, the shank, the underside, the head side, the top chamfer, the top
   face out to radius 0.7, a cylindrical pilot well of radius 0.7 down to `t - PilotDepth` below the top,
   and the drill-point cone down to the axis.
2. Verify the section is fully constrained, then revolve its one valid profile a full turn about the Z axis.
3. Fillet the one concave circle where the shank meets the underside, with radius `r`.
4. Sketch a regular hexagon on the XY plane centered on the origin, with a flat parallel to X and
   `s` across the flats. Verify it is fully constrained.
5. Extrude the hexagon from the socket floor (`k - t`) to above the top face and cut it from the body.
   The hexagon's flats lie outside the pilot well, so the well is removed above the floor.
6. Transform the finished body to Placement if one is supplied.
7. Require the body to be solid.

The sketches are anchored to their origin. Only the axis and the underside are dimensioned from it; the
rest is held by horizontal, vertical and angle constraints and by dimensions between neighboring points.

## Checks and images

Check the default build's bounds, its volume against an independent integration of the section, its face
kinds, `VerifyAll` tessellation and the full constraint of both sketches.
Check every standard length, a length between them, a placed bolt, invalid inputs and the unchanged document.
Register `m2bolt/hero` in `tools/render`.
Add the model to `models/README.md` and run `scripts/check_repository.py`.
