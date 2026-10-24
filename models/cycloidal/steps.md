# Cycloidal drive Decad construction steps

These steps implement [instructions.md](instructions.md). Lengths are millimeters. `Build(ctx, doc,
Params)` returns a `Drive`.

1. Validate `Params` with `Problems` before adding any body, then define every input and derived
   dimension in the document's parameter table and in a sketch world's table (`defineParameters`).
2. For disc `d` in `0 ..< DiscCount`, lift the sketch plane by `d (DiscThickness + DiscGap)` along the
   placement normal. Trace one lobe of the rotor profile at 2000 steps and keep a point each time the turn
   reaches 5°. Find the two inflection parameters by bisection and split the points there into three runs.
3. Draw the `L` lobes as `3 L` fit splines. Dimension every point from the sketch origin with a
   horizontal and a vertical distance, and share the point at each cut and each valley. Add the bore circle
   and the `M` output-hole circles, dimension their centers from the origin and bind their diameters to
   `disc_bore_diameter` and `output_hole_diameter`. Solve and require a fully constrained sketch.
4. Pick the one region with `1 + M` holes and `Extrude` it by `disc_thickness`. If decad refuses the
   build as unsupported, repeat steps 2 to 4 with 8°, 12° and 20° sampling.
5. Build the cam: per disc a circle of diameter `center_bearing_diameter` on that disc's center, extruded
   over the disc and gap (disc 0) or the gap and disc (disc 1), joined by `Union` for two discs. Cut the
   input bore as a cylinder that starts a plane gap below the cam and ends one above.
6. Sweep the casing contour for one pin pitch as the instructions describe and mirror it about the pin. Draw the cavity sketch: `N` copies of the pitch, three fit splines each,
   thinned by a stride of 3. Extrude the region inside the contour from one plane gap above the base
   to past the stack top. If decad refuses it as unsupported, use strides 2, 4, 6 and 1.
7. `Extrude` the housing's outer circle from `1 mm + base_thickness` below disc 1 to the stack top. `Cut`
   the cavity, then `Cut` a cylinder of diameter `housing_inner_diameter` from below the housing up into
   the cavity.
8. Draw the output plate sketch one plane gap above the stack: the plate circle and `M` socket circles
   of diameter `resolved_output_pin_diameter`. `Extrude` the region with `M` holes by
   `output_plate_thickness`. Draw `M` pin circles at the same centers, extrude each two-sided, up through
   the plate and down to the lower face of disc 1, and order the pins by angle.
9. When `ChamferSize > 0`, `Chamfer` the outer circle edges of the housing's end faces, the outer
   loop of both plate faces and both end loops of each pin. Leave the discs sharp.
10. Check each body is a solid and passes `Tessellate(0.1 mm, VerifyAll)`.
