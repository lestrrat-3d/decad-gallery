# Spinning-wheel brake-cord refinement

## Scope

Repair generic Decad source refinement needed by Gallery PR119's actual brake cord.
Use the standing generic-repair authorization in [the joinery handoff](handoff-spinning-wheel-joinery.md#scope).
Preserve the physical cord, fused terminal eyes, source dimensions and requested verification accuracy.
Keep this separate from the mother-base Cut and connector refinement implementation slices.

## Real producer and consumer

The producer is the unchanged `builder.brake()` at `DefaultParams()` with the actual `Motion` state and head tilt.
The scratch test calls this private production helper because the complete assembly stops later at unrelated frame joinery.
It constructs all brake parts, then selects the actual returned `brake-band` Body.
No nominal or substitute cord geometry is constructed by the test.

`builder.brakeCord` constructs a native circular-profile Sweep along its real seven-span LineTo/ArcAround path.
It Unions that cord with two native Revolve terminal eyes, including the right eye's actual take-up rotation.
The intended refinement therefore needs source histories beyond the connector's circular-prism leaves.

The first consumer is `Body.Volume()`; the second is
`Tessellate(.12mm,WithVerification(VerifyAll))`.
The latter exactly matches `TestSpinningWheelNativeDefault`'s required per-part consumer.

## Default evidence

The actual brake construction and volume query succeed.
The cord volume is836.3932691662151mm^3 with bound21.73436666281278mm^3.
Its reported interval is[814.6589025034023,858.1276358290279]mm^3.
Subtracting the design's scale-dependent volume term leaves residual-4.990842817159319mm^3 with that same bound.
This single reading does not prove material conservation, and overlapping broad intervals would not prove it either.

The next actual consumer refuses:

```text
requested tolerance 0.12 mm is below the faceted body's minimum mesh bound 0.19602373443374732 mm
```

The test stops at this first failure. Mesh boundary/volume verification and subsequent sketch assertions do not run.
No alternative tolerance, parameter variants, pose screen or complete assembly rerun follows it.
The test takes36.44seconds; the command takes39.69seconds and peaks at405312KiB RSS.
It exits1 normally under an8GiB virtual-memory limit and240second timeout.

Evidence in the Gallery worktree:

- `.tmp/spinningwheel-cord-volume-default.log` preserves bounded scalar output and the failure.
- `.tmp/spinningwheel_cord_volume_test.go` contains the scratch test.
- `.tmp/spinningwheel-cord-volume-overlay.json` adds that test without replacing production code.
- `.tmp/spinningwheel-cord-snapshot-before.log` and`spinningwheel-cord-snapshot-after.log` record successful manifest checks.

The native source is the preserved `decad/.tmp/native-default-wedge-retry-overlay-20261101` snapshot.
Its source manifest, Gallery modfile and sum hashes remain unchanged before and after this run.
The joinery handoff's Current status owns those snapshot hashes.
No tracked Gallery constructor, test or module file changes for this diagnostic.

## Actual source-history capture

On 2026-11-02 the unchanged default `brake()` producer passes a targeted runtime history capture in 42.93 seconds.
Its 108-operation document contains 17 brake parts; nine operations are reachable from the returned brake-band.
Those operations are Sweep 9, Transform 10, Revolve 11, Transform 12, Revolve 13, Transforms 14 and 15,
the ordered three-input Union 16 and the final Transform 17. The indices identify this diagnostic only.
Every reachable output reports zero private warnings and zero public operation diagnostics.

The Union output is a faceted body. Its Boolean Expandable record names exactly the placed Sweep and both placed
Revolves in that order. The returned final Transform output is also faceted but is not directly Boolean-Expandable.
Generic refinement must authenticate that final Transform edge before consuming the Union's source history.
This observation does not prove primitive seals, the private Union evaluator, replay or atomic publication.

Gallery independently reads the test, probe and log, then preserves
`.tmp/spinningwheel-cord-history-e7f77eca-20261102.log`, SHA256
`e7f77eca8bd2557ccba363e6b01878c9968f89b1b841801970fb25a7835effbf`.
The native `.tmp/brake_history_capture_report.md` owns the command and run-local pointer ledger.
Its eight selected producer/module/harness hashes match before and after; this is not a complete SDK freeze.
The command uses the live full-candidate worktree and a 240-second timeout.
This test calls no Volume, Tessellate or Refine consumer. The finer mesh and conservation gates below remain open.

## Remaining gates

The native design/tasklist is `feat-connector-source-refine/.tmp/brake-cord-gallery119-prerequisite-tasklist.md`.
It records the final tilt Transform and ordered three-input Union as source-history gates beyond the two primitive types.
The native owner is now porting the missing ordinary mapped-path evaluator and its route witness
in the isolated `decad/.worktrees/feat-union-many-source-witness` worktree.
Generic Sweep/Revolve source seals, bounded meshing and private Union-many route witnesses are explicit prerequisites.
The runtime capture above supplies public operation order, warnings and Boolean expansion inputs.
Private association, immutable source/face witnesses and all finer-mesh gates remain missing.

The later actual producer capture observes the private three-source mapped-path Union rather than a pairwise fold.
Its public input order is placed Sweep10, placed Revolve12 and placed Revolve15; evaluator order is[1,0,2].
It creates producer16 with span1, followed by the final Transform17.
The output has6760vertices,13524facets and11faces, with valid observed facet-to-face/group indices.
Its reported1156064path work excludes complete source, storage, stitch, construction and verification costs.
Gallery reads the native `.tmp/brake-route-capture-report.md` in that isolated worktree.
The report records an18942-file closure and before/after checks; Gallery has not independently verified that closure.
The capture is not a sealed primitive-owner or immutable route witness, and supplies no accepted finer mesh.
Current main and PR2996 lack that ordinary evaluator and its path/multiway support.
The owner must port a coherent generic slice, then bind the actual route/ordered inputs/span/source faces atomically.
The final faceted Transform witness can follow separately; circle-prism placement snapshots cannot authenticate it.

The isolated port later reaches the actual cord's three-source Union with13524source-owner records.
Its ordinary path evaluation and diagnostic VerifyAll pass; this is not the refused0.12mm consumer or bounded Refine.
Both five-source spring Unions subsequently pass ordinary evaluation using pair-local held-source gates.
The captured spring order is[Revolve21,prism25,Coil19,prism27,Revolve23], with49180facets in the held Coil.
That Coil's held floor is0.0082093852568275588mm; pair-local mesh copies preserve original sources and warnings.
Full `brake()` then stops at the take-up screw's binary Union, whose held source exceeds its pair chord.
The owner pauses new binary implementation while comparing this with the already passing PR2996 screw callback.
Use that coherent dependency if the source/history/warning records match; do not create a second dirty replay variant.
The current comparison still needs exact recorded warning codes and field values, rather than cross-document pointer equality.
The route witness, final Transform witness, full bounded replay,0.12mm cord and conservation gates remain open.

Gallery's `.tmp/goal-cord-invariance-plan.md` now derives the seven actual source span lengths and endpoint placement.
It identifies complementary straight tails and a right-eye rotation about the circular tail's axis.
That source-level derivation is not accepted native material proof; bind it to the actual primitive records,
rounded placement and nonlocal separation before claiming conservation.

Authenticate the actual warning-free ordered Union inputs and original Sweep/Revolve source records.
Prove eligibility before adding generic refinement support for those source types.
Reuse the native Refine operation if its source and error proofs admit these types; do not add a brake-specific evaluator.
Do not relabel or subdivide the held facets to lower their bound without a source-based error proof.
Require the real source producer and0.12mm VerifyAll consumer to pass with truthful displacement and volume bounds.
Then update the construction authority and obtain fresh emission if the model needs an explicit refinement call.

The mount authority also now requires0.5stock nominal clearance for the cord's captive terminal supports.
The current emitted brake-band record still uses0; fresh joinery emission is queued after the source freeze releases.
This is existing eye free play, not a geometry change. Gallery `.tmp/goal-cord-retention-clearance.md` records its derivation.
Use the actual verified meshes to check that the spring supports provide a path home after refinement.

The pending material-conservation check is in Gallery `.tmp/goal-cord-invariance-plan.md`.
It still requires actual native evidence across the existing assembly cases after the default integration succeeds.
If the volume uncertainty cannot distinguish a stretched tail, prove the actual tail sections and rigid splice geometry.
Do not claim conservation from the centreline formula or from broad overlapping intervals.
