# Cycloidal drive package

The `models/cycloidal` package builds the drive that the
[cycloidal drive gallery page](../../models/cycloidal/) shows. Its source is
[`instructions.md`](../../models/cycloidal/instructions.md), which follows the Fusion add-in's
`fusion360-gear-generator/spec/cycloidal`. The [steps](../../models/cycloidal/steps.md) map the contract
to Sketch and Decad operations. Dimensions are in millimeters.

## Using the package

`Build(ctx, doc, Params)` returns a `Drive` whose bodies carry the add-in's names, and `Drive.Named` lists
them in order. `DefaultParams()` is the add-in's default: 16 pins on a 90 mm pin circle, 1.5 mm
eccentricity, one 8 mm disc and six output pins on a 50 mm circle. `Params` carries every dialog input of
the add-in.

`Placement` (an `r3.Frame`) replaces the add-in's plane and anchor point. The frame origin is a point on
the drive axis, and the first disc's lower face lies on its plane.

`Problems` and `Validate` apply the add-in's validity table, with its wording, before anything is built.

The document passed to `Build` needs a chain tolerance, and the ways this build differs from the Fusion
add-in are listed in the doc comment on `Build` in `build.go`.

## Checks

`go test ./models/cycloidal` builds the default drive, a two-disc drive and twelve option variants
(solid cam, pin overrides, 12 and 24 pins, five output pins, 20 pins with two discs, a placement frame,
a thin housing, larger clearances, and four with chamfered discs). It checks the chamfered disc's volume
against the removed wedge. Every body passes 0.1 mm `VerifyAll`, every sketch is fully constrained, and
the rotor clears the casing, the cam and the output pins at the home pose.

The [renderer](../rendering.md) produces the page's images from the two-disc drive.
