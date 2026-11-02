# Spinning-wheel connector tessellation refusal

## Scope

Repair generic Decad history-backed refinement needed by Gallery PR119's real flexible connector.
The user explicitly prefers generic Decad fixes over Gallery workarounds and has authorized continuing prerequisites.
Keep all source geometry, hole sizes, requested accuracy and verification gates.
Do not special-case the Gallery model, part names or default dimensions.
Do not lower a held mesh bound by subdividing the same facets without a source-based error proof.
Keep this independent of the late wedge Cut repair in
[handoff-spinning-wheel-joinery.md](handoff-spinning-wheel-joinery.md).

## Current published construction gate

Decad PR3006 merged at `ad0caeffc8ee4578e81ede701398ce8ae2b6d91e` after 23 successful and three skipped checks.
Both Gallery modules now select `v0.0.0-20261102052910-ad0caeffc8ee` through the automatic updater.
The real connector completes 11 public operations, two Cuts, one live body and four constrained sketches.
Both 0.12 mm boundary/volume checks and the independent source-volume intervals pass.
The complete strict assembly next stops at the mother hinge after 225 operations; the joinery handoff owns that gate.
The staged preview includes this connector. Fine 0.05/0.01 mm Refine, motion checks and the rectangular-tool case remain pending.

The rectangular fixture's two private Cuts succeed. Its final Union requires an unproved gap shift on a shared cutter plane.
The strict warning gate correctly rejects that result. Do not treat this refusal as proved non-circular-tool coverage.
Continue the generic coplanar-Union proof separately without changing the fixture or allowing an unproved shift.

### Earlier published first-Cut refusal

The actual full default on published Decad `v0.0.0-20261102034530-29332da9647f` passes the gimbal,
then fails at `treadle-flex-connector` after 178 public operations, zero Coils and 76 live bodies.
The first public connector Cut refuses its target's 0.25802008593688447 mm held bound against the
0.0011721461110818445 mm pair chord. The test takes 0.421 seconds; Build has not returned.
This precedes the finer Refine consumers described below.

The run uses the actual current assembly source, no native overlays or local SDK replace,
an 8 GiB hard cgroup limit and a 240-second command limit.
Both Gallery modules select the same published version and module/go.mod checksums after the automatic updater.
The source and module inputs were hashed before execution. All 4252 recorded files match afterward and on independent reread.
The packet `.tmp/published-default-pr3005-29332/` contains the exact command, SDK metadata, manifests and original log.
Its before/after manifest SHA256 values are
`b23f2fa99aa66df8b51b0dec9aef6cc9380992f7b80a6928897a085b1cc36843` and
`7d703e9849981c3319ed4fb4515f540a136ff2dd0839967d19e825a56db12dfc`.
The failure log hashes to `b4225ded3a3d3a2b8c4a46b4e054f088efe06b63d9dd2568706d4b502f12fe0c`.
Gallery's independent check is `.tmp/published-default-pr3005-verified.json`.

A separate passive public-history observer confirms that this is the first connector Cut.
At zero-based indices 171/172 and 173/174, the original two legs are extruded and transformed.
Operation 175 Unions outputs 172 and 174. Operations 176/177 create and place the first hole tool.
The Cut fails before any connector Cut is committed. The second tool has not been constructed.
The observer changes only failure logging in one test overlay and leaves all original producer/module bytes unchanged.
Its separate packet `.tmp/published-default-pr3005-history/` verifies 4255 inputs before and after execution.
The before manifest hashes to `8f218fbe3f674a0930c8ee7e9e211e2141d02e34cb545fd711a08a216a4fd8c9`;
the diagnostic log hashes to `85f2c48b800764c0200c691e5f2bd7a91e4b979d72b6aadb2523fdc11814285f`.
The original run above remains the authoritative unmodified default result.

The frozen actual caller is `.tmp/connector-construction-producer/gallery/`.
Its 159-file caller manifest hashes to `9617e31d4a16c66839af745941b778debc2038b04e9ba83f9275ba112241600b`.
Every Go/module file matches the original run's source manifests. Five authority documents are explicitly post-run captures.
The packet copies both original run records and the separate observer records, and selects the same published module.
Use its real `BuildConnector` producer for the isolated repair proof; it calls the same `flexConnector` as the assembly.

