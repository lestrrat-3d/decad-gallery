# Dovetail joint

The feature in `features/dovetail` adds a straight sliding dovetail tail to one host body and cuts the matching socket
into another. It makes no part of its own.

## Construction and API

`Params` holds `TailWidth`, `Angle`, `Length`, `Depth` and `TailOn`. `DefaultParams()` returns a 10 mm tail width,
10 degrees, 16 mm length, 6 mm depth and `TailOnFirst`. `Params.TipWidth()` returns the tail's wide-end width,
`TailWidth + 2 Depth tan(Angle)`. `Params.Validate()` rejects an angle outside 5 to 20 degrees, a tail width under
3 mm, a depth under 3 mm, a length under 5 mm and an unknown `TailOn`.

`Apply(ctx, doc, p, f, first, second)` joins two host bodies at the frame `f`. The frame's origin is the centre of the
joint's start on the mating plane, U runs across the joint, V runs along the slide direction and N = U x V points from
the first host into the second. The joint runs from V = 0 to V = `Length`. `Apply` refuses hosts that do not span that
length, do not cover the joint's width plus 0.8 mm each side, or are too shallow, and says which. It retires both hosts,
as `decad.Union` and `decad.Cut` do, and returns a `Joint` with the finished `First` and `Second`, the two profile
sketches in `TailSketch` and `SocketSketch`, `Sketches()`, `TailBody(p)` and `SocketBody(p)`.

Both profile sketches come from one function and are verified fully constrained before they are extruded. The halves
are built at exact nominal size with no clearance. The sketch, the tool bodies and the corner rules are in
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

The tests run the demo for six cases: the defaults, the tail on the second block, both ends of the angle range, the
smallest joint and a large joint. For each case they check:

- All four demo sketches are fully constrained with DOF 0, no redundant constraints and no conflicts.
- The bounds of each block, and that the tail adds exactly the volume the socket removes.
- A section at mid-length has a vertex at each of the profile's four corners on both blocks.
- `meshgap` reports the halves touch with no gap, and decad's `LocatePoint` finds no point inside both blocks.
- A block moved along the slide direction by `Length` + 1 clears the other by at least 0.5 mm.
- The container names and membership, and `doc.VerifyContainers()`.
- `cornercheck.Check` with the slide direction as the print axis finds only the mating corners listed in
  `instructions.md`.

They also check `Apply` at a frame turned and moved from the world axes, the refusal of hosts that do not fit and the
rejection of invalid inputs.

`Document.Verify`, `decad.Union` and `decad.Intersect` refuse the finished halves in exact contact in most cases, so
the no-overlap check uses point classification and volumes instead.
