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

### Body containers

This rule applies to every model that builds more than one body, across the whole
gallery. Put similar bodies in one `decad.Container` whose name says what they
are, such as `Planet Gears`, `Bolts`, `Nuts`, or `Housing`. Create the container
with `NewChild` on the document root or on a parent container.

- Group bodies of the same kind: the same part reused, or parts with the same role.
- Give a container one plural name per kind. Do not name containers `Group 1` or
  after construction steps.
- Keep a single body in a container only when it is the one member of a named
  kind, such as `Housing`; nest kinds that belong to one unit, as a pair holds a
  driving and a driven gear.
- Sibling names are unique, and `NewChild` rejects a duplicate. Use
  `Container.ChildNamed` for one container shared by many bodies of a kind
  (all planets in `Planet Gears`). Use `Container.NewUniqueChild` for a unit that
  can repeat (`Pair`, `Pair 2`). Do not write a per-model helper.
- Move each body into its container with `Container.AddBody`.
- Test each multi-body model: assert the container names and that every body
  sits in the container for its kind, for each parameter branch that changes the
  body set. Require `doc.VerifyContainers().Err()` to be nil, which fails on a
  body left directly in the root and on a container with no body in or below it.

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

A press-fit peg and its socket, such as a carrier pin in a carrier plate, take 0.05 mm of clearance on each
side, and the 0.2 mm Tight minimum does not apply. The pin is a separate part that stays in the plate once pushed
in.

A bore or pocket that holds a purchased bearing fixed takes 0.05 mm of radial
clearance, and the 0.2 mm Tight minimum does not apply. The bearing is a nominal part and the seat grips its outer race. A
shaft or pin that turns in a bearing bore keeps its fit class.

Mark each model in [models/README.md](../models/) as **Adjustable**, **Nominal**,
or **Mixed**. Use **Adjustable** when the model's custom fits can be changed to
provide clearance. Use **Nominal** when the model preserves fixed standard or
purchased dimensions. Use **Mixed** when a model combines adjustable custom
fits with fixed nominal components.

Start with the smallest real case, using the generated candidate staged in
the actual package. If it fails at a Sketch or Decad boundary, stop adding
cases and hand off the blocker. Resume only when the gallery's real build
passes against the upstream fix. Once the first case passes, run the other
parameter cases and the consuming model checks.

### Surface edge finishing

Chamfer every qualifying convex boundary edge on a large planar surface:

- A planar surface is large when its net area is at least 400 mm². Combine
  connected coplanar faces when measuring area.
- An edge run is long when its arc length is at least 20 mm. Combine tangent-
  continuous boundary pieces when measuring length, and include circular rims.
- Measure the surface and edge runs before adding any edge finishing. Ignore
  internal seams between coplanar faces.
- Use a 0.5 mm equal-setback chamfer, producing a 45° chamfer where the faces
  meet at a right angle.
- Document and test any smaller chamfer or retained sharp edge required by
  functional geometry, nominal dimensions, fit clearances, or minimum material.

### Gear train clearances

These rules apply to spur and helical gears that share a plane and have
parallel axes. A helical gear uses its normal module in place of `m`.

For two meshing gears, set the center distance to half the sum of the pitch
diameters. The tip circle of each gear must then clear the root circle of its
mate by at least `max(0.25·m, 0.5 mm)`. A standard tooth with a `1.25·m`
dedendum gives exactly `0.25·m`, so a gear of module 1 needs a deeper root to
reach 0.5 mm. Deepen the root; do not widen the center distance, which also
adds backlash.

Two gears in the same plane that do not mesh keep their tip circles at least
0.5 mm apart, the Loose fit in [Part clearances](#part-clearances).

Set backlash with the gear's backlash parameter. Backlash does not count
toward the tip-to-root clearance.

Measure the tip-to-root clearance on the placed gears at several rotation
angles across one tooth pitch, for every module and tooth-count case the model
builds.

### Look-alike part marks

This rule applies to every multi-body model whose `instructions.md` has a
"Look-alike parts" section. A model without that section needs no marks.

- List each set of parts a person could mix up, with the reason, under "Look-alike
  parts" in `instructions.md`. The author decides what looks alike.
- Cut 1 to 6 round through holes in every part of a set. Give different parts of
  one set different hole counts, and give identical copies of one part the same
  count. Split a set that needs more than 6 counts into smaller sets.
- Make each hole 1.5 mm across. Space the holes 25° apart on one circle about the
  part's axis, away from any flat or keyway. Keep at least 1 mm of material between
  a hole and every edge of the face, bore or pocket.
- Place the circle on a face region that touches no mating part, and cut the
  holes through the whole part.
- Build each hole from its own sketch on the part's axis frame, with the centre
  placed by horizontal and vertical distances from the sketch origin. Apply
  [Sketch constraints](../AGENTS.md#sketch-constraints).
- Do not mark fasteners, bearings, other bought parts, or a part with no round face.
- Show each part's hole count in the model `README.md` parts table, with a
  one-line legend that says what the counts mean.
- Test each declared set: no two different parts share a count, and each hole
  passes through its part.

## Spur gear example

[`models/spurgear/instructions.md`](../models/spurgear/instructions.md) is
the construction contract. Its [steps](../models/spurgear/steps.md) map both
ordinary involute flanks through a root disc, one lofted tooth per angle, a
union, root fillet, bore, and cap chamfer, and embedded flanks through one
lofted outline of all teeth. `spurgear.Build` is the
reusable Go entry point, and the package tests exercise the resulting bodies.
