# One-way clutch

The model in `models/onewayclutch` constructs a printed one-way clutch for a D-flat shaft: a hub with three flexing
pawls, a ring with 24 internal teeth, a back plate and a cover, and three M2 x 16 bolts with their nuts. The shaft is
not modeled.

## Construction and API

`Params` holds the shaft preset's name, the freewheel `Direction`, the bore clearance and an optional `Placement`
frame. `DefaultParams()` returns the `605ZZ` preset with `Clockwise` freewheel and `ballbearing.PressFitClearance`.
`Shafts()` returns the three presets and `LookupShaft(name)` finds one. `Params.Derive()` returns the first failure in
the order `instructions.md` lists, or `Dimensions`: the shaft, the direction's sign, the D bore's radius and flat, and
the pawl values of the counter-clockwise layout. The fixed dimensions are exported constants.

Every hub and ring sketch is drawn in the counter-clockwise layout and mirrored in the plane x = 0 for `Clockwise`,
through `Direction.Sign()`. The plates, bolts and nuts do not depend on the direction.

`BuildParts(ctx, doc, params)` adds the hub, the ring, the back plate and the cover and returns them with every sketch,
in no container. `Build(ctx, doc, params)` adds those four, the three bolts and the three nuts in the `Hub`, `Ring`,
`Plates`, `Bolts` and `Nuts` containers. A rejected build adds no body.

## Build and images

```sh
go test ./models/onewayclutch
go -C tools/render run . -model onewayclutch -shot readme -out ../../models/onewayclutch/images
```

The readme shot builds the default clutch, its counter-clockwise mirror and the other two shaft presets once each. It
writes `hero.png`, `index.png`, `front.png`, `back.png`, `side.png`, `section.png`, `cutaway.png`, `exploded.png`,
`directions.png`, `sizes.png`, and `parts/hub.png`, `ring.png`, `back-plate.png`, `cover.png`, `bolt.png` and `nut.png`.
The presets `onewayclutch/hero`, `front`, `back`, `side`, `section`, `cutaway`, `exploded`, `directions` and `sizes`
render one view each.

## Verification

Tests build the three shaft presets in both directions, a 0.35 mm bore clearance on `605ZZ`, and a placed default. For
each they check:

- every sketch's full constraint (`TestOneWayClutchSketches`), the ten bodies, their bounds and `VerifyAll` meshes
  (`TestOneWayClutchDefault`), and the containers (`TestOneWayClutchContainers`);
- the validation errors and that a rejected build adds no body (`TestOneWayClutchRejectsInvalidInputs`);
- that the clockwise hub and ring are the mirror of the counter-clockwise ones, and the plates, bolts and nuts are equal
  (`TestOneWayClutchMirror`);
- that the ring turns freely in the freewheel direction and locks the other way, by turning a copy of the ring
  (`TestOneWayClutchFreewheelDirection`);
- that the placed model equals the local one moved by the frame (`TestOneWayClutchPlacement`);
- the part fits of `instructions.md` and that no two bodies overlap (`TestOneWayClutchFits`);
- the pawl's arm thickness, slot margin, strain and lock contact (`TestOneWayClutchPawls`);
- that each bolt head ends flush in its pocket, each nut sits inside its cavity and no fastener overlaps a part
  (`TestOneWayClutchFasteners`);
- the chamfers of the plates' and the hub's faces (`TestOneWayClutchEdgeFinishing`);
- `cornercheck` on the four printed parts, with the retained sharp corners of `instructions.md` listed
  (`TestOneWayClutchCorners`).

The `readme` shot writes every image the README shows.
