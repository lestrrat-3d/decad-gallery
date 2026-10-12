# Herringbone gear

![A bored herringbone gear with opposing tooth slopes](images/hero.png)

`Build(ctx, doc, Params)` lofts an involute tooth to a twisted section at the
body midpoint, mirrors that half, and joins the two halves. It patterns the
chevron tooth around a full-thickness root disc, then applies the inherited
helical root fillet and optional spur bore and outer-cap chamfer.

The [instructions](instructions.md) define the source parameters and operation
order. The [steps](steps.md) map them to Sketch and Decad.

The real eight-tooth, 2 mm thick, +14.5° gear and the source-default 17-tooth
gear pass `Tessellate(0.1 mm, VerifyAll)` with one source-order variadic `Union`.
The real eight-tooth gear with a 1 mm bore and the default 17-tooth gear with a
3 mm bore also pass `VerifyAll`; both meshes have one through-hole. The hero
renders the default 17-tooth gear with a 3 mm bore and no cap chamfer. Decad's
approved outer-cap Chamfer repair is still under review.

The mirrored tooth alone passes `VerifyAll` for signed 14.5°, 35°, and 90°
angles. Its full-height bounds and midpoint centroid check the chevron symmetry.
