# Spinning-wheel head placement failure

## Real consumer

Gallery worktree `.worktrees/feat-pedal-spindle`, uncommitted reference corrections after `71c2e50`.
The failing build used Decad `v0.0.0-20261030221427-96bc82a990de` in both modules.
The complete arrangement inventory is `docs/spinning-wheel-reference-audit.md`.
Geometry authority is `models/spinningwheel/instructions.md`; operation recipe is `steps.md` alongside it.

## First integration result

`GOWORK=off go test ./models/spinningwheel -run '^TestSpinningWheelNativeDefault$' -v -timeout 5m`
fails during the default Build before final-solid verification.
Log: `.tmp/spinningwheel-reference-native-default.log`.
Error: `spinning wheel flyer-drive-whorl: apply head tilt: decad: not supported by the current evaluator:
placed bore no longer has an exact shared axis`.

The whorl is one exact four-V-groove revolved body, cut by a matched-cap D16.2/flat6.1 tool.
The local cut succeeds and the first placement succeeds. A second common rigid transform for the hinged head fails.
Default HeadTilt is zero, so this failure happens even with a neutral hinge.
The generic route is `internal/revolvebore/payload.go`, Payload.Placed's exact shared-axis validation.

## Requested read-only diagnosis

Determine whether supported placement composition avoids this without changing physical geometry,
or whether generic Decad placement bookkeeping needs repair. Estimate the smallest repair and checks.
Do not modify Gallery source, relax the guard, substitute facets or start a new kernel repair without scope approval.
Gallery has stopped parameter expansion and image generation until its real default consumer passes.

## Read-only diagnosis

The Decad session confirmed that the inverse round trip in Payload.Placed can lose the exact shared-axis relation.
One supported construction composes final placement and head tilt with Transform.Then before one native Transform.
Its estimate is a few hours for Gallery construction changes and focused checks.
A generic Decad repair would preserve the relative source records through repeated placement without relaxing MapTool.
Its estimate is several hours to one day for the repair and repeated-placement regressions.
Continue the previously approved generic one-piece-whorl repair in the existing Decad session.
The user previously instructed contacting that session for generic fixes and explicitly approved the one-piece route.
This is a repeated-placement defect in that existing feature, not a request for a new topology class.
Preserve the exact shared-axis guard. If diagnosis requires a new geometry class, return scope and cost before expanding.

## Native repair merged

[Decad PR2932](https://github.com/lestrrat-3d/decad/pull/2932) merged as
`5622742903f0cb7a0b7e5f25539fe11c2e9d7742` after all required CI checks passed.
The native regression covers repeated placements without relaxing MapTool.
The real Gallery default passed against that repair with185bodies and204sketches before the final frame refinements.
Both modules now pin `v0.0.0-20261030233210-5622742903f0`.
The final frame uses stepped native Revolve bolts instead of touching-cylinder Unions.
The corrected default passes final-solid and sketch verification with181bodies and198sketches.
