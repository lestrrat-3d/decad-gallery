# M2 bolt

The model in `models/m2bolt` constructs an M2 socket head cap screw with a configurable length.

## Construction and API

`Params` exposes `Length` and an optional `Placement` frame. `DefaultParams()` returns length 10. `Length` must be one of `StandardLengths()`: 6, 10, 16, 18, 20 or 25 mm.
The other dimensions are exported constants taken from ISO 4762 and JIS B 1176.
The local frame puts the center of the head's underside at the origin, the head along +Z and the shank along -Z.

`Build(ctx, doc, params)` returns a `Model` with `Body`, the axial `Profile` sketch and the hexagonal
`Socket` sketch. Both sketches are fully constrained, and Build checks that before it uses them.

The body is one revolve of the section, a fillet under the head and one hexagonal cut. The section also
holds the round pilot well and drill-point cone under the socket, which the hexagonal cut widens.
Decad's boolean keeps the drill-point cone as a faceted face after that cut. The other faces keep their
exact surfaces.

The thread is not modeled, so the shank is the nominal 2 mm diameter along its whole length.

## Build and images

```sh
go test ./models/m2bolt
go -C tools/render run . -model m2bolt -shot readme -out ../../models/m2bolt/images
```

The readme shot writes `hero.png`, `index.png`, `head.png`, `side.png`, `underside.png` and `lengths.png`.
`-shot` also takes `hero`, `head`, `side`, `underside` or `lengths` alone.

## Verification

Tests check the default bolt's bounds, volume (against an integral of the section), face kinds, mesh
verification and sketches. They also check every standard length, placement and invalid inputs.
