# Plaque stand

The model in `models/plaquestand` constructs a one-piece stand for a flat rectangular plaque, and a reference plaque
body posed in its slot.

## Construction and API

`Params` holds `PlaqueWidth`, `PlaqueHeight`, `PlaqueThickness`, `Recline`, `LipHeight` and `SlotClearance`.
`DefaultParams()` returns the three placeholder plaque constants (`DefaultPlaqueWidth`, `DefaultPlaqueHeight`,
`DefaultPlaqueThickness`: 300, 250 and 6 mm) and the 15° recline, 10 mm lip and 0.55 mm slot clearance. Only the three
placeholder constants change when the real plaque's size is known.

`Params.Derive()` checks the limits and returns `Dimensions`: the stand width, back rest height, slot width, seat, depth,
height, play angle, the three tip angles and the `Outline` of 18 segments. `Params.Validate()` returns the same error.
Every error starts with `plaquestand: `.

`Build(ctx, doc, params)` returns a `Model` with the `Stand` and `Plaque` bodies, their sketches `Profile` and
`PlaqueSketch`, and the `Dimensions`. It adds exactly two bodies, each in its container (`Stand`, `Plaque`). The stand is
the extrusion of one fully constrained profile sketch along +X, with both side faces' outlines chamfered by 0.5 mm. The
plaque is a box from a fully constrained rectangle sketch. A document built with `decad.New()` is enough, because the
build has no Boolean.

The slot takes the whole clearance, because the plaque is a nominal part. The tip angles and the stand's size rules are in
`models/plaquestand/instructions.md`.

## Build and images

```sh
go test ./models/plaquestand
go -C tools/render run . -model plaquestand -shot readme -out ../../models/plaquestand/images
```

The readme shot writes `hero.png`, `index.png`, `stand.png`, `side.png`, `back.png`, `exploded.png`, `print.png`,
`sizes.png`, `parts/stand.png` and `parts/plaque.png`. `-shot` also takes `hero`, `stand`, `side`, `back`, `exploded`,
`print`, `sizes`, `part-stand` or `part-plaque` alone.

## Verification

- `params_test.go` checks the defaults, the six pinned cases' derived values, and that each of the 12 out-of-range values
  and a NaN are rejected with the field name.
- `outline_test.go` checks the 18 segments in every case: line and arc alternation, joined ends, tangent joins, the
  convex and concave arcs, the area and the length.
- `plaquestand_test.go` builds every case: both bodies solid and in their containers, both sketches fully constrained,
  the bounds, the stand's volume, sections in the middle and near each side face, verified meshes and the print fit. It
  also checks that invalid inputs, a cancelled context and nil arguments build nothing.
- `fit_test.go` checks the Loose clearance between the plaque and the stand on tessellated bodies, and reads the front,
  back and side tip angles off the built bodies.
- `corners_test.go` runs `internal/cornercheck` on the stand with the print axis +X.
