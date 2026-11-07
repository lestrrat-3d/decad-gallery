# Model build pipeline

Each model follows the same path from a human-readable construction contract
to reusable Decad Go code. A coding agent compiles the contract into Decad
steps, then a fresh agent emits Go from those checked steps. Geometry tests
check the staged code against the contract. The pipeline ends at a Decad body.
Fusion add-ins consume the Go construction downstream.

## 1. Write the construction contract

Create `models/<name>/instructions.md`. State the inputs and defaults, units,
derived values, geometric construction, feature order, valid parameter range,
and representative cases. Describe the resulting geometry rather than Fusion
API calls. Include independent conditions that can be exercised separately,
such as a bore and a cap chamfer.

## 2. Compile the contract into Decad steps

Create `models/<name>/steps.md` before writing the model code. Map each
construction requirement to a Sketch or Decad operation, name the intermediate
profile or body, and state the geometry check at each boundary. Keep the
sampled curves and exact operations visible. If an operation is missing or
incorrect, record the smallest real model case. Do not replace the required
geometry with a faceted or otherwise approximate substitute.

Make the steps complete enough for an emitter that cannot inspect an existing
implementation. State the exact operations, dimensions, constraints, profile
selection, feature order, exported API, and failure checks. Refer to
`instructions.md` for the governing geometry rule; copy a formula into steps
when the emitter needs it to implement that operation.

## 3. Emit Go from the checked steps

Give a fresh agent the instructions, steps, declared dependency contracts, and
one fixed emission prompt. Do not give it the existing model Go source. It
writes a candidate in `.tmp/`; the candidate is then copied into the model
package without hand edits. If the emitter finds a missing construction fact,
correct the instructions or steps and emit again. Do not add a gear-specific
hint to the one-off prompt or repair only the generated Go.

For spur, run `python3 scripts/modelgen/spurgear.py prepare`. Give the fresh
agent the printed bundle path and its `inputs/scripts/modelgen/spurgear_prompt.md`.
Copy the three generated Go files from `output/` to `models/spurgear/`. The
bundle includes only instructions, steps, the fixed prompt, shared constraint
helpers, and module versions. Run
`python3 scripts/modelgen/spurgear.py record <bundle>` only after the staged
files pass the focused checks. The record
contains hashes of all emission inputs and the staged Go files. Run
`python3 scripts/modelgen/spurgear.py check` to detect any later drift.

The emitted package must expose `Build(ctx, doc, params)` and any model API
named by its steps. Return contextual errors when Sketch or Decad fails.
Keep the model's README, instructions, steps, code, and focused geometry
tests in the same directory.

## 4. Verify the real model

Run `go test ./models/<name>` for the affected model. Test the default and
every distinct construction branch with real Sketch profiles and Decad
bodies. Apply [Sketch constraints](../AGENTS.md#sketch-constraints). Check
topology, dimensions, and the intended feature surfaces. Use `Tessellate`
with `VerifyAll` for complex final bodies. Check a shared
model again through its consuming assembly. CI runs `go test ./...` and the
repository layout check after a push.

### Part clearances

Choose the clearance for each distinct part fit by its intended feel:

| Fit | Intended feel | Minimum surface-to-surface clearance |
|---|---|---:|
| Tight | Not intended to be removable | 0.2 mm |
| Regular | Some leeway, but not loose | 0.35 mm |
| Loose | Loose fit | 0.5 mm |

Do not use zero-clearance or overlapping fits between separate parts. Measure
the placed parts in the assembled geometry and require the proven lower bound
to meet the selected minimum. Include geometry and measurement uncertainty in
that bound. Check every parameter case that changes the fit.

Keep purchased and standard components, such as bolts, nuts, and bearings, at
their nominal dimensions. Put the selected clearance in custom geometry that
mates with a fixed nominal part. When two fixed nominal parts mate, record the
interface as a nominal exception; do not alter either part or claim that it
meets a clearance class.

Mark each model in [models/README.md](../models/) as **Adjustable**, **Nominal
only**, or **Mixed**. Use **Adjustable** when the model's custom fits can be
changed to provide clearance. Use **Nominal only** when the model preserves
fixed standard or purchased dimensions. Use **Mixed** when a model combines
adjustable custom fits with fixed nominal components.

Start with the smallest real case, using the generated candidate staged in
the actual package. If it fails at a Sketch or Decad boundary, stop adding
cases and hand off the blocker. Resume only when the gallery's real build
passes against the upstream fix. Once the first case passes, run the other
parameter cases and the consuming model checks.

## Spur gear example

[`models/spurgear/instructions.md`](../models/spurgear/instructions.md) is
the construction contract. Its [steps](../models/spurgear/steps.md) map both
ordinary involute flanks through a root disc, one lofted tooth per angle, a
union, root fillet, bore, and cap chamfer, and embedded flanks through one
lofted outline of all teeth. `spurgear.Build` is the
reusable Go entry point, and the package tests exercise the resulting bodies.