Repair this actual public construction gate generically in Decad before rerunning the complete default.
Keep both original connector Cuts and their geometry. Coordinate with the active bounded connector replay owner;
its later fine-replay work alone does not prove that the published construction dispatch accepts this Cut.
No variants, Refine consumers, pose screens or images were run after this failure.
The successful constructors and refinement diagnostics below use earlier complete native candidates.
They are historical evidence, not proof that the current published constructor succeeds.

### Ordinary construction dispatch comparison

The published29332 `boolean.go` tries the ordinary pair, brake retry and generic regroup, but has no
`refinedUnionCut` or `refinedPairedCut` dispatch.
The complete frozen SDK in `feat-mother-base-blind-group-cut/.tmp/mother-base-freeze-6/base`
contains those routes in `boolean_refined_cut.go` and calls them before ordinary regroup.
`refinedUnionCut` authenticates a two-source Union, privately Cuts both original sources at a finer chord,
joins them and verifies the result through `tessellateBooleanOperand` before returning a three-producer result.
That helper raises the request to the result's held floor. It does not prove a mesh at the original pair chord.
`refinedPairedCut` handles the following Cut on that recorded source tree.
The isolated causal diagnosis below now confirms that this dispatch difference causes the first construction refusal.
Review a generic ordinary-construction extraction separately from the bounded finer-Refine implementation.
Keep all resource, warning, history and transaction requirements; do not import every frozen route.
Neither that investigation nor the bounded first-pair work accepts the real public construction yet.

### Ordinary construction causal diagnosis

The separate native scratch packet is `decad/.tmp/connector-refined-diagnosis/`.
Its baseline calls the real frozen public `BuildConnector` against published29332 and fails at seven operations,
two live bodies, with the same first-Cut held-bound refusal as the full assembly.
The comparison changes three SDK files only: the first retry dispatch, its source-authenticated helper,
and a chord-scale wrapper around the existing ordinary evaluator. It changes no kernel code, caps or Gallery geometry.
The trial uses a byte-checked published SDK copy because Go rejects overlays within the module cache.

The first public Cut then succeeds at zero-based operation seven, with zero warnings and 0.12 mm VerifyAll.
The unchanged constructor proceeds to its second Cut and refuses after ten operations and two live bodies.
That target's held bound is 0.0623011296519062 mm; the pair chord is 0.0011209307785054448 mm.
The baseline and trial take 0.61 and 5.47 seconds under race checking and hard 8 GiB/240-second limits.
The trial's passing diagnostic assertion records the expected next refusal; it does not mean BuildConnector succeeds.
No production SDK edits or full Gallery rerun follow this diagnosis.
The native coordinator must review the route, resource contract and publication scope before implementation continues.
The bounded finer-Refine repair remains independently required.

## Current shared prerequisites

Decad PR2999 publishes bounded rational-conversion accounting at
`68ac09f9740f9119e2164d2f0415a3a6b595316d`.
PR2998 publishes the shared circle primitive candidate and construction tokens at squash commit
`ac7341e3d0f2a9eadcf5203853552d69af288fd5`.
Gallery independently checks PR2998's final remote head `c90a62591fc7a9a15cad3b6edb6beab9674d7526`:
22 checks succeed, three skip, and none remain pending or fail.
The saved remote evidence is `.tmp/gh-native2998-final-ci.json`.
These helpers publish neither a Boolean result nor a public Refine result.

PR3001 publishes only `ReplayStoreMax`, `ReplaySymDiff` and their tests at squash commit
`96bcd2469e7cf2e2f7f3960568207c6b78f4a998`.
Gallery independently verifies its final head `a24521ec6678b446dce933fb5934a80d491e4f0e`:
22 checks succeed, three skip, and none remain pending or fail.
The saved remote evidence is `.tmp/gh-native3001-final-ci.json`.
This helper publication changes no dispatch, proof flag or cache admission.

