# Seven-segment Geneva index boundary

This contract covers the Geneva index and its fit with the existing display.
The full push-button assembly is outside its scope.

Use millimetres and degrees. Build the actual ten-slot Geneva wheel, overhung
crank, retained drive rivet, input shaft and transverse crank key described in
`button-index-steps.md`. Exercise them with the existing seven-segment display
at digit 9, including its actual output shaft, cams and sliders. The crank must
move the output through one 36° index from digit 9 to digit 0. Preserve the
existing manual display's public build behavior. Do not expose a complete button
entry point or claim that a physical button has been built.

Use a private builder in package `sevensegment` that accepts a non-nil context
and document and a finite driver angle between 0° and 360° inclusive. Return
named references for the wheel, crank, drive rivet, input shaft and crank key.
Keep all bodies in the provided document and put them in named containers. Reject
invalid inputs with contextual errors. The builder must use the real bodies
and fully constrained sketches, with no fixture-generated replacement solids.

The initial test uses a real `Build` display at digit 9 and its actual shaft,
cam and slider bodies. It tests the wheel-to-shaft fit, the crank and rivet's
interaction with the real wheel across the index, and the real cam-to-slider
fits as the output advances. The existing manual knob, one-way pawls and manual
ratchet are not part of the button assembly; the scoped test must explicitly
exclude those obsolete bodies from its fit assertions. This boundary does not
prove physical assembly, output-wheel axial retention, dwell locking, push-button
pickup, reverse recovery or complete assembly clearance.

Verify every sketch with `Sketch.Verify`: fully constrained, zero DOF, no
redundancy and no conflicts. Test actual bodies with their verified tessellation
bounds included. The emitted boundary is an incremental implementation and does
not change the gallery model's unbuilt status or its production images.
