# Decad D-coupling construction failures

## Current gallery status

The real default gallery consumer passes on published Decad `c5375fa37f6866b5a641b8679d4103422e979092`
and Sketch `6dfdb462bc53`. The gallery does not require the additional D-only loft optimization in Decad PR2881.
Both production gallery module files now pin that published Decad version; no local replace is committed.

The input shaft uses one combined flat-cut tool. Bore checks follow the actual OperationHistory to the D cutter,
then read its circle-prism and rectangle-prism inputs with Body.Section at1e-6mm. The derived tool and final
gear contours use SectionOutline at0.01mm, with OffSurface below0.01mm, unchanged positive-clearance checks,
and a90degree relative-rotation interference control. Journal checks prove the flat tools miss the slice and
verify actual output roundness. Nominal0.1mm fit clearances are unchanged.

Whole-gear exact Body.Section remains unsupported for its53-input tooth/root Union; this is not claimed as fixed.
The mesh-read refusal below the held bound was correct. A bounded D profile is not silently treated as exact.
The original reports and integration attempts below record the diagnosis.

## Clearance integration diagnosis

The focused coupling, scale-variant and 216-pose linkage checks pass on the published dependency above.
The previous `TestPedalSpindleFootmanClearsInputShaft` reported `undecided_clearance` with no clearance row.
It isolated the actual input shaft and footman at the upper crank position. Their axial faces have a nominal
5 mm gap: shaft front Y=-116 mm and footman rear Y=-121 mm. The report also marks area, volume and centroid
measurements beyond tolerance. The full diagnostic explicitly says the pair is proven disjoint, but its
requested closest-distance measurement is unavailable. The generic verifier's box-separation path already
settles intersection; it cannot supply this faceted/analytic pair's closest-distance reading.
The bounded diagnostic log is `.tmp/d-shaft-clearance-diagnostic.log`.

The regression now checks the actual bodies' conservative bounds directly. The nominal axial gap is 5 mm;
the summed bound is 0.009450162 mm. The certified lower gap must exceed 4.9 mm. This proves separation
of the entire solids without requiring unrelated area/centroid tolerance gates or a closest-distance reading.

```sh
go test ./models/pedalspindle -run '^TestPedalSpindleFootmanClearsInputShaft$' -count=1 -v -timeout 2m
```

The Decad session has the diagnosis; no core collision repair is needed for this pair.

## Scope and requested result

The pedal spindle in `decad-gallery` needs matching D-shaped shaft/gear engagements while preserving
round bearing journals. The user explicitly requested trying a combined shaft-cut tool, then contacting
a Decad session to repair the library if that attempt failed. Diagnose and fix the relevant Decad path
without loosening verification accuracy or substituting approximate coupling geometry.
Keep gallery changes in this worktree with the gallery session; perform dependency work in a Decad worktree.

Gallery worktree:
`/home/lestrrat/dev/src/github.com/lestrrat-3d/decad-gallery/.worktrees/feat-pedal-spindle`.
Current published gallery PR119 remains at `a3963644028bfd5299d9081f78a1f8ccfff279e2`.
The D-coupling changes are uncommitted candidates in that worktree.
The initial failures used `github.com/lestrrat-3d/decad v0.0.0-20261027054635-bacba52e1142`.

## Passing smallest real connection

The generated probe `.tmp/d-coupling-probe/probe_test.go` builds a real imported17-tooth Module3 gear,
without a bore. A native8.2mm circular tool minus a rectangular cap at X>3.1 cuts its D bore.
An8mm shaft has its own flat at X=3. Both final bodies pass VerifyAll at0.03mm chord tolerance.
Their real shared-plane sections clear in assembly and interfere after rotating only the shaft90degrees.
The passing log is `.tmp/d-coupling-probe.log`.

Run from the gallery worktree:

```sh
go test ./.tmp/d-coupling-probe -run '^TestRealGearDCoupling$' -count=1 -v -timeout 90s
```

## First full-assembly failure

The input shaft is diameter8mm and181mm long. In its local Z coordinates it needs flats at X=3
over Z[85,181] and[0,35], leaving a fully round journal between them.
Two successive native cuts fail at the second flat. Log: `.tmp/d-coupling-first.log`.

Decad reports a held mesh bound0.005456053372429413mm versus the next pair's chord tolerance
0.003650129320468598mm, and refuses the Boolean chain.
The first candidate is preserved in `.tmp/d-coupled-spindle-output/`.

## Combined-cut retry and current failure

The revised input shaft first unions two disjoint rectangular cap tools, then cuts the pristine cylinder once.
This builds the full48-part assembly with58 verified sketches, but the first default geometry check fails
while reading the input gear's D-bore section at0.005mm chord tolerance:

```text
decad: not supported by the current evaluator: requested tolerance 0.005 mm is below
the faceted body's minimum mesh bound 0.005438546085480087 mm
```

Log: `.tmp/single-cut-spindle-first.log`.
The staged construction and its identical emission candidate are in
`models/pedalspindle/{motion,sections,assembly,couplings}.go` and `.tmp/single-cut-spindle-output/`.
The failure is from `requireDBore` in `models/pedalspindle/couplings_test.go`, called by
`TestPedalSpindleDefault` through the consuming assembly's geometry checks.

```sh
go test ./models/pedalspindle -run '^TestPedalSpindleDefault$' -count=1 -v -timeout 3m
```

The source tool uses native verified circles and rectangles; its D bore is circle-minus-cap, then
gear-minus-D-tool. Check whether that path can retain analytic section provenance or use another supported
exact construction. A global Boolean tolerance reduction or weaker gallery test is not the requested repair.
Check current Decad main first; this gallery dependency is several days old.

## Integration and reply

Return the diagnosis, focused regression results, fix commit or PR, and a usable dependency version.
The gallery session will update both module files as needed, rerun the real consuming build, regenerate
the coupling close-up and animation, push PR119 and watch CI.
Reply to the originating Codex task `01a18a3c-0dc8-74b2-830b-de169f365c44` through session messaging.

## Exact-section integration retry

The Decad session added an exact D-profile path in `.worktrees/fix-d-shaft-coupling`.
Gallery's dimensional helpers now use Body.Section with analytic arc, chord and circle assertions at1e-6mm.
The0.01mm SectionOutline clearance and90degree relative-rotation checks are unchanged.
The real gallery consumer still refuses the first input-gear section:

```text
decad: not supported by the current evaluator: section needs an original certified smooth loft
or an exact two-body Boolean history
```

The failing log is `.tmp/d-coupling-integration-first.log`.
The temporary modfile replaces Decad with that worktree; production module files remain unchanged.

```sh
GOWORK=off go test -mod=mod -modfile=.tmp/d-coupling-integration.mod ./models/pedalspindle \
  -run '^TestPedalSpindleDefault$' -count=1 -v -timeout 3m
```

The section frame is centered on the shaft and aligned with its original phase, using the input angle
directly instead of recovering it through atan2. The failing gear is the real imported spur construction.