PR3000 publishes the ordered source/history/placement adapter at squash commit
`d81258578054f965d8e12b940849f2ac22b3ed62`, based on merged PR2998.
Gallery independently verifies its final remote head `22183ba4426c5cad3c3d0197057aa24c09e0b15e`:
22 checks succeed, three skip, and none remain pending or fail.
The saved remote evidence is `.tmp/gh-native3000-final-ci.json`.
The native owner reports real public Extrude plus two noncommuting Transforms through the private candidate,
exact and three one-short controls, cancellation, and rational displacement/area/volume/chord checks.
Gallery reads the producer tests, source replay and earlier focused logs; it separately verifies final remote CI.
The test also constructs a reference candidate, so its logged cost is not one production candidate path's cost.

The next native slice integrates the ordinary connector retry against this published adapter on a clean worktree.
The adapter alone does not wire the frozen `refinedUnionCut` and `refinedPairedCut` routes.
Do not rerun their already-known clean-baseline refusal and call it a new Gallery geometry defect.
The actual unchanged Gallery connector, complete ordinary retry, public Refine and both accuracy consumers remain pending.

## Real producer and failure

The actual producer is `spinningwheel.BuildConnector(ctx,doc,spinningwheel.DefaultParams())`.
Its construction is the same `builder.flexConnector` used by the assembly.
It Unions two posed radius3 circular legs, then Cuts both stock radius2.2 transverse clamp passages.
All dimensions and frames come from the current Gallery public constructor and Motion API.
Do not hand-write substitute geometry. Read mounting.md for the construction contract.

The public constructor succeeds and leaves one live solid.
The first focused `TestSpinningWheelConnectorDefault` fails at its next real consumer:

```text
requested tolerance 0.05 mm is below the faceted body's minimum mesh bound 0.07583204301757442 mm;
retry with a tolerance of at least 0.07583204301757442 mm to restate the held mesh
```

The test uses `Tessellate(.05,WithVerification(VerifyAll))`, matching the actual animation's connector request.
The 72-pose clearance screen requests0.01mm for each rebuilt connector.
That screen has not run; the standalone default sequence stopped at its first0.05mm refusal.
No boundary, volume, topology, material or sketch assertions after tessellation have passed for this test.
The failure took3.75seconds and exited1 normally under an8GiB virtual-memory cap and240second timeout.
Evidence: Gallery `.tmp/spinningwheel-connector-default.log`.

## Native diagnosis

The native session reproduced the same public failure. Its immediate owner is
`internal/facetedrecord/restate.go`: tessellation sends faceted bodies straight to Restate without consulting
operation history. Boolean replay has history lookup and expansion eligibility checks, but tessellation does not
currently use them. This connector's eligibility for a complete generic replay is not yet proved.
Native evidence: `.tmp/connector-starter-diagnosis.log`,3.82seconds for the real test.
Compilation and testing took19.94seconds with peak RSS422016KiB. No later geometry assertions or pose screen ran.

The native design now prefers a generic source-history refinement operation returning a fresh verified Body.
It would consume the live source only after a successful replay, use fresh face identities and preserve one live body.
The original body's below-bound Tessellate refusal would remain correct. This is a design, not an available API.
The native design owner is `.tmp/connector-refinement-design.md` in the full-candidate worktree.

On2026-11-02 Gallery confirms that design's public signature:
`Refine(ctx context.Context, source *Body, maxBound units.Value) (*Body, error)`.
The fresh result belongs to the source document; success consumes only the old source and records a real Refine operation.
Failure or cancellation leaves history and live bodies unchanged. Unrelated assembly bodies stay live.
The operation must follow the source subtree inside a larger document;11operations is diagnostic evidence, not an API guard.
This is an agreed contract, not an implemented or accepted public API.

The current Gallery constructor and tracked test do not call Refine and have no missing-Refine compile gate.
The first native consumer must add the agreed call after the real BuildConnector in a scratch test overlay.
Keep the production constructor unchanged while proving the0.05mm path, then the real0.01mm consumer.
Require one live connector and compare the returned body pointer directly with that live body's pointer.
Report bounded counts or IDs on failure; do not pass body graphs to a deep equality or pointer formatter.

After the real native producer/refinement/VerifyAll gate passes and the API is agreed, update mounting.md and steps.md,
then obtain fresh connector emission to use that generic operation. Do not hand-edit emitted construction.
Require the returned body to belong to the supplied document, with no superseded body left live.
The0.05mm animation and0.01mm clearance requests need separate real proofs; one does not establish the other.

