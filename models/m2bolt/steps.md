# M2 bolt Decad steps

Follow instructions.md for the API and governing dimensions. Emit one construction file.

## 1. Validate

Implement Params, DefaultParams, Validate and Model as specified. Build validates ctx, doc and params,
then ctx.Err, before it creates a sketch.

## 2. Revolve the blank

Create a fresh sketch.World per call and a sketch on world.XZ().
Create these points as (U, V) = (radius, height), joined in this order by lines, closing on the first:
tip center (0, -l); tip edge (ds/2 - u, -l); tip shoulder (ds/2, -l + u); shank top (ds/2, 0);
head bottom (dk/2, 0); head side (dk/2, k - v*tan30); head top edge (dk/2 - v, k);
pilot top (0.7, k); pilot floor (0.7, k - t - PilotDepth); pilot apex (0, pilot floor V - 0.7/tan59).
The last line is the axis, from the pilot apex back to the tip center.
Constrain: the axis vertical and through the origin; tip center, shank top and pilot floor heights from
the origin; the tip face horizontal with its length; the tip shoulder radius and its height above the tip
edge; the shank vertical; the underside horizontal and head bottom radius; the head side vertical; the top
face horizontal; the chamfer's height and width between its end points; the pilot top radius and height;
the pilot wall vertical; and the angle from the axis to the drill-point line, 180 - 59 degrees.
Solve, then internal/sketchcheck.VerifyFullyConstrained. Require one valid profile.
Call decad.Revolve with decad.SketchLine from (0,0) to (0,1) and decad.FullRevolution{}.

## 3. Fillet the underhead edge

Select concave edges and keep the one whose start lies at radius ds/2 from the Z axis. The pilot floor
edge is also concave. Fillet it with decad.Fillet, selecting by decad.EndpointAt, with radius r.

## 4. Cut the hexagonal socket

Create a sketch on world.XY(), sketch.CreatePolygon(0, 0, 6, s/sqrt(3)), constrain the center coincident with the
origin, the second side horizontal, and the distance from the center to that side to s/2.
Verify it is fully constrained. Extrude its profile along +Z by t plus 0.5, lift it by k - t with
r3.Translation, and decad.Cut it from the body.

## 5. Place

If Placement is set, apply r3.FromFrame(*p.Placement) with decad.Transform. Require the body solid.
Return &Model{Body, Profile, Socket}. Wrap every error with its step.

## Known decad limit

After the socket cut decad holds the drill-point cone as a faceted face instead of a cone. The revolve
builds it exactly, and the volume stays within decad's bound of the exact section.
