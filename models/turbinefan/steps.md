# Turbine fan Decad steps

Follow instructions.md for the API, the dimensions, the presets and the validation order. The package has seven files:
`params.go` (constants, `Hand`, `Params`, the presets, `Validate`, `Derive`, `Dimensions`), `size.go` (the rotor size
constants, `Size`, `SizeFor`, `Size.Beta`), `document.go`
(`NewDocument`), `blade.go` (the blade and its joint), `sketches.go` (the build run, the hub's outline sketch, the nut slot sketch and the tools) and
`turbinefan.go` (package doc, container names, `Parts`, `Model`, `BuildParts`, `Build`, the hub and the blades). Every error
message starts with `turbinefan: `.

No nut is embedded: the set screw's nut, the blade nuts and the adapter nuts sit in hex slots open to a face (AGENTS.md "Fasteners");
instructions.md "Fasteners" says why each slot works and why a back-face slot does not for the blade nuts.

## 1. Validate and derive (size.go, params.go)

`SizeFor(D_r)` (size.go) makes no geometry calls. With `s = D_r / 128`: `HubRadius = min(14 s, 18.9)`, `j = HubRadius / 14`,
`HubLength = 20 s`, `HubMid = HubLength / 2`, `TipRadius = D_r / 2`, `RootRadius = HubRadius + 0.4`, `RefRadius = 48 s`,
`HoldRadius = 21.5 s`, `BladeSocketStart = 5.7 j`, `BladeStudStart = BladeSocketStart + 0.4`, `BladeStudRadius = 3 j`,
`BladeSocketRadius = BladeStudRadius + 0.4`, `BladeStudEnd = 16 j`, `BladeBoltAt = 9.5 j`, `BladeBoltLength` = the longest
`m2fastener.BoltSize.StandardLengths` entry `L` with `HubLength - HeadPocketDepth - L >= 2`, `BladeBoltCavity = BladeBoltLength - 0.4`.
It fails for a diameter outside [128, 224] or not finite, when no length fits, and when `HubLength - BladeBoltCavity` is
above `HubMid - BladeSocketRadius - 1`. `Size.Beta(r, pitch) = 5 deg + atan(tan(25 deg - 5 deg) RefRadius / max(r,
HoldRadius)) + (pitch - 25 deg)`.

`Derive` makes no geometry calls. It runs the rules of instructions.md in order, stores `Size`, and fills the bore values
and the set screw: `BoreRadius`, `BoreFlat`, `BoreFlatWidth = 2 sqrt(r_b^2 - f_b^2)`, `BoreLoop`, `ShaftFlatWidth = 2
sqrt(d (D - d))`, `SetScrewTip = D/2 - d - c`, `SetScrewBolt` = the longest standard length `L` with `SetScrewTip + 2 + L <=
sqrt(HubRadius^2 - 1.9^2)`, `SetScrewHead = SetScrewTip + 2 + SetScrewBolt`, `CounterboreFloor = SetScrewHead - 2 - 0.4`.
`SetScrewSite` is `m2fastener.Site{Head: (0, -SetScrewHead, 4), Dir: (0, 1, 0), Corner: (0, 0, 1), Length: SetScrewBolt,
Cavity: SetScrewHead - f_b - 1.6 - 2}`. `BladeSites[k]` is `Head: BladeBoltAt u_k + (0, 0, HubLength), Dir: (0, 0, -1), Corner:
u_k, Length: BladeBoltLength, Cavity: BladeBoltCavity`. `AdapterSites[k]` (k = 0, 1, `u` at `EarBoltAngle(k+2)`, 30 and 150 degrees) is `Head: 9.5 u + (0, 0, -6.25), Dir: (0, 0,
1), Corner: u, Length: 10, Cavity: 7.75`, both validated; `Derive` fills it and the set screw's values in both mounts.

## 2. Sketches (sketches.go)

Each sketch is on a fresh `sketch.NewWorld()` plane made from an `r3.Frame`, recorded in the run, solved, and checked with
`sketchcheck.VerifyFullyConstrained` before its profile is used.

`hubSketch()` on the plane z = 0 (U = +X, V = +Y). Make every point with its exact initial coordinate.

- `disc = CreateCircle(O, HubRadius)` with `NewRadius`.
- The D bore: `qR = (w_b/2, -f_b)`, `qL = (-w_b/2, -f_b)`; `bore = CreateArc(O, qR, qL)` (counter-clockwise, the long way
  round over +Y); `flat = CreateLine(qL, qR)`; `NewRadius(bore, r_b)`, `NewHorizontal(flat)`,
  `NewDistancePointLine(O, flat, f_b)`.
- For each blade a circle of radius 1.3 whose centre, `BladeBoltAt` from the origin along `u_k`, is placed by
  `NewHorizontalDistance` and `NewVerticalDistance` from the origin, with `NewRadius`. Select the profile with four holes.

`nutSlotSketch(f, reach)` on the plane `f`, with `R = 2.51` and `h = R sqrt(3)/2`: points `A = (reach, -h)`, `B = (-R/2, -h)`,
`C = (-R, 0)`, `D = (-R/2, h)`, `E = (reach, h)` and the lines AB, BC, CD, DE, EA; `NewHorizontal(AB)`, `NewHorizontal(DE)`,
`NewVertical(EA)`, `NewDistancePointLine(O, line, h)` for AB, BC, CD and DE, `NewDistancePointLine(O, EA, reach)`,
`NewAngle(AB, BC, 60)` and `NewAngle(BC, CD, 60)`.

Tools, each recording its sketch:

1. `counterboreTool()`: a revolved meridian on the plane `r3.NewFrame((0, -rho_f, 4), U = (0, -1, 0), V = (1, 0, 0))`. Vertices
   (u, v) placed with `sketchcheck.DimensionPoint` and joined in order: `(0, 0)`; the nine round points `(rr + rr cos t_i,
   Rc - rr + rr sin t_i)`, `t_i = pi - (pi/2)(i/8)`, `rr = 0.5`, `Rc = 2.2`; `(a_out, Rc)`; `(a_out, 0)` with `a_out = HubRadius + 1 -
   rho_f`. Solve, verify, select the profile with no holes, then `decad.Revolve` about the axis from `(0, -rho_f, 4)` along
   `(0, -1, 0)` with `decad.FullRevolution{}`.
2. `setScrewSlotTool()`: `nutSlotSketch(r3.NewFrame((0, -(f_b + 1.6), 4), U = (0, 0, -1), V = (1, 0, 0)), 4 + 1)`, extruded 2
   (`NutCavityHeight`) along its normal, -Y. Then `decad.Fillet` of both caps' outer loops by `NutSlotRound`.
3. `slotTool(site, u, reach)` (formerly `bladeSlotTool(k)`): `nutSlotSketch(r3.NewFrame((site.Head.X, site.Head.Y, z), U = u,
   V = (-u.Y, u.X, 0)), reach)`, extruded 2 along +Z, with `z = site.Head.Z - site.CavityFar()` for a `Dir` -Z site and `z =
   site.Head.Z + site.Cavity` for a `Dir` +Z site. Blade k calls it with its site, `u_k` and reach `HubRadius + 1 - BladeBoltAt`, so the plane
   is z = `HubLength - BladeBoltCavity - 2` (2.4 at 128, 3.4 at 192). Adapter bolt k calls it with its site, its `Corner` and the reach `HubRadius + 1 - 9.5`, so the plane is z = 1.5. Then fillet the three edges along Z through the hexagon's corners B, C and D (C at 6.99 `u_k`), each picked by
   `decad.EndpointAt` at its stored position and joined with `.Or`, `Exactly(3)`, by `NutSlotRound`.

## 3. Parts (turbinefan.go)

1. Hub: extrude the outline `HubLength` along +Z (profile with four holes); `edgefinish.Chamfer` of both faces by 0.5 over the long
   loops; then `decad.Cut`, in this order, the socket tool and its two pattern copies, for each blade site `m2fastener.PocketTool`
   then `bladeSlotTool` (a call of `slotTool`), then `counterboreTool`, `m2fastener.HoleTool(SetScrewSite)` and `setScrewSlotTool`, then for each adapter site
   `m2fastener.HoleTool(AdapterSites[k])` and `slotTool(AdapterSites[k], Corner, HubRadius + 1 - 9.5)`. The socket tool is
   `BladeSocketTool` on `r3.NewFrame((0, BladeSocketStart, HubMid), (0, 0, 1), (1, 0, 0))`; its copies come from
   `decad.PatternCopies` with `decad.CircularPattern{Axis: (0, 0, 1), Count: 3}`. Record every tool's sketch.
2. Blades: `BuildBlade(ctx, doc, BladePitch, Size)` (`BuildBladeMirrored` when `Hand` is `LeftHand`); record its
   sketches; `decad.Transform` by `r3.FromFrame(r3.NewFrame((0, 0, HubMid), (-1, 0, 0), (0, 0, 1)))`; `decad.PatternCopies` with
   the same circular pattern.
3. Require each part solid, then `placement.Place(ctx, body, f)`, where `f` is `Placement` moved back from the placement point
   to the local origin: origin `Placement.Apply((0, 0, -anchor))` with `Placement`'s U and V (anchor `HubLength` in `ShaftMount`,
   `ClutchAnchorZ` in `ClutchMount`; `f` is nil when `Placement` is nil). Carry each site's `Head` through `f` with
   `r3.FromFrame(*f).Apply`, and `Dir` and `Corner` through `ApplyDir`.

