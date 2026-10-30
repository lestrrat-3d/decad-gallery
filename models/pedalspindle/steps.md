# Pedal spindle Decad steps

Follow `instructions.md` for all dimensions, names, phases and API types.
Emit only this package's Go construction; keep all imported spur geometry in its owner.

## 1. Validate and resolve motion

Implement Params, DefaultParams, Validate, State and Motion from the contract.
Validate context and document at Build entry. Validate parameters before creating geometry.
Reduce the crank angle modulo 2*pi before every motion or phase calculation to prevent overflow.
Solve the four-bar circle intersection in unscaled coordinates, then multiply vector coordinates
and CenterDistance by Scale. Leave angles and Ratio unchanged. Store all six motion coordinates
at Y=-125*Scale, except Input/Intermediate/Output which have Y=0.
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
For stock diameters passed to a scaling helper, divide their input by Scale so the final diameter stays stock.
Use ordinary rectangle and circle helpers for all repeated shape types.
Each body has its own local section, so no sketch is partially consumed or reused after mutation.

## 3. Place solids

Place local XY extrusions with `r3.NewFrame`, `r3.FromFrame`, then `decad.Transform`.
For a rectangle in the world XZ plane, local U is (cos(t),0,sin(t)),
V is (-sin(t),0,cos(t)), and N is -Y. Its frame origin is the required back Y face.
For world axis-aligned boxes, use the XY plane and translate to the required lower corner or center.
Build shafts, flywheel rings and hubs as local annular or circular extrusions,
then orient their extrusion axes toward -Y with U=+X and V=+Z.
Use positive extrusion length; select the larger Y coordinate as the starting plane.
Use rotated XZ frames for spokes, crank, treadle and footman.
The footman's U points from Q to P; its holes are at local (0,0) and (320*Scale,0).
The treadle's U uses the solved PedalAngle; its heel hole is at local Origin.
For the stepped treadle, create the eight shared-endpoint lines in the order specified in the contract.
Add Horizontal to edges0,2,4,6 and Vertical to edges1,3,5,7.
Anchor point0 from Origin with horizontal and vertical distances -18*Scale.
Dimension horizontal edges0,2,4 by36*Scale,194*Scale,-194*Scale.
Dimension vertical edges1,3,5 by6*Scale,24*Scale,6*Scale.
Add the22mm heel bore at Origin and5.2mm toe bore at(200*Scale,0).
Solve and fully verify before selecting the sole two-hole profile and extruding70*Scale toward-Y.
This is one sketch and one body; do not construct overlapping replacement rectangles or use booleans.
Build the bearing board and guard front as holed rectangles in local XZ coordinates.
Their hole centers come from shaft centers minus each rectangle's frame origin.
All other guard members are ordinary boxes.

## 4. Build the spindle

Create a local axial polygon in the world XZ plane with points (0,0), (2.5,0), (2.5,275*Scale),
(0.3,300*Scale), (0,300*Scale), closing to the first point. Only axial coordinates scale.
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

Call `spurgear.Build` for51 and17 teeth using the declared module, thickness and samples, with BoreDiameter=0.
Build locally at Origin; never run the tooth booleans on a translated or rotated gear.
Two solid prototypes produce three D-bore variants and four distinct transformed bodies.
Cut the large prototype with the8.2 D tool, copy for input, and consume the original for intermediate-gear.
Copy the solid17-tooth prototype for its8.2 D bore and cut the5.2 D bore in the original for output.
Perform all D-bore cuts before world placement, then apply the declared shaft/gear phase.
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

## 7. Import the purchased bearings

Use models/ballbearing.DefaultParams() for608-2RS and Params605ZZ() for605ZZ.
Its public Params has BoreDiameter, OuterDiameter, Width float64 and Placement *r3.Frame.
Build(ctx,doc,p) returns *Model with Body *decad.Body and Sketch *sketch.Sketch, verifies the local
annulus sketch and returns one placed body. Do not draw a bearing annulus in this package.
Create each bearing frame with U=+X,V=+Z,N=-Y, at its actual scaled X/Z center and larger Y endpoint.
Pass actual millimeter coordinates to this frame; do not scale them again in the bearing builder.
Leave all bearing dimensions unchanged. Append its Body as group bearing and append its Sketch.
Follow the part table exactly for all ten names, centers and axial spans.
Place footman at Y[-129,-121]*Scale, in front of the input shaft's front end at -116*Scale.
Keep treadle at Y[-210,-140]*Scale; bridge gaps with the pins specified in the contract.
At scale0.8 the two7mm shaft bearings still fit inside the14.4mm board; no bearing is scaled.
Increase footman width/end lengths and treadle thickness as specified so14mm pivot bores have material around them.
Use608-2RS for the two heel bearings and an8mm through-shaft. Keep the heel bore22mm at every scale.
Construct both heel brackets, two collar envelopes and two inner-ring spacers exactly from the part table.
The spacers use stock outer diameter12mm; the collar OD16 scales. All six added support pieces use ordinary verified sections.
The former single heel bracket is replaced by two brackets; the complete assembly has48 parts.

## 8. Positive D couplings

Preserve the exported API. Emit a separate couplings.go if needed for this construction.
Construct each D-bore tool locally: verified circle of the declared bore radius, on XY at Z=-1,
extruded through Z=thickness+1. Cut off its +X cap with a verified rectangular prism:
X[flat,radius+1],Y[-radius-1,radius+1],Z[-2,thickness+2].
Use the resulting exact D prism with decad.Cut to bore the real local gear, crank or flywheel hub.
Use the8.2 D tool for the51-tooth prototype, intermediate17-tooth, crank and flywheel hub;
use the5.2 D tool for the output17-tooth. Each D tool adds one circle and one rectangle sketch.
Build crank's local rectangle with only its round pin hole, then cut its D bore.
Build flywheel-hub as a solid disc before cutting its D bore. Place both at phase a.
Do not replace D outlines with polygons or approximate circular arcs.

Cut shaft flats before world placement with verified rectangular prisms in the shaft's local XY section.
For8mm shafts remove X>3, for5mm spindle remove X>1.75. Extend each tool to X=radius+1 and Y=±(radius+1).
Input local cylinder starts at world Y=65*Scale and extends toward-Y. Build both rectangular cap tools:
local Z[85*Scale,181*Scale+1] for the front flat and Z[-1,35*Scale] for the rear flat.
Union those two disjoint tools first, then cut the unchanged cylinder once with the combined tool.
Do not chain two cuts on the input shaft: the first cut's mesh bound prevents the second cut.
Place the completed shaft at phase a. This preserves the same final dimensions and four total shaft-tool sketches.
Intermediate starts at world Y=26*Scale: remove local Z[46*Scale,109*Scale+1], then place at phase b.
Spindle starts at world Y=30*Scale: cut local Z[80*Scale,114*Scale] after its exact revolve, then place at phase c.
These four flat tools add four rectangle sketches. Radial cut coordinates never scale; axial endpoints scale.
Record and fully verify every tool sketch, including offset planes, before consuming its geometry.
All tools are consumed by their cuts; leave only48 assembly parts, with58 recorded sketches.
