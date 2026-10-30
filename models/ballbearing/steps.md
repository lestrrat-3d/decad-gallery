# Bearing-envelope Decad steps

Follow instructions.md for the public API and governing dimensions.
Emit one cohesive model construction file, not tests or renderer code.

## 1. Validate

Implement Params, DefaultParams, Validate and Model exactly as specified.
DefaultParams returns BoreDiameter8, OuterDiameter22 and Width7, Placement nil.
Use math.IsNaN/IsInf, positive checks, ordered diameters and Placement.IsValid.
Build validates ctx/doc and params, then ctx.Err before creating the sketch or document geometry.

## 2. Build and verify the section

Create a fresh sketch.World per call. Create a sketch on world.XY().
Create the outer and bore circles at s.Origin(), with radii OuterDiameter/2 and BoreDiameter/2.
Add sketch.NewDiameter constraints with each specified diameter. No other constraints are needed.
Call s.Solve(ctx), then internal/sketchcheck.VerifyFullyConstrained(ctx,s).
Require exactly one valid profile with len(Holes)==1 among s.Profiles().
Fail on a missing or ambiguous annulus; do not pass the inner disc to extrusion.

## 3. Extrude and place

Call decad.Extrude(ctx,doc,s,profile,
decad.Distance{D:units.Millimeters(p.Width),Dir:decad.Along}).
The result locally spans Z0..Width, is centered on Z, and has an open bore.
If Placement is nonnil, call r3.FromFrame(*p.Placement), then decad.Transform(ctx,body,transform).
Require the final body solid and return &Model{Body:body,Sketch:s}.
The completed transformation replaces the unplaced body; no prototype is left in doc.
Keep all construction state local, return contextual wrapped errors, and do not copy another model's source.

## 4. Add the 605ZZ preset

For this emission, output only a new presets.go file in package ballbearing.
Implement `Params605ZZ() Params` returning a struct literal with BoreDiameter5, OuterDiameter14,
Width5 and nil Placement. Do not duplicate or modify Build, Model, Params, validation or the default608 preset.
The parent supplies the existing public Params/Build API; the emitted preset only selects dimensions.
