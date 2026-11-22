# Compression spring

Use `models/compressionspring` for a nominal round-wire spring at a known installed length. Each build returns one solid.
The default is a Lee LC 024C 14 M detent spring installed at 43.80 mm.

## Construction and API

`LeeLC024C14M()`, `LeeLC026D16S()` and `LeeLC022D03S()` return independent catalog specifications.
Pass a `Params{Spec: spec, InstalledLength: length}` to `Build(ctx, doc, params)`. Lengths are in millimeters.
The model places the first wire-section center in the XZ plane on the positive X side and winds around the +Y axis.
The first end extends half a wire diameter below Y = 0. Use a rigid placement to position the body in an assembly.

`Model.Profile` is the fully constrained circular wire sketch. `Model.Body` is the Decad coil.
`Model.Pitch` gives the millimeters per turn. `Model.Force` gives the nominal force in newtons, calculated
from the catalog rate and the difference between free and installed length. The calculation does not establish
fatigue life. The constant-pitch ends approximate the squared and ground ends of the purchased springs.

| Part | Wire diameter | Total turns | Active turns | Solid height | Approximate load at solid |
| --- | ---: | ---: | ---: | ---: | ---: |
| LC 024C 14 M | 0.61 mm | 19.64 | 17.64 | 12.32 mm | 19.13 N |
| LC 026D 16 S | 0.66 mm | 18.33 | 16.33 | 12.17 mm | 15.93 N |
| LC 022D 03 S | 0.56 mm | 5.25 | 3.25 | 3.10 mm | 9.27 N |

The catalog pages specify the variants. [Music-wire detent spring](https://www.leespring.com/product/compression-spring-lc024c14m-music-wire)
[stainless return spring](https://www.leespring.com/product/compression-spring-lc026d16s-stainless-steel) and
[stainless click-indexer spring](https://www.leespring.com/product/compression-spring-lc022d03s-stainless-steel).

## Build and checks

```sh
go test ./models/compressionspring
go -C tools/render run . -model compressionspring -shot readme -out ../../models/compressionspring/images
```

The geometry tests check every catalog variant, their winding dimensions and installed forces, the fully constrained
wire profiles and the verified boundary and volume of the tessellated solids.
