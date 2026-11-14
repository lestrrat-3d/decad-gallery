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

Start with the smallest real case, using the generated candidate staged in
the actual package. If it fails at a Sketch or Decad boundary, stop adding
cases and hand off the blocker. Resume only when the gallery's real build
passes against the upstream fix. Once the first case passes, run the other
parameter cases and the consuming model checks.

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

A press-fit part and its socket, such as a carrier pin or a shaft pressed into a carrier plate, take 0.05 mm of clearance on each
side, and the 0.2 mm Tight minimum does not apply. The pin is a separate part that stays in the plate once pushed
in.
A gear bore that holds a shaft so the gear turns with it, such as a sun gear's D-shaped bore, is also a press fit and
takes 0.05 mm of radial clearance.

A bore or pocket that holds a purchased bearing fixed takes 0.05 mm of radial
clearance, and the 0.2 mm Tight minimum does not apply. The bearing is a nominal part and the seat grips its outer race.

A shaft or pin that passes through a purchased bearing's bore takes no clearance: make its diameter equal the bore
diameter. The bearing is a nominal part, so no fit class applies. Apply this to every model's bearing shafts and pins.
Do not shrink a shaft to leave a gap against the bearing's inner race. The shaft's other fits, such as a D-flat bore in a
gear or carrier, keep their own fit class.

Mark each model in [models/README.md](../models/) as **Adjustable**, **Nominal**,
or **Mixed**. Use **Adjustable** when the model's custom fits can be changed to
provide clearance. Use **Nominal** when the model preserves fixed standard or
purchased dimensions. Use **Mixed** when a model combines adjustable custom
fits with fixed nominal components.

### Arbor collars

This rule applies to every model in the gallery. It covers every collar on a printed shaft, arbor or pin, and every
printed shaft that carries another part, such as a gear, wheel, drum, lever or knob. The carried part may sit beside a
flush side (a plate face, a wall face, or the face of a bearing set flush in either), may be pressed on a shaft end
outside the frame, or may sit against a collar with no flush side nearby.

Put a collar on the shaft between the flush side and the carried part. A collar is a raised ring or shoulder of the
shaft's own body, wider than the carried part's bore. It holds the carried part clear of the flush side, so the part
does not rub it. A boss on the carried part's own body is not a collar and does not meet this rule.

A separate printed collar ring also counts as a collar. It is pressed on the shaft's flat, is located by a
counterbore floor, the end of a flat or a step of the shaft, and meets every clause of this rule. Use one where a
collar of the shaft's own body would stop a part or a bearing from being fitted, as on a shaft that runs in two
bearings and carries parts between them.

A shaft that runs in two bearings may use a step of its own body as the collar on the side its parts are fitted
from. The step is the end of the section the parts are pressed on, and it faces the bearing's inner ring with a
Tight gap. Its diameter clears the inner ring's bore chamfer by at least 0.3 mm on the round side. A part pressed on
the flat and stopped by a collar on its other side needs no further collar on the fitting side. Its face there
stands at least 1.0 mm from the flush side. A part that turns on the shaft, or slides on the flat without a press
fit, still takes a collar on both sides. List each such step in the collar table with its diameter and the
clearance of the part's face. The 3.0 mm minimum applies to the section the step ends, not to the step.

Make every collar at least 3.0 mm long along the shaft's axis, and measure its whole length, including any part inside
a counterbore. A collar about 1 mm long cannot be printed reliably. A collar that only stops a pressed part, with no
flush side beside it, takes the same minimum. A part pressed on a shaft end outside the frame takes a collar between
it and the frame's outer face.

Choose the first of these that fits:

1. **Collar in the gap.** Measure the space between the flush side and the carried part, less the collar's fit gap at
   each end. When that space is 3.0 mm or longer, put the whole collar in the gap.