`Build` then calls `m2fastener.PlaceBolt` and `PlaceNut` at each placed blade site and, in `ShaftMount`, at the set screw site
or, in `ClutchMount`, at each placed adapter site, and adds each body to `doc.Root().ChildNamed(name)` with `AddBody`: the hub
in `Hub`, the blades in `Blades`, blade bolts 1 to 3 then the set screw or adapter bolts 1 and 2 in `Bolts`, and their nuts in
the same order in `Nuts`.

The blade's stud union needs a document with a chain tolerance of 0.1 mm; `NewDocument` sets it.

## 4. Blade (blade.go)

`BuildBlade(ctx, doc, pitch, size)` builds blade 1 in the blade frame and returns its body and its twelve sketches: the ten
sections, the stud circle and the bolt-hole circle, in that order. `BuildBladeMirrored` differs only in `mirror`. The
first failing step stops the rest.

- Blade frame: sections on world-XY planes at Z = r (radius) with the sketch origin on the stacking line; local Z is the
  radial line, local X the blade's direction of motion and local Y downstream. A local point `(u, v)` of a section at
  angle `beta` for chord position `xi` (mm from the leading edge) and offset `eta` toward the suction side is
  `(xiP - xi)(cos beta, -sin beta) + (eta - etaP)(-sin beta, -cos beta)`, with `xiP = c/2` and `etaP = c yc(0.5)`.
