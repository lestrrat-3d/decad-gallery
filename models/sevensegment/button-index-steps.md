# Seven-segment Geneva index boundary steps

Read `button-index-instructions.md`, the gallery `docs/model-pipeline.md`, and
the existing seven-segment `instructions.md` and `steps.md` as dependency
contracts. The existing display's public `Build(ctx, doc, Params{Digit: 9})`
returns the real `Model` with `Shaft`, `Cams[7]`, `Sliders[7]`, `Dimensions`
and segment truth table. The default placement is the world frame. Do not
inspect existing Go source as an emission input. Implement the private boundary
in new package-local files and leave existing manual behavior unchanged.

## Coordinate and phase conventions

The output centre O is world (X,Z)=(0,28). The crank centre C is (10,52),
26 mm from O. Let a=atan2(24,10), A=a+108°, and
B=atan2(20 sin A−24, 20 cos A−10). For the crank and its shaft let local U
point along (cos B,0,sin B), V along (−sin B,0,cos B), and local W toward −Y.
The crank origin is (10,−69.75,52). Define driver phase φ in degrees, with
zero at the parked 9 position. Construct and finish the shaft, key, crank and
rivet at phase zero before applying any rotation. Rotate those four finished
bodies together by φ about the world −Y axis through (10,−69.75,52). Use the same
frame for the Ø5 input shaft, whose positive axial station s points toward −Y.

For output increment q, use q=0 for φ≤108°, q=36° for φ≥252°, and otherwise
q=18°−atan2(sin(18°) sin φ, 1+sin(18°) cos φ). Angles inside trigonometric
functions are radians. Rotate the wheel and display shaft together from the
actual digit-9 pose by q about world +Y. At the two endpoints verify the seven
segment truth values for 9 and 0; do not substitute digit-state geometry for
the moving cams or sliders.

## Geneva wheel

1. In a radial/axial sketch, constrain the closed meridian
   (0,0),(6,0),(6,3.9),(25.75,3.9),(25.75,7.9),(6,7.9),(6,12),(0,12).
   Revolve it 360° about the axial Z line. Constrain stations and radii from
   the sketch origin, repeated radii and stations by equal-coordinate relations.
2. Cut a through single-D bore, R2.55 with its flat at 2.05 from centre, over
   Z−1..13. This fits the existing output shaft stub. The cutter is one closed
   arc-and-line sketch; constrain the circle to the origin, its radius and flat
   distance, and the line endpoints to the circle.
3. For i=0..9 place each slot at angle a−18°+36°i in the wheel XY plane.
   The slot is a closed capsule whose inner end is an R2 semicircle centred
   on radius 17.966, with parallel sides at ±2 and an open outer end beyond
   radius 25.75. Use a rectangle X17.966..27,Y±2 and R2 inner circle,
   cutting from Z3 to 9. Constrain each circle and rectangle to the common
   wheel origin and angular reference. Cut ten Ø4 holes on R20 at a+36°i
   over the same axial range. Verify all sketches before consuming them.
4. Place the wheel with local Z toward world −Y and local X along world +X,
   local Y along world +Z, origin (0,−61.45,28). Its active 4 mm band is
   world Y−69.35..−65.35. Apply the digit-9 shaft clock and q consistently;
   match the actual shaft's single flat, and verify the fit using the actual
   shaft rather than an independently fabricated shaft fixture.

## Input shaft and crank

1. Machine the Ø5 steel input shaft, length 91.75 from s=0, with an integral
   Ø7×1 head at s=0. Cut Ø4.8×0.7 retaining-ring grooves at s=62.25..62.95,
   79.85..80.55 and 90.55..91.25. Drill Ø2 through cross holes at s=2.6,
   22.35, 37.25 and 59.25 along local U. Constrain each tool sketch relative
   to the shaft axis and its station. Model the crank's Ø2×20 steel key in the
   s=2.6 hole as a separate body.
2. In the crank UV sketch make an R11.20 disk and a single keyed inner loop.
   Its bore is R2.55. The two key-slot ends are at U=±10.10. In each quadrant
   draw the side from the bore intersection at |V|=1.06 through
   (|U|,|V|)=(8.3,1.06),(8.8,1.01),(9.6,1.01),(10,1.06),(10.10,1.06).
   Horizontal lands, straight ramps and vertical end walls join the upper and
   lower origin-centred bore arcs. Dimension the X stations and middle land;
   use symmetry and shared points. Extrude W=0..4.80. Cut the shaft-head recess
   Ø7.20 from W−0.10 to +1.10.
3. The drive-rivet axis is at throw r=26 sin(18°), at local angle a−B.
   Drill Ø3.10 through the crank. Counterbore Ø4.16 from W0 to 0.78 and
   from W4.00 to 4.80. Each circle is dimensioned from the crank origin to
   this axis and by its radius. The nominal bearing land is W0.78..4.00.
4. Build the retained rivet as one steel body with Ø3 working cylinder from
   s=−4.40 to 4.74 and Ø3.85 shoulders at s=0.05..0.70 and 4.10..4.74.
   Constrain the circles on their common axis. Join overlapping shoulder and
   stem solids. Place its axis on the drilled crank axis. Its active nose
   projects into the wheel band while the crank/wheel face gap is 0.40 mm.
   The machined rivet must be installed before the crank is fitted to the shaft;
   its actual forming, tolerance and load capacity are physical checks.

## Real producer-to-consumer verification

Build the actual existing display at digit 9 and the new boundary in one
`decad.Document`. Retain and test the actual display shaft and the seven actual
cams and sliders. Do not count the manual knob, detent plate and manual ratchet
as button components. Build a distinct real body for every new component.
Verify the crank/key/shaft and wheel/shaft clearance with the actual meshes;
use each tessellation's `VerifyAll` and reported error bound in `meshgap.New`.

Test φ=0,104,108,115,122,150,180,210,238,245,252,256 and 354°. At each,
place the actual rivet and wheel according to their rigid motions. Screen both
±0.286° output park offsets, and check the Ø3.45 gauging envelope around the
rivet at the working nose. Check that no unintended wheel/rivet intersection
occurs at the sampled positions. At φ=180°, turn the wheel by q−3° and q+3°
separately and require each imposed offset to contact the nominal Ø3 rivet;
the nominal centered pose must clear. Check the actual crank and
key against the shaft and the rivet's two axial retention stops.

For each of the seven actual segment cams and sliders, check φ=108,132,156,
180,204,228,252°. Rotate the real digit-9 cam by q. At the slider pin's depth
read the two real groove-wall intersections and translate the whole real slider
to their midpoint. Test real cam/slider clearance and check that the observed
lifts match the digit-9 and digit-0 truth tables within 0.10 mm at the endpoints.
Record all seven intermediate lifts; segments e and g exchange their states.
The 9→0 groove is closed in this sampled interval. Do not claim that this
prescribed-position test simulates forces or proves autonomous motion.

Run the focused new geometry test, the existing display's affected focused
checks, and the repository structure checker. Report any unsupported operation,
missing constraint, clearance failure or unavailable feature; do not weaken the
test or silently change the geometry. Keep tests' failure output bounded.
