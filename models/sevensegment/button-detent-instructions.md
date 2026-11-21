# Seven-segment button detent-cam boundary

This is a scoped production contract for the input detent cam and its shaft key.
It extends the private Geneva index boundary in `button-index-instructions.md`;
it does not define a complete push-button assembly. Use millimetres and degrees.
The existing manual seven-segment build must remain unchanged.

Build the real keyed cam and a separate Ø2×20 mm steel key at input shaft station
37.25 mm. Use the existing `buildButtonIndex` output as the actual shaft consumer
and the `models/ballbearing` producer for the Ø3 bore, Ø6 outside, 2.5 mm wide
roller envelope. The complete follower is under separate qualification; do not
substitute hand-written fork or spring geometry or claim a complete assembled button.

The private builder accepts a non-nil context, document and finite input phase in
[0,360] degrees. It returns the actual cam and key as named body references in
named containers. Construct both bodies at zero phase and rotate the finished
bodies together about the real index shaft axis. Reject invalid inputs with
contextual errors. The actual input shaft and all new bodies must share one
document. Preserve the existing index and manual display behavior.

Follow `button-detent-steps.md` for the profile, constraints, feature order,
placement and checks. Verify every sketch as fully constrained, with zero DOF,
no redundant constraints and no conflicts before consuming it. Exercise the
actual cam, key, input shaft and purchased bearing at the stated phase positions;
charge the reported verified mesh bounds in all mesh comparisons. Geometry
checks do not establish spring force, physical assembly or fatigue life.
