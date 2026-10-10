# Spur gear

This item builds one straight-toothed involute gear. Its source is
[instructions.md](instructions.md); `spurgear.go` constructs the model with
decad. Dimensions are in millimeters, and the pressure angle is in radians.

The Go package exposes `Build(ctx, doc, Params)`. Its caller supplies a decad
document and receives one gear body. Run `go test ./models/spurgear` to check
the supported parameter cases.

`Build` draws one closed profile with fit-spline flanks and circular root
fillets, then lofts matching profiles at the two end planes. The default
model uses 15 fit points per flank and has 17 teeth and 34 root fillets.
`BoreDiameter` and `CapChamfer` values
above zero return an error; embedded flanks also return an error.
