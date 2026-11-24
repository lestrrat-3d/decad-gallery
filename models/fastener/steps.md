# Fastener Decad steps

Follow instructions.md for the API and governing dimensions. The construction files are `catalog.go`, `bolt.go`,
`bolt_socket_cap.go`, `bolt_hex_head.go` and `nut.go`.

## 1. Validate

Implement the catalog, `BoltParams`, `NutParams`, their `Validate` methods, `Bolt` and `Nut` as specified. Each build
function validates ctx, doc and params, then ctx.Err, before it creates a sketch. Look the preset up by name and
dispatch on `Bolt.Head`.

## 2. Socket-cap bolt

Create a fresh sketch.World per call and a sketch on world.XZ().
Create these points as (U, V) = (radius, height), joined in this order by lines, closing on the first:
tip center (0, -l); tip edge (ds/2 - u, -l); tip shoulder (ds/2, -l + u); shank top (ds/2, 0);
head bottom (dk/2, 0); head side (dk/2, k - v*tan(chamfer angle)); head top edge (dk/2 - v, k);
pilot top (base radius, k); pilot floor (base radius, k - t - PilotDepth);
pilot apex (0, pilot floor V - base radius / tan(half the drill point angle)).
The last line is the axis, from the pilot apex back to the tip center.
Constrain: the axis vertical and through the origin; tip center, shank top and pilot floor heights from
the origin; the tip face horizontal with its length; the tip shoulder radius and its height above the tip
edge; the shank vertical; the underside horizontal and head bottom radius; the head side vertical; the top
face horizontal; the chamfer's height and width between its end points; the pilot top radius and height;
the pilot wall vertical; and the angle from the axis to the drill-point line, 180 degrees minus half the drill point angle.
Solve, then internal/sketchcheck.VerifyFullyConstrained. Require one valid profile.
Call decad.Revolve with decad.SketchLine from (0,0) to (0,1) and decad.FullRevolution{}.

Select concave edges and keep the one whose start lies at radius ds/2 from the Z axis. The pilot floor
edge is also concave. Fillet it with decad.Fillet, selecting by decad.EndpointAt, with radius r.

Create a sketch on world.XY(), sketch.CreatePolygon(0, 0, 6, s/sqrt(3)), constrain the center coincident with the
origin, the second side horizontal, and the distance from the center to that side to s/2.
Verify it is fully constrained. Put the sketch on the plane z = k - t (the socket floor) and cut the body with
decad.CutExtrude of its profile along +Z by t plus 0.5.

## 3. Hex-head bolt

Call hexnut.Build with Size{Thread: ds, AcrossFlats: s, Thickness: k, BearingDiameter: dw}, Options{Bore: false,
ChamferLower and ChamferUpper true} and a nil placement. The placement is applied last, to the joined body. The
result's Hexagon is `Bolt.Head`.

Create a fresh sketch.World and a sketch on world.XZ(); it is `Bolt.Profile`. Points are (U, V) =
(radius, height), with r = ds/2, c = u:
tip center (0, -l); tip edge (r-c, -l); tip shoulder (r, -l+c); shank top (r, 1); top center (0, 1).
Lines in order: axis = line(tip center, top center); tip = line(tip center, tip edge);
chamfer = line(tip edge, tip shoulder); wall = line(tip shoulder, shank top); top = line(shank top, top center).
Constraints: Vertical(axis), PointOnLine(origin, axis); VerticalDistance(origin, tipCenter, -l) and
VerticalDistance(origin, topCenter, 1); Horizontal(tip), Distance(tipCenter, tipEdge, r-c);
HorizontalDistance(origin, tipShoulder, r), VerticalDistance(tipEdge, tipShoulder, c); Vertical(wall),
Horizontal(top). Solve, verify fully constrained, require one valid profile, and revolve it as in step 2.

Join with decad.Union(head, shank). The shank's top lies 1 mm inside the head and clears the chamfer cone, so the bodies
overlap in a volume. Require that the build added exactly one body (compare `len(doc.Bodies())` before and after).

## 4. Nut

Call hexnut.Build with Size{Thread: d, AcrossFlats: s, Thickness: m, BearingDiameter: dw}, Options{Bore,
ChamferLower, ChamferUpper all true} and p.Placement. internal/hexnut holds the whole construction: the hexagon
extrusion (sketch on world.XY(), polygon of radius s/sqrt(3), centered on the origin, second side horizontal, center
to that side s/2, bore circle with a diameter constraint, one valid profile with one hole, extrusion along +Z by m),
the two 30 degree chamfer cuts, the placement and the solid check. Do not copy that geometry here. Wrap a hexnut error
with the fastener prefix. Return &Nut{Body, Hexagon, LowerChamfer, UpperChamfer} from the hexnut Result.

Intersecting the hexagon with a revolved ring also gives the shape, but decad then holds the bore and both chamfers as
faceted faces. Cutting the triangles from an extrusion that already holds the bore keeps them exact.

## 5. Place

For a bolt, if Placement is set, apply r3.FromFrame(*p.Placement) with decad.Transform. Require the body solid.
Return &Bolt{Body, Profile, Socket or Head}. Wrap every error with its step.

## Known decad limit

After the socket cut decad holds the drill-point cone of the socket head cap screw as a faceted face instead of a cone.
The revolve builds it exactly, and the volume stays within decad's bound of the exact section.
