# Model build pipeline

Each model follows the same path from a human-readable construction contract
to reusable Decad Go code. A coding agent or contributor performs the
translation; the checked-in artifacts and geometry tests make the result
reviewable. The pipeline ends at a Decad body. Fusion add-ins consume the Go
construction downstream.

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
incorrect, record the smallest real model case and use the Decad/Sketch
handoff in [AGENTS.md](../AGENTS.md). Do not replace the required geometry
with a faceted or otherwise approximate substitute.

The steps document is the short implementation recipe, not a second source
of dimensions. Refer back to `instructions.md` for formulas and input rules.

## 3. Build a reusable Go package

Implement `models/<name>` as a Go package with a parameter type and a
`Build(ctx, doc, params)` entry point. Keep the construction reusable by
other model packages. Return the Decad body and a contextual error when a
Sketch or Decad operation fails. Put the model's README, instructions, steps,
code, and focused geometry tests in the same directory.

## 4. Verify the real model

Run `go test ./models/<name>` for the affected model. Test the default and
every distinct construction branch with real Sketch profiles and Decad
bodies. Apply [Sketch constraints](../AGENTS.md#sketch-constraints):
build code calls `modelsketch.Verify` on each Sketch, and tests assert it
with `sketchassert.FullyConstrained`. Check
topology, dimensions, and the intended feature surfaces. Use `Tessellate`
with `VerifyAll` for complex final bodies. Check a shared
model again through its consuming assembly. CI runs `go test ./...` and the
repository layout check after a push.

The first end-to-end case should be the smallest real case. If it fails at a
Sketch or Decad boundary, stop expanding that path and hand off the blocker
before adding more cases. Resume only when the real gallery build passes
against the upstream fix.

## Spur gear example

[`models/spurgear/instructions.md`](../models/spurgear/instructions.md) is
the construction contract. Its [steps](../models/spurgear/steps.md) map both
ordinary and embedded involute flanks through the Sketch profile, loft,
root fillet, bore, and cap chamfer operations. `spurgear.Build` is the
reusable Go entry point, and the package tests exercise the resulting bodies.