- Sections: one `sketch.NewWorld()`. For each of the ten stations of the instructions' table (`stationsFor(size)`: stations 1
  and 2 from `RootRadius` and `HubScale`, stations 3 to 10 from the airfoil table times `Scale`), a sketch on
  `CreateOffsetPlane(XY, r)`. It holds 21 fit points, each made by `sketchcheck.DimensionPoint`, in the order suction
  surface at x_k for k = 10..1, the leading edge, then the pressure surface for k = 1..10, with
  `x_k = (1 - cos(pi k / 10)) / 2`. A tail point carries `NewMidpointOf(tail, first, last)`. There is one `CreateFitSpline`
  through the 21 points and the lines `last -> tail` and `tail -> first`. Solve, require `sketchcheck.VerifyFullyConstrained`,
  and select the valid profile with no holes and three outer edges. The sections are always drawn at the 25 degree blade
  angle, with every X coordinate negated for `BuildBlade` (not for `BuildBladeMirrored`).
- Loft: `decad.LoftSections` through the ten sections in ascending r. When the pitch is not 25 degrees, `decad.Transform`
  the loft by `r3.RotationAround(origin, +Z, -(pitch - 25 degrees))` (the opposite turn when mirrored).
- Root stud: a circle of radius `BladeStudRadius` on the frame origin `(0, 0, BladeStudStart)`, U = +X, V = +Y, extruded
  `BladeStudEnd - BladeStudStart`. `decad.Chamfer` its start cap's outer loop with 0.3. `Cut` it by a circle of radius 1.3
  on the frame origin `(0, -(BladeStudRadius + 2), BladeBoltAt)`, U = +Z, V = +X, extruded `2 (BladeStudRadius + 2)` (the
  bolt hole). The cut happens before the union, so the faceted loft never takes part in it.
- `Union` the loft and the stud: they overlap in volume and share no face.
- `BladeSocketTool(ctx, doc, f, size)`: a circle of radius `BladeSocketRadius` on `f`, extruded `HubRadius + 1 -
  BladeSocketStart`, with `decad.Fillet` of its
  start cap's outer loop by 0.5. It makes one sketch.
- The document must come from `NewDocument`: without its chain tolerance, Decad refuses the blade-stud union because the
  loft's mesh bound is coarser than the pair's chord tolerance.

## 5. Clutch mount

`Mount = ClutchMount` changes only what `Build` adds. The hub, the blades and their sketches are the `ShaftMount` ones
(sections 1 to 3), so the hub has the set screw's cuts and the two adapter holes and slots in both mounts. See
instructions.md "Clutch mount" for the numbers.

- `MountZ` is 26.25 (`onewayclutch.AdapterTo + ClutchSeam`). `MountFrame` returns the placement point: origin (0, 0,
  `onewayclutch.CoverTo`) in the clutch frame, turned `ClutchTurn` (180 degrees) about Z. `ClutchAnchorZ` is `CoverTo - MountZ` = -7.75.
- `ParamsForClutch(c, opts...)` takes `WithBladePitch(pitch)` and `WithRotorDiameter(d)` to replace the defaults, then runs
  `Validate`.
- `BuildParts` places `SetScrewSite` and both `AdapterSites` in both modes. `Build` in `ClutchMount` places an M2 x 10 bolt and
  nut at each adapter site and no set screw. Bodies: 14. Sketches from `BuildParts`: 27, the same as `ShaftMount`.
- `Build` does not build the adapter or the clutch. The caller builds them with `onewayclutch.Build` and
  `onewayclutch.BuildAdapter` in the clutch's own `onewayclutch.NewDocument()`, and the fan in its own document, because both
  models file their hub in a container named `Hub`. The adapter's `FaceSites[1]` and `FaceSites[2]` are the fan's
  `AdapterSites[0]` and `[1]` through `MountFrame` and the anchor offset; `FaceSites[0]` stays empty.
