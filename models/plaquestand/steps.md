# Plaque stand Decad steps

Follow instructions.md for the dimensions, limits, derived values and the outline table. The package has four files:
`params.go` (constants, `Params`, `DefaultParams`, `Validate`, `Derive`, `Dimensions`), `outline.go` (`Segment`,
`Outline`, `Area`, `Length`, `IsArc`, `Sweep`, unexported `buildOutline`), `sketches.go` (`profileSketch`,
`plaqueSketch`) and `plaquestand.go` (package doc, container names, `Model`, `Build`). Every error message starts with
`plaquestand: `, and every Sketch or Decad error is wrapped with the step that failed.

## 1. Validate and derive (params.go, outline.go)

`Derive` makes no geometry calls. It checks `PlaqueWidth`, `PlaqueHeight`, `PlaqueThickness`, `Recline`, `LipHeight` and
`SlotClearance` against their limits in that order (a value fails when `!(v >= min && v <= max)`), computes every value
of "Derived values" in instructions.md, then returns the side-tip-angle error when `TipSide < MinTipAngle` and the
extent error when the stand width, depth or height exceeds `BedSize - 2 BedMargin`. `SlotClearance`'s minimum is
`LooseFit + FitMargin`. `Depth` is rounded up to a whole millimeter.

`buildOutline` returns the 18 segments of the outline table in order, with points `P(a, b) = C + a n + b u`.
`Outline.Area` and `Outline.Length` follow the formulas in instructions.md. `Segment.Sweep` is `2 asin(min(1, chord / (2
Radius)))` for an arc and 0 for a line.

## 2. Sketches (sketches.go)

Each sketch is on a fresh `sketch.NewWorld()` plane made from an `r3.Frame` with `CreatePlaneFromFrame`, solved with
`Solve`, checked with `sketchcheck.VerifyFullyConstrained`, and its one valid profile with no holes is selected (an
error when there is none or several). A `Verify` failure is a blocker: add no point-fixing constraint to force DOF 0.

### profileSketch(ctx, dm)

1. Frame `r3.NewFrame((0, 0, 0), (0, 1, 0), (0, 0, 1))`. Sketch (u, v) is world (y, z), and the sketch normal is +X.
2. One point per segment Start at its exact (y, z), `p[0]` to `p[17]`, and one point per arc Center, seeded from
   `dm.Outline`. No constraint touches a point except as listed in step 4.
3. Lines: `L_k = CreateLine(p[2k-2], p[2k-1])` for k = 1 to 9 (segments 0, 2, ..., 16, with `p[18]` = `p[0]`). Arcs, counter-
   clockwise from start to end: a convex `A_k = CreateArc(center, p[2k-1], p[2k])`; a concave one (A3, A5, A7)
   `A_k = CreateArc(center, p[2k], p[2k-1])`, for k = 1 to 9 (segments 1, 3, ..., 17).
4. Constraints, exactly these (54 equations counting the nine implicit arc-radius equalities, against 27 points x 2):
   - `NewRadius(A_k, Outline[2k-1].Radius)` for k = 1 to 9.
   - `NewTangent(L_k, A_k)` and `NewTangent(L_{k+1}, A_k)` for k = 1 to 9, with `L_10` = `L_1` (18 endpoint tangencies).
   - Orientation: `NewHorizontal(L1)`, `NewVertical(L2)`, `NewHorizontal(L3)`, `NewHorizontal(L8)`, `NewVertical(L9)`,
     `NewAngle(L1, L5, -(90 + Recline))` (L1 runs +Y and L5 runs -u), `NewParallel(L5, L4)`, `NewParallel(L5, L6)` and
     `NewParallel(L5, L7)`.
   - Position: `NewPointOnLine(Origin, L1)`, `NewPointOnLine(Origin, L9)`, `NewDistancePointLine(Origin, L2, D)`,
     `NewDistancePointLine(Origin, L3, T_b)`, `NewDistancePointLine(Origin, L8, T_b)`,
     `NewHorizontalDistance(Origin, A5.Center, C_y)`, `NewVerticalDistance(Origin, A5.Center, C_z)`,
     `NewDistance(L5.Start, L5.End, h_b)` and `NewDistance(L6.Start, L6.End, h_l)`.
5. Solve, verify fully constrained, select the profile.

### plaqueSketch(ctx, dm, p)

1. `x0 = (W_s - W)/2`. Frame origin `(x0, C_y + (t/2) cos theta, C_z - (t/2) sin theta)`, U = `(1, 0, 0)`, V = `u`. The
   normal U x V = `(0, -cos theta, sin theta)` points to the front.
2. `CreateRectangle(0, 0, W, H)` with `NewCoincident(rect.A, Origin)`, `NewHorizontalDistance(rect.A, rect.B, W)` and
   `NewVerticalDistance(rect.B, rect.C, H)`.
3. Solve, verify fully constrained, select the profile.

## 3. Build (plaquestand.go)

`Build(ctx, doc, p) (*Model, error)`; `Model` holds `Stand`, `Plaque`, `Profile`, `PlaqueSketch` and `Dimensions`.

1. A nil `ctx` returns `plaquestand: context must not be nil`, and a nil `doc` returns `plaquestand: document must not
   be nil`.
2. `p.Derive()`; return its error unchanged.
3. `ctx.Err()` returns `plaquestand: before creating geometry: %w`.
4. `profileSketch`, then `decad.Extrude(ctx, doc, sketch, profile, decad.Distance{D: W_s mm, Dir: decad.Along})`.
5. `edgefinish.Chamfer(ctx, stand, (1, 0, 0), []float64{0, W_s}, edgefinish.Outer, ChamferSize)`. Require `IsSolid()`.
6. `plaqueSketch`, then `decad.Extrude` by `t` along the normal. Require `IsSolid()`.
7. For each body: `doc.Root().ChildNamed(name)` (`Stand`, `Plaque`) and `AddBody`.
8. Return the `Model`. `Build` adds exactly two bodies to `doc`, and a failure before step 4 adds none.

`decad.New()` suffices: the build has no Boolean, so no chain tolerance is needed. A refusal in step 4 or 5 on the
default case is a decad blocker. Report the smallest failing case and the error instead of changing the geometry.