## Public history diagnostic

The unchanged real default constructor records11public operations. Its Union and two Cuts occur at0-based
operation indices4,7and10. Their ordered inputs form exactly`Cut(Cut(Union(L,U),H1),H2)`.
All three output bodies report zero public operation diagnostics, including both retired intermediate outputs.
No geometry-changing warning was recorded in those three operations.
This checks one prerequisite of source-history refinement. It does not prove immutable payload binding,
private Expandable eligibility, finer replay, final mesh verification or preservation of the source set.

The bounded public-API diagnostic took4.03seconds and passed. It adds only a scratch test through an overlay;
it changes no construction and requests no alternative tessellation tolerance.
Evidence: Gallery `.tmp/spinningwheel-connector-history.log`.
Fixture: `.tmp/spinningwheel-connector-history_test.go` and `.tmp/spinningwheel-connector-history-overlay.json`.

## Candidate refinement gate

The initial native full-candidate probe passes private Expandable checks for all three Boolean nodes,
with producer IDs4,9and16 and zero warnings. All four primitive leaves are solid one-circle prisms.
Their recorded section and axial deltas are zero, and their sketch frames are identity frames.
The actual placements include rotations. That initial generic leaf replay refused with
`bounded circle requires translation placement`, before charging work, records or bytes.
That attempt produced no finer connector mesh or complete refinement.

Evidence is in `.tmp/connector-refine-leaf-diagnostic.log` and `.tmp/connector-refine-leaf-fields.log`
in the native `feat-connector-source-refine` worktree. The unchanged0.05mm default consumer still refuses
the0.07583204301757442mm held bound, as recorded in `.tmp/connector-refine-default.log` there.
Later ordinary-path evidence follows below. It does not make the unchanged public constructor refine its result.

## Ordinary-path diagnostic

A scratch native ordinary replay now passes the real Gallery test's0.05mm mesh, topology, bore, material and sketch checks.
It retains the actual public BuildConnector producer and inserts a diagnostic refinement call before those checks.
The original constructor still does not invoke refinement.
The successful mesh has49542triangles and held bound0.02433759374448569mm.
The test took68.37seconds under the native diagnostic process limits.
Evidence: native `feat-connector-source-refine/.tmp/connector-finite-schedule.log`.
Its overlay, wrapper and helper are `.tmp/connector-functional-overlay.json`,
`.tmp/connector-functional-test.go` and`.tmp/connector-functional.go` in that worktree.

Uniform replay of the raw ordered Boolean tree fails at the first Cut, although the original public constructor succeeds.
The original public Boolean path uses a distributive Cut retry over both Union branches.
The successful diagnostic authenticates that ordered source tree, repeats the distributive route,
and refines the final Union along with its input branches until the actual final bound meets the requested accuracy.
Refining only the branches left the final Union above0.05mm.

This scratch helper uses ordinary unmetered replay and commits an OperationCopy.
It is not the designed public Refine operation and does not establish private-producer remapping or full replay accounting.
The diagnostic test checks one live body but does not yet compare that body's identity with the returned refined body.
Require that identity check when consuming the final API.
The0.01mm clearance consumer and72-pose screen have not passed.

## Bounded replay gate

A separate count-only run passes rotated circle-source construction and reaches the first distributed Cut.
It then refuses with`bounded operand tag placement requires a recorded rectangle`.
The partial totals are1413631455work,19853948records and3415112120cumulative bytes.
The run took4.60seconds. Evidence: native `feat-connector-source-refine/.tmp/connector-measure-real.log`.
These partial totals do not cover a complete connector replay and cannot set a complete admission budget.

Subsequent source-cache review requires copied arrays, precharged validation and rebuilding the source walk before reuse.
The first reviewed two-source trial exhausted its32000000work limit in the second source's contact audit.
Evidence: native `feat-connector-source-refine/.tmp/connector-circle-cache-reviewed-tests.log`
and`.tmp/connector-shared-circle-reviewed.log`.

