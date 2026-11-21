# Seven-segment button release-crank boundary

This scoped contract defines the actual machined release bell crank, its three
retained working pins and its separate pivot journal. It extends the private
index boundary and does not define a complete button. The complete mechanism
contract is `button-instructions.md` and `button-steps.md`.

Use millimetres and degrees. Build the five actual metal bodies specified in
`button-release-crank-steps.md`. Verify the three pins against the real crank,
including both axial stops, and the pivot against its real crank bore. Exercise
five linkage lift positions against the actual existing digit-9 display and
emitted index boundary. Do not use a hand-written substitute for either real
producer. The printed fork, pivot bracket, plunger and spring are separate,
unfinished consumers; this boundary does not prove their complete assembly.

Provide a private package-local builder accepting a non-nil context and document
and a finite lift in [0,2] mm. Return named references for crank, input, output
and bias pins, and pivot. Keep the stationary pivot separate from the crank and
its working pins. Use the gear-to-world transform in the steps, named body
containers and one document. Reject invalid inputs with contextual errors.
Preserve all existing manual-display and index behavior.

Verify every sketch before its geometry is consumed: fully constrained, zero
DOF, no redundancy and no conflicts. Use real `VerifyAll` meshes and charge
their reported bounds in clearance and contact checks. Document the intentionally
modeled axial fits and physical staking, load, wear and assembly limitations.

The working pins have 0.02 mm radial clearance in their bores and 0.15 mm
radial clearance at each head. Their heads leave 0.08 mm total axial float
around the crank web. The pivot journal has 0.15 mm radial clearance in its
bore. These are modeled working fits and do not meet the gallery's standard
clearance classes.

Install and form each working pin before fitting the crank to its bracket.
The model does not establish the supplier stock, staking die, head strength,
heat treatment, loaded wear, or the pivot bracket's retention. It also does
not establish fork or plunger engagement, spring return, or anti-rotation.
