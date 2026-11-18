# M5 nut

The model in `models/m5nut` constructs an M5 hex nut to ISO 4032 and JIS B 1181.

## Construction and API

`Params` exposes an optional `Placement` frame. `DefaultParams()` returns the local frame: the center of the
lower face at the origin, +Z along the axis, corners on the X axis. The dimensions are exported constants, and
`AcrossCorners()` gives the hexagon's corner-to-corner width. `BoreDiameter` equals `m5bolt.ShankDiameter`.

`Build(ctx, doc, params)` returns a `Model` with `Body` and the three sketches `Hexagon`, `LowerChamfer` and
`UpperChamfer`. Build checks that each is fully constrained before it uses it.

The body comes from `internal/hexnut`, the construction `models/m2nut` also uses: one extrusion of the hexagon
with its bore, then one cut of a revolved triangle for each chamfer. All eleven faces keep exact surfaces.

The thread is not modeled. The standards give the corner width and thickness as limits; the model uses the exact
hexagon corner and the maximum thickness.

The nominal 5 mm nut bore matches the nominal 5 mm bolt shank. This threadless
bolt-and-nut interface is a nominal-only exception and has no modeled clearance.

## Build and images

```sh
go test ./models/m5nut
go -C tools/render run . -model m5nut -shot readme -out ../../models/m5nut/images
```

The readme shot writes `hero.png`, `index.png`, `top.png`, `side.png` and `withbolt.png`.
`-shot` also takes `hero`, `top`, `side` or `withbolt` alone.

## Verification

Tests check the default nut's bounds, its volume against an integral of the chamfers, its face kinds, mesh
verification and sketches. They also check placement, the bore against the bolt's shank, and invalid inputs.
