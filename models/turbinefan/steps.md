# Turbine fan Decad steps

Follow instructions.md for the API, the dimensions, the presets and the validation order. The package has five files:
`params.go` (constants, `Hand`, `Params`, the presets, `Validate`, `Derive`, `Dimensions`), `document.go`
(`NewDocument`), `sketches.go` (the build run, the hub's outline sketch, the nut slot sketch and the tools) and
`turbinefan.go` (package doc, container names, `Parts`, `Model`, `BuildParts`, `Build`, the hub and the blades). Every error
message starts with `turbinefan: `.

No nut is embedded: the set screw's nut and the blade nuts sit in hex slots open to a face (AGENTS.md "Fasteners");
instructions.md "Fasteners" says why each slot works and why a back-face slot does not for the blade nuts.

## 1. Validate and derive (params.go)

`Derive` makes no geometry calls. It runs the rules of instructions.md in order and fills the bore values and the set screw:
`BoreRadius`, `BoreFlat`, `BoreFlatWidth = 2 sqrt(r_b^2 - f_b^2)`, `BoreLoop`, `ShaftFlatWidth = 2 sqrt(d (D - d))`,
`SetScrewTip = D/2 - d - c`, `SetScrewHead = SetScrewTip + 2 + 6`, `CounterboreFloor = SetScrewHead - 2 - 0.4`.
`SetScrewSite` is `m2fastener.Site{Head: (0, -SetScrewHead, 4), Dir: (0, 1, 0), Corner: (0, 0, 1), Length: 6, Cavity:
SetScrewHead - f_b - 1.6 - 2}`. `BladeSites[k]` is `Head: 9.5 u_k + (0, 0, 20), Dir: (0, 0, -1), Corner: u_k, Length: 16,
Cavity: 15.6`.

## 2. Sketches (sketches.go)

Each sketch is on a fresh `sketch.NewWorld()` plane made from an `r3.Frame`, recorded in the run, solved, and checked with
`sketchcheck.VerifyFullyConstrained` before its profile is used.

`hubSketch()` on the plane z = 0 (U = +X, V = +Y). Make every point with its exact initial coordinate.

- `disc = CreateCircle(O, 14)` with `NewRadius`.
- The D bore: `qR = (w_b/2, -f_b)`, `qL = (-w_b/2, -f_b)`; `bore = CreateArc(O, qR, qL)` (counter-clockwise, the long way
  round over +Y); `flat = CreateLine(qL, qR)`; `NewRadius(bore, r_b)`, `NewHorizontal(flat)`,
  `NewDistancePointLine(O, flat, f_b)`.
- For each blade a circle of radius 1.3 whose centre is placed by `NewHorizontalDistance` and `NewVerticalDistance` from the
  origin, with `NewRadius`. Select the profile with four holes.

`nutSlotSketch(f, reach)` on the plane `f`, with `R = 2.51` and `h = R sqrt(3)/2`: points `A = (reach, -h)`, `B = (-R/2, -h)`,
`C = (-R, 0)`, `D = (-R/2, h)`, `E = (reach, h)` and the lines AB, BC, CD, DE, EA; `NewHorizontal(AB)`, `NewHorizontal(DE)`,
`NewVertical(EA)`, `NewDistancePointLine(O, line, h)` for AB, BC, CD and DE, `NewDistancePointLine(O, EA, reach)`,
`NewAngle(AB, BC, 60)` and `NewAngle(BC, CD, 60)`.

Tools, each recording its sketch:

1. `counterboreTool()`: a revolved meridian on the plane `r3.NewFrame((0, -rho_f, 4), U = (0, -1, 0), V = (1, 0, 0))`. Vertices
   (u, v) placed with `sketchcheck.DimensionPoint` and joined in order: `(0, 0)`; the nine round points `(rr + rr cos t_i,
   Rc - rr + rr sin t_i)`, `t_i = pi - (pi/2)(i/8)`, `rr = 0.5`, `Rc = 2.2`; `(a_out, Rc)`; `(a_out, 0)` with `a_out = 14 + 1 -
   rho_f`. Solve, verify, select the profile with no holes, then `decad.Revolve` about the axis from `(0, -rho_f, 4)` along
   `(0, -1, 0)` with `decad.FullRevolution{}`.
2. `setScrewSlotTool()`: `nutSlotSketch(r3.NewFrame((0, -(f_b + 1.6), 4), U = (0, 0, -1), V = (1, 0, 0)), 4 + 1)`, extruded 2
   (`NutCavityHeight`) along its normal, -Y. Then `decad.Fillet` of both caps' outer loops by `NutSlotRound`.
3. `slotTool(site, u, reach)` (formerly `bladeSlotTool(k)`): `nutSlotSketch(r3.NewFrame((site.Head.X, site.Head.Y,
   site.Head.Z - site.CavityFar()), U = u, V = (-u.Y, u.X, 0)), reach)`, extruded 2 along +Z. Blade k calls it with its
   site, `u_k` and reach `14 + 1 - 9.5`, so the plane is z = 2.4 as before. Then fillet the three edges along Z through the hexagon's corners B, C and D (C at 6.99 `u_k`), each picked by
   `decad.EndpointAt` at its stored position and joined with `.Or`, `Exactly(3)`, by `NutSlotRound`.

## 3. Parts (turbinefan.go)

1. Hub: extrude the outline 20 along +Z (profile with four holes); `edgefinish.Chamfer` of both faces by 0.5 over the long
   loops; then `decad.Cut`, in this order, the socket tool and its two pattern copies, for each blade site `m2fastener.PocketTool`
   then `bladeSlotTool` (a call of `slotTool`), then `counterboreTool`, `m2fastener.HoleTool(SetScrewSite)` and `setScrewSlotTool`. The socket tool is
   `handcrankfan.BladeSocketTool` on `r3.NewFrame((0, 5.7, 10), (0, 0, 1), (1, 0, 0))`; its copies come from
   `decad.PatternCopies` with `decad.CircularPattern{Axis: (0, 0, 1), Count: 3}`. Record every tool's sketch.
2. Blades: `handcrankfan.BuildBlade(ctx, doc, BladePitch)` (`BuildBladeMirrored` when `Hand` is `LeftHand`); record its
   sketches; `decad.Transform` by `r3.FromFrame(r3.NewFrame((0, 0, 10), (-1, 0, 0), (0, 0, 1)))`; `decad.PatternCopies` with
   the same circular pattern.
3. Require each part solid, then `placement.Place(ctx, body, p.Placement)`. Carry each site's `Head` through
   `r3.FromFrame(*Placement).Apply`, and `Dir` and `Corner` through `ApplyDir`.

`Build` then calls `m2fastener.PlaceBolt` and `PlaceNut` at each placed blade site and at the set screw site, and adds each body
to `doc.Root().ChildNamed(name)` with `AddBody`: the hub in `Hub`, the blades in `Blades`, blade bolts 1 to 3 then the set screw
in `Bolts`, and their nuts in the same order in `Nuts`.

The blade's stud union needs a document with a chain tolerance of 0.1 mm; `NewDocument` sets it.

## 4. Clutch mount

Only `Mount = ClutchMount` runs this. `ShaftMount` runs sections 1 to 3 unchanged. See instructions.md "Clutch mount" for
the numbers. `Derive` fills `ClutchBoreRadius = D/2 + 0.55` and `EarSites[k]` (`Head: 23.5 (cos a, sin a, 0)`, `Dir: (0, 0,
-1)`, `Corner: (cos a, sin a, 0)`, `Length: 25`, `Cavity: 25.5`, `a = EarBoltAngle(k+1)`) and leaves the set screw fields
zero.

`clutchHubMeridian()` (sketches.go) is a meridian on `r3.NewFrame((0, 0, 0), U = (0, 0, 1), V = (1, 0, 0))`, each vertex
`(r, z)` placed with `sketchcheck.DimensionPoint(s, z, r)`, lines joining consecutive vertices and closing the loop, the
profile with no holes revolved about the Z axis by `decad.FullRevolution{}`. With `c` 0.5, `R` 28, `T` 3, `rho` 1.5, `rh` 14,
`H` 20 and `bc = 2 pi r_c >= edgefinish.LongEdgeRun`, the vertices in order are:

1. `(r_c, -T+c)` and `(r_c+c, -T)` when `bc`, else `(r_c, -T)`.
2. `(R-c, -T)`, `(R, -T+c)`, `(R, -c)`, `(R-c, 0)`.
3. `(rh+rho, 0)`, then for i = 1 to 8 `(rh+rho + rho cos a_i, rho + rho sin a_i)` with `a_i = 3 pi/2 - (pi/2)(i/8)`.
4. `(rh, H-c)`, `(rh-c, H)`.
5. `(r_c+c, H)` and `(r_c, H-c)` when `bc`, else `(r_c, H)`.

`clutchHub()` (turbinefan.go) revolves that body, then cuts in this order: the three sockets (as `hub()`); for each blade k
`m2fastener.HoleTool(BladeSites[k])`, `PocketTool(BladeSites[k])` then `slotTool(BladeSites[k], u_k, 14 + 1 - 9.5)`; for
each ear k `PocketTool(EarSites[k])` then `HoleTool(EarSites[k])`. Every tool's sketch is recorded. Errors read `clutch
hub: <step>: %w`.

`clampSketch()` (sketches.go) on `across(ClampBack)`: `CreateCircle` of radius 28 and 19 about the origin, each with
`NewRadius`, and three circles of radius 1.3 at each `EarSites[k].Head` XY, centred by `NewHorizontalDistance` and
`NewVerticalDistance` from the origin and with `NewRadius`. `clamp()` (turbinefan.go) extrudes the profile with four holes
by `ClampThickness` along +Z, then runs `edgefinish.Chamfer(ctx, body, +Z, {ClampBack, ClampBack + 4}, Long(All), 0.5)`, then
cuts `slotTool(EarSites[k], u_k, 28 + 1 - 23.5)` for each ear.

`BuildParts` in `ClutchMount` builds `clutchHub()` for the hub, the blades as before, then `clamp()`; it requires each solid
(`clamp ring` names the clamp), places each, and places the blade sites and ear sites (no set screw site). `Build` files, in
order: the hub (`Hub`); the blades (`Blades`); the clamp ring (`Clamp Ring`); blade bolts 1 to 3 then ear bolts 1 to 3
(`Bolts`); blade nuts 1 to 3 then ear nuts 1 to 3 (`Nuts`). Bodies: 17. Sketches from `BuildParts`: 33.
