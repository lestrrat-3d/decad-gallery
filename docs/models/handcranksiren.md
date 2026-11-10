# Hand-crank siren package

[instructions.md](../../models/handcranksiren/instructions.md) owns the dimensions and the orientation rules.
[steps.md](../../models/handcranksiren/steps.md) maps them to Sketch and Decad operations.

## Using the package

Import `github.com/lestrrat-3d/decad-gallery/models/handcranksiren`.
`Build(ctx, doc, params)` needs a document from `NewDocument`, which carries a chain tolerance of 0.01 mm that the
rotor's unions and the gears' bores need. It returns an `Assembly` of 73 named solid parts, whatever the slot count,
every Sketch it used, and the `State`. `Motion(params)` computes the pose without building bodies.
`PitchHz(slots, crankRevPerSecond)` returns the tone: `Slots x Ratio x crank turns per second`, 400 Hz for the default.

| Parameter | Default | Accepted values |
| --- | --- | --- |
| CrankAngle | 5π/4 | Any finite radians; forward is clockwise seen from the operator's right |
| Slots | 20 | 12 through 24, the ports in the rotor and the stator and the vanes on the rotor |

The held pose fixes the axes: operator forward is -X (the horn points there), operator right is +Y (the crank side)
and up is +Z; the table is Z = 0. The
rotor axis is `(0, y, 84)`, the crank axis `(60, y, 84)` and the intermediate axis `(28.19, y, 46.08)`, all along Y.
The gears are `models/spurgear` bodies of module 1.5, and the ratio is exactly 10. The bolts are `models/m2bolt` bodies
and the nuts `models/m2nut` bodies: 24 of each, with every head flush in a pocket and every nut in an enclosed
cavity. The shared build steps live in `internal/gearroot`, `internal/edgefinish` and `internal/m2fastener`.

The default build with every check takes about 5.7 minutes (340 s) and the two variants about 5.6 minutes together
(Slots 12 and 24, which run the port, orientation, overlap and rotor-fit checks only). Run the checks with:

```sh
go test ./models/handcranksiren -timeout 40m
```

CI splits the checks into the jobs `siren-default` and `siren-variants` (`.github/workflows/repository.yml`), and runs
the shared packages' tests in the mesh helper step.

Render the images with `go -C tools/render run . -model handcranksiren -shot readme -out ../../models/handcranksiren/images`.
