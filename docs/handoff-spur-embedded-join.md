# Embedded spur finishing regression after Decad update

## Scope

Restore the existing embedded spur builds in `decad-gallery` after the Decad
dependency update needed by spindle coupling verification. Diagnose the native
join and finishing boundary; do not loosen gallery assertions or replace the
specified native tooth/root-disc construction.

The user authorized contacting a Decad session for dependency repairs during
the spindle work. Keep gallery changes in this worktree and native changes in
a Decad worktree. Use local fixtures and repository tests for proof.

## Real consumer

Gallery worktree:
`/home/lestrrat/dev/src/github.com/lestrrat-3d/decad-gallery/.worktrees/feat-pedal-spindle`.

The failing dependency was published Decad `c5375fa37f6866b5a641b8679d4103422e979092`
and Sketch `6dfdb462bc53`. The last passing gallery CI before this dependency
update used Decad `bacba52e1142` at gallery commit `a396364` (run `41003826769`).
The gallery session owns spindle PR119; do not push there from Decad.

The spur recipe's stale `BuildProfile` sentence was corrected with user
approval. Fresh emission also required declaring its existing handwritten
Variant/run interfaces. The candidate used to reproduce these failures was staged unchanged from
`.tmp/modelgen/spurgear/20261030T074325174406Z/output/`.
It was not accepted into the generation record.

## Confirmed failures on the updated dependency

The smallest real eight-tooth gear builds and passes verified tessellation.
The real default spindle and all six positive-coupling checks pass with the
new spur candidate. Both Go module lint runs pass.

The affected spur package has three failing embedded cases:

| Case | Parameters | Failure |
| --- | --- | --- |
| Placed embedded | Module 0.5, 60 teeth, pressure angle 20 degrees, thickness 2, 15 samples, bore 3, chamfer 0.02; frame origin (10,-5,7), U=+Y, V=+Z | Root fillet expects 120 concave axial edges; parallel_to(+X) matches none. |
| Embedded without fillet | Module 0.5, 30 teeth, pressure angle 30 degrees, thickness 2, 5 samples, chamfer 0.02 | Neither cap outer-loop selector matches after the join. |
| Embedded with fillet | Module 0.5, 24 teeth, pressure angle 30 degrees, thickness 2, 5 samples | Root fillet expects 48 concave axial edges; parallel_to(+Z) matches none. |

Log: `.tmp/variant-spur-focused.log`.

The construction extrudes a verified native root disc and a verified natural
tooth with two fit-spline flanks and two circular arcs. It patterns that tooth
about the gear frame and unions the disc and all teeth once, then selects the
native axial root edges or outer cap loops for finishing. The absent selectors
suggest the join did not retain the required analytic topology; inspect the
actual producer before treating that as the diagnosis.

## Baseline control

The 60-tooth placed case fails identically with the previously checked-in
three-file spur implementation and the updated dependency. This rules out a
failure caused only by the fresh emission.

The original files are saved in `.tmp/prior-{involute,spurgear,tooth-section}.go`.
The Go overlay `.tmp/prior-spur-overlay.json` maps the three current source
paths to those unchanged originals. Its log is `.tmp/prior-spur-placement.log`.

```sh
go test -overlay .tmp/prior-spur-overlay.json ./models/spurgear \
  -run '^TestPlacedEmbeddedGearWithRootFilletsAndBore$' -count=1 -v -timeout 2m
```

Current candidate reproduction:

```sh
go test ./models/spurgear \
  -run '^TestPlacedEmbeddedGearWithRootFilletsAndBore$' -count=1 -v -timeout 2m
```

## Requested result

Identify the first native operation that loses the topology needed by these
existing selectors. Add a focused generic regression and repair the relevant
native path. Start with the real placed-gear case above; do not expand before
that boundary passes. Return the diagnosis, native regression results, PR and
usable dependency version. Keep the fix independent of gallery names and
specific gear dimensions.

The gallery session will update dependencies, finish fresh emission and its
generation record, rerun focused consumers, render the actual bodies, and
publish PR119 only after local checks pass. Reply through session messaging to
task `01a18a3c-0dc8-74b2-830b-de169f365c44`.

## Merged native repair

The Decad session diagnosed dispatch order in the multi-operand Union. The
generic prism fold returned a faceted body before the exact freeform axial
section join could run. Moving the certified route before that fallback
preserves the axial root edges and cap-loop selectors.

The native candidate in `.worktrees/fix-embedded-spur-join` passes all three
gallery failures, plus the real default spindle and six positive-coupling
checks. The gallery used `.tmp/embedded-join-integration.mod` with a temporary
local replacement; production module files have no local replacement.
Logs are `.tmp/embedded-join-first-consumer.log`,
`.tmp/embedded-join-other-consumers.log`, and
`.tmp/embedded-join-spindle-consumer.log`.
Decad PR2887 merged at `168c95c00eb48bf6fe32d60c0cb8410e965571af` after all required CI checks passed.
The production gallery modules pin its exact tested head,
`dfe706cff5aa3f809ac3081f4d69c0be30c842ba`, published as
`v0.0.0-20261030081324-dfe706cff5aa`. This includes the generic line-and-arc through-bore repair from PR2884.

The final three-file emission uses bundle `20261030T081559849296Z` with that published dependency.
The complete affected spur package and real default spindle/coupling checks pass.
The generation record was accepted only after those checks; both Go module lint runs and
the generation-record check pass. The actual spur and spindle images were rebuilt with the renderer.
