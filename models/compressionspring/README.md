# Compression spring

**Build status:** Not built yet.

![Lee LC 024C 14 M spring installed at 43.80 mm](images/hero.png)

Choose a purchased round-wire spring and set the distance between its seats. The model keeps the catalog wire diameter,
outside diameter and turn count while the pitch changes with compression.

| Part | Material | Outside diameter | Free length | Rate |
| --- | --- | ---: | ---: | ---: |
| Lee Spring LC 024C 14 M | Music wire | 6.10 mm | 50.80 mm | 0.49 N/mm |
| Lee Spring LC 026D 16 S | Stainless steel | 7.62 mm | 63.50 mm | 0.30 N/mm |
| Lee Spring LC 022D 03 S | Stainless steel | 7.62 mm | 15.88 mm | 0.736 N/mm |
| Lesjofors Cat.no 2353 (metric) | Stainless steel | 7.45 mm (computed) | 15 mm | 0.86 N/mm |
| KSSC K series 1087 (MonotaRO 01353223) | Stainless steel (SUS304WPB) | 7.0 mm | 14.0 mm | 0.9408 N/mm |

Lesjofors Cat.no 2353 comes from the Lesjofors Stock Spring Catalogue 2025, page 30. Its outside diameter is the
catalogue's mean diameter plus the wire diameter. Check that your country's Lesjofors shop or distributor sells it.

KSSC K series 1087 is sold in Japan by [MonotaRO](https://www.monotaro.com/p/0135/3223/) under order code 01353223.
Its values come from the page's spec table. Its ends are not ground. The page gives an allowable load of 7.15 N at
6.4 mm length and does not publish fatigue life.

**Clearance adjustment:** Nominal. Size the mating guides around the purchased spring.

![Side view of the music-wire spring](images/side.png)

The stainless button-return spring shortens from 55.50 mm at rest to 18.94 mm at full press. It still has 6.77 mm
of travel before reaching its catalog solid height. The modeled wire follows a constant-pitch helix throughout.
The purchased springs have squared, ground ends, so their seat contact and end-coil spacing differ.

![Stainless button-return spring at full press](images/pressed.png)

See the [construction, API and checks](../../docs/models/compressionspring.md).
