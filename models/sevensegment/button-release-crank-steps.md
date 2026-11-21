# Seven-segment button release-crank boundary steps

Read `button-release-crank-instructions.md`, `button-index-instructions.md`,
`button-index-steps.md`, the existing seven-segment `instructions.md` and
`steps.md`, and `docs/model-pipeline.md`. Do not inspect existing Go source as
an emission input. Use new package-local files and leave the existing builders
unchanged. All dimensions below are in mm.

## Coordinates and motion

Use the gear frame with world position (10+x,−140−z,52+y) for gear coordinate
(x,y,z). The release crank pivots about gear +X through (y,z)=(−26,10).
For lift L in [0,2], the crank angle is asin((1−L)/10) radians. Build the
crank and its three working pins at zero angle in one document, then rotate
those finished bodies together about this pivot; do not rotate their sketches
or cutters before Boolean operations. The pivot journal remains stationary.
Create named containers for the moving crank and four separate metal pins.

## Crank and retained pins

1. Sketch on gear X−46 a fully constrained YZ polygon with vertices, in order,
   (−23,7),(−23,22.7),(−29,22.7),(−29,13),(−38.8,13),(−38.8,7).
   Close it and extrude 3 along +X. Constrain horizontal and vertical edges,
   with independent offsets and lengths dimensioned from the gear axes.
2. Drill the crank through along +X with an Ø3.30 pivot bore at Y−26,Z10 and
   three Ø2.00 working-pin bores at input (−26,20), output (−36,10) and bias
   (−26,15). Counterbore each working-pin bore Ø3.10 by 0.50 from each X
   face; the remaining web is 2 mm thick. Anchor the circular sketches to
   the same frame and dimension each centre and radius. Verify every sketch.
3. Build three distinct formed steel pins, each as a single revolved axial
   profile. Both heads have Ø2.80 and occupy X−45.95..−45.54 and
   X−43.46..−43.05. The Ø1.96 waist spans the inner head faces. The input's
   Ø2.00 working tip extends from the positive-side head to X−39.50; the
   output and bias tips extend from the negative-side head to X−49.50 and
   X−57.50 respectively. Dimension every unique axial station and radius; use
   shared radii and axial alignment constraints in each closed meridian before
   revolving around its own X axis. The head inner faces are 2.08 mm apart,
   leaving 0.08 mm nominal total float on the 2.00 mm web.
4. Make a separate steel Ø3.00 pivot journal. Its left Ø4.00 head spans
   X−48.95..−48.63, its right Ø4.00 head spans X−42.91..−42.56, and its
   coaxial journal connects their inner faces. Use a fully constrained closed
   meridian and revolve about gear +X at Y−26,Z10. This journal is held by a
   separate future bracket; its full assembly retention is not established by
   the isolated crank bore alone.
5. Apply the lift rotation to the crank and three working pins, then place all
   five finished bodies in the world frame. The pivot retains its zero pose.
   Provide distinct names and references; a formed pin is one body, not three.

## Real fit and display checks

Use a default Decad document. Build the actual digit-9 `sevensegment.Build`
display and real `buildButtonIndex` boundary in that same document. Exclude
only the manual knob, ratchet and detent plate from button clearance checks.
Build the real crank boundary at L=0,0.5,1,1.5,2. For each working pin,
require the centered pin to clear its real crank bore and counterbores. Move
it by ±0.15 mm along its own axis in separate copies and require each head to
have a positive material-overlap witness against the real crank. Verify the
real pivot's Ø3 journal clears the Ø3.30 crank bore at every lift.

Compare the crank and all four metal pin bodies against the actual retained
existing display and index bodies at each lift, using conservative verified
mesh bounds or a provably sufficient bounds separation. Use bounded failure
messages. Verify `doc.VerifyContainers` and distinct body identities. The
physical pins are installed before the crank is fitted to its bracket; the
supplier stock, forming die, head strength, heat treatment and loaded wear are
not established by these geometric tests.

Run the new focused test and the existing index boundary tests. This boundary
does not establish crank-to-plunger or crank-to-fork engagement, grounded pivot
support, spring return, anti-rotation, printed fit or complete button behavior.
