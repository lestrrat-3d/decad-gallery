# Pin ring

The model in `models/pinring` constructs a plain flat ring, an assembly aid (jig) that holds a pin in place while a mechanism is assembled.
It is not part of the finished assembly.

## Construction and API

`Params` holds `RodRadius`, `Clearance`, `Width`, `Thickness` and an optional `Placement` frame. `DefaultParams()`
returns the 2.5 mm rod radius, 0.5 mm clearance, 5 mm width and 5 mm thickness, which give a 3 mm inner radius
and an 8 mm outer radius. The ring axis is Z and the ring spans z from 0 to `Thickness`.

`Build(ctx, doc, params)` returns a `Model` with `Body` and the sketch `Profile`. Build checks that the sketch is
fully constrained before it extrudes it. The body has four exact faces: two planes and two cylinders.

The bore takes the whole clearance because the ring is a custom part that mates with the rod. No face is large
enough for the edge chamfer rule: the flat faces are 172.8 mm², below 400 mm².

## Build and images

```sh
go test ./models/pinring
go -C tools/render run . -model pinring -shot readme -out ../../models/pinring/images
```

The readme shot writes `hero.png`, `index.png`, `top.png`, `side.png` and `low.png`.
`-shot` also takes `hero`, `top`, `side` or `low` alone.

## Verification

Tests check the bounds, volume, inner and outer radius, thickness, face kinds, mesh verification and the sketch
for the default and two other parameter branches. They also check placement and invalid inputs.
