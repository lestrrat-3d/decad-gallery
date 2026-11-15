# Crank handle

The model in `models/crankhandle` constructs a printed hand crank for a D-flat shaft: an arm that pinches the shaft, a
pin, a handle that turns on the pin, the M2 x 10 bolt and nut that clamp the pin in the arm, the two M2 x 6 cross bolts
and nuts that clamp the pin's two printed halves together, and the M2 x 16 clamp bolt and nut that close the arm's slit
on the shaft. `models/handcranksiren` builds its crank from it.

## Construction and API

`Params` holds the shaft diameter and flat depth, the bore clearance, the throw and an optional `Placement` frame. The
arm, pin and handle keep fixed sizes, exported as constants with the handle's meridian `HandleProfile` and the cross
bolts' positions `CrossZ`. `Params605ZZ()` and `Params6082RS()` read the shaft diameter from
`ballbearing.Params605ZZ()` and `ballbearing.Params6082RS()`, and the bore clearance from
`ballbearing.PressFitClearance`. `DefaultParams()` returns `Params605ZZ()`. `Params6082RS()` is the siren's crank.
`Params.Derive()` returns the first failure in the order `instructions.md` lists, or `Dimensions`: the bore's radius,
flat and outline length, the pin bolt's `m2fastener.Site`, the two cross bolts' sites in `CrossSites`, and the clamp's
layout: `ClampY`, `SideEnd`, `Reach` and `ClampSite`, with `CornerCenter()` and `EndCenter()` for its arcs.

Build in a document from `NewDocument()`, which carries a chain tolerance of 0.01 mm. The pin's cuts after its union
refuse a plain `decad.New()` document with a chain-depth error. `BuildParts(ctx, doc, params)` adds the arm, the pin
and the handle and returns them with every sketch, the pin bolt's and the cross bolts' sites in the world frame, and
`SplitNormal`, the world normal of the plane that splits the pin into its printed halves, and `ClampSite`. It files
them in no container, so an assembly files them with its own parts. `Build(ctx, doc, params)` adds those three, the
four bolts and the four nuts, in the `Arm`, `Pin`, `Handle`, `Bolts` and `Nuts` containers. A rejected build adds no
body.

The arm's outline runs from an arc round the pin down two straight sides, past the shaft, into two 4.5 mm corners and
one 22 mm end arc, every join tangent. It is extruded and both faces' outlines are chamfered; the chamfer needs decad
902e347 or later, which admits the outline's solver-placed tangent arcs in the Booleans that follow. Then the arm is
cut by the slit, the blind D bore, the hexagonal pocket for the pin's stub, the pin bolt's head pocket, and the clamp
bolt's square-topped nut cavity, head pocket and blind hole, with the cavity's faces and the hole's end rounded on
their tools. The pin joins a hexagonal stub to a revolved shank and head, then
takes the double flats, the pin bolt's hole and nut cavity with rounded ends (`m2fastener.RoundedHoleTool` and
`RoundedCavityTool`), and each cross bolt's hole, head pocket and nut cavity. The handle is a revolved meridian.

## Build and images

```sh
go test ./models/crankhandle
go -C tools/render run . -model crankhandle -shot readme -out ../../models/crankhandle/images
```

The readme shot writes `hero.png`, `index.png`, `side.png`, `back.png`, `cutaway.png`, `exploded.png`, `split.png`,
`print.png`, `sizes.png`, `clamp.png`, `clamp-nut.png`, and `parts/arm.png`, `pin.png`, `pin-half-heads.png`,
`pin-half-nuts.png`, `handle.png`, `bolt.png`, `cross-bolt.png`, `clamp-bolt.png` and `nut.png`. `clamp` and
`clamp-nut` cut the arm's hub along and across the clamp bolt; they are written only by the readme shot. The presets `crankhandle/hero`, `side`, `back`, `cutaway`, `exploded`,
`split`, `print` and `sizes` render one view each. `split` and `print` clip the real pin mesh by the seam with
`solidlens.ClipMesh`; the model keeps the pin whole.

## Verification

Tests build four cases: both presets, a 0.35 mm bore clearance, and an 80 mm throw. For each they check:

- every sketch's full constraint, the eleven bodies and their containers;
- `cornercheck` on the arm and the handle (axis +Z, with the hexagonal pocket's corners and the clamp head pocket's
  floor rim listed, and the slit's end round in the 80 mm throw) and on the pin with the split normal as its print
  axis (the pin bolt's nut cavity rounds listed);
- the bore: every point lies `BoreClearance` outside the shaft's D section, it stops at its floor apart from the slit's
  round end, and the wall round it is at least 1.6 mm, or 1.05 mm over the clamp bolt's hole;
- the slit: one section loop runs from the bore down the slit's walls, 1 mm apart, to the end arc, and the slit ends
  in the floor 1 mm past the bore's wall;
- the clamp bolt: an M2 x 16 with its head flush with the +X side face and inside its pocket, crossing the slit, its
  hole ending 1 mm inside the -X face, its nut inside its cavity, and no overlap with the arm in sections along and
  across it; and every wall round the clamp's cuts at least 1 mm, also for a 16.6 mm shaft;
- the face chamfers: 0.25 mm inside each face the section stands 0.25 mm in from the pin end, the end arc and both
  sides;
- the fits: the stub in its pocket (Tight), the handle round the shank (Regular), and the handle's gaps to the pin and
  the arm in a cut through the pin axis (Regular);
- the pin bolt: its head flush with the arm's back face, its tip inside the pin, the nut inside its cavity;
- each cross bolt: its head flush with the +Y flat and inside its pocket, its tip at the -Y flat, its nut inside its
  cavity in the -Y half, the bolt crossing the seam, no overlap with the pin in cuts parallel to the seam, and 1 mm or
  more of pin round every pocket and cavity;
- `splitcheck` on the pin with the split normal and both cross bolts' lines exempt: no violation, bridges no wider
  than 6 mm, and each half as high as the head's radius.

The default's bounds and `VerifyAll` meshes, the placed crank and the validation errors are checked too. The siren's
tests check its crank through the assembly.

## Known decad limit

The 608-2RS bore's rim in the arm's back face is not chamfered: Decad refuses the chamfer. `steps.md` quotes the error.
