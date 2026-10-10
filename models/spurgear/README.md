# Spur gear

This item builds one straight-toothed involute gear. Its source is
[instructions.md](instructions.md); `spurgear.go` constructs the model with
decad. Dimensions are in millimeters, and the pressure angle is in radians.

The Go package exposes `Build(ctx, doc, Params)`. Its caller supplies a decad
document and receives one gear body. Run `go test ./models/spurgear` to check
the supported parameter cases.

`Build` draws one closed profile with fit-spline flanks, lofts matching
profiles at the two end planes, then asks decad to fillet the axial root
corners. The default model uses 15 fit points per flank and has 17 teeth
and 34 root fillets.

`BoreDiameter` cuts a centered through-bore after the root fillets.
`CapChamfer` values above zero return an error because decad does not yet
chamfer this loft body. Embedded flanks also return an error.
