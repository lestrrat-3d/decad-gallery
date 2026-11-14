# Crank handle Decad steps

Follow instructions.md for the API, the dimensions, the presets and the validation order. The package has three files:
`params.go` (constants, `HandleProfile`, `Params`, the presets, `Validate`, `Derive`, `Dimensions`), `sketches.go` (the
build run and its sketches) and `crankhandle.go` (package doc, container names, `Parts`, `Model`, `BuildParts`, `Build`,
the arm and the pin). Every error message starts with `crankhandle: `.

## 1. Validate and derive (params.go)

`Derive` makes no geometry calls. It runs the six rules of instructions.md in order and fills `BoreRadius`, `BoreFlat`
and `BoreLoop`. `Site` is `m2fastener.Site{Head: (0, L, 0), Dir: (0, 0, 1), Corner: (1, 0, 0), Length: 10, Cavity:
4.75}`; Derive returns `Site.Validate()`'s error with the prefix.

## 2. Sketches (sketches.go)

Each sketch is on a fresh `sketch.NewWorld()` plane made from an `r3.Frame`, recorded in the run, solved, and checked
with `sketchcheck.VerifyFullyConstrained` before its profile is used.

- `across(y, z)` is the plane z = const through `(0, y)`, U = +X, V = +Y.
- `cylinder(y, z, r, length)`: a circle of radius `r` about the origin with `NewRadius`, extruded along +Z.
- `hexagon(y, z, flats, length)`: `CreatePolygon(0, 0, 6, flats/sqrt(3))`, with `NewCoincident(center, origin)`,
  `NewHorizontal(Sides[1])` and `NewDistance(center, Vertices[0], flats/sqrt(3))`, so its corners lie on X. Extrude
  along +Z.
- `armSketch()` on `across(0, 0)`: the point `top = (0, L)` with `NewVerticalPoints(O, top)` and
  `NewVerticalDistance(O, top, L)`; `r1 = (10, 0)`, `l1 = (-10, 0)`, `r2 = (10, L)`, `l2 = (-10, L)`;
  `hubArc = CreateArc(O, l1, r1)`, `pinArc = CreateArc(top, r2, l2)`, `right = CreateLine(r1, r2)`,
  `left = CreateLine(l2, l1)`; `NewRadius(hubArc, 10)`, `NewEqualRadius(hubArc, pinArc)` and `NewTangent` of each line
  with each arc. A circle of radius 1.3 about `top` with `NewRadius`. Select the profile with one hole.
- `boreTool()` on `across(0, -1)`: `q1 = (-w, f_b)`, `q2 = (w, f_b)` with `w = sqrt(r_b^2 - f_b^2)`,
  `bore = CreateArc(O, q1, q2)` (counter-clockwise, the long way through -Y), `flat = CreateLine(q2, q1)`;
  `NewRadius(bore, r_b)`, `NewHorizontal(flat)`, `NewDistancePointLine(O, flat, f_b)`. Extrude `ArmThickness`.
- `handle()`: the plane `r3.NewFrame((0, L, 7.4), (0, 0, 1), (1, 0, 0))`. Dimension every `HandleProfile` vertex from
  the origin with `sketchcheck.DimensionPoint`, join them in order, and `decad.Revolve` about
  `decad.ConstructionAxis{Origin: (0, L, 7.4), Dir: (0, 0, 1)}` with `decad.FullRevolution{}`.

## 3. Parts (crankhandle.go)

1. Arm: extrude `armSketch` by `ArmThickness`; `decad.Chamfer` of
   `decad.Edges(decad.OuterLoopOf(decad.CapStart(arm))).Or(decad.OuterLoopOf(decad.CapEnd(arm)))` by 0.5; then
   `decad.Cut` the bore tool, `hexagon(L, 3.25, 7.53, 5.75)` and `m2fastener.PocketTool(site)`, in that order.
2. Pin: `decad.Union` of `hexagon(L, 3.5, 7.03, 5.05)`, `cylinder(L, 8.25, 4.5, 47.15)` and
   `cylinder(L, 55, 5.8, 3)`; then `decad.Cut` `m2fastener.HoleTool(site)` and `m2fastener.CavityTool(site)`.
3. Handle: `handle()`.
4. Require each part solid, then `placement.Place(ctx, body, p.Placement)`. Carry the site's `Head` through
   `r3.FromFrame(*Placement).Apply`, and `Dir` and `Corner` through `ApplyDir`.

`Build` then calls `m2fastener.PlaceBolt` and `PlaceNut` at the placed site, and adds each body to
`doc.Root().ChildNamed(name)` with `AddBody` after every boolean.

## Known decad limit

The 608-2RS bore's rim in the arm's back face is a 25.3 mm run on a large face, which "Surface edge finishing" asks to
chamfer. After the bore cut Decad stores its arc as a `decad.FacetedCurve`, so `decad.Circular()` does not select it.
Selected by both endpoints and `decad.Convex()`, `decad.Chamfer` refuses it: "this evaluator chamfers a straight prism,
its one-line Sweep reduction, or a revolve only; no retained blend class reads selected edge[0]: an edge between a
retained plane and a retained cylinder". The rim stays sharp, as it was on the siren's arm.
