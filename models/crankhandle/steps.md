# Crank handle Decad steps

Follow instructions.md for the API, the dimensions, the presets and the validation order. The package has four files:
`params.go` (constants, `CrossZ`, `HandleProfile`, `Params`, the presets, `Validate`, `Derive`, `Dimensions`),
`document.go` (`NewDocument`), `sketches.go` (the build run and its sketches) and `crankhandle.go` (package doc,
container names, `Parts`, `Model`, `BuildParts`, `Build`, the arm and the pin). Every error message starts with
`crankhandle: `.

## 1. Validate and derive (params.go)

`Derive` makes no geometry calls. It runs the six rules of instructions.md in order and fills `BoreRadius`, `BoreFlat`
and `BoreLoop`. `Site` is `m2fastener.Site{Head: (0, L, 0), Dir: (0, 0, 1), Corner: (1, 0, 0), Length: 10, Cavity:
4.75}`. `CrossSites[i]` is `m2fastener.Site{Head: (0, L + 4, CrossZ[i]), Dir: (0, -1, 0), Corner: (1, 0, 0), Length: 6,
Cavity: 6}`. `ClampY = -(r_b + 1.3 + 1.05)`, `SideEnd = ClampY - 3.6`, `Reach = 22 - EndCenter().Y` with
`EndCenter() = (0, SideEnd + sqrt(17.5^2 - 5.5^2))` and `CornerCenter() = (5.5, SideEnd)`. `ClampSite` is
`m2fastener.Site{Head: (10, ClampY, 4), Dir: (-1, 0, 0), Corner: (0, 0, 1), Length: 16, Cavity: 16}`. Derive returns each
site's `Validate()` error with the prefix, a cross bolt's and the clamp bolt's also naming it, and rejects a clamp or cross
site whose `Tip()` differs from its `CavityFar()`.

## 2. Sketches (sketches.go)

Each sketch is on a fresh `sketch.NewWorld()` plane made from an `r3.Frame`, recorded in the run, solved, and checked
with `sketchcheck.VerifyFullyConstrained` before its profile is used.

- `across(y, z)` is the plane z = const through `(0, y)`, U = +X, V = +Y.
- `hexagon(y, z, flats, length)`: `CreatePolygon(0, 0, 6, flats/sqrt(3))`, with `NewCoincident(center, origin)`,
  `NewHorizontal(Sides[1])` and `NewDistance(center, Vertices[0], flats/sqrt(3))`, so its corners lie on X. Extrude
  along +Z.
- `armSketch()` on `across(0, 0)`: the point `top = (0, L)` with `NewVerticalPoints(O, top)` and
  `NewVerticalDistance(O, top, L)`; `r2 = (10, L)`, `l2 = (-10, L)`, `sR = (10, SideEnd)`, `sL = (-10, SideEnd)`;
  `pinArc = CreateArc(top, r2, l2)`, `right = CreateLine(sR, r2)`, `left = CreateLine(l2, sL)` with `NewRadius(pinArc,
  10)`, `NewTangent` of each line with `pinArc`, `NewVertical` of each line and `NewVerticalDistance(O, sR, SideEnd)`,
  `NewVerticalDistance(O, sL, SideEnd)`. Points at the corner centres `c_R`, `c_L` (its mirror) and the end centre
  `c_E`, and the tangent points `p_R = c_E + 22·(c_R - c_E)/|c_R - c_E|` and its mirror `p_L`;
  `arcR = CreateArc(c_R, p_R, sR)`, `arcL = CreateArc(c_L, sL, p_L)`, `arcB = CreateArc(c_E, p_L, p_R)` with
  `NewRadius(arcR, 4.5)`, `NewTangent(right, arcR)`, `NewRadius(arcL, 4.5)`, `NewTangent(left, arcL)`,
  `NewRadius(arcB, 22)` and `NewTangentCircles(arcB, arcR, true)`, `NewTangentCircles(arcB, arcL, true)`. A circle of
  radius 1.3 about `top` with `NewRadius`. Select the profile with one hole.
- `slitTool()` on `across(0, -1)`: with `h = 0.5`, `y0 = -Reach - 2`, `yc = -r_b + 1 - h`: `a = DimensionPoint(h, y0)`,
  `b = (h, yc)` and `c = (-h, yc)` each placed by `NewVerticalDistance(O, ·, yc)`, `d = DimensionPoint(-h, y0)`,
  `center = DimensionPoint(0, yc)`, `arc = CreateArc(center, b, c)` with `NewRadius(arc, h)`, and lines a–b, c–d, d–a.
  Placing b and c by x would meet the circle at a tangent, a singular solve. Extrude 10.
- `m2fastener.HexPocketTool(site)` on `site.Across(site.Cavity)` (U along `site.Corner`): for the clamp (U +Z, V +Y), six corner points at `2.51·(cos 60k°, sin 60k°)`
  and the six sides `corner k → corner k+1`, each with `NewDistancePointLine(O, side, 2.51·sqrt(3)/2)`;
  `NewHorizontal(sides[1])` and `NewAngle(sides[k], sides[k+1], 60)` for k = 0..4. The corners stay sharp. Extrude `depth` (the clamp: 2 ArmRadius - Cavity + `Stock` = 5; a cross site: 3, 2 plus `Stock`) along the bolt's direction. For a cross site the sketch's U is local +X, so the hexagon's corners point
  across the pin and its flats lie along the pin's axis. The clamp's U is +Z, so a corner points up and one down and its flats face ±Y.
