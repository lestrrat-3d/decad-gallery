# Hand-crank fan package

[instructions.md](../../models/handcrankfan/instructions.md) owns the dimensions and the orientation rules.
[steps.md](../../models/handcrankfan/steps.md) maps them to Sketch and Decad operations.

## Using the package

Import `github.com/lestrrat-3d/decad-gallery/models/handcrankfan`.
`Build(ctx, doc, params)` needs a document from `NewDocument`, which carries the chain tolerance the lofted blades
need. It returns an `Assembly` of `17 + 2 x Blades` named solid parts (23 for three blades, 25 for four),
every Sketch it used, and the `State`.
`Motion(params)` computes the pose without building bodies.
`Assembly.Set(doc)` returns the build as a `parts.Set` with its materials and joints, and `Assembly.Explode(ctx, set)`
pulls it apart with `Set.Explode`. They need Decad's `parts` package, which the repository's Decad version provides.

| Parameter | Default | Accepted values |
| --- | --- | --- |
| CrankAngle | 5π/4 | Any finite radians; forward is clockwise seen from the right |
| BladePitch | 25° | 10° through 40°, the blade angle at 75 % of the radius; it turns each whole blade |
| Blades | 4 | 3 or 4 |

X runs to the operator's right, Y away from the operator and Z up. The drive shaft runs along +X and the
fan shaft along -Y, meeting at the apex `(0, 0, 170)`. The crank shaft runs along +X 40.4 below the drive shaft, and a
pair of spur gears in the tray joins them.

A build takes about a minute and one gigabyte; the bevel pair is most of it. Run the checks with:

```sh
go test ./models/handcrankfan -timeout 15m
```

Render the images with `go -C tools/render run . -model handcrankfan -shot readme -out ../../models/handcrankfan/images`.
