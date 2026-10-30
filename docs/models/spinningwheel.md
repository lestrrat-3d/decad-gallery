# Spinning-wheel construction

[instructions.md](../../models/spinningwheel/instructions.md) owns the dimensions and behavior.
[steps.md](../../models/spinningwheel/steps.md) maps those dimensions to verified Sketch profiles
and native Decad operations. The [model page](../../models/spinningwheel/) explains operation.

## Reference

[George Weil's article](https://www.georgeweil.com/blog/how-does-a-spinning-wheel-work/)
identifies the wheel, treadle, footman, flyer, bobbin, orifice and tension adjustments.
[Ashford's Traditional specifications](https://www.ashford.co.nz/product/traditional-single-drive/)
give the 560 mm wheel, 690 mm orifice height, 10 mm orifice and 7/9/12/17 ratios used here.
[Ashford's spinning manual](https://www.ashford.co.nz/booklet/learn-to-spin.pdf)
describes the single drive band, bobbin brake and leader routing.
Frame dimensions, stock-bearing choices, shaft connections and adjustment slides are this gallery design.

## API and build

Import `github.com/lestrrat-3d/decad-gallery/models/spinningwheel`.
`Build(ctx, doc, params)` returns an assembly of named solid parts, its real constrained
sketches, and the motion state. `Motion(params)` computes the mechanism pose without creating bodies.
Both functions use the caller's parameters and keep construction state local.

| Parameter | Default | Accepted values |
| --- | --- | --- |
| Scale |1|0.8 through1.2|
| WheelAngle |pi/3|Finite radians, including multiple revolutions|
| Ratio |7|7,9,12 or17|
| DriveTension |6|0 through12 nominal mm of carriage travel|
| BrakeTravel |3|0 through6 actual mm of screw travel|
| YarnFeedPerTurn |100|0 through200 actual mm per wheel turn|

Stock bearings, shaft diameters, cord diameters, clearances and thread pitches do not scale.
Changing Ratio translates the head along Y to align the selected whorl groove with the wheel's band plane.
Changing DriveTension moves the head away from the wheel while preserving parallel axes.
Changing BrakeTravel trades one brake-tail length for the other and increases spring extension.
YarnFeedPerTurn prescribes relative bobbin motion; it does not derive yarn feed from brake friction.

Run the focused geometry tests from the repository root:

```sh
go test ./models/spinningwheel -timeout 5m
```

Build the actual bodies and render the documented images and animation:

```sh
go -C tools/render run . -model spinningwheel -shot readme -out ../../models/spinningwheel/images
```

The `hero` and `rear` presets render single scenes. The `front-back`, `flyer`, `brake` and `whorl`
presets render paired views. The `animation` preset writes a GIF.
The `whorl` sheet isolates the actual one-piece body and crops its bore in the second panel.
The `readme` preset shares one build across all images.

```sh
go -C tools/render test -run '^TestSpinningWheel' -timeout 5m
```

## Geometry and motion

Every final body comes from the native Decad model. Belt wraps are exact partial tori;
the two straight spans are circular extrusions tangent to both pulley pitch circles.
Each physical cord is represented by adjoining solid spans whose end caps share a seam.
The V-wall slope and the stock cord radius determine pitch radius, including the cord offset.
The spring uses native Coil. No rendered guide, belt or linkage replaces a failed native body.

The bobbin has stepped 608 seats and a 12 mm barrel passage around an 11 mm inner-race spacer.
Its nominal journal and outer race interfaces reuse the purchased bearing mounting envelopes.
Those envelopes do not represent separate inner/outer race rotations, seals or bearing friction.
The front plain sleeve uses the same parametric annulus constructor with its own dimensions.

The motion model solves one continuous branch of the crank, footman and treadle four-bar.
The flyer turns Ratio times per wheel revolution. At winding radius R, bobbin angle is
`(Ratio - YarnFeedPerTurn/(2*pi*R))*WheelAngle`. This explicitly keeps the bobbin independent.
The animation selects one full relative bobbin turn per wheel turn so its marker closes the GIF loop.
Its grey frame stays fixed. A/B mark the two fixed axes in the footman's plane; C/D follow the moving pins.
The pivot marks and dashed reference line are annotations over the rendered model.

## Checks

The default integration test tessellates every final solid with `VerifyAll` and verifies every
consumed sketch. Material probes check the axial orifice, blind end wall, radial outlet,
round journals, D flats and the bobbin's clearance around its inner-race spacer.
The purchased bearing checks read the actual placed widths, diameters and annular volumes.

The linkage screen copies the real crank, treadle, footman, pins, collars and sleeves into
72 solved poses and checks their clearance from the fixed frame. It uses separated boxes first.
When boxes overlap, it verifies constant sections along Y and checks their actual bounded section outlines.
The flyer screen encloses each actual arm, crossbar, whorl and guide in a full-turn swept cylinder.
It checks frame/brake separation and radial gaps around the independently turning bobbin.
These tests cover those named interfaces. They do not claim full three-dimensional contact detection
for every pair of fasteners, bearing races and structural joints.

The model does not calculate friction, spring rate, balance, wood strength or bearing life.
Nominal fits and mounting envelopes do not supply manufacturing allowances for a chosen material or printer.
Use the prototype to set drive-band preload and the lowest brake tension that draws in yarn.
