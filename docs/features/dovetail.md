# Dovetail joint

The feature in `features/dovetail` adds a straight sliding dovetail tail to one host body and cuts the matching socket
into another. It makes no part of its own.

## Construction and API

`Params` holds `TailWidth`, `Slope`, `Length`, `Depth` and `TailOn`. `Slope{Rise, Run}` is the flank's sideways lean
per unit of depth. `DefaultParams()` returns a 10 mm tail width, the 7:24 slope (16.26 degrees), 16 mm length, 6 mm
depth and `TailOnFirst`. `Params.TipWidth()` returns the tail's wide-end width, `TailWidth + 2 Depth Rise / Run`.
`Params.Validate()` rejects a slope outside `AllowedSlopes` (the 14 whole-number right triangles from 6.0 to 18.9
degrees, listed in the error), a depth for which `Depth * Rise / Run` is not an exact float, a tail width under 3 mm,
a depth under 3 mm, a length under 5 mm and an unknown `TailOn`. At 7:24, `Depth` must be a multiple of 3 mm times a
power of two, such as 3, 6, 7.5 or 12 mm.

`Apply(ctx, doc, p, f, first, second)` joins two host bodies at the frame `f`. The frame's origin is the centre of the
joint's start on the mating plane, U runs across the joint, V runs along the slide direction and N = U x V points from
the first host into the second. The joint runs from V = 0 to V = `Length`. `Apply` refuses hosts that do not span that
length, do not cover the joint's width plus 0.8 mm each side, or are too shallow, and says which. It retires both hosts,
as `decad.Union` and `decad.Cut` do, and returns a `Joint` with the finished `First` and `Second`, the two profile
sketches in `TailSketch` and `SocketSketch`, `Sketches()`, `TailBody(p)` and `SocketBody(p)`.

Both profile sketches come from one function and are verified fully constrained before they are extruded. The halves
are built at exact nominal size with no clearance. `Apply` then chamfers the convex joint edges with `decad.Chamfer` at
`Params.ChamferSize()` (0.50048828125 mm at 7:24, the smallest `H * j / 2048` at or above 0.5 mm, where H is the
slope's hypotenuse): the tail's two wide-end edges before the join, and the socket host's two lip edges after the
cut. The concave edges stay sharp. `Params.ChamferCut()` gives the profile area one chamfer removes. The flank is drawn
with a horizontal distance, not an angle, so the corners and the chamfer feet are exact floats. The tail's end faces
merge with the host's end faces. The sketch, the tool bodies and the corner rules are in
[instructions.md](../../features/dovetail/instructions.md) and [steps.md](../../features/dovetail/steps.md).

`BuildDemo(ctx, doc, p)` builds two blocks, each `TipWidth + 8` wide, `Length` long and `Depth + 4` tall, at the world
frame (U = X, V = Y, N = Z) and applies the joint. It returns a `Demo` with `First`, `Second`, `TailBlock`,
`SocketBlock` and `Sketches`. The bodies sit in the containers `Tail Blocks` and `Socket Blocks`.

## Build and images

```sh
go test ./features/dovetail
go -C tools/render run . -model dovetail -shot readme -out ../../features/dovetail/images
```

The readme shot writes `hero.png`, `index.png`, `end.png`, `rear.png`, `low.png`, `exploded.png` and the part images
in `parts/`. `-shot` also takes `hero`, `end`, `rear`, `low`, `exploded`, `tail-block` or `socket-block` alone. The
exploded view moves the first block 15 mm toward -Y and the second 15 mm toward +Y.

## Verification

The tests run the demo for six cases: the defaults, the tail on the second block, the shallowest and the steepest
slope, the smallest joint and a large joint. For each case they check:

- All four demo sketches are fully constrained with DOF 0, no redundant constraints and no conflicts.
- The bounds of each block, and the volumes: the tail adds its profile less two chamfers and the socket removes its
  profile plus two chamfers.
- A section at mid-length has a vertex at each sharp concave corner and at each chamfer end, and none at the chamfered
  convex corners. Straight edges along the slide direction stand at the same places.
- Every vertex of both blocks is exact. `meshgap` reports the halves touch with no gap. `Document.Verify` reports
  `Sound` with no interference and no diagnostic. `decad.Intersect` of the halves is empty, and `decad.Union` of them
  has exactly the sum of their volumes.
- Each block has one face at each end of the joint: the tail and its host share one end face.
- A block moved along the slide direction by `Length` + 1 clears the other by at least 0.5 mm.
- The container names and membership, and `doc.VerifyContainers()`.
- `cornercheck.Check` with the slide direction as the print axis finds only the sharp concave corners listed in
  `instructions.md`.

They also check `Apply` at a frame turned and moved from the world axes, the refusal of hosts that do not fit and the
rejection of invalid inputs.
