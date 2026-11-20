# Plaque stand Decad steps

Follow instructions.md for the dimensions, limits, derived values, the outline table, the rail and the fastener stack.
The package has these files: `params.go` (constants, `Params`, `DefaultParams`, `Validate`, `Derive`, `Dimensions`),
`outline.go` (`Segment`, `Outline`, `Area`, `Length`, `IsArc`, `Sweep`, unexported `buildOutline`), `sketches.go`
(`profileSketch`, `railSketch`, `plaqueSketch`) and `plaquestand.go` (package doc, container names, `Model`, `Build`).
Every error message starts with `plaquestand: `, and every Sketch or Decad error is wrapped with the step that failed.
`internal/m2fastener` gains `RoundedHexPocketTool`, which the brackets use.

## 1. Validate and derive (params.go, outline.go)

`Derive` makes no geometry calls. It checks `PlaqueWidth`, `PlaqueHeight`, `PlaqueThickness`, `Recline`, `LipHeight` and
`SlotClearance` against their limits in that order (a value fails when `!(v >= min && v <= max)`), computes every value
of "Derived values" in instructions.md, then returns the side-tip-angle error when `TipSide < MinTipAngle` and the extent
error when the span, depth or height exceeds `BedSize - 2 BedMargin`. `SlotClearance`'s minimum is `LooseFit +
FitMargin`, and `JointFit` is `RegularFit + FitMargin`. `Depth` is rounded up to a whole millimeter and takes `D_geom` with
the channel: `Q3_y + r_r k_b + ChannelLand + 2 r_m + w_c + RearLand + r_c`. The channel's walls, floor and centre, and the
two `m2fastener.Site` values (`Head` at `(x_k + w_b/2, y_ch, T_b + j)`, `Dir` -Z, `Corner` +X, `Length` `BoltLength`,
`Cavity` 6.4), come from `Depth`.

`buildOutline` returns the 26 segments of the outline table in order, with points `P(a, b) = C + a n + b u`.
`Outline.Area` and `Outline.Length` follow the formulas in instructions.md. `Segment.Sweep` is `2 asin(min(1, chord / (2
Radius)))` for an arc and 0 for a line.

## 2. Sketches (sketches.go)

Each sketch is on a fresh `sketch.NewWorld()` plane made from an `r3.Frame` with `CreatePlaneFromFrame`, solved with
`Solve`, checked with `sketchcheck.VerifyFullyConstrained` (`FullyConstrained`, DOF 0, no redundant constraint, no
conflict, and `report.Check()`), and its one valid profile with no holes is selected (an error when there is none or
several). A `Verify` failure is a blocker: add no point-fixing constraint to force DOF 0.

### profileSketch(ctx, dm, x0)

1. Frame `r3.NewFrame((x0, 0, 0), (0, 1, 0), (0, 0, 1))`. Sketch (u, v) is world (y, z), and the sketch normal is +X.
2. One point per segment Start at its exact (y, z), `p[0]` to `p[25]`, and one point per arc Center, seeded from
   `dm.Outline`. No constraint touches a point except as listed in step 4.
3. Lines `L_k = CreateLine(p[2k-2], p[2k-1])` for k = 1 to 13 (segments 0, 2, ..., 24, with `p[26]` = `p[0]`), named L1, L2,
   L3a, W1, Fl, W2, L3b, L4, L5, L6, L7, L8, L9. Arcs, counter-clockwise from start to end: a convex `A_k =
   CreateArc(center, p[2k-1], p[2k])`; a concave one (k = 4, 5, 7, 9 and 11, the outline's concave indices 7, 9, 13,
   17 and 21) `A_k = CreateArc(center, p[2k], p[2k-1])`, for k = 1 to 13 (segments 1, 3, ..., 25).
4. Constraints, exactly these (78 equations counting the 13 implicit arc-radius equalities, against 39 points x 2):
   - `NewRadius(A_k, Outline[2k-1].Radius)` for k = 1 to 13.
   - `NewTangent(L_k, A_k)` and `NewTangent(L_{k+1}, A_k)` for k = 1 to 13, with `L_14` = `L_1` (26 endpoint tangencies).
   - Orientation: `NewHorizontal` of L1, L3a, Fl, L3b and L8; `NewVertical` of L2, W1, W2 and L9;
     `NewAngle(L1, L5, -(90 + Recline))` (L1 runs +Y and L5 runs -u); `NewParallel(L5, L4)`, `(L5, L6)` and `(L5, L7)`.
   - Position: `NewPointOnLine(Origin, L1)` and `(Origin, L9)`; `NewDistancePointLine(Origin, ...)` of L2 (`D`), L3a
     (`T_b`), W1 (`y_cb`), Fl (`z_f`), W2 (`y_cf`), L3b (`T_b`) and L8 (`T_b`); `NewHorizontalDistance(Origin,
     A9.Center, C_y)` and `NewVerticalDistance(Origin, A9.Center, C_z)` on the slot round's centre; `NewDistance(L5.Start,
     L5.End, h_b)` and `NewDistance(L6.Start, L6.End, h_l)`.
5. Solve, verify fully constrained, select the profile. The conditioning is at least 6.8e-5 over the 729-point grid.

### railSketch(ctx, dm)

1. Frame `r3.NewFrame((0, y_ch, T_b - h_r/2 + j), (0, 1, 0), (0, 0, 1))`, normal +X.
2. Four lines and four convex arcs of radius `r_q`: the bottom line from `(-w_r/2 + r_q, -h_r/2)` to `(w_r/2 - r_q,
   -h_r/2)`, then counter-clockwise (arc, right line, arc, top line, arc, left line, arc). 12 points: 8 tangent points and
   4 centres.
3. `NewRadius` x 4, endpoint `NewTangent` x 8, `NewHorizontal` of the bottom and top lines, `NewVertical` of the right and
   left lines, and `NewDistancePointLine(Origin, ...)` of bottom, right, top and left by `h_r/2`, `w_r/2`, `h_r/2` and
   `w_r/2` (24 equations with the 4 implicit radius equalities, against 12 points x 2).
4. Solve, verify fully constrained, select the profile.

### plaqueSketch(ctx, dm, p)

1. `x0 = (S - W)/2`. Frame origin `(x0, C_y + (t/2) cos theta, C_z - (t/2) sin theta)`, U = `(1, 0, 0)`, V = `u`. The normal
   U x V = `(0, -cos theta, sin theta)` points to the front.
2. `CreateRectangle(0, 0, W, H)` with `NewCoincident(rect.A, Origin)`, `NewHorizontalDistance(rect.A, rect.B, W)` and
   `NewVerticalDistance(rect.B, rect.C, H)`.
3. Solve, verify fully constrained, select the profile.

## 3. Build (plaquestand.go)

`Build(ctx, doc, p) (*Model, error)`; `Model` holds `Brackets` [2], `Rail`, `Bolts` [2]`*fastener.Bolt`, `Nuts`
[2]`*fastener.Nut`, `Plaque`, `Sketches` and `Dimensions`.

1. A nil `ctx` returns `plaquestand: context must not be nil`, and a nil `doc` returns `plaquestand: document must not
   be nil`.
2. `p.Derive()`; return its error unchanged.
3. `ctx.Err()` returns `plaquestand: before creating geometry: %w`.
4. For each bracket k with `x_k` = 0 and `S - w_b`: `profileSketch(ctx, dm, x_k)`, then `decad.Extrude(ctx, doc, sketch,
   profile, decad.Distance{D: w_b mm, Dir: decad.Along})`, then `edgefinish.Chamfer(ctx, body, (1, 0, 0), []float64{x_k,
   x_k + w_b}, edgefinish.Outer, ChamferSize)`, then `decad.Cut` of `m2fastener.RoundedHexPocketTool(ctx, doc, site,
   NutCavityHeight + Stock, NutPocketRound)` and of `m2fastener.HoleTool(ctx, doc, site)`. Require `IsSolid()`.
   `RoundedHexPocketTool` is `HexPocketTool` followed by `decad.Fillet` of `decad.Edges(decad.OuterLoopOf(decad.CapStart(
   tool)))` by the round.
5. `railSketch`, then `decad.Extrude` by `S` along +X. Cut `PocketTool` and `HoleTool` for both sites. Require `IsSolid()`.
   No chamfer.
6. `plaqueSketch`, then `decad.Extrude` by `t` along the normal. Require `IsSolid()`.
7. `m2fastener.PlaceBolt` and `PlaceNut` for each site.
8. For each body: `doc.Root().ChildNamed(name)` (`Brackets`, `Rail`, `Bolts`, `Nuts`, `Plaque`) and `AddBody`.
9. Return the `Model`. `Build` adds exactly eight bodies to `doc`, and a failure before step 4 adds none.
   `Sketches` holds 12 sketches: 2 profiles, 2 hex pockets, 2 bracket holes, the rail, 2 head pockets, 2 rail holes and the
   plaque.

`decad.New()` suffices: no Boolean joins two parts, so no chain tolerance is needed. A refusal in step 4 or 5 on the
default case is a decad blocker. Report the smallest failing case and the error instead of changing the geometry.

## Why the nut sits in a pocket, not a cavity

Each nut sits in a hex pocket open to the bracket's bottom face, so no enclosed cavity and no print pause is needed.
