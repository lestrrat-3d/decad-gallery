# M5 bolt

The model in `models/m5bolt` constructs an M5 hex head bolt with a configurable length.

## Construction and API

`Params` exposes `Length` and an optional `Placement` frame. `DefaultParams()` returns length 25. `Length` must be
one of `StandardLengths()`: 16, 20, 25, 30, 35, 40, 45, 50, 60, 70 or 80 mm.
The other dimensions are exported constants taken from ISO 4017 and JIS B 1180, and `AcrossCorners()` gives the
head's corner-to-corner width.
The local frame puts the center of the head's underside at the origin, the head along +Z and the shank along -Z.
The head's corners lie on the X axis.

`Build(ctx, doc, params)` returns a `Model` with `Body`, the hexagonal `Head` sketch and the axial `Shank` sketch.
Both sketches are fully constrained, and Build checks that before it uses them.

The head comes from `internal/hexnut` without a bore, the same construction `models/m5nut` uses. The shank is one
revolve of its section, which reaches 1 mm into the head, and a union joins the two. The bolt has no underhead
fillet.

The thread is not modeled, so the shank is the nominal 5 mm diameter along its whole length.

## Build and images

```sh
go test ./models/m5bolt
go -C tools/render run . -model m5bolt -shot readme -out ../../models/m5bolt/images
```

The readme shot writes `hero.png`, `index.png`, `head.png`, `side.png` and `lengths.png`.
`-shot` also takes `hero`, `head`, `side` or `lengths` alone.

## Verification

Tests check the default bolt's bounds, volume, face kinds, mesh verification and sketches. They also check every
standard length, placement and invalid inputs.