- `boreTool()` on `across(0, -1)`: `q1 = (-w, f_b)`, `q2 = (w, f_b)` with `w = sqrt(r_b^2 - f_b^2)`,
  `bore = CreateArc(O, q1, q2)` (counter-clockwise, the long way through -Y), `flat = CreateLine(q2, q1)`;
  `NewRadius(bore, r_b)`, `NewHorizontal(flat)`, `NewDistancePointLine(O, flat, f_b)`. Extrude `ArmThickness`.
- `turnedPin(L)`: the plane `r3.NewFrame((0, L, 0), (0, 0, 1), (1, 0, 0))`. Dimension each vertex of the meridian
  (z, r) `(8.25, 0)`, `(8.25, 3.5)`, `(8.25 + 1.65 / tan 30°, 5.15)`, `(52.75, 5.15)`, `(55, 6.45)`, `(58, 6.45)`,
  `(58, 0)` with `sketchcheck.DimensionPoint`, join them in order, and `decad.Revolve` about
  `decad.ConstructionAxis{Origin: (0, L, 0), Dir: (0, 0, 1)}` with `decad.FullRevolution{}`.
- `flatTool(L, z, side)` on `across(L, z - 3.25)`: `CreateRectangle(-6.15, v0, 6.15, v1)` with v0..v1 = 4..7 for
  side +1 and -7..-4 for side -1, its corner A placed by `NewHorizontalDistance` and `NewVerticalDistance` from the
  origin and its sides by `NewHorizontalDistance(A, B, 12.3)` and `NewVerticalDistance(B, C, 3)`. Extrude 6.5.
- `handle()`: the plane `r3.NewFrame((0, L, 7.4), (0, 0, 1), (1, 0, 0))`. Dimension every `HandleProfile` vertex from
  the origin with `sketchcheck.DimensionPoint`, join them in order, and `decad.Revolve` about
  `decad.ConstructionAxis{Origin: (0, L, 7.4), Dir: (0, 0, 1)}` with `decad.FullRevolution{}`.

## 3. Parts (crankhandle.go)

1. Arm: extrude `armSketch` by `ArmThickness`; `decad.Chamfer` of
   `decad.Edges(decad.OuterLoopOf(decad.CapStart(arm))).Or(decad.OuterLoopOf(decad.CapEnd(arm)))` by 0.5; then
   `decad.Cut`, in this order, `slitTool()`, the bore tool, `hexagon(L, 3.25, 7.53, 5.75)`,
   `m2fastener.PocketTool(site)`, the clamp nut pocket (`m2fastener.HexPocketTool(ClampSite)` with `decad.Fillet` of the outer loop of
   its `CapStart`, the floor, by 0.6), `m2fastener.PocketTool(ClampSite)` and `m2fastener.HoleTool(ClampSite)`. The
   chamfer needs decad 902e347 or later, which admits the outline's solver-placed tangent arcs in the Booleans that
   follow. The slit goes before the bore: cut after it, the slit refuses the bore's rim bound (0.0101 mm) at the
   608-2RS within the 0.01 mm chain tolerance.
2. Pin: `decad.Union` of `hexagon(L, 3.5, 7.03, 5.8)` and `turnedPin(L)`; then `decad.Cut`, in this order, the four
   `flatTool(L, CrossZ[i], ±1)`, `m2fastener.RoundedHoleTool(site)`, `m2fastener.RoundedCavityTool(site)`, and for
   each cross site `HoleTool`, `PocketTool` and the nut pocket (`m2fastener.HexPocketTool(site)` with the same floor fillet by
   `NutPocketRound`, open to the -Y flat). The rounded tools revolve a meridian about the site's
   line with eight chords for each 0.55 mm round, every vertex placed with `DimensionPoint`.
3. Handle: `handle()`.
4. Require each part solid, then `placement.Place(ctx, body, p.Placement)`. Carry each site's `Head` through
   `r3.FromFrame(*Placement).Apply`, and `Dir`, `Corner` and `SplitNormal` (local +Y) through `ApplyDir`.

`Build` then calls `m2fastener.PlaceBolt` and `PlaceNut` at the placed pin bolt site, at each cross site and at the
clamp site, and adds each body to `doc.Root().ChildNamed(name)` with `AddBody` after every boolean: the pin bolt,
cross bolts 1 and 2, then the clamp bolt in `Bolts`, and their nuts in the same order in `Nuts`.

The pin's cuts after its union need a document with a chain tolerance of 0.005 mm or more. A plain `decad.New()`
refuses one with "cut's target holds its mesh, where this pair meets, only within a bound of 0.0012 mm, coarser than
this pair's chord tolerance 0.0011 mm". `NewDocument` sets 0.01 mm.

## Known decad limit

The D bore's rim in the arm's back face, a 25.3 mm run at the 608-2RS, is a long run on a large face, which "Surface
edge finishing" asks to chamfer. The slit opens that rim into three edges: two arcs and the flat, each arc ending at a
slit wall. Chamfered by their endpoints after the bore cut, with decad `affd593f7cdf` (it holds #3240, #3243 and #3246),
the run is refused: "not supported by the current evaluator: an edge end meets a face that is not a retained plane". Each arc alone is
refused with "a setback point lies behind an arc end". Decad's current main refuses it the same way. The rim stays
sharp, as it was on the siren's arm.

A bore rim the slit does not open does chamfer, so the order bore, rim chamfer, slit would work where the Booleans
admit it. At the 605ZZ the slit cut after the rim chamfer passes. At the 608-2RS it fails: the cut's target holds its
mesh within 0.0196 mm, coarser than the pair's chord tolerance of 0.0017 mm. Taking that route needs a geometry change
that a bore-first order admits at both bearings.