2. **Pierce the carried part.** When that space is shorter, extend the collar into a counterbore in the carried part's
   face until the collar is 3.0 mm long or more. The counterbore is blind: its floor stops the part on the collar, and
   the collar never passes through the flush side. Keep 1 mm or more of the part's material around and under the
   counterbore. Fit the counterbore by how the part moves on the shaft, as [Part clearances](#part-clearances) sets out:
   - A part that turns with the shaft takes the press-fit clearance, radially and between the floor and the collar.
   - A part that turns on the shaft takes a Regular radial fit and a Tight gap between the floor and the collar.
3. **Change the assembly.** When neither fits, move the carried part along the shaft, or shorten or move a
   neighbouring part, until option 1 or 2 fits. For example, a collar beside a purchased bearing cannot grow into the
   bearing, and the bearing cannot change. When the carried part also has no room for a counterbore, the parts around
   it must move.

Do not make a collar shorter than 3.0 mm to fit a gap. Do not extend a collar into a purchased bearing, and do not
change a bearing's size to make room. A shaft through a bearing keeps the bore's diameter, as Part clearances requires.

A collar that faces a purchased bearing bears on the inner ring only: keep its diameter inside the inner ring's face,
with a Tight gap to the bearing face. A collar that stops a pressed part takes the press-fit gap to that part. A
collar beside a part that turns on the shaft takes a Tight gap to it.

Purchased collars are exempt from this rule.

In the model's `instructions.md`, list each collar: its shaft, the carried part, the flush side (if any), its axial
extent and length, its diameter, and which of the three options it uses, with the reason for option 3. Apply
[Sketch constraints](../AGENTS.md#sketch-constraints) to every collar and counterbore sketch, and
[Surface edge finishing](#surface-edge-finishing) to every counterbore rim.

Test each collar on the built shaft body for every parameter case that changes it. Find the collar's two end faces on
the body, and require the distance between them to be 3.0 mm or more. Require the collar's diameter to be the listed
one. Measure each counterbore fit, and the gap between each carried part and its flush side, against their fit classes.

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

### Corners and roots

This rule applies to every printed part in the gallery. Purchased and standard parts kept at their nominal size,
such as bolts, nuts and bearings, are exempt, as [Part clearances](#part-clearances) keeps them.

Read each part in its print orientation, as the model README states it. A section is the part cut by a plane
parallel to the print bed, at a height inside a run of layers that share one outline.

- **Inside corners.** Round every inside corner of a section with a radius of at least 0.5 mm. An inside corner is
  a point where the outline turns toward the material by more than 45°. A turn of 45° or less needs no round.
- **Stressed roots.** Some features bend or carry load in use: a spring strip, a flexure, a pawl, an arm, a lever,
  a crank, a spoke, a clip, a tab, or a ratchet or pawl tooth. Round the root of such a feature with a radius of at
  least half the thinner of the two sections that meet there, and at least 0.5 mm. When a slot is too narrow for
  two such radii, end it in a full round whose radius is half the slot width. When a fit clearance or a mating
  part's path leaves less room, use the largest radius that keeps the fit, at least 0.5 mm, and list the corner.
- **Root width.** Make the root of a feature at least as wide, before its rounds, as the feature's narrowest
  section. Make a part that bends in use at least 1.2 mm thick (three 0.4 mm lines). Make every other wall at
  least 0.8 mm thick (two lines).
  Make the wall round a press-fit bore or press-fit counterbore of a separate printed part at least 1.6 mm (four
  0.4 mm lines) over its whole pressed length, and print such a ring with its axis vertical.
- **Tips.** Keep the material angle at every outward corner of a section at 60° or more. Blunt a sharper tip
  with a flat at least 0.8 mm wide or a round of at least 0.4 mm radius.
- **Corners between layers.** An inside corner along an edge parallel to the bed, such as the root of a pin
  standing on a face, takes the same radius when the corner opens upward. When it opens downward, cut it with a
  45° chamfer of the same leg length instead, because a round there starts as a flat overhang.

These corners are exempt:

- Gear teeth: the flanks, tips and root fillets of involute and cycloidal teeth follow their generators and
  [Gear train clearances](#gear-train-clearances). The rest of a gear body, such as its hub, web, windows and
  bore keys, follows this rule.
- Engagement edges: a corner whose shape is the working contact, such as a pallet's locking corner, an escape
  wheel tooth tip, or the contact corner of a pawl or ratchet tooth. The root of such a tooth is not exempt.
- Mating corners: a D-bore, keyway or joinery corner that must match a corner of its mate. Round both mates, or
  keep both sharp and list the pair.
- Chamfer faces that [Surface edge finishing](#surface-edge-finishing) adds.

List every exempt corner, and every root rounded below half its section, under "Retained sharp corners" in the
model's `instructions.md`, with its location and reason.

Test every printed part with `internal/cornercheck`. Cut a section at the middle of each run of layers
(`cornercheck.MidHeights`) and require no finding outside the listed corners. Each listed corner must match a
finding, so the list stays current. The check reads an inside corner as a turn toward the material of more than
50° within 0.4 mm of outline; a 0.5 mm round turns 46° over 0.4 mm and passes. It reads a tip as a turn away from
the material of more than 120° within 0.8 mm; a 0.4 mm round tip turns 115° over 0.8 mm and passes. Check every
parameter case that changes a part's outline. Where the face levels are not the runs of one outline, such as the
apex plane of a herringbone gear, pass explicit heights to `cornercheck.CheckAt` and say why in a comment.

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

### Exploded views

The exploded view is the assembly manual and a visual check of the part list, so
a reader must be able to tell every part apart in it. This rule applies to the
exploded image of every multi-body model.

- Render the image at 1600 × 900 pixels or larger.
- Show it in the model `README.md` at the full width of the page, with no
  `width` attribute. Never show it narrower than 800 px.
- Show one exploded image per row. Do not put two exploded images side by side
  in a table.
- Fill the frame: the exploded parts span at least 80% of the image width or
  height. Crop empty margin by moving the camera, not by shrinking the offsets.
- Make every part visible. Offset parts far enough along the explode axis that
  none hides another. A part that still hides behind another gets its own
  close-up or second angle in the README, not a sentence saying it is hidden.
- Give the image alt text that lists every part in assembly order.

## Spur gear example

[`models/spurgear/instructions.md`](../models/spurgear/instructions.md) is
the construction contract. Its [steps](../models/spurgear/steps.md) extrude
one tooth and a root disc, repeat the tooth with a circular pattern, and join
the disc and all teeth in one union. The root fillet, bore, and cap chamfer
follow. A gear whose flanks start inside the root circle uses the same steps
with a tooth closed by an arc inside the root disc. `spurgear.Build` is the
reusable Go entry point, and the package tests exercise the resulting bodies.
