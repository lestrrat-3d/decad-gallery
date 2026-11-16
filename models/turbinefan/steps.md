# Turbine fan Decad steps

Follow instructions.md for the API, the dimensions, the presets and the validation order. The package has five files:
`params.go` (constants, `Hand`, `Params`, the presets, `Validate`, `Derive`, `Dimensions`), `document.go`
(`NewDocument`), `sketches.go` (the build run, the hub's outline sketch and its helpers) and `turbinefan.go` (package
doc, container names, `Parts`, `Model`, `BuildParts`, `Build`, the hub and the blades). Every error message starts with
`turbinefan: `.

## 1. Validate and derive (params.go)

`Derive` makes no geometry calls. It runs the rules of instructions.md in order and fills the bore values and the
layout: with `W` 9, `R` 14 and `rho` 1, `RootCenter = (W + rho, -sqrt((R + rho)^2 - (W + rho)^2))`, `RootDisc = R *
RootCenter / (R + rho)`, `RootSide = RootCenter.Y`, `ClampY = RootSide - 3.6`, `SideEnd = ClampY - 3.6`,
`TailCorner, TailEnd = crankhandle.TailCenters(W, SideEnd)` and `Reach = crankhandle.TailReach(W, SideEnd)`.
`ClampSite` is `m2fastener.Site{Head: (9, ClampY, 10), Dir: (-1, 0, 0), Corner: (0, 0, 1), Length: 16, Cavity: 16}`.
`BladeSites[k]` is `Head: 9.5 u_k + (0, 0, 20), Dir: (0, 0, -1), Corner: u_k, Length: 16, Cavity: 15.6`.

## 2. Sketch (sketches.go)

Each sketch is on a fresh `sketch.NewWorld()` plane made from an `r3.Frame`, recorded in the run, solved, and checked with
`sketchcheck.VerifyFullyConstrained` before its profile is used.

`hubSketch()` on the plane z = 0 (U = +X, V = +Y). Make every point with its exact initial coordinate.

- `disc = CreateArc(O, P_R, P_L)`; right root `CreateArc(F_R, P_R, T_R)` with `T_R = (9, RootSide)`; left root
  `CreateArc(F_L, T_L, P_L)`; sides `right = CreateLine(sR, T_R)` and `left = CreateLine(T_L, sL)` with
  `sR = (9, SideEnd)`, `sL = (-9, SideEnd)`.
- `NewRadius(disc, 14)`; `NewVertical`, `NewHorizontalDistance(O, s, +-9)` and `NewVerticalDistance(O, s, SideEnd)` for
  each side; for each root `NewRadius(root, 1)`, `NewTangent(side, root)` and `NewTangentCircles(disc, root, false)`.
- `crankhandle.SketchTail(s, right, left, sR, sL, 9, SideEnd)` adds the tail.
- For each blade a circle of radius 1.3 whose centre is placed by `NewHorizontalDistance` and `NewVerticalDistance` from
  the origin, with `NewRadius`. Select the profile with three holes.

## 3. Parts (turbinefan.go)

1. Hub: extrude the outline 20 along +Z; `decad.Chamfer` of
   `decad.Edges(decad.OuterLoopOf(decad.CapStart(hub))).Or(decad.OuterLoopOf(decad.CapEnd(hub)))` by 0.5; then
   `decad.Cut`, in this order, `crankhandle.SlitTool` (plane z = -1, `from = -Reach - 2`, length 22), `crankhandle.BoreTool`
   (plane z = -1, length `1 + 20 - 1`), the socket tool and its two pattern copies, a head pocket and a nut cavity for
   each blade site (`m2fastener.PocketTool`, `CavityTool`), and the clamp's `crankhandle.NutPocketTool`,
   `m2fastener.PocketTool` and `m2fastener.HoleTool`. The socket tool is `handcrankfan.BladeSocketTool` on
   `r3.NewFrame((0, 5.7, 10), (0, 0, 1), (1, 0, 0))`; its copies come from `decad.PatternCopies` with
   `decad.CircularPattern{Axis: (0, 0, 1), Count: 3}`. Record every tool's sketch.
2. Blades: `handcrankfan.BuildBlade(ctx, doc, BladePitch)` (`BuildBladeMirrored` when `Hand` is `LeftHand`); record its
   sketches; `decad.Transform` by `r3.FromFrame(r3.NewFrame((0, 0, 10), (-1, 0, 0), (0, 0, 1)))`; `decad.PatternCopies`
   with the same circular pattern.
3. Require each part solid, then `placement.Place(ctx, body, p.Placement)`. Carry each site's `Head` through
   `r3.FromFrame(*Placement).Apply`, and `Dir` and `Corner` through `ApplyDir`.

`Build` then calls `m2fastener.PlaceBolt` and `PlaceNut` at each placed blade site and at the clamp site, and adds each
body to `doc.Root().ChildNamed(name)` with `AddBody`: the hub in `Hub`, the blades in `Blades`, blade bolts 1 to 3 then
the clamp bolt in `Bolts`, and their nuts in the same order in `Nuts`.

The blade's stud union needs a document with a chain tolerance of 0.1 mm; `NewDocument` sets it.
