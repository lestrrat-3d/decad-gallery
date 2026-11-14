# Crank handle

The model in `models/crankhandle` constructs a printed hand crank for a D-flat shaft: an arm, a pin, a handle that turns
on the pin, the M2 x 10 bolt and nut that clamp the pin in the arm, and the two M2 x 6 cross bolts and nuts that clamp
the pin's two printed halves together. `models/handcranksiren` builds its crank from it.

## Construction and API

`Params` holds the shaft diameter and flat depth, the bore clearance, the throw and an optional `Placement` frame. The
arm, pin and handle keep fixed sizes, exported as constants with the handle's meridian `HandleProfile` and the cross
bolts' positions `CrossZ`. `Params605ZZ()` and `Params6082RS()` read the shaft diameter from
`ballbearing.Params605ZZ()` and `ballbearing.Params6082RS()`, and the bore clearance from
`ballbearing.PressFitClearance`. `DefaultParams()` returns `Params605ZZ()`. `Params6082RS()` is the siren's crank.
`Params.Derive()` returns the first failure in the order `instructions.md` lists, or `Dimensions`: the bore's radius,
flat and outline length, the pin bolt's `m2fastener.Site`, and the two cross bolts' sites in `CrossSites`.

Build in a document from `NewDocument()`, which carries a chain tolerance of 0.01 mm. The pin's cuts after its union
refuse a plain `decad.New()` document with a chain-depth error. `BuildParts(ctx, doc, params)` adds the arm, the pin
and the handle and returns them with every sketch, the pin bolt's and the cross bolts' sites in the world frame, and
`SplitNormal`, the world normal of the plane that splits the pin into its printed halves. It files them in no
container, so an assembly files them with its own parts. `Build(ctx, doc, params)` adds those three, the three bolts
and the three nuts, in the `Arm`, `Pin`, `Handle`, `Bolts` and `Nuts` containers. A rejected build adds no body.

The arm is an extruded stadium with both faces' outlines chamfered, then cut by a blind D bore, the hexagonal pocket
for the pin's stub and the pin bolt's head pocket. The pin joins a hexagonal stub to a revolved shank and head, then
takes the double flats, the pin bolt's hole and nut cavity with rounded ends (`m2fastener.RoundedHoleTool` and
`RoundedCavityTool`), and each cross bolt's hole, head pocket and nut cavity. The handle is a revolved meridian.

## Build and images

```sh
go test ./models/crankhandle
go -C tools/render run . -model crankhandle -shot readme -out ../../models/crankhandle/images
```

The readme shot writes `hero.png`, `index.png`, `side.png`, `back.png`, `cutaway.png`, `exploded.png`, `split.png`,
`print.png`, `sizes.png`, and `parts/arm.png`, `pin.png`, `pin-half-heads.png`, `pin-half-nuts.png`, `handle.png`,
`bolt.png`, `cross-bolt.png` and `nut.png`. The presets `crankhandle/hero`, `side`, `back`, `cutaway`, `exploded`,
`split`, `print` and `sizes` render one view each. `split` and `print` clip the real pin mesh by the seam with
`solidlens.ClipMesh`; the model keeps the pin whole.

## Verification

Tests build four cases: both presets, a 0.35 mm bore clearance, and an 80 mm throw. For each they check:

- every sketch's full constraint, the nine bodies and their containers;
- `cornercheck` on the arm and the handle (axis +Z, the hexagonal pocket's corners listed) and on the pin with the
  split normal as its print axis (the pin bolt's nut cavity rounds listed);
- the bore: every point lies `BoreClearance` outside the shaft's D section, it stops at its floor, and the wall round it
  is at least 1.6 mm;
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
