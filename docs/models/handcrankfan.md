# Hand-crank fan package

[instructions.md](../../models/handcrankfan/instructions.md) owns the dimensions and the orientation rules.
[steps.md](../../models/handcrankfan/steps.md) maps them to Sketch and Decad operations.

## Using the package

Import `github.com/lestrrat-3d/decad-gallery/models/handcrankfan`.
`Build(ctx, doc, params)` needs a document from `NewDocument`, which carries the chain tolerance the lofted blades
need. It returns an `Assembly` of `45 + 3 x Blades` named solid parts (54 for three blades, 57 for four),
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
fan shaft along -Y, meeting at the apex `(0, 0, 170)`. The crank shaft runs along +X 40 below the drive shaft, and a
pair of spur gears in the tray joins them. Each shaft is 7.5 mm across with three grooves that the gears, the arm and the hub
sit on by three teeth in their bores, and seven bought 608-2RS ball bearings carry the shafts where they cross the
gearbox's walls. The bolts are `models/m2bolt` bodies and the nuts `models/m2nut` bodies: `11 + Blades` of each, with
every head flush in a pocket and every nut in an enclosed cavity. The gearbox is four printed pieces cut at its walls,
so each piece prints with its shaft axes vertical and no bore prints on its side. The model's document has a chain tolerance of 0.1 mm, which the bevel gears need for their bores.

A build takes about a minute and one gigabyte; the bevel pair is most of it. The full check of the default fan takes
about nine minutes (538 s at the last full run) and 1.5 GB, because the fastener and fit subtests add sections and
contact queries. These checks skip themselves unless `HANDCRANKFAN_FULL=1` is set, and CI does not run them. Run them with:

```sh
HANDCRANKFAN_FULL=1 go test ./models/handcrankfan -timeout 30m
```

The job `fan-motion` runs `TestHandCrankFanMotion` in `tools/render`. The full checks include the fastener, fit and spur
mesh checks, which run for the default parameters only. The bevel mesh has no proven clearance yet; see the open
non-compliance in [instructions.md](../../models/handcrankfan/instructions.md).

Render the images with `go -C tools/render run . -model handcrankfan -shot readme -out ../../models/handcrankfan/images`.
