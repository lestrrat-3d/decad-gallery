# Cycloidal drive Decad construction steps

These steps implement [instructions.md](instructions.md). Lengths are millimeters. `Build(ctx, doc,
Params)` returns a `Drive`.

1. Validate `Params` with `Problems` before adding any body, then define every input and derived
   dimension in the document's parameter table and in a sketch world's table (`defineParameters`).
2. For disc `d` in `0 ..< DiscCount`, lift the sketch plane by `d (DiscThickness + DiscGap)` along the
   placement normal. Trace one lobe of the rotor profile at 2000 steps and keep a point each time the turn
   reaches 5°. Find the two inflection parameters by bisection and split the points there into three runs.
3. Add the bore circle and one output-hole circle, dimension their centers from the origin and bind
   their diameters to `disc_bore_diameter` and `output_hole_diameter`. Draw one lobe as three fit splines,
   from its last valley back to its first so that it runs counterclockwise. Dimension every point from the
   sketch origin with a horizontal and a vertical distance, and share the point at each cut. Leave the end
   valley free and make `L` copies of the lobe with a circular pattern about the bore circle's center that
   closes the ring on shared valley points. Repeat the hole `M` times with a circular pattern about the bore
   circle's center. Solve and require a fully constrained sketch.
4. Pick the one region with `1 + M` holes and `Extrude` it by `disc_thickness`. If decad refuses the
   build as unsupported, repeat steps 2 to 4 with 8°, 12° and 20° sampling.
5. Build the cam: per disc a circle of diameter `center_bearing_diameter` on that disc's center, extruded
   over the disc and gap (disc 0) or the gap and disc (disc 1), joined by `Union` for two discs. With `disc_gap = 0`, stop the sections 0.25 mm short of the plane between the discs and add a bridge cylinder on the drive axis that reaches 0.2 mm into each. Cut the
   input bore as a cylinder that starts a plane gap below the cam and ends one above.
6. Sweep the casing contour for one pin pitch as the instructions describe and mirror it about the pin. Draw the cavity sketch: one pitch as three fit splines,
   thinned by a stride of 3, with its end point left free, and make `N` copies of it with a circular pattern about the
   origin that closes the ring on shared points. Extrude the region inside the contour from one plane gap above the base
   to past the stack top. If decad refuses it as unsupported, use strides 2, 4, 6 and 1.
7. `Extrude` the housing's outer circle from `1 mm + base_thickness` below disc 1 to the stack top. `Cut`
   the cavity, then `Cut` a cylinder of diameter `housing_inner_diameter` from below the housing up into
   the cavity.
8. Draw the output plate sketch one plane gap above the stack: the plate circle and one socket circle
   of diameter `resolved_output_pin_diameter`, repeated `M` times with a circular pattern about the origin. `Extrude` the region with `M` holes by
   `output_plate_thickness`. Draw pin 1's circle at its socket, extrude it two-sided, up through
   the plate and down to the lower face of disc 1, and chamfer it. `PatternCopies` repeats that pin `M` times
   about the drive axis, so pin `k` is at angle `2πk/M`.
9. When `ChamferSize > 0`, `Chamfer` the outer circle edges of the housing's end faces, the outer
   loop of both faces of each disc, the outer loop of both plate faces and both end loops of each pin.
10. Check each body is a solid and passes `Tessellate(0.1 mm, VerifyAll)`.
