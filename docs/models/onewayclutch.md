# One-way clutch

The model in `models/onewayclutch` constructs a printed one-way clutch for a D-flat shaft: a hub with three flexing
pawls, a 38.2 mm ring with 18 internal teeth and three bolt ears, a back plate and a cover, and three M2 x 16 bolts with their nuts,
each nut in a hex pocket open to the back plate's outer face. Each ear has an M2 bore and a nut pocket open to the ring's
rear face, where the standard adapter ring bolts on. The shaft is not modeled.

## Construction and API

`Params` holds the shaft preset's name, the freewheel `Direction`, the bore clearance and an optional `Placement`
frame. `DefaultParams()` returns the `605ZZ` preset with `Clockwise` freewheel and `ballbearing.PressFitClearance`.
`Shafts()` returns the three presets and `LookupShaft(name)` finds one. `Params.Derive()` returns the first failure in
the order `instructions.md` lists, or `Dimensions`: the shaft, the direction's sign, the D bore's radius and flat, and
the pawl values of the counter-clockwise layout. The fixed dimensions are exported constants.

Every hub and ring sketch is drawn in the counter-clockwise layout and mirrored in the plane x = 0 for `Clockwise`,
through `Direction.Sign()`. The plates, bolts and nuts do not depend on the direction.

Build the clutch in a document from `NewDocument()`, which carries a chain tolerance of 0.02 mm. The ring's mount nut
pockets meet held facets just past a plain document's gate, so a document from `decad.New()` makes `BuildParts` and
`Build` fail with `decad.ErrUnsupported`, as `crankhandle` documents for its own pin.

`BuildParts(ctx, doc, params)` adds the hub, the ring, the back plate and the cover and returns them with every sketch,
in no container. `Build(ctx, doc, params)` adds those four, the three bolts and the three nuts in the `Hub`, `Ring`,
`Plates`, `Bolts` and `Nuts` containers: ten bodies, the clutch alone. A rejected build adds no body.

`BuildAdapter(ctx, doc, params)` adds the standard adapter ring, three M2 x 25 bolts and three nuts that bolt it to the
ring's ears from the front, in the `Adapter`, `Bolts` and `Nuts` containers, and returns them as an `Adapter`. Call it
with the same `Params` as the clutch, in the same document (`Build` first: 17 bodies in six containers). It needs a
chain tolerance only when the document also holds the clutch. A part of another shape takes its place when it meets the ear
interface of `Dimensions.MountSites`. The adapter's face holes (`Dimensions.FaceSites`) take a part bolted to its front: the
turbine fan's hub uses two of them, with its own document (both models file their hub in a container named `Hub`).

## Build and images

```sh
go test ./models/onewayclutch
go -C tools/render run . -model onewayclutch -shot readme -out ../../models/onewayclutch/images
```

The readme shot builds the default clutch, the default clutch with its adapter, its counter-clockwise mirror and the other
two shaft presets once each. It writes `hero.png`, `index.png`, `front.png`, `back.png`, `side.png`, `section.png`,
`cutaway.png`, `exploded.png`, `adapter.png`, `adapter-exploded.png`, `directions.png`, `sizes.png`, and `parts/hub.png`,
`ring.png`, `back-plate.png`, `cover.png`, `bolt.png`, `nut.png`, `adapter.png` and `mount-bolt.png`. The presets
`onewayclutch/hero`, `front`, `back`, `side`, `section`, `cutaway`, `exploded`, `adapter`, `adapter-exploded`,
`directions` and `sizes` render one view each.

## Verification

Tests build the three shaft presets in both directions, a 0.35 mm bore clearance on `605ZZ`, and a placed default. For
each they check:

- every sketch's full constraint (`TestOneWayClutchSketches`), the ten bodies, their bounds and `VerifyAll` meshes
  (`TestOneWayClutchDefault`), and the containers (`TestOneWayClutchContainers`);
- the adapter: 17 bodies in six containers with the clutch, its sketches, bounds and chamfers, each head flush, each nut
  inside its pocket, each bolt tip, the clearances round every shank, the seam, the walls, the face interface (face pockets,
  face holes and access holes) and its corners
  (`TestOneWayClutchAdapter`, `TestOneWayClutchAdapterPlaced`);
- that a plain document refuses the clutch build (`TestOneWayClutchPlainDocumentRefusesBuild`);
- the validation errors and that a rejected build adds no body (`TestOneWayClutchRejectsInvalidInputs`);
- that the clockwise hub and ring are the mirror of the counter-clockwise ones, and the plates, bolts and nuts are equal
  (`TestOneWayClutchMirror`);
- that the ring turns freely in the freewheel direction and locks the other way, by turning a copy of the ring
  (`TestOneWayClutchFreewheelDirection`);
- that the placed model equals the local one moved by the frame (`TestOneWayClutchPlacement`);
- the part fits of `instructions.md` and that no two bodies overlap (`TestOneWayClutchFits`);
- that `TestTurbineFanOnClutch` in `models/turbinefan` checks the clutch through its consumer, the clutch-mount fan;
- the pawl's arm thickness, slot margin, strain and lock contact (`TestOneWayClutchPawls`);
- that each bolt head ends flush in its pocket, each tip lies inside the back plate's outer face and ends flush with it once
  the seams close, each nut sits in its hex pocket and no fastener overlaps a part (`TestOneWayClutchFasteners`);
- that the ring stands 1.25 mm past every shaft-side body at both ends and that its axial play stays under that
  standoff by a Regular gap (`TestOneWayClutchEnds`);
- the ring's three ears, their bores and walls, the three mount nut pockets and that the ring's outside is the same in both
  directions (`TestOneWayClutchEars`);
- the chamfers of the plates', the 605ZZ hub's (0.25 mm; the 8 mm hubs stay unchamfered) and the ring's faces (`TestOneWayClutchEdgeFinishing`,
  `TestOneWayClutchRingEdgeFinishing`);
- `cornercheck` on the four printed parts, with the retained sharp corners of `instructions.md` listed and the back plate's and the ring's hex pocket corners exempt
  (`TestOneWayClutchCorners`).

The `readme` shot writes every image the README shows.
