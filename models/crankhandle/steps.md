# Crank handle Decad steps

Follow instructions.md for the API, the symbols, the presets and the validation order. Emit three files:
`params.go` (constants, `Params`, the presets, `Validate`, `Derive`, `Dimensions`), `sketches.go` (the plate sketch and
the set screw tool) and `crankhandle.go` (package doc, container names, `Model`, `Build`, the 3D features, placement
and containers). Every error message starts with `crankhandle: `.

## 1. Validate and derive (params.go)

`Derive` makes no geometry calls. It runs the ten rules of instructions.md in order and fills every `Dimensions`
field from the symbol table. `Site` is `m2fastener.Site{Head: (0, -t, z_s), Dir: (0, 1, 0), Corner: (1, 0, 0),
Length: 6, Cavity: 4}`, and Derive returns `Site.Validate()`'s error wrapped with the prefix. `Validate` returns
Derive's error. The presets read `ShaftDiameter` from `ballbearing.Params605ZZ().BoreDiameter` and
`ballbearing.Params6082RS().BoreDiameter`, and `BoreClearance` from `ballbearing.PressFitClearance`.

## 2. Plate sketch (sketches.go)

`plateSketch(ctx, p, dm) (*sketch.Sketch, *sketch.Profile, error)`. A fresh `sketch.NewWorld()`, a sketch on
`world.XY()`, `O = s.Origin()`. Let `n_y = (R_h - R_e) / Throw` and `n_x = sqrt(1 - n_y^2)`, the outward normal of the
right tangent line. Create the points at these coordinates; the solver starts there and picks each tangent's side:

- `E = (0, Throw)`, `H_R = R_h (n_x, n_y)`, `E_R = E + R_e (n_x, n_y)`, `E_L = E + R_e (-n_x, n_y)`,
  `H_L = R_h (-n_x, n_y)`, `T_L = (-w_t, -t)`, `T_R = (w_t, -t)`.

Entities, each arc counter-clockwise from its start to its end:

- `hubRight = CreateArc(O, T_R, H_R)`, `right = CreateLine(H_R, E_R)`, `end = CreateArc(E, E_R, E_L)`,
  `left = CreateLine(E_L, H_L)`, `hubLeft = CreateArc(O, H_L, T_L)`, `tail = CreateLine(T_L, T_R)`.

Constraints: `NewVerticalPoints(O, E)`, `NewVerticalDistance(O, E, Throw)`, `NewRadius(hubRight, R_h)`,
`NewEqualRadius(hubRight, hubLeft)`, `NewRadius(end, R_e)`, `NewTangent` of `right` with `hubRight` and `end`, of
`left` with `end` and `hubLeft`, `NewHorizontal(tail)` and `NewDistancePointLine(O, tail, t)`.

The bore: `Q1 = (w_b, -f_b)`, `Q2 = (-w_b, -f_b)`, `bore = CreateArc(O, Q1, Q2)`, `flat = CreateLine(Q2, Q1)`, with
`NewRadius(bore, r_b)`, `NewHorizontal(flat)` and `NewDistancePointLine(O, flat, f_b)`.

`s.Solve(ctx)`, then `sketchcheck.VerifyFullyConstrained(ctx, s)`. Select the one valid profile with exactly one
hole; the other profile is the bore's disc. More than one or none is an error.

## 3. Set screw tool (sketches.go)

`screwTool(ctx, doc, site) (*decad.Body, *sketch.Sketch, error)`. The sketch plane is
`r3.NewFrame(site.Head, site.Dir, (0, 0, 1))`, so sketch U runs along the screw from the head (`a`) and V is the
distance from the axis (`r`). Dimension every vertex from the origin with `sketchcheck.DimensionPoint` and join them
in order with lines, the last back to the first. With `k = RimChamfer`, `rp = m2fastener.HeadPocketRadius` (2.2),
`dp = m2fastener.HeadPocketDepth` (2), `rh = m2fastener.HoleRadius` (1.3), `rn = m2fastener.NutCavityRadius` (2.51),
`near = site.Cavity` (4), `far = site.CavityFar()` (6), `end = site.Tip() + m2fastener.Stock` (9):

`(-1, 0), (-1, rp), (dp - k, rp), (dp, rp - k), (dp, rh), (near, rh), (near, rn - k), (near + k, rn), (far - k, rn),
(far, rn - k), (far, rh), (end, rh), (end, 0)`.

Solve, verify full constraint, require one valid profile, and
`decad.Revolve(ctx, doc, s, profile, decad.ConstructionAxis{Origin: site.Head, Dir: site.Dir}, decad.FullRevolution{})`.

## 4. Crank body (crankhandle.go)

In the local frame, in this order:

1. `decad.Extrude(ctx, doc, plate, profile, decad.Distance{D: T, Dir: decad.Along})`.
2. `decad.Cut(ctx, body, screwTool)`.
3. `edgefinish.Chamfer(ctx, body, (0, 0, 1), []float64{0, T}, keep, edgefinish.MinChamfer)`, where `keep` is true for
   `edgefinish.Outer` and, when `L_b >= edgefinish.LongEdgeRun`, for every loop. The only inner loop on either face
   is the bore. Use `L_b` from `Dimensions`, not `edgefinish.Long`: after the cut Decad refuses the bore arc's length,
   and `edgefinish.Long` then measures the arc the shorter way round, which is wrong for an arc of more than half a
   turn.
4. `decad.Boss(ctx, body, decad.BossSpec{Face: decad.Faces(decad.Facing((0, 0, 1))).Exactly(1), At: (0, Throw, T),
   Diameter: GripDiameter, Height: GripLength, RootFillet: rho})`.
5. Find the circular edge whose centre lies at `z = T + GripLength` with radius `GripDiameter/2`, name it by
   `decad.EndpointAt(start)` and `decad.Circular()`, and `decad.Fillet` it `Exactly(1)` with radius `r_t`.
6. Require `body.IsSolid()`, then `placement.Place(ctx, body, p.Placement)`.

## 5. Fasteners and containers (crankhandle.go)

Carry `Site.Head` through `r3.FromFrame(*Placement).Apply` and `Dir` and `Corner` through `ApplyDir`, then
`m2fastener.PlaceBolt` and `m2fastener.PlaceNut`. Put each body in `doc.Root().ChildNamed(name)` with `AddBody`, after
every boolean.

## Known decad limit

Decad cannot chamfer the faces of a plate after the set screw cut for some throws. At the 605ZZ preset with `Throw`
30 or 65 to 90 the chamfer fails: "cut's tool holds its mesh, where this pair meets, only within a bound of 0.00258 mm,
coarser than this pair's chord tolerance 0.00211 mm". `Throw` 40, 50, 55, 60, 100 and 120 build. Chamfering first does
not help: Decad then refuses the cut, because "loop 0 of this cap-loop chamfer has an arc whose recorded end is not on
the circle its start states", and no boolean may compose that chamfer. The construction keeps the cut and the native
chamfer, and the representative cases use throws that build.
