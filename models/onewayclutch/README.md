# One-way clutch

A printed one-way clutch for a D-flat shaft. A hub with three flexing pawls presses on the shaft and drives a 42 mm ring
with 24 internal teeth one way; the ring turns freely the other way. Choose the shaft from the presets (`605ZZ`,
`608ZZ`, `608ZZ-gearbox`) and the freewheel direction. All parts print without supports; the only bought parts are
three M2 x 16 bolts and nuts.

This model is under construction: only the default clutch (605ZZ, clockwise freewheel) builds so far, and the images,
views and parts table are still to come. See [instructions.md](instructions.md) for the construction contract and
[steps.md](steps.md) for the Decad steps.

```sh
go test ./models/onewayclutch
go -C tools/render run . -model onewayclutch -shot hero -out hero.png
```
