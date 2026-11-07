# M2 nut

The model in `models/m2nut` constructs an M2 hex nut to ISO 4032 and JIS B 1181.

## Construction and API

`Params` exposes an optional `Placement` frame. `DefaultParams()` returns the local frame: the center of the
lower face at the origin, +Z along the axis, corners on the X axis. The dimensions are exported constants, and
`AcrossCorners()` gives the hexagon's corner-to-corner width. `BoreDiameter` equals `m2bolt.ShankDiameter`.

`Build(ctx, doc, params)` returns a `Model` with `Body` and the three sketches `Hexagon`, `LowerChamfer` and
`UpperChamfer`. Build checks that each is fully constrained before it uses it.

The body is one extrusion of the hexagon with its bore, then one cut of a revolved triangle for each chamfer.
All eleven faces keep exact surfaces. An intersection with a revolved ring would give the same shape but leave
the bore and the chamfers faceted, so the model does not use it.

The thread is not modeled. The standards give the corner width and thickness as limits; the model uses the exact
hexagon corner and the maximum thickness.

The nominal 2 mm nut bore matches the nominal 2 mm bolt shank. This threadless
bolt-and-nut interface is a nominal-only exception and has no modeled clearance.

## Build and images

```sh
go test ./models/m2nut
go -C tools/render run . -model m2nut -shot readme -out ../../models/m2nut/images
go -C tools/render test -run '^TestM2NutRendering$'
```

The readme shot writes `hero.png`, `index.png`, `top.png`, `side.png`, `low.png` and `withbolt.png`.
`-shot` also takes `hero`, `top`, `side`, `low` or `withbolt` alone.

## Verification

Tests check the default nut's bounds, its volume against an integral of the chamfers, its face kinds, mesh
verification and sketches. They also check placement, the bore against the bolt's shank, and invalid inputs.
