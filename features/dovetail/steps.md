# Dovetail joint: Decad steps

These steps compile [instructions.md](instructions.md) into Sketch and Decad operations. Package `features/dovetail`.
Units are millimeters and degrees. Use `Stock` = 1.0 and `MinWall` = 0.8.

## Exported API

- `Params{TailWidth, Angle, Length, Depth float64; TailOn Carrier}`, `DefaultParams()`, `Params.TipWidth()`,
  `Params.Validate()`. `Carrier` is `TailOnFirst` (0) or `TailOnSecond` (1).
- `Apply(ctx, doc, p, f, first, second) (*Joint, error)`. `Joint` holds `First`, `Second`, `TailSketch`,
  `SocketSketch`, and the methods `Sketches()`, `TailBody(p)` and `SocketBody(p)`.
- `BuildDemo(ctx, doc, p) (*Demo, error)`, `DemoFrame()` and the container names `TailBlocksContainerName` and
  `SocketBlocksContainerName`.

## 1. Validate

Reject a nil ctx, doc or host and an invalid frame. Call `Params.Validate`. For each host, read `Bounds()`, map its
eight corners into the frame with `Frame.ToLocal`, and take the local minimum and maximum. Require the checks listed
under "Frame and hosts" in the instructions, with the tolerance `Bounds().Bound.Base()` + 1e-6. Check: V span, U
cover, the host's near face at N = 0, the host's depth from the mating plane.
Check at the boundary: each refusal names the failed check.

## 2. Profile sketch (twice: tail, then socket)

Let s = +1 for `TailOnFirst` and -1 otherwise. Build the plane frame with origin = O + V * y0, U axis = U, V axis = s * N.
The tail sketch uses y0 = 0. The socket sketch uses y0 = -`Stock`. The plane's normal is U x (s N) = -s V.

Create the six points with `CreatePoint`: A(-w/2, -Stock), B(w/2, -Stock), C(w/2, 0), D(W/2, Depth), E(-W/2, Depth),
G(-w/2, 0), where w = `TailWidth` and W = `TipWidth`. Create the lines AB, BC, CD, DE, EG, GA with `CreateLine`.
Add the constraints:

- `NewHorizontal(AB)`, `NewHorizontal(DE)`, `NewVertical(BC)`, `NewVertical(GA)`.
- `NewMidpointOf(origin, G, C)` and `NewHorizontalPoints(G, C)`.
- `NewDistance(G, C, w)`, `NewVerticalDistance(B, C, Stock)`, `NewVerticalDistance(C, D, Depth)`.
- `NewAngle(BC, CD, -Angle)` and `NewAngle(GA, EG, +Angle)`. Angles are signed from the first line's direction to the second's.

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

## 4. Join and cut

- `decad.Union(ctx, tailHost, tail)`.
- `decad.Cut(ctx, socketHost, socketCutter)`.
Both retire their operands and return new bodies. Return them as `First` and `Second` according to `TailOn`.
The 1 mm neck root inside the tail's host overlaps the host, so the join has volume to merge. The tail's end faces
lie in the host's end faces, which Union takes as coplanar faces.
Check at the boundary: both results are solid.

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
- Volumes: the tail's host equals the blank plus the tail's volume and the socket's host equals the blank minus it,
  with the tail's volume `(TailWidth + TipWidth)/2 * Depth * Length`.
- Profile: a section at V = `Length`/2 of each block has a vertex within 0.02 of each of the profile's four corners at
  (+-`TailWidth`/2, 0) and (+-`TipWidth`/2, s Depth).
- Mating: `meshgap` reports the blocks touch (lower bound of the gap is 0). decad's `LocatePoint` classifies a grid of
  points across the mid-length section, and no point is inside both blocks.
- Sliding: a block moved by `Length` + 1 along V is at least 0.5 from the other.
- Containers: names, membership and `VerifyContainers`.
- Corners: `cornercheck.Check` with the slide direction as the print axis finds only the two exempt corners per block.
- Misfit hosts, invalid parameters and a moved and turned frame.

## Decad gaps

`Document.Verify`, `decad.Union` and `decad.Intersect` refuse the two finished halves in exact contact in most cases
(the default, both angle limits and the smallest joint): `unsupported_pair_contact`, and a refusal that the held facets "come within the chord tolerance without provably
interpenetrating". The checks above use `LocatePoint`, volumes and `meshgap` instead. The smallest case is
`dovetail.BuildDemo` at the defaults, with the two blocks (taken out of their containers) joined by `decad.Union`.
