# Pedal spindle Decad steps

Follow `instructions.md` for all dimensions, names, phases and API types.
Emit only this package's Go construction; keep all imported spur geometry in its owner.

## 1. Validate and resolve motion

Implement Params, DefaultParams, Validate, State and Motion from the contract.
Validate context and document at Build entry. Validate parameters before creating geometry.
Reduce the crank angle modulo 2*pi before every motion or phase calculation to prevent overflow.
Solve the four-bar circle intersection in unscaled coordinates, then multiply vector coordinates
and CenterDistance by Scale. Leave angles and Ratio unchanged. Store all six motion coordinates
at Y=-101*Scale, except Input/Intermediate/Output which have Y=0.
Keep build state local; the package has no mutable globals.

## 2. Sketch builders

Use `sketch.NewWorld`, its XY plane, `CreateSketch`, and `CreateCircle` or `CreateRectangle`.
For rectangles, CreateRectangle creates horizontal/vertical constraints already.
Add HorizontalDistance and VerticalDistance from Origin to rect.A, then HorizontalDistance
from rect.A to rect.B for width and VerticalDistance rect.B to rect.C for height.
For each circular hole, use Origin when centered or dimension the center from Origin,
then add a Diameter constraint. Include expected holes in the section before extrusion.
For annuli, make two concentric circles at Origin; constrain the outer radius and inner diameter.
For each section, Solve(ctx), then `sketchcheck.VerifyFullyConstrained(ctx,s)` before reading profiles.
Select exactly one valid profile with the intended hole count. A disc surrounded by an annulus
is also a valid profile; choose the annulus when the contract requires a bore.
Record each created sketch in Assembly.Sketches. Do not use Fix constraints.

Extrude with `decad.Extrude(ctx,doc,s,profile,decad.Distance{D:units.Millimeters(length),Dir:decad.Along})`.
Apply scale while drawing local sections and choosing extrusion lengths.
Use ordinary rectangle and circle helpers for all repeated shape types.
Each body has its own local section, so no sketch is partially consumed or reused after mutation.

## 3. Place solids

Place local XY extrusions with `r3.NewFrame`, `r3.FromFrame`, then `decad.Transform`.
For a rectangle in the world XZ plane, local U is (cos(t),0,sin(t)),
V is (-sin(t),0,cos(t)), and N is -Y. Its frame origin is the required back Y face.
For world axis-aligned boxes, use the XY plane and translate to the required lower corner or center.
Build bearings, shafts, flywheel rings and hubs as local annular or circular extrusions,
then orient their extrusion axes toward -Y with U=+X and V=+Z.
Use positive extrusion length; select the larger Y coordinate as the starting plane.
Use rotated XZ frames for spokes, crank, treadle and footman.
The footman's U points from Q to P; its holes are at local (0,0) and (320*Scale,0).
The treadle's U uses the solved PedalAngle; its heel hole is at local Origin.
Build the bearing board and guard front as holed rectangles in local XZ coordinates.
Their hole centers come from shaft centers minus each rectangle's frame origin.
All other guard members are ordinary boxes.

## 4. Build the spindle

Create a local axial polygon in the world XZ plane with points (0,0), (5,0), (5,275),
(0.6,300), (0,300), closing to the first point. Apply Scale.
The local vertical coordinate becomes length along the spindle axis.
Reuse the sketch Origin for the first point. Add Horizontal to the first and fourth lines,
Vertical to the second and fifth lines. Dimension the first radial length, second axial length,
and the taper endpoint's horizontal and vertical coordinates relative to Origin.
Shared endpoints and these eight scalar constraints constrain the five-point polygon.
Use Solve and full Verify before obtaining the sole hole-free profile.
Revolve 360 degrees about local Z with `decad.ConstructionAxis{Origin:r3.Vec{},Dir:r3.NewVec(0,0,1)}`
and `decad.FullRevolution{}` (confirm the exact angular extent API from dependency source).
Transform local +Z to world -Y, with the base at output X/Z and Y=30*Scale.
The final tip is at Y=-270*Scale. Preserve the exact conical taper; do not facet it.

## 5. Reuse the spur gears

Call `spurgear.Build` for 51 and 17 teeth using the declared module, bore, thickness and samples.
Build locally at Origin; never run the tooth booleans on a translated or rotated gear.
Two prototypes produce four distinct transformed bodies, named by the part table.
Decad Transform retires its input. For the input gear and intermediate pinion, call
`decad.Copy(ctx, prototype)` and transform that copy. For the second occurrence of each prototype
(intermediate gear and output pinion), transform the original prototype after its first copy exists.
This leaves four placed gear bodies and no unused prototype bodies in the document.
To rotate a prototype through angle t about world Y after placement, use the XZ U/V frame in step3.
The clockwise/anticlockwise sense is defined by that frame's normal, -Y.
Copies must have the correct bore, normal, phase, axial positions, and center distances.
Do not handwrite a substitute tooth profile or remove root fillets.
The imported Build verifies its own sketches; Assembly.Sketches covers this package's sketches.

## 6. Assemble and check

Append one Part for every row/member in the contract, with the specified name/group.
Return all physical parts including the guard, even though some render presets hide it.
Require every part Body.IsSolid() before returning Assembly and its solved State.
Return contextual errors for sketch creation, solve, verify, profile selection, extrusion,
revolve, gear construction and transformation.

Test the first real default assembly before adding parameter variants or renderer presets.
Use bounded failure reports with part names, dimensions and sketch reports only.
Geometry tests must exercise imported gears after placement in this assembly.
