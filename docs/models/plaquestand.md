# Plaque stand

The model in `models/plaquestand` constructs a stand for a flat rectangular plaque from two brackets, a rail and two M2
bolts and nuts, and a reference plaque body posed in its slots.

## Construction and API

`Params` holds `PlaqueWidth`, `PlaqueHeight`, `PlaqueThickness`, `Recline`, `LipHeight` and `SlotClearance`.
`DefaultParams()` returns the three default plaque constants (`DefaultPlaqueWidth`, `DefaultPlaqueHeight`,
`DefaultPlaqueThickness`: 250, 305 and 12.8 mm) and the 15° recline, 10 mm lip and 0.55 mm slot clearance. Those three
constants hold the measured size of the user's plaque.

`Params.Derive()` checks the limits and returns `Dimensions`: the span, back rest height, slot width, seat, depth, height,
play angle, the three tip angles, the channel's walls, floor and centre, the two bolt `Sites` and the `Outline` of 26
segments. `Params.Validate()` returns the same error. Every error starts with `plaquestand: `.

`Build(ctx, doc, params)` returns a `Model` with the two `Brackets`, the `Rail`, the `Bolts` and `Nuts`
(`models/m2bolt`, `models/m2nut`, placed with `internal/m2fastener`), the `Plaque`, every `Sketches` entry (12) and the
`Dimensions`. It adds eight bodies, each in its container (`Brackets`, `Rail`, `Bolts`, `Nuts`, `Plaque`). A bracket is
one fully constrained profile sketch extruded 20 mm along +X, with both side faces' outlines chamfered by 0.5 mm and
the nut pocket and bolt hole cut. The rail is a rounded rectangle extruded along the span, with a head pocket and hole
cut for each bolt. The plaque is a box from a fully constrained rectangle sketch. A document built with `decad.New()` is
enough, because the build has no Boolean between parts.

The slot takes the whole clearance, because the plaque is a nominal part, and the rail takes a Regular 0.40 mm gap in its
channels. The nut pocket comes from the shared `m2fastener.RoundedHexPocketTool`. The tip angles and the sizing rules are
in `models/plaquestand/instructions.md`.

## Build and images

```sh
go test ./models/plaquestand
go -C tools/render run . -model plaquestand -shot readme -out ../../models/plaquestand/images
```

The readme shot writes `hero.png`, `index.png`, `frame.png`, `joint.png`, `side.png`, `back.png`, `exploded.png`,
`print.png`, `sizes.png`, `parts/bracket.png`, `parts/rail.png`, `parts/bolt.png`, `parts/nut.png` and
`parts/plaque.png`. `-shot` also takes `hero`, `frame`, `joint`, `side`, `back`, `exploded`, `print`, `sizes`,
`part-bracket`, `part-rail`, `part-bolt`, `part-nut` or `part-plaque` alone.

## Verification

- `params_test.go` checks the defaults, the six pinned cases' derived values and the channel and bolt sites, and that each
  of the 12 out-of-range values and a NaN are rejected with the field name.
- `outline_test.go` checks the 26 segments in every case: line and arc alternation, joined ends, tangent joins, the
  convex and concave arcs, the area and the length.
- `plaquestand_test.go` builds every case: eight solid bodies in their containers, 12 fully constrained sketches, each
  bracket's and the rail's bounds and volume, sections in the middle and near each bracket face, verified meshes and the
  print fit. It also checks that invalid inputs, a cancelled context and nil arguments build nothing, and
  `TestPlaqueStandProfileConditioning` keeps the profile sketch's conditioning above 6e-5 over the 729-point grid of
  the limits (it is not on the PR list).
- `fit_test.go` checks the Loose clearance between the plaque and every other body and the Regular fit of the rail in the
  channels on tessellated bodies, and reads the front, back and side tip angles off the built bodies.
- `fasteners_test.go` checks each bolt's head, tip and nut against its pockets, that no fastener overlaps the part it
  passes through, and the wall left round each pocket.
- `corners_test.go` runs `internal/cornercheck` on each bracket with the print axis +X and on the rail with +Z.