The subsequent source-plane exclusion proof admits both real source meshes within that unchanged work limit.
The two-source trial reports26234117work,1102610records and96785184cumulative bytes after both contact audits pass.
It takes4.50seconds. Evidence: native `feat-connector-source-refine/.tmp/connector-plane-real.log`.
This is a source audit; it does not complete a Boolean, a refined connector or a public Refine commit.
The separate placed-circle probe also accepts all four original primitive sources at its coarse diagnostic chord.
Evidence: native `feat-connector-source-refine/.tmp/connector-placed-real.log`.
That probe alone does not discharge fine-chord Boolean replay or the complete operand-placement gate.

The next count-only private replay completes both distributed Cuts on the first Union branch.
It stops during the second branch's first Cut at the source-contact audit's fixed8000000pair ceiling.
Its partial totals are13669849059work,680562366records and58875350962cumulative charged bytes.
The test takes38.06seconds; the command takes45.88seconds and peaks at417600KiB RSS.
Evidence: native `feat-connector-source-refine/.tmp/connector-full-measure.log`.
This run publishes no result and never reaches the final Union, public Refine commit or0.05mm VerifyAll consumer.
These are still partial costs. They cannot define a complete Refine admission budget.
Continue the full-route diagnostic under hard8GiB/240second process limits while keeping existing Boolean defaults unchanged.

The 2026-11-02 full-route count diagnostic passes that earlier source-contact ceiling.
It completes three distributed Cuts, then exhausts a diagnostic 100000000000 work allowance in the fourth Boolean.
The third Cut completes at 16739007281 work, 743306112 records and 70186989836 cumulative charged bytes.
The fourth Cut's source audit completes at 17046456479 work, 750611075 records and 70697644406 bytes.
Its Boolean then consumes 82953543511 additional work units before refusal at 99999999990 work,
1233853802 records and 85376514324 cumulative charged bytes. The test takes 84.43 seconds.
These byte totals count cumulative charges, not peak resident memory.
Evidence: native `feat-connector-source-refine/.tmp/connector-full-route-cost-20261102/run.log`,
SHA256 `e1cda643adac975e1b5ee3d6c42a56e8041fe291026b2c4f2502f56532f6fc1e`.
Gallery independently reads the log and verifies its hash.
The owner is tracing the fourth Boolean's internal stages before another diagnostic run.
The final Union, public Refine publication and final VerifyAll have not run. No production budget changed.
Do not treat the partial 100 billion work total as a complete admission cost.

The next scratch source-bound projection tree completes the fourth Cut's proximity check.
It visits 550280 nodes, prunes 38292, and submits 238496 facet pairs.
Proximity completes at 18417253471 work. The following `MeshBoolean` pair-cut stage exhausts the diagnostic
25 billion work allowance at 24999999965 work, before a fourth Cut result or final Union.
The test takes 45.91 seconds. Evidence: native
`feat-connector-source-refine/.tmp/connector-projection-node-20261102/run.log`,
SHA256 `0898a27ad5c9ebea397f80713f4a4d1fbd8132f993d4beedc27f8a37f63e0c04`.
Gallery independently reads the log and verifies its hash.
The adapter omits 2266 candidate-record reservations; its reported record totals are incomplete.
It also needs production source sealing across the package boundary. This is stage evidence, not admitted replay.
Correct that accounting before deriving complete work/record/byte admission limits or publishing Refine.
The following review also finds that `BoxesWithin` reads unsealed `BoolMesh.Boxes` after the tree's source capture.
Stale boxes could omit a true contact even when the projection tree itself matches its captured vertices.
The native owner is binding or recomputing those boxes from the sealed mesh and sealing the resulting near-contact set.
Require both checks before accepting the filter in production; the scratch pass alone does not prove them.

The native owner must continue the complete bounded route, including generic operand placement,
all distributed Cuts, the final Union, verification and an atomic public Refine commit.

A later generic three-prism scene fixture encounters a crossing-association proof gap at rational coordinates67/23and61/23.
That scene is not this actual connector producer. It is also not the brake cord's Sweep/Revolve Union.
Do not report that fixture's refusal as an attempted Gallery connector or cord refinement.
If its witness is a real prerequisite, name the actual source-to-consumer call path that needs it.
Keep the actual connector and [cord refinement](handoff-spinning-wheel-cord-refinement.md) acceptance gates separate.

