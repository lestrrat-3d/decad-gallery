# Spinning-wheel through-bore cut

Record of the failing consumer on2026-10-30. The model's instructions.md and steps.md own geometry;
the measurements and source hashes below capture reproductions, not new design requirements.

## Integration status on2026-10-31

The approved one-piece repair merged in [Decad PR2928](https://github.com/lestrrat-3d/decad/pull/2928).
All native CI checks passed. Both Gallery modules pin `v0.0.0-20261030221427-96bc82a990de`.
Against that production version, all five Gallery assembly cases pass, each with 142 final bodies and 155 fully constrained
sketches. The whorl retains eight native conical faces, exact D-bore material and tight analytical volume.
Every body passes VerifyAll; the whorl also passes at0.01 mm chord tolerance for its bore close-up.
The renderer integration passes and all actual model images plus the96-frame GIF are generated.
The required fresh spur emission uses the merged native revision. All focused spur checks pass and its record matches.
The old pedalspindle consumer's crank pivot check originally required `Circle3` cap rims.
Decad's public contract permits bounded `FacetedCurve` rims after a Boolean cut.
The check now requires the native cylindrical wall, stock radius and axis, two closed rims on opposite caps,
rim displacement certificates within 0.01 mm, and adjacency to both planar caps.
The old consumer's default and both scale variants pass with that check.
The blocker descriptions below are historical captures, not current repair instructions.

## Scope

Repair generic native through-bore cutting for stepped revolved pulley bodies in decad.
The user authorized contacting the existing Decad session for generic geometry fixes.
Do not add wheel-specific routing, shape-name checks, tolerance relaxation or unverified faceted substitutes.

## Real consumer

Gallery worktree `/home/lestrrat/dev/src/github.com/lestrrat-3d/decad-gallery/.worktrees/feat-pedal-spindle`.
Default spinningwheel.Build first fails in `models/spinningwheel/head.go`, builder.whorl,
at Cut(stepped full-revolve whorl, exact extruded D-shaped through-tool).
Native dependency dfe706cff5aa. Inspect source recipe in models/spinningwheel/{instructions,steps}.md.
Generated source is a producer candidate; do not hand-edit its geometry.

Reproduce: `go test ./models/spinningwheel -run '^TestSpinningWheelDefault$' -v -timeout 5m`.
Saved log `.tmp/spinningwheel-default-diagnostic.log`.
The last two live bodies at failure have 21 and4 faces:
host bounds(-43.226227447857326,-43.226227447857326,0)..(43.226227447857326,43.226227447857326,32);
tool bounds(-4.1,-4.1,0)..(3.1,4.1,32).

Error: `this chord tolerance asks for 11640 facets in one revolve mesh, whose pairwise audit exceeds
the fixed ceiling of 8000000 exact tests; retry with a coarser tolerance, or ask for a mesh that does not run that audit`.

The actual producer built the hollow shaft, radial port and two shaft flats before this failure.
Use the real whorl and tool as the initial failing pair. Generalize the native exact path to supported
coaxial stepped full-revolve targets and line/arc bore outlines; preserve containment, topology, units and rollback checks.
Verify D and other closed through-bore shapes, transformed placements, invalid/unsupported cases,
then pass the actual gallery assembly through VerifyAll.

## Revised full consumer

Gallery source now contains a second fresh emission with corrected collar clearances, guide orientation,
foot fasteners and real nut pockets. The first default case now stops earlier in `frame.go`, tension-carriage:
an X-axis8.6 circle is cut through the wooden carriage, followed by a short D-shaped16.2/flat7.1 nut pocket
on the same axis. This second Cut hits the held-mesh chain-depth guard before reaching the whorl.
Log `.tmp/spinningwheel-default-revised.log` contains the full bounded diagnostic.
The target's bound0.01960599794642134 exceeds the new pair's chord tolerance0.009818111960211576 mm.

Keep the independent minimal whorl repro when diagnosing its generic exact-through-bore gap.
For the carriage, first advise whether drawing its primary circle in a YZ extrusion section
or combining the connected tool features gives the required native geometry without a kernel change.
The parent will update the construction recipe and freshly emit any such construction change.
Do not relax the chain-depth guard. Gallery source must remain unchanged by the Decad session.

## Current consumer result

The fresh frame uses primary-sketch holes and1 mm cutter overruns. The default assembly now clears
all adjustment geometry. Against the local native repair worktree it stops at the real new whorl:
`.tmp/spinningwheel-default-native.log` reports9894facets beyond the same8M ceiling.
The real current whorl revolves in a +Y axial frame while its D through-tool extrudes in -Y,
with aligned radial flat orientation. This is valid antiparallel through-cut geometry.
Please cover opposite extrusion directions and the actual nonuniform four-groove outer profile,
not only common-transform placements of same-direction abstract step fixtures.

## Exact current public API placement data

Gallery base HEAD63c47049b8a48fe1343893a634ab1cf6b3066618; spinningwheel is an uncommitted new model.
Source hash head.go: debe481660b7ee9da05987630ad4c3d69479923fb21365fd4422369ab240955a.
Source hash sections.go: ef8ee138bc1e3df9ba15c4eb1446fee2a2ff5f969ef5b062732499604e8d2c18.
The actual paths above remain the producer source; no native payload is manually constructed by Gallery.

Default a=math.Remainder(7*pi/3,2*pi)=1.0471975511965974;
c=0.5000000000000003, sn=0.8660254037844385.
Host starts as FullRevolution of the radius/axial section on native world.XZ about local+Z,
axial range0..32. Host placement is r3.FromFrame of:
- origin(-246,-4,690)
- U=(c,0,sn)
- V=(sn,0,-c)
- N=(0,1,0)

Tool starts as an XY D outline with radius4.1, flat localX=3.1, extruded+Z over0..32.
Tool placement is r3.FromFrame of:
- origin(-246,28,690)
- U=(c,0,sn)
- V=(-sn,0,c)
- N=(0,-1,0)

Each Xform maps p to origin+U*p.X+V*p.Y+N*p.Z.
Host profile in radius/axial coordinates, closed back to the first point:
+- (0,0)
- (43.226227447857326,0)
- (43.226227447857326,1)
- (37.226227447857326,4)
- (43.226227447857326,7)
- (43.226227447857326,9)
- (34.45682846444463,9)
- (28.45682846444463,12)
- (34.45682846444463,15)
- (34.45682846444463,17)
- (26.783604353958523,17)
- (20.783604353958523,20)
- (26.783604353958523,23)
- (26.783604353958523,25)
- (20.013112491764904,25)
- (14.013112491764904,28)
- (20.013112491764904,31)
- (20.013112491764904,32)
- (0,32)

Real failure: spinningwheel: flyer-drive-whorl: cut native tool: this chord tolerance asks for9894facets
in one revolve mesh, whose pairwise audit exceeds the fixed ceiling of8000000exact tests.
See .tmp/spinningwheel-default-native.log for the complete bounded error and last two live body bounds.

## Remaining required generic class

The actual whorl outer meridian includes straight sloped segments, hence native conical groove walls.
A route limited to stacks of constant-radius cylinders does not accept this real producer.
The required bounded class is full revolutions of closed line-segment meridians with a solid central
core over the complete axial span, cut by a contained constant line/circular-arc extrusion.
The real bore radius4.1 stays below every outer radius; minimum root radius is14.013112491764904.
No free curves or toroidal outer wall are needed for this consumer.
Preserve the existing outer conical/cylindrical faces; replace the end caps with holed caps and insert
the exact prism's inner walls. Validate containment over every linear meridian interval, both extrusion
directions, placements, topology and measurements. This is a proposed scope for Decad's own design review,
not permission to skip its proofs or consumer test.

Please estimate the smallest generic conical-wall extension and confirm the actual producer passes before
calling the native prerequisite complete. The user authorized obtaining a generic Decad repair; the Gallery
deliverable remains blocked until this exact whorl and the full default assembly pass.

## Approved one-piece construction

The native session estimated4–8 engineering days for the conical-wall topology, measurements and hybrid tessellator.
Its minimal frustum/D-prism fallback returned a solid with zero Cone faces, so its analytic-surface regression failed.
The broad native implementation was stopped pending user choice on2026-10-30.
[Whorl options](spinning-wheel-whorl-options.md) describes the dimensioned18 mm bonded D-carrier alternative.
On2026-10-31 the user explicitly chose the one-piece whorl with the generic Decad extension.
The existing native session has been asked to resume that scope. Keep the current one-piece geometry.

Current Gallery worktree is feat-pedal-spindle, associated open PR119. New spinningwheel work is uncommitted.
Do not merge PR119 without user authorization.
The earlier pending-decision failure log is .tmp/spinningwheel-pending-decision-default.log.
Current successful logs are .tmp/spinningwheel-one-piece-all-checks.log and
.tmp/spinningwheel-one-piece-final-render-integration.log. Repository layout now passes.
Frame platforms now end before belt planeY0, with a forward crossbeam and leadscrew atY=-260*Scale.
Front yarn guide is atY=-224*Scale to clear the stationary maiden throughout rotation.

Require the actual one-piece whorl to retain eight native conical groove faces and the D-shaped through-bore.
Run one real default through VerifyAll before variants, renderer integration, images and animation.
Any production module-pin update also requires a fresh spur generation record per docs/model-pipeline.md.
After the native repair, finish images/docs, focused checks, repository layout, root/render lints,
PR119 update and CI watch. Do not ask again for the approved one-piece scope.
