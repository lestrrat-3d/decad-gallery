# Spinning-wheel construction

[instructions.md](../../models/spinningwheel/instructions.md) owns the dimensions and behavior.
[steps.md](../../models/spinningwheel/steps.md) maps those dimensions to verified Sketch profiles
and native Decad operations. The [model page](../../models/spinningwheel/) explains operation.
[mounting.md](../../models/spinningwheel/mounting.md) owns the final fitted joints, hardware retention and part grouping.

## Reference

[George Weil's article](https://www.georgeweil.com/blog/how-does-a-spinning-wheel-work/)
identifies the wheel, treadle, footman, flyer, bobbin, orifice and tension adjustments.
[Ashford's Traditional specifications](https://www.ashford.co.nz/product/traditional-single-drive/)
give the 560 mm wheel, 690 mm orifice height, 10 mm orifice and 7/9/12/17 ratios used here.
[Ashford's spinning manual](https://www.ashford.co.nz/booklet/learn-to-spin.pdf)
describes the single drive band, bobbin brake and leader routing.
Frame dimensions, stock-bearing choices, shaft connections, alignment slots and hinged jack are this gallery design.

## Current preview

The static images and the animation are rendered from the real default Build body: 206 parts, none omitted.
[preview-status.json](../../models/spinningwheel/images/preview-status.json) records the part counts.
Joint cuts, retention and collision checks remain unfinished; 19 part pairs still lack a conclusive
non-overlap result, so the images do not show a verified assembly.
See [the preview renderer](../rendering.md) for the regeneration command.

The sections below describe the intended complete model and its checks, not a claim that those checks pass today.

## API and build

Import `github.com/lestrrat-3d/decad-gallery/models/spinningwheel`.
`Build(ctx, doc, params)` returns an assembly of named solid parts, its real constrained
sketches, and the motion state. `Motion(params)` computes the mechanism pose without creating bodies.
Both functions use the caller's parameters and keep construction state local.
`BuildPreview(ctx, doc, params)` additionally returns completed parts and named failures for visual review.
It uses the same part constructors, keeps strict `Build` unchanged, and omits final joint cuts.

| Parameter | Default | Accepted values |
| --- | --- | --- |
| Scale |1|0.8 through1.2|
| WheelAngle |pi/3|Finite radians, including multiple revolutions|
| Ratio |7|7,9,12 or17|
| DriveTension |6|0 through12 nominal mm of jack-edge lift|
| BrakeTravel |3|0 through6 actual mm of screw travel|
| YarnFeedPerTurn |100|0 through200 actual mm per wheel turn|

Stock bearings, shaft diameters, cord diameters, clearances and thread pitches do not scale.
Changing Ratio translates the head along Y to align the selected whorl groove with the wheel's band plane.
Changing DriveTension tilts the hinged upper head while preserving parallel shaft axes.
The vertical M8 jack has constant shaft length, a pointed shelf contact and a positively keyed knob.
Changing BrakeTravel trades one brake-tail length for the other and extends each spring by half that travel.
YarnFeedPerTurn prescribes relative bobbin motion; it does not derive yarn feed from brake friction.

Run the focused geometry tests from the repository root:

```sh
go test ./models/spinningwheel -timeout 5m
```

Build the actual bodies and render the documented images and animation:

```sh
go -C tools/render run . -model spinningwheel -shot readme -out ../../models/spinningwheel/images
```

The `hero` and `rear` presets render single scenes. The `front-back`, `layout`, `flyer`, `brake` and `whorl`
presets render paired views. The `animation` preset writes a GIF.
The `whorl` sheet isolates the actual one-piece body and crops its bore in the second panel.
The `layout` sheet shows top and side views of the pedal, perpendicular pivot axes and rear footman.
The `readme` preset shares one build across all images.

```sh
go -C tools/render test -run '^TestSpinningWheelMotion$' -timeout 5m
```

## Geometry and motion

Every final body comes from the native Decad model. Drive-band wraps are exact partial tori;
the two straight spans are circular extrusions tangent to both pulley pitch circles, with adjoining end caps.
The brake cord is one swept solid with captive terminal eyes, and the yarn leader is one continuous Union body.
The V-wall slope and the stock cord radius determine pitch radius, including the cord offset.
The spring uses native Coil. No rendered guide, belt or linkage replaces a failed native body.

The bobbin has stepped 608 seats and a 12 mm barrel passage around an 11 mm inner-race spacer.
Its nominal journal and outer race interfaces reuse the purchased bearing mounting envelopes.
Those envelopes do not represent separate inner/outer race rotations, seals or bearing friction.
The front plain sleeve uses the same parametric annulus constructor with its own dimensions.

The motion model solves one continuous branch of the spatial crank, upper gimbal, footman and treadle linkage.
The treadle rotates about X; the wheel rotates about Y. The treadle bars and rear crossarm reach behind the wheel.
The gimbal's first 605 stays coaxial with the crank pin. Two more 605 envelopes share a perpendicular cross-pin.
The lower connection is one compliant body with two anchored legs. The renderer and clearance tests rebuild
that actual component at each solved pose through `BuildConnector`, rather than moving it as a rigid part.
The model does not calculate that connector's strain, fatigue or stiffness.
The flyer turns Ratio times per wheel revolution. At winding radius R, bobbin angle is
`(Ratio - YarnFeedPerTurn/(2*pi*R))*WheelAngle`. This explicitly keeps the bobbin independent.
The animation selects one full relative bobbin turn per wheel turn so its marker closes the GIF loop.
Its grey frame stays fixed. A/B mark the perpendicular fixed axes; C/D/S follow the three-dimensional moving joints.
The pivot marks and dashed reference line are annotations over the rendered model.

## Checks

The default integration test tessellates every final solid with `VerifyAll` and verifies every
consumed sketch. Material probes check the axial orifice, blind end wall, radial outlet,
round journals, removable shaft keys and the bobbin's clearance around its inner-race spacer.
The purchased bearing checks read the actual placed widths, diameters and annular volumes.
Each part must have one connected final mesh and a declared mount with a support path to the three grounded legs.
The geometric support check removes declared connections whose actual boundary gap exceeds the mount clearance
after charging both verified mesh bounds. It then requires a path to a grounded leg through the remaining connections.
A touching group with only a distant frame reference fails this check. Nearby surfaces alone do not prove retention;
the material, joint and clearance checks inspect the actual bodies separately.
The brake checks read the rivets' native circles, check their shanks and retaining heads, and probe material around
the spring and cord eyes. Closed-mesh topology checks require two eye openings in each spring and in the cord.
The spring mounts allow the specified 0.2 mm radial free play between each eye and its pin.

The linkage screen places meshes of the real moving parts in 72 solved poses and checks their clearance from the frame.
It uses separated boxes first, then projections on native face normals and crossed edge directions.
Each separating gap must exceed both verified meshes' displacement bounds.
For non-convex shapes, invariant native sections or triangle distances resolve overlapping convex enclosures.
The triangle check also rejects a component contained inside another solid.
The only bore exemptions require an actual native coaxial through-cylinder and prove the moving part fits inside it.
The flyer screen encloses each actual arm, crossbar, whorl and guide in a full-turn swept cylinder.
It checks frame/brake separation and radial gaps around the independently turning bobbin.
The default geometry test also requires zero material interferences and no unresolved pair diagnostics
from the native document verifier across all final parts. An unsupported contact result fails that check.
The motion screens cover the named moving interfaces, rather than every possible part pair at every angle.

The model does not calculate friction, spring rate, balance, wood strength or bearing life.
Nominal fits and mounting envelopes do not supply manufacturing allowances for a chosen material or printer.
Use the prototype to set drive-band preload and the lowest brake tension that draws in yarn.

The [reference audit](../spinning-wheel-reference-audit.md) lists every labelled reference part and all eight correction groups.