## Production resource acceptance

The latest first-pair audit includes required seals missing from the earlier scratch callback measurement.
Both circle candidates alone had reached a refusing callback at 30278192 work;
the later 31392006 Boolean-entry result omitted the held-Union and candidate mesh seals.
With those seals included, held-Union capture ends at 1925216 work and the left mesh seal at 22370406.
The tool's radial scan then exhausts at 31999619 work, 1391801 records and 138413146 charged bytes.
It reaches neither the source/held recheck nor Boolean collection. Those earlier entry totals cannot set admission.
The implementer is reducing repeated cap-rational copies and reviewing a generic bounded source proof.
The actual source has 598 vertices and 1192 facets; the separate brake source has 2582 vertices and 5168 facets.
Its circle-construction witness must not admit the brake mesh or an unbound facet-role label.
All remaining Cuts, stitching, output verification and public publication still require full cumulative measurement.

The production `Refine` creates `proofbound.NewReplayTrialBudget`.
Its source constants permit 32000000 work, 4194304 records and 2147483648 cumulative charged bytes per replay.
Gallery independently reads `refine.go` and `internal/proofbound/replay_budget.go` in the native worktree.
The scratch `ConnectorMeasurementBudget` does not replace those production limits.
The larger diagnostics below establish intermediate geometry and failure locations only.
Even their first private output exceeds all three production limits; they cannot admit a public Refine result.
Reduce the generic route's charged costs without widening those limits or omitting verification.
Require the complete real connector at 0.05 mm and then 0.01 mm through the production meter.

The next frozen first-source profile isolates costs before the first Boolean.
Its 1068 stations preserve the existing requested chord; source sampling alone takes 40226878 work.
The complete first-source audit takes 249827039 work, 6745086 records and 470161086 cumulative bytes.
Both work and record limits fail before the second source. Contact filtering alone cannot admit even this first source.
The native profile identifies station/placement work, vertex lifts, cap triangulation and contact partitioning as separate costs.
Gallery reads the stage counters and preserves `.tmp/spinningwheel-connector-source-profile-e368a665-20261102.log`,
SHA256 `e368a665d5e2f429eafc4b78a2c62ddedb821d89f5615aa1c49dbb5605e1c0ae`.
The test intentionally returns a diagnostic sentinel after the first source; it is not a passed production call.

## Intermediate-output verification

The next native gate checks an actual private Cut result before allowing the following Cut to consume it.
The original circle-leaf seal cannot authenticate a faceted Boolean result.
The required source, producer, face-table, contact, meter and publication contract is native
`feat-connector-source-refine/.tmp/refine-faceted-intermediate-capability.md`.

The real first distributed Cut returns producer 17 with four faces, 8032 vertices and 16064 facets.
Its new metered topology and exact held-mesh volume checks pass; the held volume is positive.
The following facet-contact audit stops at the unchanged 8000000-pair ceiling.
At refusal it reports 10899428178 work, 109594452 records and 15956187376 cumulative charged bytes.
The test takes 21.63 seconds. Evidence: native
`feat-connector-source-refine/.tmp/connector-output-contact-20261102/run.log`,
SHA256 `604ddef999fd8eb06050a5dc55ab48aee1abac496c5cc7a0be98a2c25a3e90a1`.
Gallery independently reads the log and verifies its hash.

This new intermediate gate produces no output capability and does not reach the following Cut.
The earlier diagnostic's three completed Cuts and fourth-Cut proximity pass did not include this complete gate.
They do not prove that a production Refine can consume those intermediate outputs.
Finish the generic contact proof within the existing ceiling, authenticate the result, and recheck that capability
at the next real Cut. Keep the final 0.05 mm VerifyAll and then 0.01 mm acceptance gates.

The following frozen actual first-output run passes topology, positive exact held volume, facet area and contact checks.
Generic three-dimensional candidate filtering leaves 3348599 classified pairs under the unchanged 8000000 ceiling.
It reports 15527570928 work, 150978331 records and 24656292816 cumulative charged bytes in 23.77 seconds.
Gallery saves the observed log as `.tmp/spinningwheel-connector-audit-c1480471-20261102.log`,
SHA256 `c14804713e32024fc113c15345266cd0d317654fbb21fe9189d3397605af6c1a`.
Gallery independently reads that log and verifies its hash. This closes the geometric contact-audit refusal above.

