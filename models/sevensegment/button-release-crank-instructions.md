# Seven-segment button release-crank boundary

This scoped contract defines the actual machined release bell crank, its three
retained working pins and its separate pivot journal. It extends the private
index boundary and does not define a complete button. The fork, plunger,
spring, and pivot bracket have not yet been specified in this repository.

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

The working pins and pivot use the Regular fit class. The nominal radial
clearance is at least 0.36 mm and the head-to-web axial clearance is 0.38 mm
on each side. The pivot's right head clears the crank face by 0.38 mm. The
minimum crank ligaments at the input end, output end, and between working
counterbores are 0.52, 0.62, and 0.64 mm respectively. The remaining web is
1.32 mm thick.

Install and form each working pin before fitting the crank to its bracket.
The model does not establish the supplier stock, staking die, head strength,
heat treatment, loaded wear, or the pivot bracket's retention. It also does
not establish fork or plunger engagement, spring return, or anti-rotation.
Pivot and pin radial play can displace a working tip by up to 0.76 mm before
angular and assembly errors. Recheck engagement and bracket clearance when
those parts are designed.
