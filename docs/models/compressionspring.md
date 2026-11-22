# Compression spring

Use `models/compressionspring` for a nominal round-wire spring at a known installed length. Each build returns one solid.
The default is a Lee LC 024C 14 M detent spring installed at 43.80 mm.

## Construction and API

`LeeLC024C14M()`, `LeeLC026D16S()`, `LeeLC022D03S()`, `LesjoforsStock2353()` and `KSSC1087()` return independent catalog specifications.
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
| Lesjofors Cat.no 2353 | 0.60 mm | 6.3 | not published | 3.78 mm (computed) | 9.65 N (computed) |
| KSSC 1087 | 0.60 mm | 6.5 | not published | 4.5 mm | 8.94 N (computed) |

`Spec.ActiveTurns` is optional. Zero means the supplier does not publish it. `Build` returns an error for a negative or
non-finite value, or one above `TotalTurns`. For Lesjofors Cat.no 2353 the outside diameter 7.45 mm is the catalogue's
mean diameter 6.85 mm plus the wire diameter. Its solid height is the wire times the total turns, and its load at solid is
0.86 N/mm x (15 mm - 3.78 mm).
For KSSC 1087 the load at solid is 0.9408 N/mm x (14.0 mm - 4.5 mm). The supplier's allowable load is 7.15 N at
6.4 mm length, and fatigue life is not published.

The catalog pages specify the variants. [Music-wire detent spring](https://www.leespring.com/product/compression-spring-lc024c14m-music-wire)
[stainless return spring](https://www.leespring.com/product/compression-spring-lc026d16s-stainless-steel),
[stainless click-indexer spring](https://www.leespring.com/product/compression-spring-lc022d03s-stainless-steel) and
the metric [Lesjofors stock spring Cat.no 2353](https://www.lesjofors.com/siteassets/corporate-page-blocks-media-and-files/products/products-categories/stock-springs/stock_spring-catalog_2025_stock-spring-catalog.pdf)
(Stock Spring Catalogue 2025, page 30). The Japanese
[KSSC K series 1087 at MonotaRO](https://www.monotaro.com/p/0135/3223/) (order code 01353223) takes its values from
the page's spec table.

## Build and checks

```sh
go test ./models/compressionspring
go -C tools/render run . -model compressionspring -shot readme -out ../../models/compressionspring/images
```

The geometry tests check every catalog variant, their winding dimensions and installed forces, the fully constrained
wire profiles and the verified boundary and volume of the tessellated solids.