A subsequent actual held-output snapshot probe initially refused before its audits because it required a nil body owner.
The real private output has a valid non-nil owner; the corrected seal captures and rechecks that identity.
Gallery preserves the refusal as `.tmp/spinningwheel-connector-held-refusal-75de2c7e-20261102.log`,
SHA256 `75de2c7eb8dd3efc2d6eda6ef3ef5c4c68a2fbdfd410685f1fd7a4edf3f744bb`.
The corrected actual output passes held-geometry checks and rejects a changed vertex or a different meter.
Later probes also bind every output facet to an ordered input face and recheck the original source graph/history.
The latest combined probe passes with 15533358404 work, 151067337 records and 24658524160 cumulative charged bytes
in 23.47 seconds. Changed source ordinal, output vertex and meter identity refuse before restored success.
Gallery preserves that completed log as `.tmp/spinningwheel-connector-combined-e90e8dcc-20261102.log`,
SHA256 `e90e8dcce3d8bc8299dc7004b5084bb592761e89bb7816f6e82f6cbe11498fb9`.
The owner-only, history and lineage intermediate logs are also copied under their hash prefixes in Gallery `.tmp`.

The next actual-output probe also captures retained Plane/Cylinder tags, per-face normal/departure fields and CapRecords.
Changing a face normal bound or cap level refuses; restoration and history rechecks pass.
It reports 15540322413 work, 151067432 records and 24658544152 cumulative charged bytes in 23.03 seconds.
Gallery independently reads the log and preserves `.tmp/spinningwheel-connector-tags-e30d81e6-20261102.log`,
SHA256 `e30d81e6fbdf7318e248b3054c8ba5e05fe9987c595d661cf7fbc5e5f8454b25`.
Production resource acceptance above explains why this diagnostic cannot admit the result.

These probes have not minted a complete source/output capability. Source leaf construction seals, complete payload/bounds,
original input denotation and a private Boolean execution witness remain unconnected.
Truthful source and volume-error bounds, atomic publication and the 0.05/0.01 mm consumers remain open.

## Native baseline comparison

Against Decad commit`9fcb86432`, the same Gallery constructor fails at its first Cut, before the refinement request.
It records7operations and only the Union at index4. The held bound is0.25802008593688447mm;
the pair chord is0.0011721461110818445mm.
The full candidate's11-operation producer success therefore depends on earlier native bounded-cut support.
The native session must extract that prerequisite before publishing a clean-baseline Refine slice.
Keep the real refinement proof moving separately in the preserved full candidate.

The native ordinary constructor/history gate is prepared in Decad root `.tmp/connector-prerequisite`.
Its `status.json` records separate implementation and verification owners; no source or acceptance run is admitted.
Gallery reads the strengthened fixture and verifies SHA256
`83cf23ae6d7ef09c30a6fc64f0dd0511426f9332217685dd9a3fedc0d3e54fb9`.
Run it only against the coherent metered retry extraction. The old unmetered producer pass is historical evidence.

## Reproduction

Use the real Gallery worktree:
`/home/lestrrat/dev/src/github.com/lestrrat-3d/decad-gallery/.worktrees/feat-pedal-spindle`.
The candidate modfile points to the native full-candidate worktree; the production native pin is unchanged.

```sh
ulimit -v 8388608
GOWORK=off timeout 240s go test -modfile=.tmp/spinningwheel-arc-candidate.mod \
  ./models/spinningwheel -run '^TestSpinningWheelConnectorDefault$' -count=1 -v -timeout 3m50s
```

The public connector construction is frozen while the native session diagnoses this failure.
The new test consumes the actual returned body. It requires one connected verified solid, two through-hole handles,
both actual transverse passages, material around each bore and fully constrained consumed sketches.
The sequence must pass before expanding connector cases. The final assembly,72poses and actual animation must
then consume the repaired constructor and tessellation path before claiming end-to-end success.
