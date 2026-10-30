# Ball-bearing mounting envelope

The model in `models/ballbearing` constructs the nominal annular envelope of a purchased bearing.
It is intended for positioning the bearing in a consuming assembly and checking shaft and housing space.

## Construction and API

`DefaultParams()` returns the 608-2RS bore, outside diameter and width as 8, 22 and 7 mm.
These dimensions are listed in [JTEKT's 608 2RS catalogue](https://www.jtekt.co.jp/products/bearing/en/p/detail/?pno=608+2RS).
`Params` exposes `BoreDiameter`, `OuterDiameter`, `Width` and an optional `Placement` frame.
The placement origin is the center of the lower axial face; its normal gives the bearing axis.

`Params605ZZ()` returns a 5 mm bore, 14 mm outside diameter and 5 mm width from
[EZO's 605ZZ catalogue](https://www.ezo-brg.co.jp/english/product/spec.php?eid=00529).
Both presets use the same `Build` entry point, and callers supply placement independently.

`Build(ctx, doc, params)` returns a `Model` with `Body` and the actual `Sketch` consumed by extrusion.
The sketch remains in its local XY plane when the body is placed. It contains two origin-centered
diameter-constrained circles, and Build verifies its full constraint before using the annular profile.
The body spans 0 to Width along local +Z, and Build leaves one final body in the supplied document.

The model does not change the nominal dimensions for a shaft fit or housing fit. Those allowances belong
to the consuming assembly and depend on its materials, machining process and loading.
The annulus includes the volume occupied by rings, balls, cage and closures; no internal geometry is asserted.
Its volume is therefore the envelope volume rather than the steel volume of a physical bearing.
Do not use that volume to calculate bearing mass or inertia.

## Build and images

Run the focused geometry checks from the repository root:

```sh
go test ./models/ballbearing
```

Generate the hero and index from the real Decad envelope:

```sh
go -C tools/render run . -model ballbearing -shot readme -out ../../models/ballbearing/images
```

The hero preset uses 1600 × 900 pixels; the index uses 480 × 270 pixels.
The readme shot also writes `605zz.png` at 1600 × 900; `-shot 605zz` renders it separately.
Run the renderer integration check with:

```sh
go -C tools/render test -run '^TestBearing(608|605)Rendering$'
```

## Verification

Tests build the real 608-2RS envelope and check its nominal bounds, annular volume, through-bore,
analytic faces and fully constrained sketch. Tessellation uses VerifyAll.
Other cases check placed geometry, custom dimensions, repeated builds and invalid inputs.
Reuse through a consuming assembly must be checked there before claiming that its bearing mounting works.
