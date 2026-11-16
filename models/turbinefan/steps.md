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
3. `bladeSlotTool(k)`: `nutSlotSketch(r3.NewFrame(9.5 u_k + (0, 0, 2.4), U = u_k, V = v_k), 14 + 1 - 9.5)`, extruded 2 along +Z.
   Then fillet the three edges along Z through the hexagon's corners B, C and D (C at 6.99 `u_k`), each picked by
   `decad.EndpointAt` at its stored position and joined with `.Or`, `Exactly(3)`, by `NutSlotRound`.

## 3. Parts (turbinefan.go)

1. Hub: extrude the outline 20 along +Z (profile with four holes); `edgefinish.Chamfer` of both faces by 0.5 over the long
   loops; then `decad.Cut`, in this order, the socket tool and its two pattern copies, for each blade site `m2fastener.PocketTool`
   then `bladeSlotTool`, then `counterboreTool`, `m2fastener.HoleTool(SetScrewSite)` and `setScrewSlotTool`. The socket tool is
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
