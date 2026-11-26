# Dovetail joint: Decad steps

These steps compile [instructions.md](instructions.md) into Sketch and Decad operations. Package `features/dovetail`.
Units are millimeters. Use `Stock` = 1.0 and `MinWall` = 0.8.

## Exported API

- `Params{TailWidth float64; Slope Slope; Length, Depth float64; TailOn Carrier}`, `DefaultParams()`,
  `Params.TipWidth()`, `Params.Lean()` (`Depth * Rise / Run`) and `Params.Validate()`. `Slope{Rise, Run int}` has
  `Angle()` and `DepthStep()`. `DefaultSlope` is 7:24 and `AllowedSlopes` lists the accepted ones. `Carrier` is
  `TailOnFirst` (0) or `TailOnSecond` (1).
- `Apply(ctx, doc, p, f, first, second) (*Joint, error)`. `Joint` holds `First`, `Second`, `TailSketch`,
  `SocketSketch`, and the methods `Sketches()`, `TailBody(p)` and `SocketBody(p)`.
- `Params.ChamferSize()`, the smallest `H * j / 2048` at or above `edgefinish.MinChamfer` (0.50048828125 at 7:24),
  and `Params.ChamferCut()`, the profile area one chamfer removes.
- `BuildDemo(ctx, doc, p) (*Demo, error)`, `DemoFrame()` and the container names `TailBlocksContainerName` and
  `SocketBlocksContainerName`.

## 1. Validate

Reject a nil ctx, doc or host and an invalid frame. Call `Params.Validate`: `Slope` must be in `AllowedSlopes`, and
`Depth * Rise / Run` must be an exact float (checked with `big.Rat`), else the error names the allowed slopes or the
depth step. For each host, read `Bounds()`, map its
eight corners into the frame with `Frame.ToLocal`, and take the local minimum and maximum. Require the checks listed
under "Frame and hosts" in the instructions, with the tolerance `Bounds().Bound.Base()` + 1e-6. Check: V span, U
cover, the host's near face at N = 0, the host's depth from the mating plane.
Check at the boundary: each refusal names the failed check.

## 2. Profile sketch (twice: tail, then socket)

Let s = +1 for `TailOnFirst` and -1 otherwise. Build the plane frame with origin = O + V * y0, U axis = U, V axis = s * N.
The tail sketch uses y0 = 0. The socket sketch uses y0 = -`Stock`. The plane's normal is U x (s N) = -s V.

Create the six points with `CreatePoint`: A(-w/2, -Stock), B(w/2, -Stock), C(w/2, 0), D(W/2, Depth), E(-W/2, Depth),
G(-w/2, 0), where w = `TailWidth` and W = `TipWidth` (`w + 2 Lean()`). Create the lines AB, BC, CD, DE, EG, GA with `CreateLine`.
Add the constraints:

- `NewHorizontal(AB)`, `NewHorizontal(DE)`, `NewVertical(BC)`, `NewVertical(GA)`.
- `NewMidpointOf(origin, G, C)` and `NewHorizontalPoints(G, C)`.
- `NewDistance(G, C, w)`, `NewVerticalDistance(B, C, Stock)`, `NewVerticalDistance(C, D, Depth)`.
- `NewHorizontalDistance(C, D, Lean())` and `NewHorizontalDistance(E, G, Lean())`. The flank's sideways distance
  replaces an angle, so the corner solves to an exact float.

The computed points A to E are not fixed by coordinates. Their positions come from the lines and dimensions.
Solve, then call `sketchcheck.VerifyFullyConstrained`: status `FullyConstrained`, DOF 0, no redundant constraints, no
conflicts. Select the one valid profile with no holes.
Check at the boundary: the profile count is 1 and the sketch verifies.

## 3. Extrude

- Tail: extrude the tail profile by `Length`.
- Socket cutter: extrude the socket profile by `Length` + 2 `Stock`.
- Direction: the plane normal is -s V, so use `decad.Against` when s = +1 and `decad.Along` when s = -1. Both then
  sweep toward +V.
Check at the boundary: each tool is a solid body.

## 4. Chamfer, join and cut

