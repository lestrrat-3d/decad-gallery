# One-way clutch Decad steps

Follow instructions.md for the API, the dimensions, the presets and the validation order. The package has five files:
`params.go` (constants, `Direction`, `Shaft`, `Params`, `Validate`, `Derive`, `Dimensions`), `sketches.go` (the build
run and its sketches), `onewayclutch.go` (package doc, container names, `Parts`, `Model`, `BuildParts`, `Build`, the
four part builders and the meridian tables), `adapter.go` (`Adapter`, `BuildAdapter` and the adapter's builder) and
`document.go` (`NewDocument`). Every error message starts with `onewayclutch: `. Build the clutch in a document from
`onewayclutch.NewDocument()`, which carries a chain tolerance of 0.02 mm: a plain `decad.New()` refuses the back plate's
cut after the ring's mount pockets. `BuildAdapter` alone needs no chain tolerance.

## 1. Validate and derive (params.go)

`Derive` makes no geometry calls. It runs the validation rules of instructions.md in order and fills `BoreRadius`,
`BoreFlat`, `BoreLoop` and `Sign`. `Sites[i]` is `m2fastener.Site{Head: (8.1 cos a_i, 8.1 sin a_i, 18.5), Dir: (0, 0,
-1), Corner: (cos(a_i + 30°), sin(a_i + 30°), 0), Length: 16, Cavity: 16.5}` for `a_i` in 30°, 150°, 270°. `MountSites[i]`
is `m2fastener.Site{Head: (21.6 cos e_i, 21.6 sin e_i, 26.0), Dir: (0, 0, -1), Corner: (cos(e_i + 30°), sin(e_i + 30°), 0),
Length: 25, Cavity: 25.25}` for the ear angles `e_i`. Each passes `Validate()`.

The pawl values come from the seated tooth gap. `P = 13.75∠70°`, `Q = 16.1∠90°`. `ramp = (Q - P)/|Q - P|`;
`RampAngle = atan2(ramp)` (145.940°); the unit normal `n` of the ramp pointing away from the axis gives
`RampDistance = n·P` (13.338) and `FlankDistance = RampDistance - HeadGap` (12.938). The lock line is `x = HeadGap`, so
`Apex = (0.4, (FlankDistance - n.x·0.4)/n.y)`. `TipCenter = (0.8, (FlankDistance - 0.4 - n.x·0.8)/n.y)`, `HeadReach =
|TipCenter| + 0.4`, `HeadTravel = HeadReach - 13.75`. `LockEnd = (0.4, sqrt(12.0² - 0.4²))`. `FlankEnd` is the
intersection of the flank with the circle of radius 13.2 nearer `Apex`: `foot = n·FlankDistance`, `foot ± ramp·sqrt(13.2²
- FlankDistance²)`.

## 2. Sketches (sketches.go)

Each sketch is on a fresh `sketch.NewWorld()` plane made from an `r3.Frame`, recorded in the run, solved, and checked
with `sketchcheck.VerifyFullyConstrained` before its profile is used (`run.profile`). `across(z)` is the plane z = const
with U = +X, V = +Y. `run.mirror(v)` maps `(x, y)` to `(σx, y)`; `run.angle(θ)` is `90 + σ(θ - 90)` in (-180, 180];
`run.arc(center, a, b)` is `CreateArc(center, a, b)` for `σ = 1` and `CreateArc(center, b, a)` for `σ = -1`.
`newPolar` adds the horizontal reference line (`NewHorizontal`, `NewHorizontalDistance(O, end, 20)`); `axis(θ)` adds a
construction line to the mirrored point `20∠θ` with `NewAngle(ref, axis, angle(θ))` and `NewDistance(O, end, 20)`.

- `boreLoop`: `q1 = (-w, f_b)`, `q2 = (w, f_b)`, `bore = CreateArc(O, q1, q2)`, `flat = CreateLine(q2, q1)`,
  `NewRadius(bore, r_b)`, `NewHorizontal(flat)`, `NewDistancePointLine(O, flat, f_b)`.
- `boltHoles`: per angle an axis (`NewAngle`, `NewDistance` 20), a centre with `NewPointOnLine` and `NewDistance(O,
  centre, 8.1)`, a circle `NewRadius(1.3)`.
- `plateSketch(z)` on `across(z)`: a circle about the origin with `NewRadius(16.7)`, `boreLoop`, `boltHoles`. Select
  the profile with four holes.
- `hubSketch()` on `across(4.25)`: for each pawl `k` (turn `d = 120k`), points at the mirrored, turned layout positions
  of `a` (core and web corner), `W`, `H_f`, `A`, `H_l`, `S` (arm inner arc end at `12.0∠-6°`), `T` (core end of the
  slot round, `10.4∠-6°`) and the slot centre `C` (`11.2∠-6°`). Entities: `core = arc(O, T_{k-1}, a)`, `web =
  CreateLine(a, W)`, `outer = arc(O, W, H_f)`, `flank = CreateLine(H_f, A)`, `lock = CreateLine(A, H_l)`, `inner =
  arc(O, S, H_l)`, `slot = arc(C, T, S)`. Constraints per pawl: `NewRadius` of `core` (10.4), `outer` (13.2), `inner`
  (12.0); `NewParallel(webAxis, web)` and `NewPointOnLine(W, webAxis)`; `NewAngle(ref, flank, angle(RampAngle + d))`
  and `NewDistancePointLine(O, flank, FlankDistance)`; `NewParallel(lockAxis, lock)` and `NewDistancePointLine(O, lock,
  0.40)`; `NewPointOnLine(C, slotAxis)`; `NewTangentCircles(slot, inner, true)` and `NewTangentCircles(slot,
  core_{k+1}, false)`. After all constraints, `sketch.CreateFillet(core, web, a, 1.0)` and `sketch.CreateFillet(flank,
  lock, A, 0.4)` for every pawl (a fillet removes its corner point and its constraints). Then `boreLoop` and
  `boltHoles`. Select the profile with four holes. Extrude 10 along +Z.
- `toothLoop(s, ps)` adds to a sketch: 18 tip points `P_j` and root points `Q_j` at the mirrored `13.75∠φ_j` and
  `16.1∠φ_j`, `φ_j = 90 + 20j`; `lock_j = CreateLine(Q_j, P_j)`, `ramp_j = CreateLine(P_j, Q_{j+1})`. Per `j`: axis at
  `φ_j`; `NewPointOnLine(P_j, axis)`, `NewDistance(O, P_j, 13.75)`, `NewParallel(axis, lock_j)`,
  `NewDistancePointLine(O, ramp_j, 13.338)`. Then `CreateFillet(ramp_{j-1}, lock_j, Q_j, 0.8)` at every `Q_j`. It is
  the ring outline's tooth hole.
- `ringSketch()` on `across(RingFrom)`: the outline with three ears as its outer loop, and as holes the tooth loop and
  three ear bores. See the ears design for the ear construction.
- `meridian(vertices)`: the plane `r3.NewFrame((0, 0, 0), (0, 0, 1), (1, 0, 0))`; `DimensionPoint(z, r)` for each vertex,
  lines in order, closed; `decad.Revolve` about `ConstructionAxis{Origin: (0, 0, 0), Dir: (0, 0, 1)}` with
  `decad.FullRevolution{}`.

## 3. Parts (onewayclutch.go)

- `hub()`: `hubSketch`, extrude 10 along +Z, then one `edgefinish.ChamferRuns(body, +Z, [HubFrom, HubFrom+10],
  Long(All), Long(Inner), 0.25)` call: the arm arcs and inner runs of 20 mm or more (two decad calls, edges chosen on
  the prism). The head run stays sharp. At the 8 mm presets the faces are under 400 mm², the call returns
  `edgefinish.ErrNoEdge` and the hub stays the exact prism. No Boolean.
- `ring()`: `ringSketch` extruded `RingHeight` (4 holes), one `edgefinish.Chamfer(body, +Z, [RingFrom, RingTo],
  Long(Outer), 0.5)`, then `decad.Cut` with `meridian(RingBackTool)` and `meridian(RingFrontTool)`, then
  `m2fastener.HexPocketTool` at each mount site. The outline holds the teeth, so there is no tooth cut. The meridian
  tables are written from the constants.
- `plate(z, length, outerZ, inner, tools)`: `plateSketch(z)`, extrude `length` along +Z, then one
  `edgefinish.Chamfer(body, +Z, [outerZ], Long(All), 0.5)` call, which takes every outer-face loop of 20 mm or more (the
  outer circle, and the D loop at 8 mm), then `decad.Cut` with `meridian(inner)` and each fastener tool in order.
- `backPlate()`: `plate(0, 4, 0, BackPlateInner, HexPocketTool per site)`. `adapter()` (adapter.go): `adapterSketch` on
  `across(AdapterFrom)` (the outer circle, and as holes the bore and three ear holes: four holes), extrude 6 along +Z, one
  `edgefinish.Chamfer(body, +Z, [AdapterFrom, AdapterTo], Long(All), 0.5)`, then `m2fastener.PocketTool` at each mount
  site. `BuildAdapter` places it, its bolts and its nuts, and files them in `Adapter`, `Bolts` and `Nuts`. `cover()`: `plate(14.5, 4, 18.5, CoverInner,
  PocketTool per site)`.
- `BuildParts` builds the four bodies in the order hub, ring, back plate, cover, requires each to be solid, and moves
  each by `placement.Place` last. It places the sites with `placeSite`.
- `Build` places `m2fastener.PlaceBolt` and `PlaceNut` at each site and files every body with
  `doc.Root().ChildNamed(name)` and `AddBody` in `Hub`, `Ring`, `Plates`, `Bolts`, `Nuts`.

## 4. Checks (onewayclutch_test.go)

`TestOneWayClutchDefault`: ten solids; every sketch fully constrained; each body's bounds from the stack; every body
tessellates with `VerifyAll`; no two bodies overlap (a bolt in its nut and a head on its pocket floor touch by design;
the bolt-to-cover clearance is measured in sections at z 17.5 and 15.5, at least 0.28 mm). `TestOneWayClutchContainers`:
the five names, each body in its container, `doc.VerifyContainers().Err()` nil. `TestOneWayClutchAdapter` builds the clutch and
`BuildAdapter` in one `NewDocument()`: 17 bodies in six containers, the adapter's bounds and chamfers, flush heads, nuts
inside their pockets, bolt tips, the shank clearances, the seam and the walls.