- Chamfer the tail body's two wide-end edges: the straight edges along V between (+-W/2, s Depth) at V = 0 and
  V = `Length`, in the joint frame. Find each on `tail.Edges()` by its end points, name it with
  `decad.EndpointAt(start)` and `decad.ParallelTo(direction)`, and call `decad.Chamfer(tail,
  query.Exactly(2), units.Millimeters(p.ChamferSize())).Do(ctx)`. Check that the result is solid.
- `decad.Union(tailHost, tail).Do(ctx)`.
- `decad.Cut(socketHost, socketCutter).Do(ctx)`.
- Chamfer the cut host's two lip edges the same way: the edges along V at (+-w/2, 0), `Exactly(2)`, `p.ChamferSize()`.
  Check that the result is solid.
The socket cutter is not chamfered, so the socket's floor corners stay sharp, and the tail's root edges stay sharp.
Union and Cut retire their operands and return new bodies; so does each chamfer. Return them as `First` and `Second` according to `TailOn`.
The 1 mm neck root inside the tail's host overlaps the host, so the join has volume to merge. The tail's end faces
lie in the host's end faces, and Union merges them into one face with no edge between them.
Check at the boundary: both results are solid, and the chamfer finds exactly two edges on each body.

## 5. Demo

- Build two block sketches on the frame's U and V plane with origin at the midpoint of the V = 0 edge: four lines,
  two horizontal, two vertical, `NewMidpoint(origin, bottom edge)`, `NewDistance` for the width and
  `NewVerticalDistance` for the length. Verify each.
- Extrude the first block `Depth` + 4 `decad.Against` and the second the same `decad.Along`.
- `Apply` at `DemoFrame()`.
- Create the containers with `doc.Root().ChildNamed("Tail Blocks")` and `ChildNamed("Socket Blocks")`, and add the tail's
  block and the socket's block with `AddBody`.
Check at the boundary: `doc.VerifyContainers().Err()` is nil.

## 6. Geometry checks

For every case in the instructions:

- All four demo sketches verify as in step 2.
- Bounds of each block: U within +-(`TipWidth`/2 + 4), V from 0 to `Length`, N as the instructions give, with the tail
  host reaching `Depth` past the mating plane.
- Volumes: with the profile area A = `(TailWidth + TipWidth)/2 * Depth` and the chamfer area `ChamferCut()`, the
  tail's host equals the blank plus `(A - 2 ChamferCut()) Length`, and the socket's host equals the blank minus
  `(A + 2 ChamferCut()) Length`.
- Profile: a section at V = `Length`/2 of the tail block has a vertex within 0.02 of each root corner at
  (+-`TailWidth`/2, 0) and of each chamfer end, and none within 0.1 of the wide-end corner (+-`TipWidth`/2, s Depth).
  The socket block has a vertex at each floor corner (+-`TipWidth`/2, s Depth) and each chamfer end, and none within 0.1
  of a lip (+-`TailWidth`/2, 0). The chamfer ends are the points `p.ChamferSize()` from the corner along each of its two
  faces.
- Edges: each block has straight edges along V at the concave corners and at the four chamfer ends, and none at the
  convex corners.
- Mating: every vertex of both blocks has a bound of zero. `meshgap` reports the blocks touch (lower bound of the gap
  is 0). `doc.Verify` reports `Sound` with no interference and no diagnostic. With both blocks in one container,
  `decad.Intersect` fails with `BooleanEmpty`, and `decad.Union` has exactly the sum of the two volumes.
- End faces: each block has exactly one face lying wholly in the plane V = 0 and one in V = `Length`, with one loop. Its
  area is the block's end area plus the tail profile less its chamfers for the tail block, and less the profile and its
  chamfers for the socket block.
- Sliding: a block moved by `Length` + 1 along V is at least 0.5 from the other.
- Containers: names, membership and `VerifyContainers`.
- Corners: `cornercheck.Check` with the slide direction as the print axis finds only the two sharp concave corners per
  block: the tail block's root corners and the socket block's floor corners.
- Misfit hosts, invalid parameters and a moved and turned frame.
